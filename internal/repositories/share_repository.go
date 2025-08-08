package repositories

import (
	"go-cloud-storage/internal/models"
	"gorm.io/gorm"
)

type ShareRepository interface {
	CreateShare(share *models.ShareRecord) error
	GetShareById(id int) (*models.ShareRecord, error)
	GetShareByUser(userId int) ([]models.ShareRecord, error)
	DeleteShare(id int) error
	GetShareByLink(link string) (*models.ShareRecord, error)
}

type shareRepo struct {
	db *gorm.DB
}

func NewShareRepository(db *gorm.DB) ShareRepository {
	return &shareRepo{db: db}
}

// CreateShare 创建分享记录
func (r *shareRepo) CreateShare(share *models.ShareRecord) error {
	return r.db.Create(share).Error
}

func (r *shareRepo) GetShareById(id int) (*models.ShareRecord, error) {
	var share models.ShareRecord
	err := r.db.First(&share, id).Error
	return &share, err
}

func (r *shareRepo) GetShareByUser(userId int) ([]models.ShareRecord, error) {
	var shares []models.ShareRecord
	err := r.db.Where("owner_id = ?", userId).Find(&shares).Error
	return shares, err
}

func (r *shareRepo) DeleteShare(id int) error {
	return r.db.Delete(&models.ShareRecord{}, id).Error
}

func (r *shareRepo) GetShareByLink(link string) (*models.ShareRecord, error) {
	var share models.ShareRecord
	err := r.db.Where("share_link = ?", link).First(&share).Error
	return &share, err
}
