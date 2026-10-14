package application

import (
	"context"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"gorm.io/gorm"

	ports "go-cloud-storage/backend/pkg/ports"
	repository "go-cloud-storage/backend/services/file-service/internal/adapters/repository"
)

type cleanupStorage struct {
	ports.Storage
	err   error
	calls int
}

func (s *cleanupStorage) DeleteObjectWithThumbnail(context.Context, string) error {
	s.calls++
	return s.err
}
func TestCleanupRetainsJobOnStorageFailureAndRetries(t *testing.T) {
	db, mock := mockDB(t)
	storage := &cleanupStorage{err: errors.New("offline")}
	for attempt := 0; attempt < 2; attempt++ {
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT.*object_deletion.*FOR UPDATE SKIP LOCKED").WillReturnRows(sqlmock.NewRows([]string{"id", "object_key", "attempts", "next_attempt_at"}).AddRow("job", "key", attempt, time.Now()))
		mock.ExpectQuery("SELECT count.*file.*oss_object_key").WithArgs("key").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		if attempt == 0 {
			mock.ExpectExec("UPDATE.*object_deletion").WillReturnResult(sqlmock.NewResult(0, 1))
		} else {
			storage.err = nil
			mock.ExpectExec("DELETE FROM.*object_deletion").WithArgs("job").WillReturnResult(sqlmock.NewResult(0, 1))
		}
		mock.ExpectCommit()
		done, err := ProcessObjectDeletion(context.Background(), db, storage)
		if err != nil || !done {
			t.Fatalf("done=%v err=%v", done, err)
		}
	}
	if storage.calls != 2 {
		t.Fatal(storage.calls)
	}
}
func TestCleanupPreservesReferencedObject(t *testing.T) {
	db, mock := mockDB(t)
	storage := &cleanupStorage{}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT.*object_deletion.*FOR UPDATE SKIP LOCKED").WillReturnRows(sqlmock.NewRows([]string{"id", "object_key"}).AddRow("job", "key"))
	mock.ExpectQuery("SELECT count.*file.*oss_object_key").WithArgs("key").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectExec("DELETE FROM.*object_deletion").WithArgs("job").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	_, err := ProcessObjectDeletion(context.Background(), db, storage)
	if err != nil || storage.calls != 0 {
		t.Fatalf("calls=%d err=%v", storage.calls, err)
	}
}

func TestPurgeRollsBackWhenCleanupJobCannotBeSaved(t *testing.T) {
	db, mock := mockDB(t)
	expectPurgeLock(mock)
	mock.ExpectQuery("SELECT.*file_id.*is_deleted").WillReturnRows(sqlmock.NewRows([]string{"file_id"}).AddRow("f"))
	mock.ExpectQuery("WITH RECURSIVE descendants").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("f"))
	mock.ExpectQuery("SELECT.*file.*FOR UPDATE").WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "is_deleted", "size", "oss_object_key"}).AddRow("f", 1, true, 4, "key"))
	mock.ExpectQuery("SELECT.*oss_object_key.*GROUP BY").WillReturnRows(sqlmock.NewRows([]string{"oss_object_key"}))
	for _, table := range []string{"recycle_bin", "favorite", "file"} {
		mock.ExpectExec("DELETE FROM.*" + table).WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectExec("UPDATE.*storage_quota").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO.*object_deletion").WillReturnError(errors.New("disk full"))
	mock.ExpectRollback()
	storage := &cleanupStorage{}
	s := NewRecyclePurgeService(db, storage, repository.NewRecycleRepository(db), repository.NewFileRepository(db), shareInvalidationStub{}, repository.NewFavoriteRepository(db), repository.NewStorageQuotaRepository(db))
	if err := s.PurgeFiles(context.Background(), []string{"f"}); err == nil {
		t.Fatal("must roll back")
	}
	if storage.calls != 0 {
		t.Fatal("uncommitted deletion reached storage")
	}
}

type shareInvalidationStub struct{ ShareIndex }

func (shareInvalidationStub) DeleteBatch(*gorm.DB, []string) error { return nil }
