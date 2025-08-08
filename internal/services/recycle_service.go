package services

//
//import (
//	"errors"
//	"fmt"
//	"go-cloud-storage/internal/models"
//	"go-cloud-storage/internal/repositories"
//	"log"
//)
//
//type RecycleService interface {
//	MoveToRecycle(file *models.File, userID int) error
//	ListRecycleBin(userID int) ([]models.RecycleBin, error)
//	RestoreFile(fileID string, userID int) error
//	PermanentDelete(fileID string, userID int) error
//	CleanExpiredRecycle() (int, error)
//}
//
//type recycleService struct {
//	recycleRepo repositories.RecycleRepository
//	fileRepo    repositories.FileRepository
//}
//
//func (s recycleService) MoveToRecycle(file *models.File, userId int) error {
//	//record := &models.RecycleBin{
//	//	FileId: file.Id,
//	//	UserId: userId,
//	//	Name
//	//}
//}
//
//func (s recycleService) ListRecycleBin(userID int) ([]models.RecycleBin, error) {
//
//	panic("implement me")
//}
//
//func (s recycleService) RestoreFile(fileID string, userID int) error {
//	// 1. 验证文件所有权
//	record, err := s.recycleRepo.GetByFileId(fileID)
//	if err != nil {
//		return err
//	}
//
//	if record.UserId != userID {
//		return errors.New("unauthorized operation")
//	}
//
//	// 2. 根据类型执行恢复
//	if record.IsDir {
//		// 恢复文件夹及其所有内容
//		if err := s.fileRepo.RestoreFolder(fileID); err != nil {
//			return fmt.Errorf("failed to restore folder: %w", err)
//		}
//	} else {
//		// 恢复单个文件
//		if err := s.fileRepo.RestoreFile(fileID); err != nil {
//			return fmt.Errorf("failed to restore file: %w", err)
//		}
//	}
//
//	// 3. 更新存储配额（假设有配额服务）
//	if err := s.quotaService.UpdateAfterRestore(userID, record.Size); err != nil {
//		// 注意：这里需要权衡是否回滚，建议记录日志但继续
//		log.Printf("quota update failed after restore: %v", err)
//	}
//
//	// 4. 标记回收站记录为已恢复
//	return s.recycleRepo.MarkAsRestored(fileID)
//}
//
//func (r recycleService) PermanentDelete(fileID string, userID int) error {
//
//	panic("implement me")
//}
//
//func (r recycleService) CleanExpiredRecycle() (int, error) {
//
//	panic("implement me")
//}
//
//func NewRecycleService(recycleRepo repositories.RecycleRepository, fileRepo repositories.FileRepository) RecycleService {
//	return &recycleService{
//		recycleRepo: recycleRepo,
//		fileRepo:    fileRepo,
//	}
//}
