package services

import (
	"context"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"go-cloud-storage/backend/internal/models"
	"go-cloud-storage/backend/internal/ports"
	"go-cloud-storage/backend/internal/repositories"
	"testing"
	"time"
)

type shareListRepo struct {
	repositories.ShareRepository
	shares []*models.Share
}

func (r shareListRepo) GetUserShares(int, int, int) ([]*models.Share, int64, error) {
	return r.shares, int64(len(r.shares)), nil
}

type shareFiles struct{ ShareFileReader }

func (shareFiles) GetFileByIds([]string) ([]models.File, error) {
	return []models.File{{Id: "f", UserId: 1, ShareVersion: 2}}, nil
}
func TestShareListHidesInvalidatedVersionBeforeEventDelivery(t *testing.T) {
	s := NewShareService(shareListRepo{shares: []*models.Share{{Id: 1, FileId: "f", FileVersion: 1}, {Id: 2, FileId: "f", FileVersion: 2}}}, shareFiles{}, nil, nil)
	items, _, err := s.GetUserShares(1, 1, 20)
	if err != nil || len(items) != 1 || items[0].Id != 2 {
		t.Fatalf("items=%v err=%v", items, err)
	}
}

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
	for _, table := range []string{"recycle_bin", "share", "favorite", "file"} {
		mock.ExpectExec("DELETE FROM.*" + table).WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectExec("UPDATE.*storage_quota").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO.*object_deletion").WillReturnError(errors.New("disk full"))
	mock.ExpectRollback()
	storage := &cleanupStorage{}
	s := NewRecyclePurgeService(db, storage, repositories.NewRecycleRepository(db), repositories.NewFileRepository(db), repositories.NewShareRepository(db), repositories.NewFavoriteRepository(db), repositories.NewStorageQuotaRepository(db))
	if err := s.PurgeFiles(context.Background(), []string{"f"}); err == nil {
		t.Fatal("must roll back")
	}
	if storage.calls != 0 {
		t.Fatal("uncommitted deletion reached storage")
	}
}

// The other request wins between the initial lookup and INSERT.
type racedShareRepo struct {
	repositories.ShareRepository
	lookups int
	winner  *models.Share
}

func (r *racedShareRepo) IsShared(string) (bool, *models.Share) {
	r.lookups++
	return r.lookups > 1, r.winner
}
func (r *racedShareRepo) CreateShare(*models.Share) (*models.Share, error) {
	return nil, &mysqlDriver.MySQLError{Number: 1062, Message: "duplicate file"}
}
func (shareFiles) GetUserFileByID(int, string) (*models.File, error) {
	return &models.File{Id: "f", UserId: 1, ShareVersion: 2}, nil
}
func TestConcurrentShareCreationReturnsExistingWinner(t *testing.T) {
	winner := &models.Share{Id: 8, FileId: "f", FileVersion: 2}
	repo := &racedShareRepo{winner: winner}
	got, err := NewShareService(repo, shareFiles{}, nil, nil).CreateShare(1, "f", 1, "")
	if err != nil || got != winner {
		t.Fatalf("got=%v err=%v", got, err)
	}
}
