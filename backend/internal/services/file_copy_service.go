package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"go-cloud-storage/backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (s *fileService) CopyFile(ctx context.Context, userId int, fileId, targetFolderId string) error {
	if fileId == targetFolderId {
		return errors.New("不能复制到自身")
	}

	file, err := s.fileRepo.GetFileById(fileId)
	if err != nil {
		return fmt.Errorf("找不到源文件: %w", err)
	}

	// 检查配额
	if !file.IsDir {
		quota, _ := s.storageQuotaRepo.GetByUserID(userId)
		if quota != nil && quota.Used+file.Size > quota.Total {
			return errors.New("存储空间不足")
		}
	}

	// 生成新文件名（避免冲突）
	baseName := file.Name
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)
	newName := baseName
	counter := 1
	for {
		exist, _ := s.fileRepo.GetFileByParentAndName(ctx, userId, targetFolderId, newName)
		if exist == nil {
			break
		}
		if ext != "" {
			newName = fmt.Sprintf("%s_副本%d%s", nameWithoutExt, counter, ext)
		} else {
			newName = fmt.Sprintf("%s_副本%d", nameWithoutExt, counter)
		}
		counter++
	}

	if file.IsDir {
		return s.copyFolder(ctx, userId, file, targetFolderId, newName)
	}
	return s.copySingleFile(ctx, userId, file, targetFolderId, newName)
}

func (s *fileService) copySingleFile(ctx context.Context, userId int, src *models.File, targetParentId, newName string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		return s.copyFileRecord(ctx, tx, userId, src, targetParentId, newName)
	})
}

func (s *fileService) copyFolder(ctx context.Context, userId int, src *models.File, targetParentId, newName string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		newId := uuid.New().String()
		if err := tx.Model(&models.File{}).Create(&models.File{
			Id: newId, UserId: userId, Name: newName, IsDir: true, ParentId: sql.NullString{String: targetParentId, Valid: true}, Size: 0, SizeStr: "-",
		}).Error; err != nil {
			return fmt.Errorf("创建文件夹记录失败: %w", err)
		}
		return s.copyChildren(ctx, tx, userId, src.Id, newId)
	})
}

func (s *fileService) copyChildren(ctx context.Context, tx *gorm.DB, userId int, srcId, targetParentId string) error {
	children, _, err := s.fileRepo.GetFiles(ctx, userId, srcId, 1, 10000, "created_at", "desc")
	if err != nil {
		return err
	}
	for _, child := range children {
		childFile, _ := s.fileRepo.GetFileById(child.Id)
		if childFile == nil {
			continue
		}
		if childFile.IsDir {
			newId := uuid.New().String()
			if err := tx.Model(&models.File{}).Create(&models.File{
				Id: newId, UserId: userId, Name: childFile.Name, IsDir: true, ParentId: sql.NullString{String: targetParentId, Valid: true}, Size: 0, SizeStr: "-",
			}).Error; err != nil {
				return err
			}
			if err := s.copyChildren(ctx, tx, userId, childFile.Id, newId); err != nil {
				return err
			}
		} else {
			if err := s.copyFileRecord(ctx, tx, userId, childFile, targetParentId, childFile.Name); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *fileService) copyFileRecord(ctx context.Context, tx *gorm.DB, userId int, src *models.File, targetParentId, newName string) error {
	// 生成新的 OSS key 并复制 MinIO 对象
	newKey := s.minio.GenerateObjectKey(userId, targetParentId, newName)
	if err := s.minio.CopyObject(ctx, src.OssObjectKey, newKey); err != nil {
		return fmt.Errorf("复制文件对象失败: %w", err)
	}

	// 如果源文件有缩略图，也复制缩略图对象
	newThumbnailURL := src.ThumbnailURL
	if src.ThumbnailURL != "" {
		thumbKey := s.minio.GenerateObjectKey(userId, targetParentId, "thumb_"+newName)
		// 从 OSS URL 中解析缩略图 key（样式：.../bucket/objectKey）
		parts := strings.Split(src.OssObjectKey, "/")
		thumbSrcKey := strings.Replace(src.OssObjectKey, parts[len(parts)-1], "thumb_"+parts[len(parts)-1], 1)
		if err := s.minio.CopyObject(ctx, thumbSrcKey, thumbKey); err == nil {
			newThumbnailURL = s.minio.GenerateObjectURL(thumbKey)
		}
	}

	newFileURL := s.minio.GenerateObjectURL(newKey)

	if err := tx.Model(&models.File{}).Create(&models.File{
		Id: uuid.New().String(), UserId: userId, Name: newName, Size: src.Size, SizeStr: src.SizeStr,
		IsDir: false, FileExtension: src.FileExtension, FileHash: src.FileHash, FileURL: newFileURL,
		ThumbnailURL: newThumbnailURL, OssObjectKey: newKey, ParentId: sql.NullString{String: targetParentId, Valid: true},
	}).Error; err != nil {
		return err
	}
	return s.storageQuotaRepo.UpdateUsedSpace(tx, userId, src.Size)
}

