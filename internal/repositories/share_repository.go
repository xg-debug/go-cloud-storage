package repositories

import (
	"go-cloud-storage/internal/models"
	"gorm.io/gorm"
	"time"
)

type ShareRepository interface {
	CountSharedFiles(userId int) (int64, error)
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
