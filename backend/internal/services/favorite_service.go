package services

import (
	"go-cloud-storage/backend/internal/models"
	"go-cloud-storage/backend/internal/models/dto"
	"go-cloud-storage/backend/internal/repositories"
	"time"
)

type FavoriteService interface {
	GetFavorites(userId, page, pageSize int) ([]dto.FavoriteDTO, int64, error)
	AddToFavorite(fileId string, userId int) error
	CancelFavorite(userId int, fileId string) error
}

type favoriteService struct {
	favoriteRepo repositories.FavoriteRepository
	fileRepo     repositories.FileRepository
	fileService  FileService
}

func NewFavoriteService(favRepo repositories.FavoriteRepository, fileRepo repositories.FileRepository, fs FileService) FavoriteService {
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

	fileMap := make(map[string]*models.File, len(files))
	for i := range files {
		fileMap[files[i].Id] = &files[i]
	}

	var result []dto.FavoriteDTO
	for _, f := range favs {
		file, ok := fileMap[f.FileId]
		if !ok {
			continue
		}
		fullPath, err := s.fileService.GetFilePath(file)
		if err != nil {
			fullPath = "/" + file.Name
		}

		result = append(result, dto.FavoriteDTO{
			FileId:    f.FileId,
			Name:      file.Name,
			IsDir:     file.IsDir,
			Path:      fullPath,
			SizeStr:   file.SizeStr,
			FileURL:   file.FileURL,
			CreatedAt: f.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return result, total, nil
}

func (s favoriteService) AddToFavorite(fileId string, userId int) error {
	var favorite = &models.Favorite{
		UserId:    userId,
		FileId:    fileId,
		CreatedAt: time.Now(),
	}
	return s.favoriteRepo.AddToFavorite(favorite)
}

func (s favoriteService) CancelFavorite(userId int, fileId string) error {
	return s.favoriteRepo.CancelFavorite(userId, fileId)
}
