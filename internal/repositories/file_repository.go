package repositories

import (
	"context"
	"database/sql"
	"errors"
	"go-cloud-storage/internal/models"
	"go-cloud-storage/internal/pkg/utils"
	"time"

	"gorm.io/gorm"
)

type FileRepository interface {
	InitFolder(folder *models.File) error
	GetFiles(ctx context.Context, userId int, parentId string, page int, pageSize int) ([]models.File, int64, error)
	GetFilesByCategory(ctx context.Context, userId int, fileType string, sortBy string, sortOrder string, page int, pageSize int) ([]models.File, int64, error)

	GetRecentFiles(userId int, since time.Time) ([]models.File, error)
	GetAllUserFiles(userId int) ([]models.File, error)

	CreateFile(file *models.File) error
	GetFileById(id string) (*models.File, error)
	GetFileByIds(fileIds []string) ([]models.File, error)
	GetUserFileByID(userId int, fileId string) (*models.File, error)
	GetFileByMD5(userId int, fileMD5 string) (*models.File, error)
	UpdateFile(file *models.File, updateFields map[string]interface{}) error
	UpdateFileNameById(fileId, newName string) error

	SoftDeleteFile(db *gorm.DB, userId int, fileId string) error
	AddToRecycle(db *gorm.DB, recycleEntry *models.RecycleBin) error
	DeletePermanent(db *gorm.DB, fileIds []string) error
	DeleteByUserId(db *gorm.DB, userId int) error

	MarkAsNotDeleted(db *gorm.DB, fileIds []string, userId *int) error

	CheckDuplicateName(userId int, parentId, name string) (bool, error)
	RestoreFolder(folderId string) error

	CreateFolder(userId int, folderName string, parentId string) (*models.File, error)
	ListUserFiles(userId int, parentId *string, page, pageSize int) ([]*models.File, error)
	FindByHash(hash string) (*models.File, error)
	RenameFile(userId int, fileId, newName string) error
	MoveFile(userId int, fileId, newParentId string) error
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

// GetFiles 获取用户文件列表
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
	if err := query.Order("is_dir desc, created_at desc").Offset(offset).Limit(pageSize).Find(&files).Error; err != nil {
		return nil, 0, err
	}

	return files, total, nil
}

// GetFilesByCategory 根据文件类型获取文件列表
func (r *fileRepo) GetFilesByCategory(ctx context.Context, userId int, fileType string, sortBy string, sortOrder string, page int, pageSize int) ([]models.File, int64, error) {
	var files []models.File
	var total int64

	query := r.db.WithContext(ctx).Where("user_id = ? AND is_dir = ? AND is_deleted = ?", userId, false, false)

	// 根据文件类型筛选
	switch fileType {
	case "image":
		query = query.Where("file_extension IN ('jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp', 'svg')")
	case "video":
		query = query.Where("file_extension IN ('mp4', 'avi', 'mov', 'wmv', 'flv', 'webm', 'mkv')")
	case "audio":
		query = query.Where("file_extension IN ('mp3', 'wav', 'flac', 'aac', 'ogg', '.m4a')")
	case "document":
		query = query.Where("file_extension IN ('txt', 'md', 'pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx')")
	}

	// 计算总数
	if err := query.Model(&models.File{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 排序
	if sortOrder == "asc" {
		query = query.Order(sortBy + " asc")
	} else {
		query = query.Order(sortBy + " desc")
	}

	// 分页
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&files).Error; err != nil {
		return nil, 0, err
	}

	return files, total, nil
}

func (r *fileRepo) GetRecentFiles(userId int, since time.Time) ([]models.File, error) {
	var files []models.File
	if err := r.db.Where("user_id = ? AND is_dir = ? AND is_deleted = ? AND updated_at >= ?", userId, false, false, since).Order("updated_at DESC").Find(&files).Error; err != nil {
		return nil, err
	}
	return files, nil
}

// CreateFile 创建文件或文件夹：根据file.isDir
func (r *fileRepo) CreateFile(file *models.File) error {
	return r.db.Create(file).Error
}

// GetFileById 根据id获取文件（包括软删除的文件）
func (r *fileRepo) GetFileById(id string) (*models.File, error) {
	var file models.File
	err := r.db.Where("id = ?", id).First(&file).Error
	return &file, err
}

func (r *fileRepo) GetFileByIds(fileIds []string) ([]models.File, error) {
	var res []models.File
	err := r.db.Where("id IN ?", fileIds).Find(&res).Error
	return res, err
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
func (r *fileRepo) SoftDeleteFile(db *gorm.DB, userId int, fileId string) error {
	return db.Model(&models.File{}).
		Where("id = ? AND user_id = ?", fileId, userId).
		Updates(map[string]interface{}{
			"is_deleted": 1,
		}).Error
}

func (r *fileRepo) DeletePermanent(db *gorm.DB, fileIds []string) error {
	return db.Where("id IN ?", fileIds).Delete(&models.File{}).Error
}

func (r *fileRepo) DeleteByUserId(db *gorm.DB, userId int) error {
	return db.Where("user_id = ? AND is_deleted = ?", userId, true).Delete(&models.File{}).Error
}

func (r *fileRepo) AddToRecycle(db *gorm.DB, recycleEntry *models.RecycleBin) error {
	return db.Create(&recycleEntry).Error
}

// MarkAsNotDeleted 恢复文件（单个/多个） userId 传 nil 表示不限制用户
func (r *fileRepo) MarkAsNotDeleted(db *gorm.DB, fileIds []string, userId *int) error {
	query := db.Model(&models.File{})
	if len(fileIds) > 0 {
		query = query.Where("id IN ?", fileIds)
	}
	if userId != nil {
		query = query.Where("user_id = ?", *userId)
	}
	return query.Updates(map[string]interface{}{"is_deleted": false}).Error
}

// CheckDuplicateName 检查同级目录下是否存在同名文件
func (r *fileRepo) CheckDuplicateName(userId int, parentId, name string) (bool, error) {
	var count int64
	err := r.db.Model(&models.File{}).
		Where("user_id = ? AND parent_id = ? AND name = ? AND is_deleted = ?", userId, parentId, name, false).
		Count(&count).Error
	return count > 0, err
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
	folder := &models.File{
		Id:        utils.NewUUID(), // 需要实现生成Id的函数
		UserId:    userId,
		Name:      folderName,
		IsDir:     true,
		ParentId:  pId,
		Size:      0,
		SizeStr:   "-",
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
		Update("parent_id", newParentId).Error
}

// GetAllUserFiles 获取用户所有文件（不包括已删除的）
func (r *fileRepo) GetAllUserFiles(userId int) ([]models.File, error) {
	var files []models.File
	err := r.db.Where("user_id = ? AND is_deleted = ?", userId, false).Find(&files).Error
	return files, err
}

// GetFileByMD5 根据MD5查找用户文件（用于秒传）
func (r *fileRepo) GetFileByMD5(userId int, fileMD5 string) (*models.File, error) {
	var file models.File
	err := r.db.Where("user_id = ? AND file_hash = ? AND is_deleted = ?", userId, fileMD5, false).First(&file).Error
	return &file, err
}
