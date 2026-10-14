package eventing

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"go-cloud-storage/backend/infrastructure/persistence"
	"go-cloud-storage/backend/internal/ports"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func database(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: conn, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
		conn.Close()
	})
	return db, mock
}
func TestOutboxFailureRollsBackBusinessTransaction(t *testing.T) {
	db, mock := database(t)
	failure := errors.New("outbox unavailable")
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE file").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO `event_outbox`").WillReturnError(failure)
	mock.ExpectRollback()
	publisher := &Publisher{DB: db}
	err := persistence.NewGormTransactionManager(db).WithinTransaction(context.Background(), func(ctx context.Context) error {
		if err := persistence.TransactionDB(ctx, db).Exec("UPDATE file SET size = 1 WHERE id = 'test'").Error; err != nil {
			return err
		}
		return publisher.Publish(ctx, ports.Event{ID: "event-1", Type: "file.uploaded.v1"})
	})
	if !errors.Is(err, failure) {
		t.Fatalf("expected atomic rollback, got %v", err)
	}
}
func TestInboxDuplicateSkipsMutationAndFailureRollsBackMarker(t *testing.T) {
	for _, test := range []struct {
		name     string
		affected int64
		failure  bool
		called   bool
	}{{"new", 1, false, true}, {"duplicate", 0, false, false}, {"retryable", 1, true, true}} {
		t.Run(test.name, func(t *testing.T) {
			db, mock := database(t)
			mock.ExpectBegin()
			mock.ExpectExec("INSERT INTO `event_inbox`").WillReturnResult(sqlmock.NewResult(0, test.affected))
			if test.failure {
				mock.ExpectRollback()
			} else {
				mock.ExpectCommit()
			}
			called := false
			err := Apply(db, "event-1", func(tx *gorm.DB) error {
				called = true
				if test.failure {
					return errors.New("business write failed")
				}
				return nil
			})
			if called != test.called || (err != nil) != test.failure {
				t.Fatalf("called=%v err=%v", called, err)
			}
		})
	}
}
