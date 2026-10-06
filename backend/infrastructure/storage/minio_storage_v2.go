package storage

import (
	"context"
	"io"
	"time"

	miniosrv "go-cloud-storage/backend/infrastructure/minio"
	"go-cloud-storage/backend/internal/ports"

	"github.com/minio/minio-go/v7"
)

// MinIOStorageV2 is the model-free object-storage adapter used by the refactored
// application layer. Unlike the legacy adapter, uploads return StoredObject and
// never construct application/domain models inside infrastructure code.
type MinIOStorageV2 struct {
	service *miniosrv.MinioService
}

func NewMinIOStorageV2(service *miniosrv.MinioService) *MinIOStorageV2 {
	return &MinIOStorageV2{service: service}
}

func (s *MinIOStorageV2) UploadFromStream(ctx context.Context, userID int, r io.Reader, fileName string, fileSize int64, _ string, parentID string) (*ports.StoredObject, error) {
	objectKey := s.service.GenerateObjectKey(userID, parentID, fileName)
	size, err := s.service.PutObjectStream(ctx, objectKey, r, fileSize, "application/octet-stream")
	if err != nil {
		return nil, err
	}
	url := s.service.GenerateObjectURL(objectKey)
	return &ports.StoredObject{
		ObjectKey:    objectKey,
		Size:         size,
		URL:          url,
		ThumbnailURL: url,
	}, nil
}

func (s *MinIOStorageV2) GenerateObjectKey(userID int, parentID, fileName string) string {
	return s.service.GenerateObjectKey(userID, parentID, fileName)
}

func (s *MinIOStorageV2) InitiateMultipartUpload(ctx context.Context, objectKey string) (string, error) {
	return s.service.InitiateMultipartUpload(ctx, objectKey)
}

func (s *MinIOStorageV2) UploadPart(ctx context.Context, objectKey, uploadID string, partNumber int, r io.Reader, chunkSize int64, expectedHash string) (*ports.UploadedPart, string, error) {
	part, hash, err := s.service.UploadPart(ctx, objectKey, uploadID, partNumber, r, chunkSize, expectedHash)
	if err != nil {
		return nil, hash, err
	}
	return &ports.UploadedPart{PartNumber: part.PartNumber, ETag: part.ETag}, hash, nil
}

func (s *MinIOStorageV2) CompleteMultipartUpload(ctx context.Context, objectKey, uploadID string, parts []ports.CompletedPart) (*ports.StoredObject, error) {
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

func (s *MinIOStorageV2) AbortMultipartUpload(ctx context.Context, objectKey, uploadID string) error {
	return s.service.AbortMultipartUpload(ctx, objectKey, uploadID)
}

func (s *MinIOStorageV2) GetObjectInfo(ctx context.Context, objectKey string) (int64, error) {
	return s.service.GetObjectInfo(ctx, objectKey)
}

func (s *MinIOStorageV2) ComputeObjectSHA256(ctx context.Context, objectKey string) (string, error) {
	return s.service.ComputeObjectSHA256(ctx, objectKey)
}

func (s *MinIOStorageV2) DownloadFile(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	return s.service.DownloadFile(ctx, objectKey)
}

func (s *MinIOStorageV2) DownloadFileRange(ctx context.Context, objectKey string, start, end int64) (io.ReadCloser, error) {
	return s.service.DownloadFileRange(ctx, objectKey, start, end)
}

func (s *MinIOStorageV2) PutObjectStream(ctx context.Context, objectKey string, r io.Reader, size int64, contentType string) (int64, error) {
	return s.service.PutObjectStream(ctx, objectKey, r, size, contentType)
}

func (s *MinIOStorageV2) PresignedGetPreviewURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	return s.service.PresignedGetPreviewURL(ctx, objectKey, expiry)
}

func (s *MinIOStorageV2) PresignStoredObjectURL(ctx context.Context, storedURL string, expiry time.Duration) string {
	return s.service.PresignStoredObjectURL(ctx, storedURL, expiry)
}

func (s *MinIOStorageV2) PresignedDownloadURL(ctx context.Context, objectKey, fileName string, expiry time.Duration) (string, error) {
	return s.service.PresignedDownloadURL(ctx, objectKey, fileName, expiry)
}

func (s *MinIOStorageV2) DeleteFile(ctx context.Context, objectKey string) error {
	return s.service.DeleteFile(ctx, objectKey)
}

func (s *MinIOStorageV2) DeleteObjectWithThumbnail(ctx context.Context, objectKey string) error {
	return s.service.DeleteObjectWithThumbnail(ctx, objectKey)
}

func (s *MinIOStorageV2) DeleteThumbnailForObject(ctx context.Context, objectKey string) error {
	return s.service.DeleteThumbnailForObject(ctx, objectKey)
}

func (s *MinIOStorageV2) DeleteFiles(ctx context.Context, objectKeys []string) error {
	return s.service.DeleteFiles(ctx, objectKeys)
}

func (s *MinIOStorageV2) CopyObject(ctx context.Context, srcKey, dstKey string) error {
	return s.service.CopyObject(ctx, srcKey, dstKey)
}

func (s *MinIOStorageV2) GenerateObjectURL(objectKey string) string {
	return s.service.GenerateObjectURL(objectKey)
}

func (s *MinIOStorageV2) ThumbnailObjectKey(objectKey string) string {
	return miniosrv.ThumbnailObjectKey(objectKey)
}

func (s *MinIOStorageV2) GenerateThumbnailForObject(ctx context.Context, objectKey string) (string, error) {
	return s.service.GenerateThumbnailForObject(ctx, objectKey)
}

var _ ports.Storage = (*MinIOStorageV2)(nil)
