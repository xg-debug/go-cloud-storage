package services

import (
	"context"
	"go-cloud-storage/internal/pkg/aliyunoss"
	"go-cloud-storage/internal/repositories"
	"gorm.io/gorm"
)

type RecycleService interface {
	GetRecycleFiles(userId int) ([]map[string]interface{}, error)
	DeleteOne(ctx context.Context, fileId string) error
	DeleteSelected(ctx context.Context, fileIds []string) error
	ClearRecycles(ctx context.Context, userId int) error
	RestoreOne(fileId string) error
	RestoreSelected(fileIds []string) error
	CleanExpiredItems() error
}

type recycleService struct {
	db          *gorm.DB
	ossService  *aliyunoss.OSSService
	recycleRepo repositories.RecycleRepository
	fileRepo    repositories.FileRepository
}

func NewRecycleService(db *gorm.DB, oss *aliyunoss.OSSService, recycleRepo repositories.RecycleRepository, fileRepo repositories.FileRepository) RecycleService {
	return &recycleService{
		db:          db,
		ossService: oss,
		recycleRepo: recycleRepo,
		fileRepo:    fileRepo,
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

func (s *recycleService) DeleteOne(ctx context.Context, fileId string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1.获取文件信息
		file, err := s.fileRepo.GetFileById(fileId)
		if err != nil {
			return err
		}
		// 2.删除回收站记录
		if err := s.recycleRepo.DeleteOne(tx, fileId); err != nil {
			return err
		}
		// 3.删除文件记录
		if err := s.fileRepo.DeletePermanent(tx, []string{fileId}); err != nil {
			return err
		}
		// 4.OSS删除
		if err := s.ossService.DeleteFile(context.Background(), file.OssObjectKey); err != nil {
			return err
		}
		return nil
	})
}
func (s *recycleService) DeleteSelected(ctx context.Context, fileIds []string) error {
	objectKeys, err := s.fileRepo.GetObjectKeysByIds(fileIds)
	if err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1.删除回收站的记录
		if err := s.recycleRepo.DeleteBatch(tx, fileIds); err != nil {
			return err
		}
		// 2.删除对应的文件记录
		if err := s.fileRepo.DeletePermanent(tx, fileIds); err != nil {
			return err
		}
		// 3. OSS 删除
		if err := s.ossService.DeleteFiles(context.Background(), objectKeys); err != nil {
			return err
		}
		return nil
	})
}
func (s *recycleService) ClearRecycles(ctx context.Context, userId int) error {
	objectKeys, err := s.fileRepo.GetObjectKeysByUserId(userId)
	if err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1.清空回收站记录
		if err := s.recycleRepo.DeleteAll(tx, userId); err != nil {
			return err
		}
		// 2.删除file表中该用户软删除的记录
		if err := s.fileRepo.DeleteByUserId(tx, userId); err != nil {
			return err
		}
		// 3. OSS 删除
		if err := s.ossService.DeleteFiles(context.Background(), objectKeys); err != nil {
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

// CleanExpiredItems 清理过期的回收站项目
func (s *recycleService) CleanExpiredItems() error {
	_, err := s.recycleRepo.CleanExpiredRecords()
	return err
}
