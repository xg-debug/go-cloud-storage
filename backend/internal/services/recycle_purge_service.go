package services

import (
	"context"
	"fmt"

	"go-cloud-storage/backend/infrastructure/minio"
	"go-cloud-storage/backend/internal/repositories"

	"gorm.io/gorm"
)

type RecyclePurgeService interface {
	PurgeOne(ctx context.Context, fileID string) error
	PurgeFiles(ctx context.Context, fileIDs []string) error
}

type recyclePurgeService struct {
	db          *gorm.DB
	minio       *minio.MinioService
	recycleRepo repositories.RecycleRepository
	fileRepo    repositories.FileRepository
	shareRepo   repositories.ShareRepository
	starRepo    repositories.FavoriteRepository
	quotaRepo   repositories.StorageQuotaRepository
}

func NewRecyclePurgeService(
	db *gorm.DB,
	minioService *minio.MinioService,
	recycleRepo repositories.RecycleRepository,
	fileRepo repositories.FileRepository,
	shareRepo repositories.ShareRepository,
	starRepo repositories.FavoriteRepository,
	quotaRepo repositories.StorageQuotaRepository,
) RecyclePurgeService {
	return &recyclePurgeService{
		db:          db,
		minio:       minioService,
		recycleRepo: recycleRepo,
		fileRepo:    fileRepo,
		shareRepo:   shareRepo,
		starRepo:    starRepo,
		quotaRepo:   quotaRepo,
	}
}

func (s *recyclePurgeService) PurgeOne(ctx context.Context, fileID string) error {
	return s.PurgeFiles(ctx, []string{fileID})
}

func (s *recyclePurgeService) PurgeFiles(ctx context.Context, fileIDs []string) error {
	if len(fileIDs) == 0 {
		return nil
	}

	allFileIDs, err := s.expandDescendantIDs(ctx, fileIDs)
	if err != nil {
		return err
	}
	if len(allFileIDs) == 0 {
		return nil
	}

	files, err := s.fileRepo.GetFileByIds(allFileIDs)
	if err != nil {
		return err
	}

	objectKeys := make([]string, 0, len(files))
	releasedByUser := make(map[int]int64)
	for _, file := range files {
		if file.IsDir {
			continue
		}
		if file.OssObjectKey != "" {
			objectKeys = append(objectKeys, file.OssObjectKey)
		}
		if file.Size > 0 {
			releasedByUser[file.UserId] += file.Size
		}
	}

	// 跨用户秒传场景：同一个 MinIO 对象可能被多个用户引用，只删除无人引用的对象
	keysToDelete, err := s.fileRepo.GetObjectKeysByIdsExcludeRefs(allFileIDs, objectKeys)
	if err != nil {
		return err
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.recycleRepo.DeleteBatch(tx, allFileIDs); err != nil {
			return err
		}
		if err := s.shareRepo.DeleteBatch(tx, allFileIDs); err != nil {
			return err
		}
		if err := s.starRepo.DeleteBatch(tx, allFileIDs); err != nil {
			return err
		}
		if err := s.fileRepo.DeletePermanent(tx, allFileIDs); err != nil {
			return err
		}
		for userID, released := range releasedByUser {
			if err := s.quotaRepo.UpdateUsedSpace(tx, userID, -released); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	if len(keysToDelete) == 0 {
		return nil
	}
	if err := s.minio.DeleteFiles(ctx, keysToDelete); err != nil {
		return fmt.Errorf("delete minio objects failed: %w", err)
	}

	return nil
}

func (s *recyclePurgeService) expandDescendantIDs(ctx context.Context, fileIDs []string) ([]string, error) {
	var ids []string
	err := s.db.WithContext(ctx).Raw(`
		WITH RECURSIVE descendants AS (
			SELECT id FROM file WHERE id IN ?
			UNION ALL
			SELECT f.id FROM file f
			INNER JOIN descendants d ON f.parent_id = d.id
		)
		SELECT DISTINCT id FROM descendants
	`, fileIDs).Scan(&ids).Error
	return ids, err
}
