package services

import (
	"context"
	"go-cloud-storage/internal/pkg/minio"
	"go-cloud-storage/internal/repositories"

	"gorm.io/gorm"
)

type RecycleService interface {
	GetRecycleFiles(userId int) ([]map[string]interface{}, error)
	DeleteOne(ctx context.Context, userid int, fileId string) error
	DeleteSelected(ctx context.Context, fileIds []string) error
	ClearRecycles(ctx context.Context, userId int) error
	RestoreOne(fileId string) error
	RestoreSelected(fileIds []string) error
	CleanExpiredItems() error
}

type recycleService struct {
	db          *gorm.DB
	minio       *minio.MinioService
	recycleRepo repositories.RecycleRepository
	fileRepo    repositories.FileRepository
	shareRepo   repositories.ShareRepository
	starRepo    repositories.FavoriteRepository
}

func NewRecycleService(db *gorm.DB, minio *minio.MinioService, recycleRepo repositories.RecycleRepository, fileRepo repositories.FileRepository,
	shareRepo repositories.ShareRepository, starRepo repositories.FavoriteRepository) RecycleService {
	return &recycleService{
		db:          db,
		minio:       minio,
		recycleRepo: recycleRepo,
		fileRepo:    fileRepo,
		shareRepo:   shareRepo,
		starRepo:    starRepo,
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
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1.获取文件信息
		file, err := s.fileRepo.GetFileById(fileId)
		if err != nil {
			return err
		}
		// 2.删除分享记录和收藏记录(如果有的话)
		exist, shareInfo := s.shareRepo.IsShared(fileId)
		if exist {
			// 需要删除对应的分享记录
			if err := s.shareRepo.Delete(tx, shareInfo.Id); err != nil {
				return err
			}
		}
		// 3.删除收藏记录(如果有的话)
		started, _ := s.starRepo.IsFavorited(userId, fileId)
		if started {
			// 需要删除对应的收藏记录
			if err := s.starRepo.Delete(tx, fileId); err != nil {
				return err
			}
		}
		// 4.删除回收站记录
		if err := s.recycleRepo.DeleteOne(tx, fileId); err != nil {
			return err
		}
		// 5.删除文件记录
		if err := s.fileRepo.DeletePermanent(tx, []string{fileId}); err != nil {
			return err
		}
		// 6.OSS删除
		if err := s.minio.DeleteFile(context.Background(), file.OssObjectKey); err != nil {
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
		// 1.批量删除回收站的记录
		if err := s.recycleRepo.DeleteBatch(tx, fileIds); err != nil {
			return err
		}
		// 2.批量删除分享记录(如果有的话)
		if err := s.shareRepo.DeleteBatch(tx, fileIds); err != nil {
			return err
		}
		// 3.批量删除收藏记录(如果有的话)
		if err := s.starRepo.DeleteBatch(tx, fileIds); err != nil {
			return err
		}

		// 4.删除对应的文件记录
		if err := s.fileRepo.DeletePermanent(tx, fileIds); err != nil {
			return err
		}
		// 5. OSS 删除
		if err := s.minio.DeleteFiles(context.Background(), objectKeys); err != nil {
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
		// 1.收集回收站所有记录的file_id
		fileIds, err := s.recycleRepo.GetAllFileIds(tx, userId)
		if err != nil {
			return err
		}
		// 删除对应的收藏记录(如果有)
		if err := s.starRepo.DeleteBatch(tx, fileIds); err != nil {
			return err
		}
		// 删除对应的分享记录(如果有)
		if err := s.shareRepo.DeleteBatch(tx, fileIds); err != nil {
			return err
		}
		// 2.清空回收站记录
		if err := s.recycleRepo.DeleteAll(tx, userId); err != nil {
			return err
		}
		// 3.删除file表中该用户软删除的记录
		if err := s.fileRepo.DeleteByUserId(tx, userId); err != nil {
			return err
		}
		// 4. OSS 删除
		if err := s.minio.DeleteFiles(context.Background(), objectKeys); err != nil {
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
