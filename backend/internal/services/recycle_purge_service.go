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
}

func NewRecyclePurgeService(
	db *gorm.DB,
	minioService *minio.MinioService,
	recycleRepo repositories.RecycleRepository,
	fileRepo repositories.FileRepository,
	shareRepo repositories.ShareRepository,
	starRepo repositories.FavoriteRepository,
) RecyclePurgeService {
	return &recyclePurgeService{
		db:          db,
		minio:       minioService,
		recycleRepo: recycleRepo,
		fileRepo:    fileRepo,
		shareRepo:   shareRepo,
		starRepo:    starRepo,
	}
}

func (s *recyclePurgeService) PurgeOne(ctx context.Context, fileID string) error {
	return s.PurgeFiles(ctx, []string{fileID})
}

func (s *recyclePurgeService) PurgeFiles(ctx context.Context, fileIDs []string) error {
	if len(fileIDs) == 0 {
		return nil
	}

	objectKeys, err := s.fileRepo.GetObjectKeysByIds(fileIDs)
	if err != nil {
		return err
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.recycleRepo.DeleteBatch(tx, fileIDs); err != nil {
			return err
		}
		if err := s.shareRepo.DeleteBatch(tx, fileIDs); err != nil {
			return err
		}
		if err := s.starRepo.DeleteBatch(tx, fileIDs); err != nil {
			return err
		}
		if err := s.fileRepo.DeletePermanent(tx, fileIDs); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	if len(objectKeys) == 0 {
		return nil
	}
	if err := s.minio.DeleteFiles(ctx, objectKeys); err != nil {
		return fmt.Errorf("delete minio objects failed: %w", err)
	}

	return nil
}
