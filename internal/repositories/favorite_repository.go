package repositories

import (
	"go-cloud-storage/internal/models"
	"gorm.io/gorm"
)

type FavoriteRepository interface {
	AddToFavorite(favorite *models.Favorite) error
	CancelFavorite(userId int, fileId string) error
	ListFavorite(userId, page, pageSize int) ([]*models.Favorite, int64, error)
	IsFavorited(userId int, fileId string) (bool, *models.Favorite)
	CountFavorites(userId int) (int64, error)
	Delete(tx *gorm.DB, fileId string) error
	DeleteBatch(tx *gorm.DB, fileIds []string) error
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
func (r *favoriteRepo) IsFavorited(userId int, fileId string) (bool, *models.Favorite) {
	var count int64
	var favorite models.Favorite
	r.db.Model(&models.Favorite{}).Where("user_id = ? AND file_id = ?", userId, fileId).First(&favorite).Count(&count)
	if count > 0 {
		return true, &favorite
	} else {
		return false, nil
	}
}

// CountFavorites 统计收藏数量
func (r *favoriteRepo) CountFavorites(userId int) (int64, error) {
	var count int64
	err := r.db.Model(&models.Favorite{}).Where("user_id = ?", userId).Count(&count).Error
	return count, err
}

func (r *favoriteRepo) Delete(tx *gorm.DB, fileId string) error {
	db := r.db // 默认使用非事务DB连接

	if tx != nil {
		// 如果传入了事务对象，则使用事务
		db = tx
	}
	return db.Delete(&models.Favorite{}, "fileId = ?", fileId).Error
}

func (r *favoriteRepo) DeleteBatch(tx *gorm.DB, fileIds []string) error {
	db := r.db // 默认使用非事务DB连接

	if tx != nil {
		// 如果传入了事务对象，则使用事务
		db = tx
	}
	return db.Where("file_id IN ?", fileIds).Delete(&models.Favorite{}).Error
}
