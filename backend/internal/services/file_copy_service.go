package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"go-cloud-storage/backend/internal/models"

	miniosrv "go-cloud-storage/backend/infrastructure/minio"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (s *fileService) CopyFile(ctx context.Context, userId int, fileId, targetFolderId string) error {
	if fileId == targetFolderId {
		return errors.New("不能复制到自身")
	}

	file, err := s.fileRepo.GetUserFileByID(userId, fileId)
	if err != nil {
		return fmt.Errorf("找不到源文件或无权限: %w", err)
	}
	if isProtectedRootFolder(file) {
		return errors.New("不能复制根目录")
	}
	if err := s.ensureTargetFolder(ctx, userId, targetFolderId); err != nil {
		return err
	}
	if file.IsDir && targetFolderId != "" {
		isSub, err := s.fileRepo.IsSubFolder(ctx, userId, file.Id, targetFolderId)
		if err != nil {
			return err
		}
		if isSub {
			return errors.New("不能复制到子文件夹")
		}
	}

	// 检查配额：文件夹先统计子树总大小，避免复制到一半才失败
	if file.IsDir {
		totalSize, sizeErr := s.fileRepo.SumSubtreeSize(ctx, file.Id)
		if sizeErr != nil {
			return fmt.Errorf("统计复制大小失败: %w", sizeErr)
		}
		quota, _ := s.storageQuotaRepo.GetByUserID(userId)
		if quota != nil && quota.Used+totalSize > quota.Total {
			return errors.New("存储空间不足")
		}
	} else {
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

	copiedKeys := make([]string, 0)
	var copyErr error
	if file.IsDir {
		copyErr = s.copyFolder(ctx, userId, file, targetFolderId, newName, &copiedKeys)
	} else {
		copyErr = s.copySingleFile(ctx, userId, file, targetFolderId, newName, &copiedKeys)
	}
	if copyErr != nil {
		s.cleanupCopiedObjects(copiedKeys)
	}
	return copyErr
}

func (s *fileService) copySingleFile(ctx context.Context, userId int, src *models.File, targetParentId, newName string, copiedKeys *[]string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		return s.copyFileRecord(ctx, tx, userId, src, targetParentId, newName, copiedKeys)
	})
}

func (s *fileService) copyFolder(ctx context.Context, userId int, src *models.File, targetParentId, newName string, copiedKeys *[]string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		newId := uuid.New().String()
		if err := tx.Model(&models.File{}).Create(&models.File{
			Id: newId, UserId: userId, Name: newName, IsDir: true, ParentId: nullableParentID(targetParentId), Size: 0, SizeStr: "-",
		}).Error; err != nil {
			return fmt.Errorf("创建文件夹记录失败: %w", err)
		}
		return s.copyChildren(ctx, tx, userId, src.Id, newId, copiedKeys)
	})
}

func (s *fileService) copyChildren(ctx context.Context, tx *gorm.DB, userId int, srcId, targetParentId string, copiedKeys *[]string) error {
	// 分页遍历子项，避免单层超过 10000 时静默丢失文件
	const pageSize = 2000
	for page := 1; ; page++ {
		children, _, err := s.fileRepo.GetFiles(ctx, userId, srcId, page, pageSize, "created_at", "desc")
		if err != nil {
			return err
		}
		if len(children) == 0 {
			return nil
		}
		for _, child := range children {
			childFile, err := s.fileRepo.GetUserFileByID(userId, child.Id)
			if err != nil {
				return err
			}
			if childFile.IsDir {
				newId := uuid.New().String()
				if err := tx.Model(&models.File{}).Create(&models.File{
					Id: newId, UserId: userId, Name: childFile.Name, IsDir: true, ParentId: nullableParentID(targetParentId), Size: 0, SizeStr: "-",
				}).Error; err != nil {
					return err
				}
				if err := s.copyChildren(ctx, tx, userId, childFile.Id, newId, copiedKeys); err != nil {
					return err
				}
			} else {
				if err := s.copyFileRecord(ctx, tx, userId, childFile, targetParentId, childFile.Name, copiedKeys); err != nil {
					return err
				}
			}
		}
		if len(children) < pageSize {
			return nil
		}
	}
}

func (s *fileService) copyFileRecord(ctx context.Context, tx *gorm.DB, userId int, src *models.File, targetParentId, newName string, copiedKeys *[]string) error {
	// 生成新的 OSS key 并复制 MinIO 对象
	newKey := s.minio.GenerateObjectKey(userId, targetParentId, newName)
	if err := s.minio.CopyObject(ctx, src.OssObjectKey, newKey); err != nil {
		return fmt.Errorf("复制文件对象失败: %w", err)
	}
	*copiedKeys = append(*copiedKeys, newKey)

	// 如果源文件有缩略图，也复制缩略图对象。
	// 缩略图 key 规则为 <原key去扩展名>_thumb.jpg（与缩略图生成逻辑一致）。
	newThumbnailURL := ""
	if src.ThumbnailURL != "" && src.OssObjectKey != "" {
		thumbSrcKey := miniosrv.ThumbnailObjectKey(src.OssObjectKey)
		newThumbKey := miniosrv.ThumbnailObjectKey(newKey)
		if err := s.minio.CopyObject(ctx, thumbSrcKey, newThumbKey); err == nil {
			*copiedKeys = append(*copiedKeys, newThumbKey)
			newThumbnailURL = s.minio.GenerateObjectURL(newThumbKey)
		}
	}

	newFileURL := s.minio.GenerateObjectURL(newKey)

	if err := tx.Model(&models.File{}).Create(&models.File{
		Id: uuid.New().String(), UserId: userId, Name: newName, Size: src.Size, SizeStr: src.SizeStr,
		IsDir: false, FileExtension: src.FileExtension, FileHash: src.FileHash, FileURL: newFileURL,
		ThumbnailURL: newThumbnailURL, OssObjectKey: newKey, ParentId: nullableParentID(targetParentId),
	}).Error; err != nil {
		return err
	}
	return s.storageQuotaRepo.UpdateUsedSpace(tx, userId, src.Size)
}

func (s *fileService) cleanupCopiedObjects(objectKeys []string) {
	if len(objectKeys) == 0 {
		return
	}
	seen := make(map[string]struct{}, len(objectKeys))
	deduped := make([]string, 0, len(objectKeys))
	for _, key := range objectKeys {
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, key)
	}
	if len(deduped) == 0 {
		return
	}
	if err := s.minio.DeleteFiles(context.Background(), deduped); err != nil {
		slog.Error("cleanup copied objects after copy failure failed", "count", len(deduped), "error", err)
	}
}
