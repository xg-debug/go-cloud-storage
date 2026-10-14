package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	contracts "go-cloud-storage/backend/pkg/contracts"
	domain "go-cloud-storage/backend/services/file-service/internal/domain"
)

// EnsureOwner provisions only file-owned data and is safe to repeat after crashes.
func EnsureOwner(db *gorm.DB, quotaGB int) func(context.Context, contracts.Identity) error {
	return func(ctx context.Context, id contracts.Identity) error {
		if id.UserID <= 0 || id.RootID == "" {
			return errors.New("invalid owner")
		}
		return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			q := domain.StorageQuota{UserID: id.UserID, Total: int64(quotaGB) * 1024 * 1024 * 1024, CreatedAt: time.Now(), UpdatedAt: time.Now()}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&q).Error; err != nil {
				return err
			}
			root := domain.File{Id: id.RootID, UserId: id.UserID, Name: "/", IsDir: true, ParentId: sql.NullString{}, CreatedAt: time.Now(), UpdatedAt: time.Now()}
			return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&root).Error
		})
	}
}
