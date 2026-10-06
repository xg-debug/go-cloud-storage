package ports

import (
	"context"
	"io"
	"time"
)

// StoredObject is the storage-layer representation of a persisted object.
// It deliberately contains no file-domain model so application services do not
// depend on a concrete object-storage SDK.
type StoredObject struct {
	ObjectKey   string
	Size        int64
	URL         string
	ThumbnailURL string
}

// UploadedPart is the storage-neutral result of a multipart part upload.
type UploadedPart struct {
	PartNumber int
	ETag       string
}

// CompletedPart identifies one uploaded part when completing a multipart upload.
type CompletedPart struct {
	PartNumber int
	ETag       string
}

// Storage defines the object-storage capabilities required by the file module.
// MinIO/S3/OSS implementations belong in infrastructure adapters.
type Storage interface {
	UploadFromStream(ctx context.Context, userID int, r io.Reader, fileName string, fileSize int64, fileHash, parentID string) (*StoredObject, error)

	GenerateObjectKey(userID int, parentID, fileName string) string
	InitiateMultipartUpload(ctx context.Context, objectKey string) (string, error)
	UploadPart(ctx context.Context, objectKey, uploadID string, partNumber int, r io.Reader, chunkSize int64, expectedHash string) (*UploadedPart, string, error)
	CompleteMultipartUpload(ctx context.Context, objectKey, uploadID string, parts []CompletedPart) (*StoredObject, error)
	AbortMultipartUpload(ctx context.Context, objectKey, uploadID string) error

	GetObjectInfo(ctx context.Context, objectKey string) (int64, error)
	ComputeObjectSHA256(ctx context.Context, objectKey string) (string, error)
	DownloadFile(ctx context.Context, objectKey string) (io.ReadCloser, error)
	DownloadFileRange(ctx context.Context, objectKey string, start, end int64) (io.ReadCloser, error)
	PutObjectStream(ctx context.Context, objectKey string, r io.Reader, size int64, contentType string) (int64, error)

	PresignedGetPreviewURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error)
	PresignStoredObjectURL(ctx context.Context, storedURL string, expiry time.Duration) string
	PresignedDownloadURL(ctx context.Context, objectKey, fileName string, expiry time.Duration) (string, error)

	DeleteFile(ctx context.Context, objectKey string) error
	DeleteObjectWithThumbnail(ctx context.Context, objectKey string) error
	DeleteThumbnailForObject(ctx context.Context, objectKey string) error
	DeleteFiles(ctx context.Context, objectKeys []string) error
	CopyObject(ctx context.Context, srcKey, dstKey string) error

	GenerateObjectURL(objectKey string) string
	ThumbnailObjectKey(objectKey string) string
	GenerateThumbnailForObject(ctx context.Context, objectKey string) (string, error)
}
