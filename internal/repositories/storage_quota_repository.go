package repositories

import (
	"go-cloud-storage/internal/models"
	"gorm.io/gorm"
)

type StorageQuotaRepository interface {
	GetByUserID(userID int) (*models.StorageQuota, error)
	Create(quota *models.StorageQuota) error
	UpdateUsedSpace(userID int, deltaSize int64) error
	EnsureUserQuota(userID int) error
}

type storageQuotaRepo struct {
	db *gorm.DB
}

func NewStorageQuotaRepository(db *gorm.DB) StorageQuotaRepository {
	return &storageQuotaRepo{db: db}
}

// GetByUserID 根据用户ID获取存储配额
func (r *storageQuotaRepo) GetByUserID(userID int) (*models.StorageQuota, error) {
	var quota models.StorageQuota
	err := r.db.Where("user_id = ?", userID).First(&quota).Error
	if err != nil {
		return nil, err
	}
	return &quota, nil
}

// Create 创建存储配额记录
func (r *storageQuotaRepo) Create(quota *models.StorageQuota) error {
	return r.db.Create(quota).Error
}

// UpdateUsedSpace 更新已使用空间
func (r *storageQuotaRepo) UpdateUsedSpace(userID int, deltaSize int64) error {
	return r.db.Model(&models.StorageQuota{}).
		Where("user_id = ?", userID).
		Update("used", gorm.Expr("used + ?", deltaSize)).Error
	// "used + ?" 表示将 used 字段的值增加 deltaSize
}

// EnsureUserQuota 确保用户有存储配额记录
func (r *storageQuotaRepo) EnsureUserQuota(userID int) error {
	var quota models.StorageQuota
	err := r.db.Where("user_id = ?", userID).First(&quota).Error

	if err == gorm.ErrRecordNotFound {
		// 如果不存在配额记录，创建默认配额
		quota = models.StorageQuota{
			UserID: userID,
			Total:  10737418240, // 10GB
			Used:   0,
		}
		return r.db.Create(&quota).Error
	}

	return err
}
