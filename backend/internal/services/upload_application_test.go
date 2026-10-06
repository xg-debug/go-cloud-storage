package services

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"go-cloud-storage/backend/internal/models"
	"go-cloud-storage/backend/internal/ports"
)

type fakeUploadFileRepo struct {
	duplicate       bool
	existing        *models.File
	createErr       error
	created         []*models.File
	thumbnailUpdate bool
}

func (f *fakeUploadFileRepo) CheckDuplicateName(context.Context, int, string, string) (bool, error) {
	return f.duplicate, nil
}
func (f *fakeUploadFileRepo) FindByHash(context.Context, int, string) (*models.File, error) {
	return f.existing, nil
}
func (f *fakeUploadFileRepo) GetUserFile(context.Context, int, string) (*models.File, error) {
	return nil, nil
}
func (f *fakeUploadFileRepo) Create(_ context.Context, file *models.File) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.created = append(f.created, file)
	return nil
}
func (f *fakeUploadFileRepo) UpdateThumbnail(context.Context, string, string) (bool, error) {
	return f.thumbnailUpdate, nil
}

type fakeUploadQuotaRepo struct {
	available int64
	addErr    error
	added     int64
}

func (f *fakeUploadQuotaRepo) GetAvailableSpace(context.Context, int) (int64, error) {
	return f.available, nil
}
func (f *fakeUploadQuotaRepo) AddUsedSpace(_ context.Context, _ int, delta int64) error {
	if f.addErr != nil {
		return f.addErr
	}
	f.added += delta
	return nil
}

type fakeTxManager struct {
	calls int
	err   error
}

func (f *fakeTxManager) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	f.calls++
	if f.err != nil {
		return f.err
	}
	return fn(ctx)
}

type fakeSessionStore struct {
	available       bool
	sessions        map[string]*ports.UploadSession
	lockResult      bool
	lockErr         error
	lockCalls       int
	releaseCalls    int
	deleteCalls     int
	savedParts      []ports.UploadPartState
	expiredSessions []ports.UploadSessionRef
}

func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{available: true, sessions: map[string]*ports.UploadSession{}, lockResult: true}
}
func sessionMapKey(userID int, hash string) string { return string(rune(userID)) + ":" + hash }
func (f *fakeSessionStore) Available() bool          { return f.available }
func (f *fakeSessionStore) Get(_ context.Context, userID int, hash string) (*ports.UploadSession, error) {
	s, ok := f.sessions[sessionMapKey(userID, hash)]
	if !ok {
		return nil, ports.ErrUploadSessionNotFound
	}
	return s, nil
}
func (f *fakeSessionStore) Save(_ context.Context, s *ports.UploadSession) error {
	if s.Parts == nil {
		s.Parts = map[int]ports.UploadPartState{}
	}
	if s.ExpiresAt.IsZero() {
		s.ExpiresAt = time.Now().Add(time.Hour)
	}
	f.sessions[sessionMapKey(s.UserID, s.FileHash)] = s
	return nil
}
func (f *fakeSessionStore) SavePart(_ context.Context, userID int, hash string, part ports.UploadPartState) error {
	s, ok := f.sessions[sessionMapKey(userID, hash)]
	if !ok {
		return ports.ErrUploadSessionNotFound
	}
	if s.Parts == nil {
		s.Parts = map[int]ports.UploadPartState{}
	}
	s.Parts[part.Index] = part
	f.savedParts = append(f.savedParts, part)
	return nil
}
func (f *fakeSessionStore) Refresh(context.Context, int, string) error { return nil }
func (f *fakeSessionStore) Delete(_ context.Context, userID int, hash string) error {
	delete(f.sessions, sessionMapKey(userID, hash))
	f.deleteCalls++
	return nil
}
func (f *fakeSessionStore) AcquireMergeLock(context.Context, int, string, time.Duration) (bool, error) {
	f.lockCalls++
	return f.lockResult, f.lockErr
}
func (f *fakeSessionStore) ReleaseMergeLock(context.Context, int, string) error {
	f.releaseCalls++
	return nil
}
func (f *fakeSessionStore) ListExpired(context.Context, time.Time, int64) ([]ports.UploadSessionRef, error) {
	return f.expiredSessions, nil
}

type fakeStorage struct {
	uploadObject       *ports.StoredObject
	uploadErr          error
	uploadCalls        int
	initUploadID       string
	initCalls          int
	uploadPart         *ports.UploadedPart
	partHash           string
	partErr            error
	completeObject     *ports.StoredObject
	completeErr        error
	completeParts      []ports.CompletedPart
	abortCalls         int
	deleteCalls        int
	deleteWithThumb    []string
	sha256             string
	thumbnailURL       string
	generateObjectKey  string
}

func (f *fakeStorage) UploadFromStream(context.Context, int, io.Reader, string, int64, string, string) (*ports.StoredObject, error) {
	f.uploadCalls++
	return f.uploadObject, f.uploadErr
}
func (f *fakeStorage) GenerateObjectKey(int, string, string) string {
	if f.generateObjectKey != "" {
		return f.generateObjectKey
	}
	return "objects/test"
}
func (f *fakeStorage) InitiateMultipartUpload(context.Context, string) (string, error) {
	f.initCalls++
	if f.initUploadID == "" {
		return "upload-1", nil
	}
	return f.initUploadID, nil
}
func (f *fakeStorage) UploadPart(context.Context, string, string, int, io.Reader, int64, string) (*ports.UploadedPart, string, error) {
	if f.uploadPart == nil {
		f.uploadPart = &ports.UploadedPart{PartNumber: 1, ETag: "etag"}
	}
	return f.uploadPart, f.partHash, f.partErr
}
func (f *fakeStorage) CompleteMultipartUpload(_ context.Context, _ string, _ string, parts []ports.CompletedPart) (*ports.StoredObject, error) {
	f.completeParts = append([]ports.CompletedPart(nil), parts...)
	return f.completeObject, f.completeErr
}
func (f *fakeStorage) AbortMultipartUpload(context.Context, string, string) error { f.abortCalls++; return nil }
func (f *fakeStorage) GetObjectInfo(context.Context, string) (int64, error) {
	if f.completeObject != nil {
		return f.completeObject.Size, nil
	}
	return 0, nil
}
func (f *fakeStorage) ComputeObjectSHA256(context.Context, string) (string, error) { return f.sha256, nil }
func (f *fakeStorage) DownloadFile(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}
func (f *fakeStorage) DownloadFileRange(context.Context, string, int64, int64) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}
func (f *fakeStorage) PutObjectStream(context.Context, string, io.Reader, int64, string) (int64, error) {
	return 0, nil
}
func (f *fakeStorage) PresignedGetPreviewURL(context.Context, string, time.Duration) (string, error) {
	return "", nil
}
func (f *fakeStorage) PresignStoredObjectURL(context.Context, string, time.Duration) string { return "" }
func (f *fakeStorage) PresignedDownloadURL(context.Context, string, string, time.Duration) (string, error) {
	return "", nil
}
func (f *fakeStorage) DeleteFile(context.Context, string) error { f.deleteCalls++; return nil }
func (f *fakeStorage) DeleteObjectWithThumbnail(_ context.Context, key string) error {
	f.deleteWithThumb = append(f.deleteWithThumb, key)
	return nil
}
func (f *fakeStorage) DeleteThumbnailForObject(context.Context, string) error { return nil }
func (f *fakeStorage) DeleteFiles(context.Context, []string) error             { return nil }
func (f *fakeStorage) CopyObject(context.Context, string, string) error         { return nil }
func (f *fakeStorage) GenerateObjectURL(string) string                          { return "" }
func (f *fakeStorage) ThumbnailObjectKey(string) string                         { return "" }
func (f *fakeStorage) GenerateThumbnailForObject(context.Context, string) (string, error) {
	return f.thumbnailURL, nil
}

type fakeEventBus struct{ events []ports.Event }

func (f *fakeEventBus) Publish(_ context.Context, event ports.Event) error {
	f.events = append(f.events, event)
	return nil
}

func newUploadTestApp(files *fakeUploadFileRepo, quotas *fakeUploadQuotaRepo, tx *fakeTxManager, sessions *fakeSessionStore, storage *fakeStorage, events *fakeEventBus) UploadApplication {
	return NewUploadApplication(files, quotas, tx, sessions, storage, events)
}

func TestUploadApplicationInstantUploadReusesExistingObject(t *testing.T) {
	files := &fakeUploadFileRepo{existing: &models.File{
		Id: "old", UserId: 1, Name: "old.pdf", Size: 128, SizeStr: "128 B",
		OssObjectKey: "shared/object", FileURL: "old-url", ThumbnailURL: "old-thumb",
	}}
	quotas := &fakeUploadQuotaRepo{available: 1024}
	tx := &fakeTxManager{}
	sessions := newFakeSessionStore()
	storage := &fakeStorage{}
	events := &fakeEventBus{}
	app := newUploadTestApp(files, quotas, tx, sessions, storage, events)

	file, err := app.UploadFile(context.Background(), strings.NewReader("ignored"), 1, "paper.pdf", 128, "hash", "")
	if err != nil {
		t.Fatal(err)
	}
	if storage.uploadCalls != 0 {
		t.Fatalf("storage upload calls = %d, want 0 for instant upload", storage.uploadCalls)
	}
	if file.Id == "old" || file.OssObjectKey != "shared/object" {
		t.Fatalf("instant upload did not clone metadata correctly: %+v", file)
	}
	if tx.calls != 1 || len(files.created) != 1 || quotas.added != 128 {
		t.Fatalf("tx=%d created=%d quota=%d", tx.calls, len(files.created), quotas.added)
	}
	if len(events.events) != 1 || events.events[0].Type != "file.uploaded.v1" {
		t.Fatalf("events = %+v", events.events)
	}
}

func TestUploadApplicationPersistenceFailureCleansObject(t *testing.T) {
	persistErr := errors.New("db write failed")
	files := &fakeUploadFileRepo{createErr: persistErr}
	quotas := &fakeUploadQuotaRepo{available: 1024}
	tx := &fakeTxManager{}
	sessions := newFakeSessionStore()
	storage := &fakeStorage{uploadObject: &ports.StoredObject{ObjectKey: "objects/new", Size: 100}}
	app := newUploadTestApp(files, quotas, tx, sessions, storage, &fakeEventBus{})

	_, err := app.UploadFile(context.Background(), strings.NewReader("data"), 1, "new.pdf", 100, "hash", "")
	if err == nil {
		t.Fatal("expected persistence failure")
	}
	if len(storage.deleteWithThumb) != 1 || storage.deleteWithThumb[0] != "objects/new" {
		t.Fatalf("cleanup calls = %+v", storage.deleteWithThumb)
	}
}

func TestInitChunkUploadResumesAndSortsParts(t *testing.T) {
	const mb = int64(1024 * 1024)
	files := &fakeUploadFileRepo{}
	quotas := &fakeUploadQuotaRepo{available: 100 * mb}
	tx := &fakeTxManager{}
	sessions := newFakeSessionStore()
	sessions.sessions[sessionMapKey(1, "hash")] = &ports.UploadSession{
		UserID: 1, FileHash: "hash", UploadID: "u1", ObjectKey: "obj", FileName: "a.pdf",
		FileSize: 15 * mb, ChunkSize: 5 * mb, TotalChunks: 3, ExpiresAt: time.Now().Add(time.Hour),
		Parts: map[int]ports.UploadPartState{2: {Index: 2, ETag: "e2"}, 0: {Index: 0, ETag: "e0"}},
	}
	storage := &fakeStorage{}
	app := newUploadTestApp(files, quotas, tx, sessions, storage, &fakeEventBus{})

	result, err := app.InitChunkUpload(context.Background(), InitChunkUploadInput{
		UserID: 1, FileName: "a.pdf", FileHash: "hash", FileSize: 15 * mb, ChunkSize: 5 * mb, TotalChunks: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if storage.initCalls != 0 {
		t.Fatalf("multipart init called %d times while resuming", storage.initCalls)
	}
	if len(result.UploadedChunks) != 2 || result.UploadedChunks[0] != 0 || result.UploadedChunks[1] != 2 {
		t.Fatalf("uploaded chunks = %v", result.UploadedChunks)
	}
}

func TestInitChunkUploadRejectsChangedParametersAfterPartialUpload(t *testing.T) {
	const mb = int64(1024 * 1024)
	sessions := newFakeSessionStore()
	sessions.sessions[sessionMapKey(1, "hash")] = &ports.UploadSession{
		UserID: 1, FileHash: "hash", UploadID: "u1", ObjectKey: "obj", FileName: "a.pdf",
		FileSize: 15 * mb, ChunkSize: 5 * mb, TotalChunks: 3, ExpiresAt: time.Now().Add(time.Hour),
		Parts: map[int]ports.UploadPartState{0: {Index: 0, ETag: "e0", Size: 5 * mb}},
	}
	app := newUploadTestApp(&fakeUploadFileRepo{}, &fakeUploadQuotaRepo{available: 100 * mb}, &fakeTxManager{}, sessions, &fakeStorage{}, &fakeEventBus{})

	_, err := app.InitChunkUpload(context.Background(), InitChunkUploadInput{
		UserID: 1, FileName: "a.pdf", FileHash: "hash", FileSize: 20 * mb, ChunkSize: 5 * mb, TotalChunks: 4,
	})
	if err == nil || !strings.Contains(err.Error(), "断点不一致") {
		t.Fatalf("err = %v, want resume-parameter conflict", err)
	}
}

func TestUploadChunkExpiredSessionAbortsAndDeletes(t *testing.T) {
	const mb = int64(1024 * 1024)
	sessions := newFakeSessionStore()
	sessions.sessions[sessionMapKey(1, "hash")] = &ports.UploadSession{
		UserID: 1, FileHash: "hash", UploadID: "u1", ObjectKey: "obj", FileName: "a.pdf",
		FileSize: 5 * mb, ChunkSize: 5 * mb, TotalChunks: 1, ExpiresAt: time.Now().Add(-time.Minute),
		Parts: map[int]ports.UploadPartState{},
	}
	storage := &fakeStorage{}
	app := newUploadTestApp(&fakeUploadFileRepo{}, &fakeUploadQuotaRepo{available: 100 * mb}, &fakeTxManager{}, sessions, storage, &fakeEventBus{})

	err := app.UploadChunk(context.Background(), 1, "hash", 0, strings.NewReader("x"), 5*mb, "")
	if err == nil || storage.abortCalls != 1 || sessions.deleteCalls != 1 {
		t.Fatalf("err=%v abort=%d delete=%d", err, storage.abortCalls, sessions.deleteCalls)
	}
}

func TestMergeChunksRespectsDistributedLock(t *testing.T) {
	sessions := newFakeSessionStore()
	sessions.lockResult = false
	app := newUploadTestApp(&fakeUploadFileRepo{}, &fakeUploadQuotaRepo{}, &fakeTxManager{}, sessions, &fakeStorage{}, &fakeEventBus{})

	_, err := app.MergeChunks(context.Background(), 1, "hash", "a.pdf", "", 1, 1, 1)
	if err == nil || !strings.Contains(err.Error(), "合并正在进行中") {
		t.Fatalf("err = %v", err)
	}
}

func TestMergeChunksCompletesPersistsAndDeletesSession(t *testing.T) {
	const mb = int64(1024 * 1024)
	files := &fakeUploadFileRepo{}
	quotas := &fakeUploadQuotaRepo{available: 100 * mb}
	tx := &fakeTxManager{}
	sessions := newFakeSessionStore()
	sessions.sessions[sessionMapKey(1, "short-hash")] = &ports.UploadSession{
		UserID: 1, FileHash: "short-hash", UploadID: "u1", ObjectKey: "obj", FileName: "a.pdf",
		FileSize: 8 * mb, ChunkSize: 4 * mb, TotalChunks: 2, ExpiresAt: time.Now().Add(time.Hour),
		Parts: map[int]ports.UploadPartState{
			1: {Index: 1, ETag: "e1", Size: 4 * mb},
			0: {Index: 0, ETag: "e0", Size: 4 * mb},
		},
	}
	storage := &fakeStorage{completeObject: &ports.StoredObject{ObjectKey: "obj", Size: 8 * mb, URL: "url"}}
	events := &fakeEventBus{}
	app := newUploadTestApp(files, quotas, tx, sessions, storage, events)

	file, err := app.MergeChunks(context.Background(), 1, "short-hash", "a.pdf", "", 8*mb, 4*mb, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(storage.completeParts) != 2 || storage.completeParts[0].PartNumber != 1 || storage.completeParts[1].PartNumber != 2 {
		t.Fatalf("parts = %+v", storage.completeParts)
	}
	if file.OssObjectKey != "obj" || len(files.created) != 1 || quotas.added != 8*mb {
		t.Fatalf("file=%+v created=%d quota=%d", file, len(files.created), quotas.added)
	}
	if sessions.deleteCalls != 1 || sessions.releaseCalls != 1 {
		t.Fatalf("delete=%d release=%d", sessions.deleteCalls, sessions.releaseCalls)
	}
	if len(events.events) != 1 || events.events[0].Type != "file.uploaded.v1" {
		t.Fatalf("events=%+v", events.events)
	}
}

var _ ports.UploadFileRepository = (*fakeUploadFileRepo)(nil)
var _ ports.UploadQuotaRepository = (*fakeUploadQuotaRepo)(nil)
var _ ports.TransactionManager = (*fakeTxManager)(nil)
var _ ports.UploadSessionStore = (*fakeSessionStore)(nil)
var _ ports.Storage = (*fakeStorage)(nil)
var _ ports.EventBus = (*fakeEventBus)(nil)
