package services

import (
	"context"
	"fmt"

	"go-cloud-storage/backend/internal/repositories"

	"gorm.io/gorm"
)

const defaultExpiredJobScanLimit = 200

type RecycleJobPublisher interface {
	PublishExpiredFilePurge(ctx context.Context, fileID string) error
}

type RecycleService interface {
	GetRecycleFiles(userId int, page, pageSize int) ([]map[string]interface{}, int64, error)
	DeleteOne(ctx context.Context, userId int, fileId string) error
	DeleteSelected(ctx context.Context, userId int, fileIds []string) error
	ClearRecycles(ctx context.Context, userId int) error
	RestoreOne(userId int, fileId string) error
	RestoreSelected(userId int, fileIds []string) error
	DispatchExpiredPurgeJobs(ctx context.Context, limit int) (int, error)
}

type recycleService struct {
	db          *gorm.DB
	recycleRepo repositories.RecycleRepository
	fileRepo    repositories.FileRepository
	purge       RecyclePurgeService
	publisher   RecycleJobPublisher
}

func NewRecycleService(
	db *gorm.DB,
	recycleRepo repositories.RecycleRepository,
	fileRepo repositories.FileRepository,
	purge RecyclePurgeService,
	publisher RecycleJobPublisher,
) RecycleService {
	return &recycleService{
		db:          db,
		recycleRepo: recycleRepo,
		fileRepo:    fileRepo,
		purge:       purge,
		publisher:   publisher,
	}
}

// TrashItem 回收站项目响应结构
type TrashItem struct {
	FileId      string `json:"file_id"`
	Name        string `json:"name"`
	IsDir       bool   `json:"is_dir"`
	Size        int64  `json:"size"`
	DeletedDate string `json:"deleted_date"`
	ExpireTime  int    `json:"expire_time"`
}

// GetRecycleFiles 获取用户的回收站项目
func (s *recycleService) GetRecycleFiles(userId int, page, pageSize int) ([]map[string]interface{}, int64, error) {
	items, total, err := s.recycleRepo.GetFiles(userId, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	var res []map[string]interface{}

	// 准备返回数据
	for _, item := range items {
		res = append(res, map[string]interface{}{
			"fileId":      item.FileId,
			"name":        item.Name,
			"isDir":       item.IsDir,
			"size_str":    item.SizeStr,
			"deletedDate": item.DeletedAt.Format("2006-01-02 15:04:05"),
			"expireDays":  int(item.ExpireAt.Sub(item.DeletedAt).Hours() / 24),
		})
	}

	return res, total, nil
}

func (s *recycleService) DeleteOne(ctx context.Context, userId int, fileId string) error {
	if userId > 0 {
		if err := s.verifyOwnership(nil, userId, []string{fileId}); err != nil {
			return err
		}
	}
	return s.purge.PurgeOne(ctx, fileId)
}

func (s *recycleService) DeleteSelected(ctx context.Context, userId int, fileIds []string) error {
	// userId <= 0 表示系统级操作（如过期清理），跳过权限校验
	if userId > 0 {
		if err := s.verifyOwnership(nil, userId, fileIds); err != nil {
			return err
		}
	}
	return s.purge.PurgeFiles(ctx, fileIds)
}

func (s *recycleService) ClearRecycles(ctx context.Context, userId int) error {
	fileIDs, err := s.recycleRepo.GetAllFileIds(nil, userId)
	if err != nil {
		return err
	}
	return s.purge.PurgeFiles(ctx, fileIDs)
}
func (s *recycleService) RestoreOne(userId int, fileId string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.verifyOwnership(tx, userId, []string{fileId}); err != nil {
			return err
		}
		if err := s.recycleRepo.DeleteOne(tx, fileId); err != nil {
			return err
		}
		if err := s.fileRepo.MarkAsNotDeleted(tx, []string{fileId}, nil); err != nil {
			return err
		}
		return nil
	})
}
func (s *recycleService) RestoreSelected(userId int, fileIds []string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.verifyOwnership(tx, userId, fileIds); err != nil {
			return err
		}
		if err := s.recycleRepo.DeleteBatch(tx, fileIds); err != nil {
			return err
		}
		if err := s.fileRepo.MarkAsNotDeleted(tx, fileIds, nil); err != nil {
			return err
		}
		return nil
	})
}

func (s *recycleService) verifyOwnership(tx *gorm.DB, userId int, fileIds []string) error {
	count, err := s.recycleRepo.CountByUserAndFileIds(tx, userId, fileIds)
	if err != nil {
		return err
	}
	if int(count) != len(fileIds) {
		return fmt.Errorf("无权操作该文件")
	}
	return nil
}

// CleanExpiredItems 清理过期的回收站项目
func (s *recycleService) DispatchExpiredPurgeJobs(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		limit = defaultExpiredJobScanLimit
	}
	fileIDs, err := s.recycleRepo.GetExpiredFileIds(limit)
	if err != nil {
		return 0, err
	}
	if len(fileIDs) == 0 {
		return 0, nil
	}

	if s.publisher == nil {
		if err := s.purge.PurgeFiles(ctx, fileIDs); err != nil {
			return 0, err
		}
		return len(fileIDs), nil
	}

	for _, fileID := range fileIDs {
		if err := s.publisher.PublishExpiredFilePurge(ctx, fileID); err != nil {
			return 0, err
		}
	}
	return len(fileIDs), nil
}
