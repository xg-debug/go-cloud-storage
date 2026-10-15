package application

import (
	"context"
	"crypto/sha256"
	"fmt"
	ports "go-cloud-storage/backend/pkg/ports"
	domain "go-cloud-storage/backend/services/file-service/internal/domain"
	fileports "go-cloud-storage/backend/services/file-service/internal/ports"
	"io"
	"strings"
	"testing"
	"time"
)

type auditParentRepo struct {
	fakeUploadFileRepo
	alive bool
}

func (r *auditParentRepo) GetUserFile(context.Context, int, string) (*domain.File, error) {
	if r.alive {
		return &domain.File{Id: "parent", IsDir: true, UserId: 1}, nil
	}
	return nil, nil
}

type auditUploadStorage struct {
	fakeStorage
	onUpload   func()
	onComplete func()
}

func (s *auditUploadStorage) UploadFromStream(ctx context.Context, u int, r io.Reader, n string, size int64, h, p string) (*ports.StoredObject, error) {
	if s.onUpload != nil {
		s.onUpload()
	}
	return s.fakeStorage.UploadFromStream(ctx, u, r, n, size, h, p)
}
func (s *auditUploadStorage) CompleteMultipartUpload(ctx context.Context, k, id string, p []ports.CompletedPart) (*ports.StoredObject, error) {
	if s.onComplete != nil {
		s.onComplete()
	}
	return s.fakeStorage.CompleteMultipartUpload(ctx, k, id, p)
}
func TestRegressionUploadMustRecheckParentBeforeCommit(t *testing.T) {
	r := &auditParentRepo{alive: true}
	st := &auditUploadStorage{fakeStorage: fakeStorage{uploadObject: &ports.StoredObject{ObjectKey: "key", Size: 3}}}
	st.onUpload = func() { r.alive = false }
	app := NewUploadApplication(r, &fakeUploadQuotaRepo{available: 100}, &fakeTxManager{}, newFakeSessionStore(), st, &fakeEventBus{})
	_, err := app.UploadFile(context.Background(), strings.NewReader("abc"), 1, "a.txt", 3, payloadHash("abc"), "parent")
	if err == nil || len(r.created) != 0 || len(st.deleteWithThumb) != 1 {
		t.Fatalf("parent deletion must reject upload and clean object: err=%v created=%d", err, len(r.created))
	}
}
func TestRegressionNormalUploadMustVerifyHash(t *testing.T) {
	r := &fakeUploadFileRepo{}
	st := &fakeStorage{uploadObject: &ports.StoredObject{ObjectKey: "key", Size: 3}}
	app := newUploadTestApp(r, &fakeUploadQuotaRepo{available: 100}, &fakeTxManager{}, newFakeSessionStore(), st, &fakeEventBus{})
	expected := fmt.Sprintf("%x", sha256.Sum256([]byte("good")))
	_, err := app.UploadFile(context.Background(), strings.NewReader("bad"), 1, "a.txt", 3, expected, "")
	if err == nil {
		t.Fatalf("wrong SHA256 saved as %s", r.created[0].FileHash)
	}
}
func TestRegressionOldMergeMustNotDeleteNewSession(t *testing.T) {
	ctx := context.Background()
	sessions := newFakeSessionStore()
	hash := strings.Repeat("a", 64)
	sessions.Save(ctx, &fileports.UploadSession{UserID: 1, FileHash: hash, UploadID: "old", ObjectKey: "old-key", FileName: "a.txt", FileSize: 3, ChunkSize: 3, TotalChunks: 1, ExpiresAt: time.Now().Add(time.Hour), Parts: map[int]fileports.UploadPartState{0: {Index: 0, ETag: "etag", Size: 3}}})
	st := &auditUploadStorage{fakeStorage: fakeStorage{completeObject: &ports.StoredObject{ObjectKey: "old-key", Size: 3}, sha256: hash}}
	app := NewUploadApplication(&fakeUploadFileRepo{}, &fakeUploadQuotaRepo{available: 100}, &fakeTxManager{}, sessions, st, &fakeEventBus{})
	st.onComplete = func() {
		// Simulate replacement after a lease expiry; conditional cleanup must
		// still protect the new session even if the old worker resumes.
		sessions.Save(ctx, &fileports.UploadSession{UserID: 1, FileHash: hash, UploadID: "new", ObjectKey: "new-key", FileName: "a.txt", FileSize: 3, ChunkSize: 3, TotalChunks: 1})
	}
	if _, err := app.MergeChunks(ctx, 1, hash, "a.txt", "", 3, 3, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := sessions.Get(ctx, 1, hash); err != nil {
		t.Fatal("old merge removed replacement upload session")
	}
}

func TestCancelAndInitRespectActiveSessionLock(t *testing.T) {
	sessions := newFakeSessionStore()
	sessions.lockResult = false
	storage := &fakeStorage{}
	app := newUploadTestApp(&fakeUploadFileRepo{}, &fakeUploadQuotaRepo{available: 100}, &fakeTxManager{}, sessions, storage, &fakeEventBus{})
	if err := app.CancelChunkUpload(context.Background(), 1, "hash"); err == nil {
		t.Fatal("cancel ignored active merge")
	}
	if _, err := app.InitChunkUpload(context.Background(), InitChunkUploadInput{UserID: 1, FileName: "a.txt", FileHash: "hash", FileSize: 3, ChunkSize: 1024 * 1024, TotalChunks: 1}); err == nil {
		t.Fatal("init ignored active merge")
	}
	if storage.initCalls != 0 || storage.abortCalls != 0 || sessions.deleteCalls != 0 {
		t.Fatal("busy session was modified")
	}
}
func TestCleanupRechecksExpiryAfterScan(t *testing.T) {
	ctx := context.Background()
	sessions := newFakeSessionStore()
	sessions.expiredSessions = []fileports.UploadSessionRef{{UserID: 1, FileHash: "hash"}}
	sessions.Save(ctx, &fileports.UploadSession{UserID: 1, FileHash: "hash", UploadID: "new", ObjectKey: "key", ExpiresAt: time.Now().Add(time.Hour)})
	storage := &fakeStorage{}
	app := NewUploadApplication(&fakeUploadFileRepo{}, &fakeUploadQuotaRepo{}, &fakeTxManager{}, sessions, storage, &fakeEventBus{}).(*uploadApplication)
	app.cleanupExpiredChunkUploads(ctx, 100)
	if storage.abortCalls != 0 || sessions.deleteCalls != 0 {
		t.Fatal("cleanup deleted session refreshed after scan")
	}
}
