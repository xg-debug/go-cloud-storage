package application

import (
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"go-cloud-storage/backend/pkg/utils"
	repository "go-cloud-storage/backend/services/user-service/internal/adapters/repository"
	domain "go-cloud-storage/backend/services/user-service/internal/domain"
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

type userRepoStub struct {
	repository.UserRepository
	user  *domain.User
	reset *domain.PasswordResetToken
}

func (r userRepoStub) GetUserInfoById(int) (*domain.User, error) { return r.user, nil }
func (r userRepoStub) GetPasswordResetToken(string) (*domain.PasswordResetToken, error) {
	return r.reset, nil
}

func TestPasswordVersionRevokesAllSessionsWithoutRedis(t *testing.T) {
	utils.InitJWTSecret("test-secret-at-least-thirty-two-bytes-long")
	user := &domain.User{Id: 1, Password: "old-bcrypt-hash"}
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
			s := &userService{db: db, userRepo: userRepoStub{reset: &domain.PasswordResetToken{Id: 7, UserId: 1, ExpiresAt: time.Now().Add(time.Hour)}}}
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
