package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"go-cloud-storage/internal/models"
	"go-cloud-storage/utils"
	"gorm.io/gorm"
	"time"
)

type FileRepository interface {
	InitFolder(folder *models.File) error
	GetFiles(ctx context.Context, userId int, parentId string, page int, pageSize int) ([]models.File, int64, error)
	CreateFile(file *models.File) error
	GetFileById(id string) (*models.File, error)
	GetUserFileByID(userId int, fileId string) (*models.File, error)
	UpdateFile(file *models.File, updateFields map[string]interface{}) error
	UpdateFileNameById(fileId, newName string) error
	SoftDeleteFile(userId int, fileId string) error
	HardDeleteFile(fileId string) error
	CheckDuplicateName(userId int, parentId, name string) (bool, error)
	GetFilePath(userId int, fileId string) (string, error)
	RestoreFile(userId int, fileId string) error
	RestoreFolder(folderId string) error

	CreateFolder(userId int, folderName string, parentId string) (*models.File, error)
	ListUserFiles(userId int, parentId *string, page, pageSize int) ([]*models.File, error)
	FindByHash(hash string) (*models.File, error)
	RenameFile(userId int, fileId, newName string) error
	MoveFile(userId int, fileId, newParentId string) error
	GetUserUsedStorage(userId int) (int64, error)
	GetFileCount(userId int) (int64, error)
}

type fileRepo struct {
	db *gorm.DB
}

func NewFileRepository(db *gorm.DB) FileRepository {
	return &fileRepo{db: db}
}

func (r *fileRepo) InitFolder(folder *models.File) error {
	return r.db.Create(folder).Error
}

// 基础文件操作方法

func (r *fileRepo) GetFiles(ctx context.Context, userId int, parentId string, page int, pageSize int) ([]models.File, int64, error) {
	var files []models.File
	var total int64

	query := r.db.WithContext(ctx).Where("user_id = ?", userId)
	query = query.Where("is_deleted = ?", false)
	if parentId != "" {
		query = query.Where("parent_id = ?", parentId)
	} else {
		query = query.Where("parent_id IS NULL")
	}

	if err := query.Model(&models.File{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := query.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&files).Error; err != nil {
		return nil, 0, err
	}

	return files, total, nil
}

// CreateFile 创建文件或文件夹：根据file.isDir
func (r *fileRepo) CreateFile(file *models.File) error {
	return r.db.Create(file).Error
}

func (r *fileRepo) GetFilesByParentId(parentId string) ([]models.File, error) {
	var files []models.File
	err := r.db.Where("parent_id = ?", parentId).Find(&files).Error
	return files, err
}

// GetFileById 根据id获取文件（包括软删除的文件）
func (r *fileRepo) GetFileById(id string) (*models.File, error) {
	var file models.File
	err := r.db.Where("id = ?", id).First(&file).Error
	return &file, err
}

// GetUserFileByID 获取用户文件(不包括软删除的文件)
func (r *fileRepo) GetUserFileByID(userId int, fileId string) (*models.File, error) {
	var file models.File
	err := r.db.Where("id = ? AND user_id = ? AND is_deleted = ?", fileId, userId, false).First(&file).Error
	return &file, err
}

// UpdateFile 更新文件信息
func (r *fileRepo) UpdateFile(file *models.File, updateFields map[string]interface{}) error {
	return r.db.Model(file).Updates(updateFields).Error
}

func (r *fileRepo) UpdateFileNameById(fileId, newName string) error {
	return r.db.Model(&models.File{}).Where("id = ?", fileId).Update("name", newName).Error
}

// SoftDeleteFile 软删除文件
func (r *fileRepo) SoftDeleteFile(userId int, fileId string) error {
	return r.db.Model(&models.File{}).
		Where("id = ? AND user_id = ?", fileId, userId).
		Updates(map[string]interface{}{
			"is_deleted": 1,
		}).Error
}

// HardDeleteFile 硬删除文件
func (r *fileRepo) HardDeleteFile(fileId string) error {
	return r.db.Where("id = ?", fileId).Delete(&models.File{}).Error
}

// CheckDuplicateName 检查同级目录下是否存在同名文件
func (r *fileRepo) CheckDuplicateName(userId int, parentId, name string) (bool, error) {
	var count int64
	err := r.db.Model(&models.File{}).
		Where("user_id = ? AND parent_id = ? AND name = ? AND is_deleted = ?", userId, parentId, name, false).
		Count(&count).Error
	return count > 0, err
}

// GetFilePath 获取文件完整路径
func (r *fileRepo) GetFilePath(userId int, fileId string) (string, error) {
	// 使用递归CTE查询完整路径（MySQL 8.0+ / PostgreSQL）
	var path string
	// 实现递归查询逻辑...
	return path, nil
}

// RestoreFile 恢复软删除文件
func (r *fileRepo) RestoreFile(userId int, fileId string) error {
	return r.db.Model(&models.File{}).Where("id = ? AND user_id = ?", fileId, userId).
		Update("is_deleted", false).Error
}

func (r *fileRepo) RestoreFolder(folderId string) error {
	// 递归恢复文件夹及其所有内容
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 恢复文件夹本身
		if err := tx.Model(&models.File{}).
			Where("id = ?", folderId).
			Update("is_deleted", false).Error; err != nil {
			return err
		}

		// 递归恢复所有子文件和子文件夹
		// 使用闭包实现递归CTE
		recursiveQuery := `
            WITH RECURSIVE children AS (
                SELECT id FROM files WHERE parent_id = ?
                UNION ALL
                SELECT f.id FROM files f
                INNER JOIN children c ON f.parent_id = c.id
            )
            UPDATE files SET is_deleted = false
            WHERE id IN (SELECT id FROM children)
        `

		return tx.Exec(recursiveQuery, folderId).Error
	})
}

// 目录操作方法

// CreateFolder 创建目录
func (r *fileRepo) CreateFolder(userId int, folderName string, parentId string) (*models.File, error) {
	var pId sql.NullString
	if parentId == "" {
		pId = sql.NullString{Valid: false} // NULL
	} else {
		pId = sql.NullString{String: parentId, Valid: true} // 有值
	}
	fmt.Println("pId: ", pId)
	folder := &models.File{
		Id:        utils.NewUUID(), // 需要实现生成Id的函数
		UserId:    userId,
		Name:      folderName,
		IsDir:     true,
		ParentId:  pId,
		Size:      0,
		CreatedAt: time.Now(),
	}
	err := r.db.Create(folder).Error
	return folder, err
}

// 查询方法

// ListUserFiles 列出用户文件(不包括软删除的文件)
func (r *fileRepo) ListUserFiles(userId int, parentId *string, page, pageSize int) ([]*models.File, error) {
	var files []*models.File
	query := r.db.Where("user_id = ? AND is_deleted = ?", userId, false)
	if parentId == nil || *parentId == "" {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", parentId)
	}
	err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&files).Error
	return files, err
}

// FindByHash 根据文件哈希查找文件（用于去重）
func (r *fileRepo) FindByHash(hash string) (*models.File, error) {
	var file models.File
	err := r.db.Where("file_hash = ? AND is_deleted = ?", hash, false).First(&file).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &file, err
}

// 文件操作

// RenameFile 重命名文件/目录
func (r *fileRepo) RenameFile(userId int, fileId, newName string) error {
	return r.db.Model(&models.File{}).
		Where("id = ? AND user_id = ?", fileId, userId).
		Update("name", newName).Error
}

// MoveFile 移动文件
func (r *fileRepo) MoveFile(userId int, fileId, newParentId string) error {
	return r.db.Model(&models.File{}).
		Where("id = ? AND user_id = ?", fileId, userId).
		Update("parent_id = ?", newParentId).Error
}

// 统计

// GetUserUsedStorage 获取用户已用存储空间
func (r *fileRepo) GetUserUsedStorage(userId int) (int64, error) {
	var totalSize int64
	err := r.db.Model(&models.File{}).
		Where("user_id = ? AND is_deleted = ? AND is_dir = ?", userId, false, false).
		Select("COLALESCE(SUM(size), 0)").
		Scan(&totalSize).Error
	return totalSize, err
}

// GetFileCount 获取用户文件数量
func (r *fileRepo) GetFileCount(userId int) (int64, error) {
	var count int64

	err := r.db.Model(&models.File{}).
		Where("user_id = ? AND is_deleted = ? AND is_dir = ?", userId, false, false).
		Count(&count).Error
	return count, err
}
