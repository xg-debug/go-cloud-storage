package repositories

import (
	"errors"

	"go-cloud-storage/backend/internal/models"
	"gorm.io/gorm"
)

type StorageQuotaRepository interface {
	GetByUserID(userID int) (*models.StorageQuota, error)
	Create(quota *models.StorageQuota) error
	UpdateUsedSpace(tx *gorm.DB, userID int, deltaSize int64) error
	EnsureUserQuota(userID int) error
	GetAvailableSpace(userId int) (int64, error)
}

type storageQuotaRepo struct {
	db *gorm.DB
}

func NewStorageQuotaRepository(db *gorm.DB) StorageQuotaRepository {
	return &storageQuotaRepo{db: db}
}

// GetByUserID 根据用户ID 获取存储配额 (总空间大小)
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

// UpdateUsedSpace 更新已使用空间（原子操作，防止超配额；释放空间时向下钳制为 0）
func (r *storageQuotaRepo) UpdateUsedSpace(tx *gorm.DB, userID int, deltaSize int64) error {
	db := r.db
	if tx != nil {
		db = tx
	}

	query := db.Model(&models.StorageQuota{}).Where("user_id = ?", userID)
	updateExpr := gorm.Expr("GREATEST(used + ?, 0)", deltaSize)
	if deltaSize > 0 {
		query = query.Where("used + ? <= total", deltaSize)
		updateExpr = gorm.Expr("used + ?", deltaSize)
	}

	result := query.Update("used", updateExpr)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		if deltaSize < 0 {
			// 负增量且无配额记录（或已钳制为 0）时视为成功：没有可回退的配额
			return nil
		}
		return errors.New("存储空间不足")
	}
	return nil
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

func (r *storageQuotaRepo) GetAvailableSpace(userId int) (int64, error) {
	var quota models.StorageQuota
	err := r.db.Where("user_id = ?", userId).First(&quota).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, gorm.ErrRecordNotFound
		}
		return 0, err
	}

	// 计算剩余空间
	availableSpace := quota.Total - quota.Used

	// 确保返回的值非负
	if availableSpace < 0 {
		return 0, nil
	}

	return availableSpace, nil
}
