package services

import (
	"go-cloud-storage/internal/models"
	"go-cloud-storage/internal/repositories"
	"gorm.io/gorm"
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
	db          *gorm.DB
	recycleRepo repositories.RecycleRepository
	fileRepo    repositories.FileRepository
}

func NewRecycleService(db *gorm.DB, recycleRepo repositories.RecycleRepository, fileRepo repositories.FileRepository) RecycleService {
	return &recycleService{
		db:          db,
		recycleRepo: recycleRepo,
		fileRepo:    fileRepo,
	}
}

// GetRecycleFiles 获取用户的回收站项目
func (s *recycleService) GetRecycleFiles(userId int) ([]models.RecycleBin, error) {
	return s.recycleRepo.GetFiles(userId)
}

func (s *recycleService) DeleteOne(fileId string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.recycleRepo.DeleteOne(tx, fileId); err != nil {
			return err
		}
		if err := s.fileRepo.DeletePermanent(tx, []string{fileId}); err != nil {
			return err
		}
		return nil
	})
}
func (s *recycleService) DeleteSelected(fileIds []string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1.删除回收站的记录
		if err := s.recycleRepo.DeleteBatch(tx, fileIds); err != nil {
			return err
		}
		// 2.删除对应的文件记录
		if err := s.fileRepo.DeletePermanent(tx, fileIds); err != nil {
			return err
		}
		return nil
	})
}
func (s *recycleService) ClearRecycles(userId int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1.清空回收站记录
		if err := s.recycleRepo.DeleteAll(tx, userId); err != nil {
			return err
		}
		// 2.删除file表中该用户软删除的记录
		if err := s.fileRepo.DeleteByUserId(tx, userId); err != nil {
			return err
		}
		return nil
	})
}
func (s *recycleService) RestoreOne(fileId string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1.删除回收站的记录
		if err := s.recycleRepo.DeleteOne(tx, fileId); err != nil {
			return err
		}
		// 2.更新file表中软删除的标志
		if err := s.fileRepo.MarkAsNotDeleted(tx, []string{fileId}, nil); err != nil {
			return err
		}
		return nil
	})
}
func (s *recycleService) RestoreSelected(fileIds []string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1.删除回收站的记录
		if err := s.recycleRepo.DeleteBatch(tx, fileIds); err != nil {
			return err
		}
		// 2.更新file表中软删除的标志
		if err := s.fileRepo.MarkAsNotDeleted(tx, fileIds, nil); err != nil {
			return err
		}
		return nil
	})
}
func (s *recycleService) RestoreAll(userId int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1.删除回收站的记录
		if err := s.recycleRepo.DeleteAll(tx, userId); err != nil {
			return err
		}
		// 2.更新file表中所有软删除的标志
		if err := s.fileRepo.MarkAsNotDeleted(tx, []string{}, &userId); err != nil {
			return err
		}
		return nil
	})
}

// CleanExpiredItems 清理过期的回收站项目
func (s *recycleService) CleanExpiredItems() error {
	_, err := s.recycleRepo.CleanExpiredRecords()
	return err
}
