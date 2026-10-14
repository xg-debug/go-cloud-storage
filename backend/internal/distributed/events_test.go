package distributed

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"go-cloud-storage/backend/infrastructure/eventing"
	"go-cloud-storage/backend/internal/ports"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func TestDelayedDeleteEventOnlyRemovesOlderShares(t *testing.T) {
	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: conn, SkipInitializeWithVersion: true}), &gorm.Config{NamingStrategy: schema.NamingStrategy{SingularTable: true}})
	if err != nil {
		t.Fatal(err)
	}
	event := ports.Event{ID: "delete-v2", Type: "file.deleted.v1", Data: map[string]int64{"file-1": 2}}
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `event_inbox`").WillReturnResult(sqlmock.NewResult(0, 1))
	// A newly shared restored file has version 2 and must survive this event.
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `share` WHERE file_id = ? AND file_version < ?")).WithArgs("file-1", int64(2)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := ShareEventHandler(db)(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	// Redelivery must not perform a second business mutation.
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `event_inbox`").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()
	if err := ShareEventHandler(db)(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMalformedEventPayloadsArePermanentFailures(t *testing.T) {
	ctx := context.Background()
	for _, data := range []any{map[string]int64{"file": -1}, map[string]int64{"": 1}, "invalid"} {
		if err := ShareEventHandler(nil)(ctx, ports.Event{Type: "file.deleted.v1", Data: data}); !errors.Is(err, eventing.ErrInvalidEvent) {
			t.Fatalf("unexpected result: %v", err)
		}
	}
	if err := NotificationEventHandler(nil, nil)(ctx, ports.Event{Type: "file.uploaded.v1", UserID: 1, Data: map[string]string{}}); !errors.Is(err, eventing.ErrInvalidEvent) {
		t.Fatalf("unexpected result: %v", err)
	}
}
