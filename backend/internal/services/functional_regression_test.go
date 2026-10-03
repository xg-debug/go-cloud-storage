package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"go-cloud-storage/backend/internal/models"
	"go-cloud-storage/backend/internal/repositories"
	"go-cloud-storage/backend/pkg/utils"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func mockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: conn, SkipInitializeWithVersion: true}), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true}, Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})
	return db, mock
}

func expectPurgeLock(mock sqlmock.Sqlmock) {
	mock.ExpectQuery("SELECT DISTINCT.*user_id.*FROM.*file").WithArgs("f").WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(1))
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT.*FROM.*user.*FOR UPDATE").WithArgs(1).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
}

func TestExpiredPurgeRechecksCurrentRecycleState(t *testing.T) {
	db, mock := mockDB(t)
	expectPurgeLock(mock)
	// A restored file or a new recycle entry whose expiry is in the future
	// must not reach descendant expansion, metadata deletion, or object storage.
	mock.ExpectQuery("SELECT.*file_id.*is_deleted.*expire_at <=.*").
		WithArgs("f", true, sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"file_id"}))
	mock.ExpectCommit()
	s := &recyclePurgeService{db: db}
	if err := s.PurgeExpired(context.Background(), []string{"f"}); err != nil {
		t.Fatal(err)
	}
}

func TestManualPurgeAlsoSkipsRestoredFile(t *testing.T) {
	db, mock := mockDB(t)
	expectPurgeLock(mock)
	mock.ExpectQuery("SELECT.*file_id.*is_deleted").WithArgs("f", true).
		WillReturnRows(sqlmock.NewRows([]string{"file_id"}))
	mock.ExpectCommit()
	if err := (&recyclePurgeService{db: db}).PurgeFiles(context.Background(), []string{"f"}); err != nil {
		t.Fatal(err)
	}
}

func TestPurgeRefusesLiveDescendant(t *testing.T) {
	db, mock := mockDB(t)
	expectPurgeLock(mock)
	mock.ExpectQuery("SELECT.*file_id.*expire_at <=.*").WithArgs("f", true, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"file_id"}).AddRow("f"))
	mock.ExpectQuery("WITH RECURSIVE descendants").WithArgs("f").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("f").AddRow("child"))
	mock.ExpectQuery("SELECT.*FROM.*file.*FOR UPDATE").WithArgs("f", "child").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "is_dir", "is_deleted"}).
			AddRow("f", 1, true, true).AddRow("child", 1, false, false))
	mock.ExpectRollback()
	if err := (&recyclePurgeService{db: db}).PurgeExpired(context.Background(), []string{"f"}); err == nil {
		t.Fatal("live descendant must prevent deletion")
	}
}

type recycleRepoStub struct{ repositories.RecycleRepository }

func (recycleRepoStub) GetExpiredFileIds(int) ([]string, error) { return []string{"expired"}, nil }

type purgeStub struct {
	RecyclePurgeService
	calls int
}

func (s *purgeStub) PurgeExpired(_ context.Context, ids []string) error { s.calls++; return nil }

func TestCleanupWithoutRabbitMQUsesExpiryCheckedPurge(t *testing.T) {
	purge := &purgeStub{}
	service := NewRecycleService(nil, recycleRepoStub{}, nil, purge, nil)
	n, err := service.DispatchExpiredPurgeJobs(context.Background(), 0)
	if err != nil || n != 1 || purge.calls != 1 {
		t.Fatalf("n=%d calls=%d err=%v", n, purge.calls, err)
	}
}

type userRepoStub struct {
	repositories.UserRepository
	user  *models.User
	reset *models.PasswordResetToken
}

func (r userRepoStub) GetUserInfoById(int) (*models.User, error) { return r.user, nil }
func (r userRepoStub) GetPasswordResetToken(string) (*models.PasswordResetToken, error) {
	return r.reset, nil
}

func TestPasswordVersionRevokesAllSessionsWithoutRedis(t *testing.T) {
	utils.InitJWTSecret("test-secret-at-least-thirty-two-bytes-long")
	user := &models.User{Id: 1, Password: "old-bcrypt-hash"}
	s := &userService{userRepo: userRepoStub{user: user}}
	version := utils.CredentialVersion(user.Password)
	if err := s.ValidateSession(1, version); err != nil {
		t.Fatal(err)
	}
	user.Password = "new-bcrypt-hash"
	if err := s.ValidateSession(1, version); err == nil {
		t.Fatal("old session accepted")
	}
	if err := s.ValidateSession(1, ""); err == nil {
		t.Fatal("legacy unversioned session accepted")
	}
	if err := s.ValidateSession(1, utils.CredentialVersion(user.Password)); err != nil {
		t.Fatal(err)
	}
}

func TestResetPasswordAtomicallyConsumesToken(t *testing.T) {
	for _, stale := range []bool{false, true} {
		t.Run(map[bool]string{false: "rollback-on-password-write-failure", true: "reject-reused-token"}[stale], func(t *testing.T) {
			db, mock := mockDB(t)
			s := &userService{db: db, userRepo: userRepoStub{reset: &models.PasswordResetToken{Id: 7, UserId: 1, ExpiresAt: time.Now().Add(time.Hour)}}}
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT.*FROM.*user.*FOR UPDATE").WithArgs(1, 1).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
			affected := int64(1)
			if stale {
				affected = 0
			}
			mock.ExpectExec("UPDATE.*password_reset_token.*SET.*used.*expires_at >").
				WithArgs(true, 7, false, sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, affected))
			if !stale {
				mock.ExpectExec("UPDATE.*user.*SET.*password").WithArgs(sqlmock.AnyArg(), 1).WillReturnError(errors.New("database write failed"))
			}
			mock.ExpectRollback()
			if err := s.ResetPassword("token", "NewPassword123"); err == nil {
				t.Fatal("expected atomic failure")
			}
		})
	}
}

func TestResetTokenFitsPersistedColumn(t *testing.T) {
	token, err := utils.GenerateResetToken(1, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(token) != 64 {
		t.Fatalf("reset token length = %d, want 64", len(token))
	}
}
