package services

import (
	"context"
	"database/sql"
	"errors"
	"go-cloud-storage/internal/models"
	"go-cloud-storage/internal/pkg/utils"
	"go-cloud-storage/internal/repositories"
	"gorm.io/gorm"
	"time"
)

type FileItem struct {
	Id           string `json:"id"`
	Name         string `json:"name"`
	IsDir        bool   `json:"is_dir"`
	Size         int64  `json:"size"`
	SizeStr      string `json:"size_str"`
	Extension    string `json:"extension"`
	Modified     string `json:"modified"`
	FileURL      string `json:"file_url"`
	ThumbnailURL string `json:"thumbnail_url"`
}

type FileService interface {
	GetFiles(ctx context.Context, userId int, parentId string, page int, pageSize int) ([]FileItem, int64, error)
	CreateFolder(userId int, folderName string, parentId string) (*models.File, error)
	UploadFile(fileName, extension string, size int64, parentId string) (*models.File, error)
	Rename(userId int, fileId, newName string) error
	Delete(fileId string, userId int) error
	CreateFileInfo(file *models.File) error
}

type fileService struct {
	db       *gorm.DB
	fileRepo repositories.FileRepository
}

func NewFileService(db *gorm.DB, repo repositories.FileRepository) FileService {
	return &fileService{db: db, fileRepo: repo}
}

func (s *fileService) GetFiles(ctx context.Context, userId int, parentId string, page int, pageSize int) ([]FileItem, int64, error) {
	files, total, err := s.fileRepo.GetFiles(ctx, userId, parentId, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	var fileList []FileItem
	for _, file := range files {
		fileList = append(fileList, FileItem{
			Id:           file.Id,
			Name:         file.Name,
			IsDir:        file.IsDir,
			Size:         file.Size,
			SizeStr:      file.SizeStr,
			Extension:    file.FileExtension,
			Modified:     file.UpdatedAt.Format("2006-01-02 15:04:05"),
			FileURL:      file.FileURL,
			ThumbnailURL: file.ThumbnailURL,
		})
	}
	return fileList, total, nil
}

func (s *fileService) CreateFolder(userId int, folderName string, parentId string) (*models.File, error) {
	folder, err := s.fileRepo.CreateFolder(userId, folderName, parentId)
	if err != nil {
		return nil, err
	}
	return folder, nil
}

func (s *fileService) UploadFile(fileName, extension string, size int64, parentId string) (*models.File, error) {
	var pId sql.NullString
	if parentId == "" {
		pId = sql.NullString{Valid: false} // NULL
	} else {
		pId = sql.NullString{String: parentId, Valid: true} // 有值
	}

	file := &models.File{
		Id:            utils.NewUUID(),
		Name:          fileName,
		Size:          size,
		IsDir:         false,
		FileExtension: extension,
		FileHash:      "hash",
		ParentId:      pId,
		IsDeleted:     false,
		CreatedAt:     time.Now(),
	}
	err := s.fileRepo.CreateFile(file)
	return file, err
}

func (s *fileService) Rename(userId int, fileId, newName string) error {
	// 检查是否存在
	file, err := s.fileRepo.GetFileById(fileId)
	if err != nil {
		return err
	}
	// 重名检查
	exists, err := s.fileRepo.CheckDuplicateName(userId, file.ParentId.String, newName)
	if err != nil {
		return err
	}
	typeStr := "文件夹"
	if file.IsDir == false {
		typeStr = "文件"
	}
	if exists {
		return errors.New("该目录下已有同名的" + typeStr)
	}
	return s.fileRepo.UpdateFileNameById(fileId, newName)
}

func (s *fileService) Delete(fileId string, userId int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1.软删除文件
		if err := s.fileRepo.SoftDeleteFile(tx, userId, fileId); err != nil {
			return err
		}
		// 2.构造回收站记录
		recycleEntry := &models.RecycleBin{
			FileId:    fileId,
			UserId:    userId,
			DeletedAt: time.Now(),
			ExpireAt:  time.Now().Add(10 * 24 * time.Hour),
		}
		if err := s.fileRepo.AddToRecycle(tx, recycleEntry); err != nil {
			return err
		}
		// 如果到这里都没报错，事务会自动提交
		return nil
	})
}

func (s *fileService) CreateFileInfo(file *models.File) error {
	return s.fileRepo.CreateFile(file)
}
