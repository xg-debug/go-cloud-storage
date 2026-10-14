package storage

import (
	"context"
	"io"
	"time"

	miniosrv "go-cloud-storage/backend/infrastructure/minio"
	"go-cloud-storage/backend/internal/ports"

	"github.com/minio/minio-go/v7"
)

// MinIOStorage adapts the existing MinioService to the application Storage port.
// Keeping this wrapper separate lets the file module depend on a stable interface
// while the underlying object-storage SDK remains an infrastructure concern.
type MinIOStorage struct {
	service *miniosrv.MinioService
}

func NewMinIOStorage(service *miniosrv.MinioService) *MinIOStorage {
	return &MinIOStorage{service: service}
}

func (s *MinIOStorage) UploadFromStream(ctx context.Context, userID int, r io.Reader, fileName string, fileSize int64, fileHash, parentID string) (*ports.StoredObject, error) {
	file, err := s.service.UploadFromStream(ctx, userID, r, fileName, fileSize, fileHash, parentID)
	if err != nil {
		return nil, err
	}
	return &ports.StoredObject{
		ObjectKey:    file.OssObjectKey,
		Size:         file.Size,
		URL:          file.FileURL,
		ThumbnailURL: file.ThumbnailURL,
	}, nil
}

func (s *MinIOStorage) GenerateObjectKey(userID int, parentID, fileName string) string {
	return s.service.GenerateObjectKey(userID, parentID, fileName)
}

func (s *MinIOStorage) InitiateMultipartUpload(ctx context.Context, objectKey string) (string, error) {
	return s.service.InitiateMultipartUpload(ctx, objectKey)
}

func (s *MinIOStorage) UploadPart(ctx context.Context, objectKey, uploadID string, partNumber int, r io.Reader, chunkSize int64, expectedHash string) (*ports.UploadedPart, string, error) {
	part, hash, err := s.service.UploadPart(ctx, objectKey, uploadID, partNumber, r, chunkSize, expectedHash)
	if err != nil {
		return nil, hash, err
	}
	return &ports.UploadedPart{PartNumber: part.PartNumber, ETag: part.ETag}, hash, nil
}

func (s *MinIOStorage) CompleteMultipartUpload(ctx context.Context, objectKey, uploadID string, parts []ports.CompletedPart) (*ports.StoredObject, error) {
	completed := make([]minio.CompletePart, 0, len(parts))
	for _, part := range parts {
		completed = append(completed, minio.CompletePart{PartNumber: part.PartNumber, ETag: part.ETag})
	}
	fileURL, thumbnailURL, err := s.service.CompleteMultipartUpload(ctx, objectKey, uploadID, completed)
	if err != nil {
		return nil, err
	}
	size, err := s.service.GetObjectInfo(ctx, objectKey)
	if err != nil {
		return nil, err
	}
	return &ports.StoredObject{ObjectKey: objectKey, Size: size, URL: fileURL, ThumbnailURL: thumbnailURL}, nil
}

func (s *MinIOStorage) AbortMultipartUpload(ctx context.Context, objectKey, uploadID string) error {
	return s.service.AbortMultipartUpload(ctx, objectKey, uploadID)
}

func (s *MinIOStorage) GetObjectInfo(ctx context.Context, objectKey string) (int64, error) {
	return s.service.GetObjectInfo(ctx, objectKey)
}

func (s *MinIOStorage) ComputeObjectSHA256(ctx context.Context, objectKey string) (string, error) {
	return s.service.ComputeObjectSHA256(ctx, objectKey)
}

func (s *MinIOStorage) DownloadFile(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	return s.service.DownloadFile(ctx, objectKey)
}

func (s *MinIOStorage) DownloadFileRange(ctx context.Context, objectKey string, start, end int64) (io.ReadCloser, error) {
	return s.service.DownloadFileRange(ctx, objectKey, start, end)
}

func (s *MinIOStorage) PutObjectStream(ctx context.Context, objectKey string, r io.Reader, size int64, contentType string) (int64, error) {
	return s.service.PutObjectStream(ctx, objectKey, r, size, contentType)
}

func (s *MinIOStorage) PresignedGetPreviewURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	return s.service.PresignedGetPreviewURL(ctx, objectKey, expiry)
}

func (s *MinIOStorage) PresignStoredObjectURL(ctx context.Context, storedURL string, expiry time.Duration) string {
	return s.service.PresignStoredObjectURL(ctx, storedURL, expiry)
}

func (s *MinIOStorage) PresignedDownloadURL(ctx context.Context, objectKey, fileName string, expiry time.Duration) (string, error) {
	return s.service.PresignedDownloadURL(ctx, objectKey, fileName, expiry)
}

func (s *MinIOStorage) DeleteFile(ctx context.Context, objectKey string) error {
	return s.service.DeleteFile(ctx, objectKey)
}

func (s *MinIOStorage) DeleteObjectWithThumbnail(ctx context.Context, objectKey string) error {
	return s.service.DeleteObjectWithThumbnail(ctx, objectKey)
}

func (s *MinIOStorage) DeleteThumbnailForObject(ctx context.Context, objectKey string) error {
	return s.service.DeleteThumbnailForObject(ctx, objectKey)
}

func (s *MinIOStorage) DeleteFiles(ctx context.Context, objectKeys []string) error {
	return s.service.DeleteFiles(ctx, objectKeys)
}

func (s *MinIOStorage) CopyObject(ctx context.Context, srcKey, dstKey string) error {
	return s.service.CopyObject(ctx, srcKey, dstKey)
}

func (s *MinIOStorage) GenerateObjectURL(objectKey string) string {
	return s.service.GenerateObjectURL(objectKey)
}

func (s *MinIOStorage) ThumbnailObjectKey(objectKey string) string {
	return miniosrv.ThumbnailObjectKey(objectKey)
}

func (s *MinIOStorage) GenerateThumbnailForObject(ctx context.Context, objectKey string) (string, error) {
	return s.service.GenerateThumbnailForObject(ctx, objectKey)
}
