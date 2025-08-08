package repositories

import (
	"errors"
	"go-cloud-storage/internal/models"
	"gorm.io/gorm"
	"time"
)

type FavoriteRepository interface {
	AddToFavorite(userId int, fileId string) error
	getFileByID(tx *gorm.DB, fileId string) (*models.File, error)
	CancelFavorite(userId int, fileId *string) error
	ListFavorite(userId int, page, pageSize int) ([]*models.Favorite, int64, error)
	IsFavorited(userId int, fileId string) (bool, error)
	CountFavorites(userId int) (int64, error)
}

type favoriteRepo struct {
	db *gorm.DB
}

func NewFavoriteRepository(db *gorm.DB) FavoriteRepository {
	return &favoriteRepo{db: db}
}

type FavoriteDTO struct {
	FileID    string    `json:"file_id"`
	CreatedAt time.Time `json:"created_at"`
	Name      string    `json:"name"`
	IsDir     bool      `json:"is_dir"` // true=文件夹，false=文件
}

// AddToFavorite 添加收藏
func (r *favoriteRepo) AddToFavorite(userId int, fileId string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1.检查目标是否是有效文件
		_, err := r.getFileByID(tx, fileId)
		if err != nil {
			return err
		}
		// 2.检查是否已经收藏
		if exist, _ := r.IsFavorited(userId, fileId); exist {
			return errors.New("该文件已收藏")
		}
		// 3.创建收藏记录
		favorite := &models.Favorite{
			UserId: userId,
			FileId: fileId,
		}
		return tx.Create(favorite).Error
	})
}

// getFileByID 获取文件信息（用于校验）
func (r *favoriteRepo) getFileByID(tx *gorm.DB, fileId string) (*models.File, error) {
	var file models.File
	err := tx.Where("id = ?", fileId).First(&file).Error
	if err != nil {
		return nil, errors.New("文件不存在")
	}
	return &file, nil
}

// CancelFavorite 取消收藏
func (r *favoriteRepo) CancelFavorite(userId int, fileId *string) error {
	return r.db.Where("user_id = ? AND file_id = ?", userId, fileId).Delete(&models.Favorite{}).Error
}

// ListFavorite 获取收藏列表（分页）
func (r *favoriteRepo) ListFavorite(userId int, page, pageSize int) ([]*models.Favorite, int64, error) {
	var count int64
	err := r.db.Model(&models.Favorite{}).
		Where("user_id = ?", userId).
		Count(&count).Error
	if err != nil {
		return nil, 0, err
	}

	var dtos []*models.Favorite
	err = r.db.Raw(`
		SELECT f.file_id, f.created_at, file.name, file.is_dir
		FROM favorite f
		JOIN file ON f.file_id = file.id
		WHERE f.user_id = ?
		ORDER BY f.created_at DESC 
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
