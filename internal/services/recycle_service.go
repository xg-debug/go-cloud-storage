package services

import (
	"go-cloud-storage/internal/models"
	"go-cloud-storage/internal/repositories"
)

type RecycleService interface {
	GetRecycleFiles(userId int) ([]models.RecycleBin, error)
	DeleteOne(fileId string) error
	DeleteSelected(fileIds []string) error
	ClearRecycles(userId int) error
	RestoreOne(fileId string) error
	RestoreSelected(fileIds []string) error
	RestoreAll(userId int) error
	CleanExpiredItems() error
}

type recycleService struct {
	recycleRepo repositories.RecycleRepository
	fileRepo    repositories.FileRepository
}

func NewRecycleService(recycleRepo repositories.RecycleRepository, fileRepo repositories.FileRepository) RecycleService {
	return &recycleService{
		recycleRepo: recycleRepo,
		fileRepo:    fileRepo,
	}
}

// MoveToTrash 将文件移动到回收站
//func (ts *recycleService) MoveToTrash(fileID string, userID int, originalPath string) error {
//	// 标记文件为已删除
//	err := ts.fileRepo.SoftDeleteFile(userID, fileID)
//	if err != nil {
//		return err
//	}
//
//	// 添加到回收站记录
//	recycleRecord := &models.RecycleBin{
//		FileId:       fileID,
//		UserId:       userID,
//		OriginalPath: originalPath,
//		DeletedAt:    time.Now(),
//	}
//
//	return ts.recycleRepo.AddToRecycle(recycleRecord)
//}

func (s *recycleService) DeleteOne(fileId string) error {
	return s.recycleRepo.DeleteOne(fileId)
}
func (s *recycleService) DeleteSelected(fileIds []string) error {
	return s.recycleRepo.DeleteBatch(fileIds)
}
func (s *recycleService) ClearRecycles(userId int) error {
	return s.recycleRepo.DeleteAll(userId)
}
func (s *recycleService) RestoreOne(fileId string) error {
	return s.recycleRepo.RestoreOne(fileId)
}
func (s *recycleService) RestoreSelected(fileIds []string) error {
	return s.recycleRepo.RestoreBatch(fileIds)
}
func (s *recycleService) RestoreAll(userId int) error {
	return s.recycleRepo.RestoreAll(userId)
}

// GetRecycleFiles 获取用户的回收站项目
func (s *recycleService) GetRecycleFiles(userId int) ([]models.RecycleBin, error) {
	return s.recycleRepo.GetFiles(userId)
}

// CleanExpiredItems 清理过期的回收站项目
func (s *recycleService) CleanExpiredItems() error {
	_, err := s.recycleRepo.CleanExpiredRecords()
	return err
}
