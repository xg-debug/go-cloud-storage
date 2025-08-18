package services

import (
	"context"
	"go-cloud-storage/internal/repositories"

	"gorm.io/gorm"
)

type CategoryService interface {
	GetFilesByCategory(userId int, fileType string, sortBy string, sortOrder string, page int, pageSize int) ([]FileItem, int64, error)
}

type categoryService struct {
	db       *gorm.DB
	fileRepo repositories.FileRepository
}

func NewCategoryService(db *gorm.DB, fileRepo repositories.FileRepository) CategoryService {
	return &categoryService{
		db:       db,
		fileRepo: fileRepo,
	}
}

// GetFilesByCategory 根据文件类型获取文件列表
func (s *categoryService) GetFilesByCategory(userId int, fileType string, sortBy string, sortOrder string, page int, pageSize int) ([]FileItem, int64, error) {
	ctx := context.Background()
	files, total, err := s.fileRepo.GetFilesByCategory(ctx, userId, fileType, sortBy, sortOrder, page, pageSize)
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
			CreatedAt:    file.CreatedAt.Format("2006年01月02日"),
			FileURL:      file.FileURL,
			ThumbnailURL: file.ThumbnailURL,
		})
	}
	return fileList, total, nil
}
