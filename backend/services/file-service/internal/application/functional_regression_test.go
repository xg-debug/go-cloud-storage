package application

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	repository "go-cloud-storage/backend/services/file-service/internal/adapters/repository"
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
	mock.ExpectQuery("SELECT.*FROM.*storage_quota.*FOR UPDATE").WithArgs(1).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(1))
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

type recycleRepoStub struct{ repository.RecycleRepository }

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
