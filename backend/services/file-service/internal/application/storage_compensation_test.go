package application_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	ports "go-cloud-storage/backend/pkg/ports"
	application "go-cloud-storage/backend/services/file-service/internal/application"
	domain "go-cloud-storage/backend/services/file-service/internal/domain"
	fileports "go-cloud-storage/backend/services/file-service/internal/ports"
)

type memoryStorage struct {
	ports.Storage
	mu        sync.Mutex
	objects   map[string]string
	parts     map[int]string
	thumbnail chan struct{}
}

func newStorage() *memoryStorage {
	return &memoryStorage{objects: map[string]string{}, parts: map[int]string{}}
}
func (s *memoryStorage) PutObjectStream(_ context.Context, key string, r io.Reader, size int64, contentType string) (int64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects[key] = string(data)
	return int64(len(data)), nil
}
func (s *memoryStorage) GenerateObjectURL(key string) string {
	return "http://objects.test/bucket/" + key
}
func (s *memoryStorage) GetObjectInfo(_ context.Context, key string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.objects[key]
	if !ok {
		return 0, errors.New("missing object")
	}
	return int64(len(data)), nil
}
func (s *memoryStorage) DownloadFile(ctx context.Context, key string) (io.ReadCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.objects[key]
	if !ok {
		return nil, errors.New("missing object")
	}
	return io.NopCloser(strings.NewReader(data)), nil
}
func (s *memoryStorage) DownloadFileRange(ctx context.Context, key string, start, end int64) (io.ReadCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.objects[key]
	if end >= int64(len(data)) {
		return nil, errors.New("range")
	}
	return io.NopCloser(strings.NewReader(data[start : end+1])), nil
}
func (s *memoryStorage) ComputeObjectSHA256(ctx context.Context, key string) (string, error) {
	r, err := s.DownloadFile(ctx, key)
	if err != nil {
		return "", err
	}
	defer r.Close()
	h := sha256.New()
	_, err = io.Copy(h, r)
	return hex.EncodeToString(h.Sum(nil)), err
}
func (s *memoryStorage) InitiateMultipartUpload(context.Context, string) (string, error) {
	return "upload-1", nil
}
func (s *memoryStorage) UploadPart(_ context.Context, key, id string, n int, r io.Reader, size int64, hash string) (*ports.UploadedPart, string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.parts[n] = string(data)
	return &ports.UploadedPart{PartNumber: n, ETag: "etag"}, hash, nil
}
func (s *memoryStorage) CompleteMultipartUpload(_ context.Context, key, id string, parts []ports.CompletedPart) (*ports.StoredObject, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var data string
	for _, p := range parts {
		data += s.parts[p.PartNumber]
	}
	s.objects[key] = data
	return &ports.StoredObject{ObjectKey: key, Size: int64(len(data)), URL: s.GenerateObjectURL(key)}, nil
}
func (s *memoryStorage) AbortMultipartUpload(context.Context, string, string) error { return nil }
func (s *memoryStorage) DeleteFile(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.objects, key)
	return nil
}
func (s *memoryStorage) DeleteObjectWithThumbnail(ctx context.Context, key string) error {
	return s.DeleteFile(ctx, key)
}
func (s *memoryStorage) DeleteThumbnailForObject(context.Context, string) error { return nil }
func (s *memoryStorage) DeleteFiles(ctx context.Context, keys []string) error {
	for _, key := range keys {
		_ = s.DeleteFile(ctx, key)
	}
	return nil
}
func (s *memoryStorage) CopyObject(_ context.Context, source, destination string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects[destination] = s.objects[source]
	return nil
}
func (s *memoryStorage) GenerateThumbnailForObject(context.Context, string) (string, error) {
	if s.thumbnail != nil {
		close(s.thumbnail)
	}
	return "", nil
}
func (s *memoryStorage) PresignedGetPreviewURL(_ context.Context, key string, expiry time.Duration) (string, error) {
	return s.GenerateObjectURL(key) + "?preview=1", nil
}
func (s *memoryStorage) PresignedDownloadURL(_ context.Context, key, name string, expiry time.Duration) (string, error) {
	return s.GenerateObjectURL(key) + "?download=1", nil
}
func (s *memoryStorage) PresignStoredObjectURL(_ context.Context, value string, expiry time.Duration) string {
	return value + "?signed=1"
}

func read(t *testing.T, r io.ReadCloser) string {
	t.Helper()
	defer r.Close()
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func (s *memoryStorage) UploadFromStream(ctx context.Context, user int, r io.Reader, name string, size int64, hash, parent string) (*ports.StoredObject, error) {
	key := name
	n, err := s.PutObjectStream(ctx, key, r, size, "")
	return &ports.StoredObject{ObjectKey: key, Size: n, URL: s.GenerateObjectURL(key)}, err
}

type fileRepository struct {
	fileports.UploadFileRepository
	file *domain.File
	fail bool
}

func (r *fileRepository) CheckDuplicateName(context.Context, int, string, string) (bool, error) {
	return false, nil
}
func (r *fileRepository) FindByHash(context.Context, int, string) (*domain.File, error) {
	return nil, nil
}
func (r *fileRepository) Create(ctx context.Context, f *domain.File) error {
	if r.fail {
		return errors.New("database unavailable")
	}
	r.file = f
	return nil
}
func (r *fileRepository) GetUserFile(ctx context.Context, userID int, id string) (*domain.File, error) {
	if r.file != nil && r.file.UserId == userID && r.file.Id == id {
		return r.file, nil
	}
	return nil, nil
}

type quota struct{ used int64 }

func (q *quota) GetAvailableSpace(context.Context, int) (int64, error)     { return 1024, nil }
func (q *quota) AddUsedSpace(ctx context.Context, user int, n int64) error { q.used += n; return nil }

type transaction struct{}

func (transaction) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type sessions struct{ fileports.UploadSessionStore }
type events struct{ items []ports.Event }

func (e *events) Publish(ctx context.Context, event ports.Event) error {
	e.items = append(e.items, event)
	return nil
}

func TestUploadApplicationStorageCompensation(t *testing.T) {
	store := newStorage()
	store.thumbnail = make(chan struct{})
	client := store
	files := &fileRepository{}
	q := &quota{}
	bus := &events{}
	app := application.NewUploadApplication(files, q, transaction{}, sessions{}, client, bus)
	file, err := app.UploadFile(context.Background(), strings.NewReader("payload"), 7, "a.txt", 7, "hash", "")
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-store.thumbnail:
	case <-time.After(time.Second):
		t.Fatal("thumbnail call not received")
	}
	if files.file != file || q.used != 7 || len(bus.items) != 1 || bus.items[0].Type != "file.uploaded.v1" {
		t.Fatal("metadata/quota/event missing")
	}
	download := application.NewDownloadApplication(files, client)
	r, _, size, err := download.DownloadRange(context.Background(), 7, file.Id, 0, 6)
	if err != nil {
		t.Fatal(err)
	}
	if size != 7 || read(t, r) != "payload" {
		t.Fatal("download mismatch")
	}
	if _, _, _, err = download.DownloadRange(context.Background(), 8, file.Id, 0, 6); err == nil {
		t.Fatal("another user accessed file")
	}
	// Database failure compensates the object and must not emit a success event.
	files.fail = true
	if _, err = app.UploadFile(context.Background(), strings.NewReader("bad"), 7, "b.txt", 3, "other", ""); err == nil {
		t.Fatal("database failure ignored")
	}
	store.mu.Lock()
	count := len(store.objects)
	store.mu.Unlock()
	if count != 1 || len(bus.items) != 1 || q.used != 7 {
		t.Fatal("compensation/event invariant failed")
	}
}
