package services

import (
	"context"
	"go-cloud-storage/internal/repositories"

	"gorm.io/gorm"
)

const defaultExpiredJobScanLimit = 200

type RecycleJobPublisher interface {
	PublishExpiredFilePurge(ctx context.Context, fileID string) error
}

type RecycleService interface {
	GetRecycleFiles(userId int) ([]map[string]interface{}, error)
	DeleteOne(ctx context.Context, userid int, fileId string) error
	DeleteSelected(ctx context.Context, fileIds []string) error
	ClearRecycles(ctx context.Context, userId int) error
	RestoreOne(fileId string) error
	RestoreSelected(fileIds []string) error
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
func (s *recycleService) GetRecycleFiles(userId int) ([]map[string]interface{}, error) {
	items, err := s.recycleRepo.GetFiles(userId)
	if err != nil {
		return nil, err
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

	return res, nil
}

func (s *recycleService) DeleteOne(ctx context.Context, userId int, fileId string) error {
	_ = userId
	return s.purge.PurgeOne(ctx, fileId)
}

func (s *recycleService) DeleteSelected(ctx context.Context, fileIds []string) error {
	return s.purge.PurgeFiles(ctx, fileIds)
}

func (s *recycleService) ClearRecycles(ctx context.Context, userId int) error {
	fileIDs, err := s.recycleRepo.GetAllFileIds(nil, userId)
	if err != nil {
		return err
	}
	return s.purge.PurgeFiles(ctx, fileIDs)
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
