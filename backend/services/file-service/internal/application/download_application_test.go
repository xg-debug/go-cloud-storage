package application

import (
	"context"
	"testing"
	"time"

	ports "go-cloud-storage/backend/pkg/ports"
	domain "go-cloud-storage/backend/services/file-service/internal/domain"
	fileports "go-cloud-storage/backend/services/file-service/internal/ports"
)

type fakeFileReadRepo struct {
	file *domain.File
	err  error
}

func (f *fakeFileReadRepo) GetUserFile(context.Context, int, string) (*domain.File, error) {
	return f.file, f.err
}

type downloadStorageFake struct {
	*fakeStorage
	presignedURL string
}

func (f *downloadStorageFake) PresignedDownloadURL(context.Context, string, string, time.Duration) (string, error) {
	return f.presignedURL, nil
}

func TestDownloadApplicationRangeUsesStoragePort(t *testing.T) {
	files := &fakeFileReadRepo{file: &domain.File{
		Id: "f1", UserId: 1, Name: "video.mp4", OssObjectKey: "objects/video", Size: 100,
	}}
	storage := &downloadStorageFake{fakeStorage: &fakeStorage{
		completeObject: &ports.StoredObject{ObjectKey: "objects/video", Size: 100},
	}}
	app := NewDownloadApplication(files, storage)

	reader, file, objectSize, err := app.DownloadRange(context.Background(), 1, "f1", 10, 19)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if file.Id != "f1" || objectSize != 100 {
		t.Fatalf("file=%+v objectSize=%d", file, objectSize)
	}
}

func TestDownloadApplicationDownloadInfoChoosesChunkStrategy(t *testing.T) {
	const mb = int64(1024 * 1024)
	files := &fakeFileReadRepo{file: &domain.File{
		Id: "f1", UserId: 1, Name: "archive.bin", OssObjectKey: "objects/archive", Size: 120 * mb,
	}}
	storage := &downloadStorageFake{fakeStorage: &fakeStorage{}, presignedURL: "https://download.example/file"}
	app := NewDownloadApplication(files, storage)

	info, err := app.GetDownloadInfo(context.Background(), 1, "f1")
	if err != nil {
		t.Fatal(err)
	}
	if info.ChunkSize != 10*mb || info.Chunks != 12 {
		t.Fatalf("chunkSize=%d chunks=%d", info.ChunkSize, info.Chunks)
	}
	if info.DirectDownloadURL != storage.presignedURL || info.PreferredStrategy != "presigned" || !info.SupportsRange {
		t.Fatalf("unexpected download info: %+v", info)
	}
}

var _ fileports.FileReadRepository = (*fakeFileReadRepo)(nil)
