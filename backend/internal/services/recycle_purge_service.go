package services

import (
	"context"
	"fmt"
	"go-cloud-storage/backend/internal/models"
	"gorm.io/gorm/clause"
	"time"

	"go-cloud-storage/backend/infrastructure/minio"
	"go-cloud-storage/backend/internal/repositories"

	"gorm.io/gorm"
)

type RecyclePurgeService interface {
	PurgeOne(ctx context.Context, fileID string) error
	PurgeExpired(ctx context.Context, fileIDs []string) error
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
	return s.purgeFiles(ctx, fileIDs, false)
}

func (s *recyclePurgeService) PurgeExpired(ctx context.Context, fileIDs []string) error {
	return s.purgeFiles(ctx, fileIDs, true)
}

func (s *recyclePurgeService) purgeFiles(ctx context.Context, fileIDs []string, expiredOnly bool) error {
	if len(fileIDs) == 0 {
		return nil
	}
	// Discover owners before opening the transaction. Both restore and purge lock
	// these user rows, then re-read current recycle state (never trust a queued ID).
	var owners []int
	if err := s.db.WithContext(ctx).Model(&models.File{}).Where("id IN ?", fileIDs).Distinct().Order("user_id").Pluck("user_id", &owners).Error; err != nil {
		return err
	}
	if len(owners) == 0 {
		return nil
	}
	var keysToDelete []string
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var users []models.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id IN ?", owners).Order("id").Find(&users).Error; err != nil {
			return err
		}
		var roots []string
		query := tx.Table("recycle_bin AS rb").Joins("JOIN file f ON f.id = rb.file_id").
			Where("rb.file_id IN ? AND f.is_deleted = ?", fileIDs, true)
		if expiredOnly {
			query = query.Where("rb.expire_at <= ?", time.Now())
		}
		if err := query.Pluck("rb.file_id", &roots).Error; err != nil {
			return err
		}
		if len(roots) == 0 {
			return nil
		} // restored, already purged, or re-deleted with a new expiry
		var allFileIDs []string
		if err := tx.Raw(`
   WITH RECURSIVE descendants AS (
    SELECT id FROM file WHERE id IN ?
    UNION ALL SELECT f.id FROM file f INNER JOIN descendants d ON f.parent_id = d.id
   ) SELECT DISTINCT id FROM descendants`, roots).Scan(&allFileIDs).Error; err != nil {
			return err
		}
		var files []models.File
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id IN ?", allFileIDs).Find(&files).Error; err != nil {
			return err
		}
		keys := make([]string, 0, len(files))
		released := make(map[int]int64)
		for _, file := range files {
			// Fail closed: FK cascades must never remove a live descendant.
			if !file.IsDeleted {
				return fmt.Errorf("目录包含已恢复文件，请先移动或恢复父目录")
			}
			if file.IsDir {
				continue
			}
			if file.OssObjectKey != "" {
				keys = append(keys, file.OssObjectKey)
			}
			released[file.UserId] += file.Size
		}
		var err error
		keysToDelete, err = repositories.NewFileRepository(tx).GetObjectKeysByIdsExcludeRefs(allFileIDs, keys)
		if err != nil {
			return err
		}
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
		for userID, size := range released {
			if err := s.quotaRepo.UpdateUsedSpace(tx, userID, -size); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(keysToDelete) > 0 {
		if err := s.minio.DeleteFiles(ctx, keysToDelete); err != nil {
			return fmt.Errorf("delete minio objects failed: %w", err)
		}
	}
	return nil
}
