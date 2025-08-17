package services

import (
	"go-cloud-storage/internal/models"
	"go-cloud-storage/internal/models/dto"
	"go-cloud-storage/internal/repositories"
	"time"
)

type FavoriteService interface {
	GetFavorites(userId, page, pageSize int) ([]dto.FavoriteDTO, int64, error)
	AddToFavorite(fileId string, userId int) error
	CancelFavorite(userId int, fileId string) error
}

type favoriteService struct {
	favoriteRepo repositories.FavoriteRepository
	fileService  FileService
}

func NewFavoriteService(favRepo repositories.FavoriteRepository, fs FileService) FavoriteService {
	return &favoriteService{favoriteRepo: favRepo, fileService: fs}
}

func (s favoriteService) GetFavorites(userId, page, pageSize int) ([]dto.FavoriteDTO, int64, error) {
	// 调用 repo 层获取收藏列表和总数
	favs, total, err := s.favoriteRepo.ListFavorite(userId, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	var result []dto.FavoriteDTO

	for _, f := range favs {
		// 查询 file 表获取完整路径
		file, err := s.fileService.GetFileById(f.FileId)
		if err != nil {
			return nil, 0, err
		}
		// 计算完整路径，例如 /parent1/parent2/fileName
		fullPath, err := s.fileService.GetFilePath(file)

		dto := dto.FavoriteDTO{
			FileId:    f.FileId,
			Name:      file.Name,
			IsDir:     file.IsDir,
			Path:      fullPath,
			SizeStr:   file.SizeStr,
			FileURL:   file.FileURL,
			CreatedAt: f.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		result = append(result, dto)
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
