package ports

import (
	"context"

	"go-cloud-storage/backend/internal/models"
)

// FileReadRepository is the minimal metadata boundary required by download and
// preview streaming use cases.
type FileReadRepository interface {
	GetUserFile(ctx context.Context, userID int, fileID string) (*models.File, error)
}
