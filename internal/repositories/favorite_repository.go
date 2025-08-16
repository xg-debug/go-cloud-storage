package repositories

import (
	"go-cloud-storage/internal/models"
	"gorm.io/gorm"
)

type FavoriteRepository interface {
	AddToFavorite(favorite *models.Favorite) error
	CancelFavorite(userId int, fileId string) error
	ListFavorite(userId, page, pageSize int) ([]*models.Favorite, int64, error)
	IsFavorited(userId int, fileId string) (bool, error)
	CountFavorites(userId int) (int64, error)
}

type favoriteRepo struct {
	db *gorm.DB
}

func NewFavoriteRepository(db *gorm.DB) FavoriteRepository {
	return &favoriteRepo{db: db}
}

// AddToFavorite 添加收藏
func (r *favoriteRepo) AddToFavorite(favorite *models.Favorite) error {
	return r.db.Create(favorite).Error
}

// CancelFavorite 取消收藏
func (r *favoriteRepo) CancelFavorite(userId int, fileId string) error {
	return r.db.Where("user_id = ? AND file_id = ?", userId, fileId).Delete(&models.Favorite{}).Error
}

// ListFavorite 获取收藏列表（分页）
func (r *favoriteRepo) ListFavorite(userId, page, pageSize int) ([]*models.Favorite, int64, error) {
	var count int64
	err := r.db.Model(&models.Favorite{}).
		Where("user_id = ?", userId).
		Count(&count).Error
	if err != nil {
		return nil, 0, err
	}

	var dtos []*models.Favorite
	err = r.db.Raw(`
		SELECT file_id, created_at
		FROM favorite
		WHERE user_id = ?
		ORDER BY created_at DESC 
		LIMIT ? OFFSET ?`, userId, pageSize, (page-1)*pageSize).Scan(&dtos).Error
	return dtos, count, err
}

// IsFavorited 检查是否已收藏
func (r *favoriteRepo) IsFavorited(userId int, fileId string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Favorite{}).Where("user_id = ? AND file_id = ?", userId, fileId).Count(&count).Error
	return count > 0, err
}

// CountFavorites 统计收藏数量
func (r *favoriteRepo) CountFavorites(userId int) (int64, error) {
	var count int64
	err := r.db.Model(&models.Favorite{}).Where("user_id = ?", userId).Count(&count).Error
	return count, err
}
