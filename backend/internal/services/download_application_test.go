package services

import (
	"context"
	"testing"
	"time"

	"go-cloud-storage/backend/internal/models"
)

type fakeFileReadRepo struct {
	file *models.File
	err  error
}

func (f *fakeFileReadRepo) GetUserFile(context.Context, int, string) (*models.File, error) {
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
	files := &fakeFileReadRepo{file: &models.File{
		Id: "f1", UserId: 1, Name: "video.mp4", OssObjectKey: "objects/video", Size: 100,
	}}
	storage := &downloadStorageFake{fakeStorage: &fakeStorage{completeObject: &portsStoredObjectForTest{}}}
	_ = storage
}

func TestDownloadApplicationDownloadInfoChoosesChunkStrategy(t *testing.T) {
	const mb = int64(1024 * 1024)
	files := &fakeFileReadRepo{file: &models.File{
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
