package services

import (
	"context"
	"errors"
	"io"
	"time"

	"go-cloud-storage/backend/internal/models"
	"go-cloud-storage/backend/internal/ports"
)

const downloadURLTTL = 10 * time.Minute

type DownloadInfo struct {
	FileID            string `json:"fileId"`
	FileName          string `json:"fileName"`
	FileSize          int64  `json:"fileSize"`
	ChunkSize         int64  `json:"chunkSize"`
	Chunks            int64  `json:"chunks"`
	SupportsRange     bool   `json:"supportsRange"`
	PreferredStrategy string `json:"preferredStrategy"`
	ExpiresInSeconds  int64  `json:"expiresInSeconds"`
	DirectDownloadURL string `json:"directDownloadUrl"`
}

type DownloadApplication interface {
	GetObjectSize(ctx context.Context, userID int, fileID string) (int64, error)
	GetPresignedDownloadURL(ctx context.Context, userID int, fileID string) (string, *models.File, error)
	DownloadRange(ctx context.Context, userID int, fileID string, start, end int64) (io.ReadCloser, *models.File, int64, error)
	GetDownloadInfo(ctx context.Context, userID int, fileID string) (*DownloadInfo, error)
}

type downloadApplication struct {
	files   ports.FileReadRepository
	storage ports.Storage
}

func NewDownloadApplication(files ports.FileReadRepository, storage ports.Storage) DownloadApplication {
	if files == nil {
		panic("NewDownloadApplication requires a FileReadRepository")
	}
	if storage == nil {
		panic("NewDownloadApplication requires a Storage implementation")
	}
	return &downloadApplication{files: files, storage: storage}
}

func (s *downloadApplication) getFile(ctx context.Context, userID int, fileID string) (*models.File, error) {
	file, err := s.files.GetUserFile(ctx, userID, fileID)
	if err != nil || file == nil || file.IsDir || file.IsDeleted {
		return nil, errors.New("要下载的文件不存在")
	}
	return file, nil
}

func (s *downloadApplication) GetObjectSize(ctx context.Context, userID int, fileID string) (int64, error) {
	file, err := s.getFile(ctx, userID, fileID)
	if err != nil {
		return 0, err
	}
	return s.storage.GetObjectInfo(ctx, file.OssObjectKey)
}

func (s *downloadApplication) GetPresignedDownloadURL(ctx context.Context, userID int, fileID string) (string, *models.File, error) {
	file, err := s.getFile(ctx, userID, fileID)
	if err != nil {
		return "", nil, err
	}
	u, err := s.storage.PresignedDownloadURL(ctx, file.OssObjectKey, file.Name, downloadURLTTL)
	if err != nil {
		return "", nil, err
	}
	return u, file, nil
}

func (s *downloadApplication) DownloadRange(ctx context.Context, userID int, fileID string, start, end int64) (io.ReadCloser, *models.File, int64, error) {
	file, err := s.getFile(ctx, userID, fileID)
	if err != nil {
		return nil, nil, 0, err
	}
	objectSize, err := s.storage.GetObjectInfo(ctx, file.OssObjectKey)
	if err != nil {
		return nil, nil, 0, err
	}
	reader, err := s.storage.DownloadFileRange(ctx, file.OssObjectKey, start, end)
	if err != nil {
		return nil, nil, 0, err
	}
	return reader, file, objectSize, nil
}

func (s *downloadApplication) GetDownloadInfo(ctx context.Context, userID int, fileID string) (*DownloadInfo, error) {
	file, err := s.getFile(ctx, userID, fileID)
	if err != nil {
		return nil, errors.New("文件不存在")
	}

	const (
		midChunkSize   int64 = 5 * 1024 * 1024
		largeChunkSize int64 = 10 * 1024 * 1024
	)
	var chunkSize int64
	switch {
	case file.Size <= 10*1024*1024:
		chunkSize = 0
	case file.Size <= 100*1024*1024:
		chunkSize = midChunkSize
	default:
		chunkSize = largeChunkSize
	}
	var chunks int64
	if chunkSize > 0 {
		chunks = (file.Size + chunkSize - 1) / chunkSize
	}

	directURL, _ := s.storage.PresignedDownloadURL(ctx, file.OssObjectKey, file.Name, downloadURLTTL)
	return &DownloadInfo{
		FileID: file.Id, FileName: file.Name, FileSize: file.Size,
		ChunkSize: chunkSize, Chunks: chunks, SupportsRange: true,
		PreferredStrategy: "presigned", ExpiresInSeconds: int64(downloadURLTTL.Seconds()),
		DirectDownloadURL: directURL,
	}, nil
}

var _ DownloadApplication = (*downloadApplication)(nil)
