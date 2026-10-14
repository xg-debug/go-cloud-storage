package application

import (
	"context"
	"errors"
	"time"

	repository "go-cloud-storage/backend/services/file-service/internal/adapters/repository"
	domain "go-cloud-storage/backend/services/file-service/internal/domain"
	dto "go-cloud-storage/backend/services/file-service/internal/domain/dto"
)

type FavoriteService interface {
	GetFavorites(userId, page, pageSize int) ([]dto.FavoriteDTO, int64, error)
	AddToFavorite(fileId string, userId int) error
	CancelFavorite(userId int, fileId string) error
}

type favoriteService struct {
	favoriteRepo repository.FavoriteRepository
	fileRepo     repository.FileRepository
	fileService  FileService
}

func NewFavoriteService(favRepo repository.FavoriteRepository, fileRepo repository.FileRepository, fs FileService) FavoriteService {
	return &favoriteService{favoriteRepo: favRepo, fileRepo: fileRepo, fileService: fs}
}

func (s favoriteService) GetFavorites(userId, page, pageSize int) ([]dto.FavoriteDTO, int64, error) {
	favs, total, err := s.favoriteRepo.ListFavorite(userId, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	if len(favs) == 0 {
		return []dto.FavoriteDTO{}, total, nil
	}

	// 批量查询：收集所有 fileId，一次查询获取所有文件
	fileIds := make([]string, len(favs))
	for i, f := range favs {
		fileIds[i] = f.FileId
	}

	files, err := s.fileRepo.GetFileByIds(fileIds)
	if err != nil {
		return nil, 0, err
	}

	fileMap := make(map[string]*domain.File, len(files))
	for i := range files {
		fileMap[files[i].Id] = &files[i]
	}

	var result []dto.FavoriteDTO
	for _, f := range favs {
		file, ok := fileMap[f.FileId]
		if !ok || file.UserId != userId || file.IsDeleted {
			continue
		}
		fullPath, err := s.fileService.GetFilePath(file)
		if err != nil {
			fullPath = "/" + file.Name
		}

		fileURL, _ := s.fileService.ResolveFileURLs(context.Background(), file)
		result = append(result, dto.FavoriteDTO{
			Id:        f.FileId,
			FileId:    f.FileId,
			Name:      file.Name,
			IsDir:     file.IsDir,
			Path:      fullPath,
			Size:      file.Size,
			SizeStr:   file.SizeStr,
			Extension: file.FileExtension,
			FileURL:   fileURL,
			CreatedAt: f.CreatedAt.Format(time.RFC3339),
		})
	}

	return result, total, nil
}

func (s favoriteService) AddToFavorite(fileId string, userId int) error {
	file, err := s.fileRepo.GetUserFileByID(userId, fileId)
	if err != nil || file == nil {
		return errors.New("文件不存在或无权限")
	}
	var favorite = &domain.Favorite{
		UserId:    userId,
		FileId:    fileId,
		CreatedAt: time.Now(),
	}
	return s.favoriteRepo.AddToFavorite(favorite)
}

func (s favoriteService) CancelFavorite(userId int, fileId string) error {
	return s.favoriteRepo.CancelFavorite(userId, fileId)
}
