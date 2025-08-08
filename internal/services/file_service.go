package services

import (
	"context"
	"database/sql"
	"errors"
	"go-cloud-storage/internal/models"
	"go-cloud-storage/internal/pkg/oss"
	"go-cloud-storage/internal/repositories"
	"go-cloud-storage/utils"
	"time"
)

type FileItem struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	IsDir     bool   `json:"is_dir"`
	Size      int64  `json:"size"`
	Extension string `json:"extension"`
	Modified  string `json:"modified"`
}

type FileService interface {
	GetFiles(ctx context.Context, userId int, parentId string, page int, pageSize int) ([]FileItem, int64, error)
	CreateFolder(userId int, folderName string, parentId string) (*models.File, error)
	UploadFile(fileName, extension string, size int64, parentId string) (*models.File, error)
	Rename(userId int, fileId, newName string) error
	Delete(fileId string, userId int) error
	CreateFromFileInfo(file *oss.FileInfo) error
}

type fileService struct {
	fileRepo repositories.FileRepository
}

func NewFileService(repo repositories.FileRepository) FileService {
	return &fileService{fileRepo: repo}
}

func (s *fileService) GetFiles(ctx context.Context, userId int, parentId string, page int, pageSize int) ([]FileItem, int64, error) {
	files, total, err := s.fileRepo.GetFiles(ctx, userId, parentId, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	var fileList []FileItem
	for _, file := range files {
		fileList = append(fileList, FileItem{
			Id:        file.Id,
			Name:      file.Name,
			IsDir:     file.IsDir,
			Size:      file.Size,
			Extension: file.FileExtension,
			Modified:  file.UpdatedAt.Format("2006-01-02 15:04:05"),
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
	// 查询文件是否存在
	file, err := s.fileRepo.GetFileById(fileId)
	if err != nil {
		return err
	}
	if file.UserId != userId {
		return errors.New("无权限删除该文件")
	}
	return s.fileRepo.SoftDeleteFile(userId, fileId)
}

func (s *fileService) CreateFromFileInfo(file *oss.FileInfo) error {
	var pId sql.NullString
	if file.ParentId == "" {
		pId = sql.NullString{Valid: false} // NULL
	} else {
		pId = sql.NullString{String: file.ParentId, Valid: true} // 有值
	}
	dbFile := models.File{
		Id:            file.Id,
		UserId:        file.UserId,
		Name:          file.Name,
		Size:          int64(file.Size),
		IsDir:         file.IsDir,
		FileExtension: file.FileExtension,
		OssObjectKey:  file.OssObjectKey,
		FileHash:      file.FileHash,
		ParentId:      pId,
		IsDeleted:     file.IsDeleted,
		CreatedAt:     file.CreatedAt,
		UpdatedAt:     file.UpdatedAt,
	}
	return s.fileRepo.CreateFile(&dbFile)
}
