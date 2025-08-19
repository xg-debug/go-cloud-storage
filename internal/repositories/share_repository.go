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
	DeleteShare(shareID int) error
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

// DeleteShare 删除分享
func (r *shareRepo) DeleteShare(shareID int) error {
	return r.db.Delete(&models.Share{}, shareID).Error
}
