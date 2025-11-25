package repositories

import (
	"go-cloud-storage/internal/models"
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

// 获取可用的存储空间大小
func Get() {

}

// Create 创建存储配额记录
func (r *storageQuotaRepo) Create(quota *models.StorageQuota) error {
	return r.db.Create(quota).Error
}

// UpdateUsedSpace 更新已使用空间
// tx：可选的事务对象。如果为 nil，则使用 r.db
func (r *storageQuotaRepo) UpdateUsedSpace(tx *gorm.DB, userID int, deltaSize int64) error {
	db := r.db // 默认使用非事务DB连接

	if tx != nil {
		// 如果传入了事务对象，则使用事务
		db = tx
	}

	return db.Model(&models.StorageQuota{}).
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
