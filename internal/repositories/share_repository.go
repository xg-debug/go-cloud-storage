package repositories

import (
	"go-cloud-storage/internal/models"
	"time"

	"gorm.io/gorm"
)

type ShareRepository interface {
	CountSharedFiles(userId int) (int64, error)
	CreateShare(share *models.Share) (*models.Share, error)
	GetUserShares(userID int) ([]*models.Share, error)
	GetShareByID(shareID int) (*models.Share, error)
	GetShareByToken(token string) (*models.Share, error)
	UpdateShareExpireTime(shareID int, expireTime *time.Time) error
	Delete(tx *gorm.DB, shareID int) error
	DeleteBatch(tx *gorm.DB, fileIds []string) error
	IsShared(fileId string) (bool, *models.Share)
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
	err := r.db.Where("share_token = ?", token).First(&share).Error
	if err != nil {
		return nil, err
	}
	return &share, nil
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
	var count int64
	var share models.Share
	r.db.Model(&models.Share{}).Where("file_id = ?", fileId).First(&share).Count(&count)
	if count == 1 {
		return true, &share
	} else {
		return false, nil
	}
}
