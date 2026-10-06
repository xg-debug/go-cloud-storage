package persistence

import (
	"context"
	"errors"

	"go-cloud-storage/backend/internal/models"
	"go-cloud-storage/backend/internal/ports"

	"gorm.io/gorm"
)

type GormUploadFileRepository struct {
	db *gorm.DB
}

func NewGormUploadFileRepository(db *gorm.DB) *GormUploadFileRepository {
	return &GormUploadFileRepository{db: db}
}

func (r *GormUploadFileRepository) CheckDuplicateName(ctx context.Context, userID int, parentID, name string) (bool, error) {
	var count int64
	query := dbFromContext(ctx, r.db).Model(&models.File{}).
		Where("user_id = ? AND name = ? AND is_deleted = ?", userID, name, false)
	if parentID == "" {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", parentID)
	}
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *GormUploadFileRepository) FindByHash(ctx context.Context, userID int, fileHash string) (*models.File, error) {
	var file models.File
	err := dbFromContext(ctx, r.db).
		Where("user_id = ? AND file_hash = ? AND is_dir = ? AND is_deleted = ?", userID, fileHash, false, false).
		First(&file).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func (r *GormUploadFileRepository) GetUserFile(ctx context.Context, userID int, fileID string) (*models.File, error) {
	var file models.File
	err := dbFromContext(ctx, r.db).
		Where("id = ? AND user_id = ? AND is_deleted = ?", fileID, userID, false).
		First(&file).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func (r *GormUploadFileRepository) Create(ctx context.Context, file *models.File) error {
	return dbFromContext(ctx, r.db).Create(file).Error
}

func (r *GormUploadFileRepository) UpdateThumbnail(ctx context.Context, fileID, thumbnailURL string) (bool, error) {
	result := dbFromContext(ctx, r.db).Model(&models.File{}).
		Where("id = ? AND is_deleted = ?", fileID, false).
		Update("thumbnail_url", thumbnailURL)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

type GormUploadQuotaRepository struct {
	db *gorm.DB
}

func NewGormUploadQuotaRepository(db *gorm.DB) *GormUploadQuotaRepository {
	return &GormUploadQuotaRepository{db: db}
}

func (r *GormUploadQuotaRepository) GetAvailableSpace(ctx context.Context, userID int) (int64, error) {
	var quota models.StorageQuota
	if err := dbFromContext(ctx, r.db).Where("user_id = ?", userID).First(&quota).Error; err != nil {
		return 0, err
	}
	available := quota.Total - quota.Used
	if available < 0 {
		return 0, nil
	}
	return available, nil
}

func (r *GormUploadQuotaRepository) AddUsedSpace(ctx context.Context, userID int, delta int64) error {
	db := dbFromContext(ctx, r.db)
	query := db.Model(&models.StorageQuota{}).Where("user_id = ?", userID)
	updateExpr := gorm.Expr("GREATEST(used + ?, 0)", delta)
	if delta > 0 {
		query = query.Where("used + ? <= total", delta)
		updateExpr = gorm.Expr("used + ?", delta)
	}
	result := query.Update("used", updateExpr)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		if delta < 0 {
			return nil
		}
		return errors.New("存储空间不足")
	}
	return nil
}

var _ ports.UploadFileRepository = (*GormUploadFileRepository)(nil)
var _ ports.UploadQuotaRepository = (*GormUploadQuotaRepository)(nil)
