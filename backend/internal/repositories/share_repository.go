package repositories

import (
	"go-cloud-storage/backend/internal/models"
	"time"

	"gorm.io/gorm"
)

type ShareRepository interface {
	CountSharedFiles(userId int) (int64, error)
	CreateShare(share *models.Share) (*models.Share, error)
	GetUserShares(userID int) ([]*models.Share, error)
	GetShareByID(shareID int) (*models.Share, error)
	GetShareByToken(token string) (*models.Share, error)
	IncrementAccessCount(shareID int) error
	IncrementDownloadCount(shareID int) error
	UpdateShareExpireTime(shareID int, expireTime *time.Time) error
	Delete(tx *gorm.DB, shareID int) error
	DeleteBatch(tx *gorm.DB, fileIds []string) error
	IsShared(fileId string) (bool, *models.Share)
	UpdateShareInfo(shareID int, code *string, expireTime *time.Time) error
}

type shareRepo struct {
	db *gorm.DB
}

func NewShareRepository(db *gorm.DB) ShareRepository {
	return &shareRepo{db: db}
}

func (r *shareRepo) CountSharedFiles(userId int) (int64, error) {
	var count int64
	err := r.db.Where("user_id = ? AND expire_time < ?", userId, time.Now()).Model(&models.Share{}).Count(&count).Error
	return count, err
}

// CreateShare 创建分享
func (r *shareRepo) CreateShare(share *models.Share) (*models.Share, error) {
	err := r.db.Create(share).Error
	if err != nil {
		return nil, err
	}
	return share, nil
}

// GetUserShares 获取用户的分享列表
func (r *shareRepo) GetUserShares(userID int) ([]*models.Share, error) {
	var shares []*models.Share
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&shares).Error
	return shares, err
}

// GetShareByID 根据ID获取分享
func (r *shareRepo) GetShareByID(shareID int) (*models.Share, error) {
	var share models.Share
	err := r.db.Where("id = ?", shareID).First(&share).Error
	if err != nil {
		return nil, err
	}
	return &share, nil
}

// GetShareByToken 根据token获取分享
func (r *shareRepo) GetShareByToken(token string) (*models.Share, error) {
	var share models.Share
	err := r.db.Where("share_token = ? AND is_deleted = 0", token).First(&share).Error
	if err != nil {
		return nil, err
	}
	return &share, nil
}

func (r *shareRepo) IncrementAccessCount(shareID int) error {
	return r.db.Model(&models.Share{}).
		Where("id = ?", shareID).
		UpdateColumn("access_count", gorm.Expr("access_count + 1")).Error
}

func (r *shareRepo) IncrementDownloadCount(shareID int) error {
	return r.db.Model(&models.Share{}).
		Where("id = ?", shareID).
		UpdateColumn("download_count", gorm.Expr("download_count + 1")).Error
}

// UpdateShareExpireTime 更新分享过期时间
func (r *shareRepo) UpdateShareExpireTime(shareID int, expireTime *time.Time) error {
	return r.db.Model(&models.Share{}).Where("id = ?", shareID).Update("expire_time", expireTime).Error
}

// Delete 删除分享
func (r *shareRepo) Delete(tx *gorm.DB, shareID int) error {
	db := r.db // 默认使用非事务DB连接

	if tx != nil {
		// 如果传入了事务对象，则使用事务
		db = tx
	}
	return db.Delete(&models.Share{}, shareID).Error
}

func (r *shareRepo) DeleteBatch(tx *gorm.DB, fileIds []string) error {
	db := r.db // 默认使用非事务DB连接

	if tx != nil {
		// 如果传入了事务对象，则使用事务
		db = tx
	}
	return db.Where("file_id IN ?", fileIds).Delete(&models.Share{}).Error
}

func (r *shareRepo) IsShared(fileId string) (bool, *models.Share) {
	var share models.Share
	err := r.db.Where("file_id = ? AND is_deleted = 0", fileId).First(&share).Error
	if err == nil {
		return true, &share
	}
	return false, nil
}

// UpdateShareInfo 更新分享信息
func (r *shareRepo) UpdateShareInfo(shareID int, code *string, expireTime *time.Time) error {
	updates := map[string]interface{}{
		"extraction_code": code,
		"expire_time":     expireTime,
	}
	return r.db.Model(&models.Share{}).Where("id = ?", shareID).Updates(updates).Error
}
