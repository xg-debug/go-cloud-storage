package ports

import (
	"context"

	domain "go-cloud-storage/backend/services/file-service/internal/domain"
)

// FileReadRepository is the minimal metadata boundary required by download and
// preview streaming use cases.
type FileReadRepository interface {
	GetUserFile(ctx context.Context, userID int, fileID string) (*domain.File, error)
}
