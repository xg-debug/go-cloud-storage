package ports

import (
	"context"

	domain "go-cloud-storage/backend/services/file-service/internal/domain"
)

// UploadFileRepository is the minimal file-metadata boundary required by the
// upload application. Concrete ORM/query details stay in infrastructure.
type UploadFileRepository interface {
	CheckDuplicateName(ctx context.Context, userID int, parentID, name string) (bool, error)
	FindByHash(ctx context.Context, userID int, fileHash string) (*domain.File, error)
	GetUserFile(ctx context.Context, userID int, fileID string) (*domain.File, error)
	Create(ctx context.Context, file *domain.File) error
	UpdateThumbnail(ctx context.Context, fileID, thumbnailURL string) (bool, error)
}

// UploadQuotaRepository is the minimal quota boundary required by uploads.
type UploadQuotaRepository interface {
	GetAvailableSpace(ctx context.Context, userID int) (int64, error)
	AddUsedSpace(ctx context.Context, userID int, delta int64) error
}
