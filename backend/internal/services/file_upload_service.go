package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"go-cloud-storage/backend/internal/models"
	"go-cloud-storage/backend/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
)

const (
	defaultUploadChunkSize      int64 = 10 * 1024 * 1024
	chunkUploadSessionTTL             = 24 * time.Hour
	chunkUploadMetadataTTL            = 48 * time.Hour
	chunkUploadCleanupBatchSize       = 100
	chunkUploadCleanupInterval        = 5 * time.Minute
	chunkUploadSessionSet             = "upload:sessions"
)

func (s *fileService) UploadFile(ctx context.Context, r io.Reader, userId int, fileName string, fileSize int64, fileHash string, parentId string) (*models.File, error) {
	fileName = strings.TrimSpace(fileName)
	if err := validateFileName(fileName); err != nil {
		return nil, err
	}
	if err := s.ensureTargetFolder(ctx, userId, parentId); err != nil {
		return nil, err
	}
	if exists, err := s.fileRepo.CheckDuplicateName(userId, parentId, fileName); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.New("该目录下已有同名的文件")
	}

	// 秒传检查：仅在当前用户范围内复用已有对象，避免通过已知 hash 探测或引用其他用户文件
	existingFile, err := s.fileRepo.GetFileByMD5(userId, fileHash)
	if err == nil && existingFile != nil && !existingFile.IsDeleted {
		// 秒传成功：为当前用户创建新文件记录，复用已有的 MinIO 对象
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fileName), "."))
		newFile := &models.File{
			Id:            utils.NewUUID(),
			UserId:        userId,
			Name:          fileName,
			Size:          existingFile.Size,
			SizeStr:       existingFile.SizeStr,
			IsDir:         false,
			FileExtension: ext,
			OssObjectKey:  existingFile.OssObjectKey,
			FileHash:      fileHash,
			ParentId:      nullableParentID(parentId),
			IsDeleted:     false,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
			FileURL:       existingFile.FileURL,
			ThumbnailURL:  existingFile.ThumbnailURL,
		}
		err = s.db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(newFile).Error; err != nil {
				return fmt.Errorf("保存文件记录失败: %w", err)
			}
			return s.storageQuotaRepo.UpdateUsedSpace(tx, userId, newFile.Size)
		})
		if err != nil {
			return nil, err
		}
		return newFile, nil
	}

	// 检查文件大小是否超过用户配额
	availableSpace, err := s.storageQuotaRepo.GetAvailableSpace(userId)
	if err != nil {
		return nil, fmt.Errorf("获取可用空间失败: %w", err)
	}
	if fileSize > availableSpace {
		return nil, errors.New("存储空间不足，请升级存储配额")
	}

	// 上传文件至 MinIO
	uploadFile, err := s.minio.UploadFromStream(ctx, userId, r, fileName, fileSize, fileHash, parentId)
	if err != nil {
		return nil, fmt.Errorf("MinIO 上传失败: %w", err)
	}

	// 事务处理：入库 + 扣减配额
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(uploadFile).Error; err != nil {
			return fmt.Errorf("保存文件记录失败: %w", err)
		}
		return s.storageQuotaRepo.UpdateUsedSpace(tx, userId, fileSize)
	})

	if err != nil {
		if cleanupErr := s.minio.DeleteObjectWithThumbnail(context.Background(), uploadFile.OssObjectKey); cleanupErr != nil {
			slog.Error("cleanup uploaded object after db failure failed", "objectKey", uploadFile.OssObjectKey, "error", cleanupErr)
		}
		return nil, err
	}
	s.generateThumbnailAsync(uploadFile.Id, uploadFile.OssObjectKey)

	return uploadFile, nil
}

// InitChunkUpload 初始化分片上传
// 逻辑：秒传检查 -> 断点续传检查 -> 新建上传任务
func (s *fileService) InitChunkUpload(ctx context.Context, userId int, fileName, fileHash string, parentId string, fileSize int64, chunkSize int64, totalChunks int) (gin.H, error) {
	fileName = strings.TrimSpace(fileName)
	if err := validateFileName(fileName); err != nil {
		return nil, err
	}
	if err := s.ensureTargetFolder(ctx, userId, parentId); err != nil {
		return nil, err
	}
	if exists, err := s.fileRepo.CheckDuplicateName(userId, parentId, fileName); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.New("该目录下已有同名的文件")
	}
	if chunkSize <= 0 {
		chunkSize = defaultUploadChunkSize
	}
	// 分片参数强校验：防止恶意客户端用极小分片制造海量 Redis 字段（内存 DoS），
	// 或超过 MinIO 单次 Multipart Upload 的 10000 片上限。
	const (
		minChunkSize int64 = 1 * 1024 * 1024      // 1MB
		maxChunkSize int64 = 50 * 1024 * 1024     // 50MB
		maxChunks          = 10000                // MinIO/S3 硬上限
	)
	if chunkSize < minChunkSize {
		return nil, fmt.Errorf("分片大小过小（最小 %dMB）", minChunkSize/1024/1024)
	}
	if chunkSize > maxChunkSize {
		return nil, fmt.Errorf("分片大小过大（最大 %dMB）", maxChunkSize/1024/1024)
	}
	if totalChunks <= 0 && fileSize > 0 {
		totalChunks = int((fileSize + chunkSize - 1) / chunkSize)
	}
	if fileSize <= 0 || totalChunks <= 0 {
		return nil, errors.New("文件大小或分片数量无效")
	}
	if totalChunks > maxChunks {
		return nil, fmt.Errorf("分片数量过多（最多 %d 片），请增大分片大小", maxChunks)
	}

	// 1.秒传检查：仅在当前用户范围内匹配已有文件
	existingFile, err := s.fileRepo.GetFileByMD5(userId, fileHash)
	if err == nil && existingFile != nil && !existingFile.IsDeleted {
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fileName), "."))
		newFile := &models.File{
			Id:            utils.NewUUID(),
			UserId:        userId,
			Name:          fileName,
			Size:          existingFile.Size,
			SizeStr:       existingFile.SizeStr,
			IsDir:         false,
			FileExtension: ext,
			OssObjectKey:  existingFile.OssObjectKey,
			FileHash:      fileHash,
			ParentId:      nullableParentID(parentId),
			IsDeleted:     false,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
			FileURL:       existingFile.FileURL,
			ThumbnailURL:  existingFile.ThumbnailURL,
		}
		err = s.db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(newFile).Error; err != nil {
				return fmt.Errorf("保存文件记录失败: %w", err)
			}
			return s.storageQuotaRepo.UpdateUsedSpace(tx, userId, newFile.Size)
		})
		if err != nil {
			return nil, err
		}
		return gin.H{
			"finished": true,
			"file":     newFile,
			"url":      newFile.FileURL,
		}, nil
	}
	// 2.存储配额检查：如果文件大小超出配额，直接返回错误。
	remainingSpace, err := s.storageQuotaRepo.GetAvailableSpace(userId)
	if err != nil {
		return nil, fmt.Errorf("获取可用空间失败: %w", err)
	}
	if fileSize > remainingSpace {
		return nil, errors.New("存储空间不足，请升级存储配额")
	}

	// 3.断点续传检查
	sessionKey := fmt.Sprintf("upload:%d:%s", userId, fileHash)
	uploadId := ""
	objectKey := ""

	createSession := func() error {
		objectKey = s.minio.GenerateObjectKey(userId, parentId, fileName)
		var initErr error
		uploadId, initErr = s.minio.InitiateMultipartUpload(ctx, objectKey)
		if initErr != nil {
			return fmt.Errorf("初始化 OSS 上传失败: %w", initErr)
		}
		if err := s.redis.HSet(ctx, sessionKey,
			"id", uploadId,
			"key", objectKey,
			"fileName", fileName,
			"parentId", parentId,
			"fileSize", strconv.FormatInt(fileSize, 10),
			"chunkSize", strconv.FormatInt(chunkSize, 10),
			"totalChunks", strconv.Itoa(totalChunks),
		).Err(); err != nil {
			_ = s.minio.AbortMultipartUpload(ctx, objectKey, uploadId)
			return err
		}
		return s.refreshChunkUploadSession(ctx, sessionKey)
	}

	// 检查是否有进行中的上传会话
	sessionExists, err := s.redis.Exists(ctx, sessionKey).Result()

	if err != nil || sessionExists == 0 {
		// 3.1 无会话：初始化新的 MinIO 分片上传
		if err := createSession(); err != nil {
			return nil, err
		}
	} else {
		// 3.2 会话存在：从 Hash 中读取 uploadId 和 objectKey
		uploadId, _ = s.redis.HGet(ctx, sessionKey, "id").Result()
		objectKey, _ = s.redis.HGet(ctx, sessionKey, "key").Result()
		if uploadId == "" || objectKey == "" {
			return nil, errors.New("上传任务状态异常，请取消后重新上传")
		}
		if expired, _ := s.isChunkUploadSessionExpired(ctx, sessionKey); expired {
			_ = s.minio.AbortMultipartUpload(ctx, objectKey, uploadId)
			s.deleteChunkUploadSession(ctx, sessionKey)
			if err := createSession(); err != nil {
				return nil, err
			}
		} else {
			// 会话冲突检查：同一用户对相同内容(hash)的上传会话只能有一个。
			// 若 fileName/parentId 与已有会话不一致，说明该文件正在上传到其他位置，
			// 直接复用会导致分片互相污染、合并失败。
			storedName, _ := s.redis.HGet(ctx, sessionKey, "fileName").Result()
			storedParent, _ := s.redis.HGet(ctx, sessionKey, "parentId").Result()
			if storedName != "" && (storedName != fileName || storedParent != parentId) {
				return nil, errors.New("相同内容的文件正在上传到其他位置，请稍后重试或先取消该上传")
			}
			if err := s.redis.HSet(ctx, sessionKey,
				"fileName", fileName,
				"parentId", parentId,
				"fileSize", strconv.FormatInt(fileSize, 10),
				"chunkSize", strconv.FormatInt(chunkSize, 10),
				"totalChunks", strconv.Itoa(totalChunks),
			).Err(); err != nil {
				return nil, err
			} else if err := s.refreshChunkUploadSession(ctx, sessionKey); err != nil {
				return nil, err
			}
		}
	}

	// 4.获取已上传的分片列表（从 Hash 中读取 ETag 字段，排除 id/key/锁字段）
	allFields, err := s.redis.HGetAll(ctx, sessionKey).Result()

	uploadedChunks := make([]int, 0)
	if err == nil {
		for k := range allFields {
			// 跳过元数据字段和 hash 字段，只提取纯数字 key（分片索引）
			if isChunkUploadMetadataField(k) {
				continue
			}
			idx, convErr := strconv.Atoi(k)
			if convErr == nil {
				uploadedChunks = append(uploadedChunks, idx)
			}
		}
	}

	// 排序，方便前端处理
	sort.Ints(uploadedChunks)

	return gin.H{
		"finished":       false,
		"fileHash":       fileHash,
		"uploadId":       uploadId,
		"uploadedChunks": uploadedChunks,
		"chunkSize":      chunkSize,
		"totalChunks":    totalChunks,
	}, nil
}

// UploadChunk 流式上传单个分片，可选 hash 校验
// expectedChunkHash 为空时跳过校验；不为空时，服务端边上传边计算 SHA-256 并比对
func (s *fileService) UploadChunk(ctx context.Context, userId int, fileHash string, chunkIndex int, r io.Reader, chunkSize int64, expectedChunkHash string) error {
	sessionKey := fmt.Sprintf("upload:%d:%s", userId, fileHash)

	uploadId, err := s.redis.HGet(ctx, sessionKey, "id").Result()
	if err != nil || uploadId == "" {
		return errors.New("上传任务不存在或已过期，请重新初始化")
	}
	objectKey, err := s.redis.HGet(ctx, sessionKey, "key").Result()
	if err != nil {
		return errors.New("文件路径丢失")
	}
	if expired, _ := s.isChunkUploadSessionExpired(ctx, sessionKey); expired {
		_ = s.minio.AbortMultipartUpload(ctx, objectKey, uploadId)
		s.deleteChunkUploadSession(ctx, sessionKey)
		return errors.New("上传任务已过期，请重新初始化")
	}

	fileSize, chunkUnitSize, totalChunks, err := s.getChunkUploadMetadata(ctx, sessionKey)
	if err != nil {
		return err
	}
	if chunkIndex < 0 || chunkIndex >= totalChunks {
		return fmt.Errorf("分片索引越界: %d", chunkIndex)
	}
	expectedSize := chunkUnitSize
	if chunkIndex == totalChunks-1 {
		expectedSize = fileSize - int64(totalChunks-1)*chunkUnitSize
	}
	if expectedSize <= 0 {
		return errors.New("分片大小元数据异常")
	}
	if chunkSize != expectedSize {
		return fmt.Errorf("分片大小校验失败: index=%d got=%d expected=%d", chunkIndex, chunkSize, expectedSize)
	}

	partNumber := chunkIndex + 1
	partInfo, computedHash, err := s.minio.UploadPart(ctx, objectKey, uploadId, partNumber, r, chunkSize, expectedChunkHash)
	if err != nil {
		return fmt.Errorf("OSS 分片上传失败: %w", err)
	}

	// 幂等存储：ETag + 分片 hash 写入同一个 Hash
	err = s.redis.HSet(ctx, sessionKey,
		strconv.Itoa(chunkIndex), partInfo.ETag,
		strconv.Itoa(chunkIndex)+"_hash", computedHash,
		strconv.Itoa(chunkIndex)+"_size", strconv.FormatInt(chunkSize, 10),
	).Err()
	if err != nil {
		return err
	}

	// 单次 Expire 刷新整个会话的 TTL
	if err := s.refreshChunkUploadSession(ctx, sessionKey); err != nil {
		return err
	}
	return nil
}

// MergeChunks 合并分片
func (s *fileService) MergeChunks(ctx context.Context, userId int, fileHash, fileName, parentId string, fileSize int64, chunkSize int64, totalChunks int) (*models.File, error) {
	fileName = strings.TrimSpace(fileName)
	if err := validateFileName(fileName); err != nil {
		return nil, err
	}
	if err := s.ensureTargetFolder(ctx, userId, parentId); err != nil {
		return nil, err
	}

	sessionKey := fmt.Sprintf("upload:%d:%s", userId, fileHash)

	// 分布式锁：TTL 必须覆盖完整的合并流程（含合并后整文件 SHA-256 校验，
	// 大文件全量下载校验可能超过 30 秒），防止锁过期后第二个合并请求并发进入。
	lockKey := fmt.Sprintf("upload:%d:%s:lock", userId, fileHash)
	locked, err := s.redis.SetNX(ctx, lockKey, "1", 10*time.Minute).Result()
	if err != nil || !locked {
		return nil, errors.New("合并正在进行中，请稍后重试")
	}
	defer s.redis.Del(ctx, lockKey)

	uploadId, err := s.redis.HGet(ctx, sessionKey, "id").Result()
	if err != nil || uploadId == "" {
		return nil, errors.New("上传任务失败")
	}
	objectKey, err := s.redis.HGet(ctx, sessionKey, "key").Result()
	if err != nil || objectKey == "" {
		return nil, errors.New("文件路径丢失")
	}
	if expired, _ := s.isChunkUploadSessionExpired(ctx, sessionKey); expired {
		_ = s.minio.AbortMultipartUpload(ctx, objectKey, uploadId)
		s.deleteChunkUploadSession(ctx, sessionKey)
		return nil, errors.New("上传任务已过期，请重新初始化")
	}

	// 1.获取所有分片 ETag（过滤 id/key 和非数字字段）
	allFields, err := s.redis.HGetAll(ctx, sessionKey).Result()
	if err != nil || len(allFields) <= 2 { // 只有 id 和 key，没有分片
		return nil, errors.New("未找到已上传的分片数据")
	}
	storedFileSize, storedChunkSize, storedTotalChunks, err := parseChunkUploadMetadata(allFields)
	if err != nil {
		return nil, err
	}
	if storedFileName := allFields["fileName"]; storedFileName != "" && storedFileName != fileName {
		return nil, errors.New("文件名与上传会话不一致")
	}
	if storedParentId, ok := allFields["parentId"]; ok && storedParentId != parentId {
		return nil, errors.New("父目录与上传会话不一致")
	}
	if fileSize <= 0 {
		fileSize = storedFileSize
	} else if storedFileSize > 0 && fileSize != storedFileSize {
		return nil, fmt.Errorf("文件大小与上传会话不一致: got=%d expected=%d", fileSize, storedFileSize)
	}
	if chunkSize <= 0 {
		chunkSize = storedChunkSize
	} else if storedChunkSize > 0 && chunkSize != storedChunkSize {
		return nil, fmt.Errorf("分片大小与上传会话不一致: got=%d expected=%d", chunkSize, storedChunkSize)
	}
	if chunkSize <= 0 {
		chunkSize = defaultUploadChunkSize
	}
	if totalChunks <= 0 {
		totalChunks = storedTotalChunks
	} else if storedTotalChunks > 0 && totalChunks != storedTotalChunks {
		return nil, fmt.Errorf("分片数量与上传会话不一致: got=%d expected=%d", totalChunks, storedTotalChunks)
	}
	if totalChunks <= 0 && fileSize > 0 {
		totalChunks = int((fileSize + chunkSize - 1) / chunkSize)
	}
	if totalChunks <= 0 {
		return nil, errors.New("分片数量无效")
	}
	if exists, err := s.fileRepo.CheckDuplicateName(userId, parentId, fileName); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.New("该目录下已有同名的文件")
	}

	var completeParts []minio.CompletePart
	seenParts := make(map[int]bool, totalChunks)
	var uploadedSize int64
	for k, v := range allFields {
		if isChunkUploadMetadataField(k) {
			continue
		}
		idx, convErr := strconv.Atoi(k)
		if convErr == nil {
			if idx < 0 || idx >= totalChunks {
				return nil, fmt.Errorf("分片索引越界: %d", idx)
			}
			seenParts[idx] = true
			if partSize, sizeErr := strconv.ParseInt(allFields[strconv.Itoa(idx)+"_size"], 10, 64); sizeErr == nil {
				uploadedSize += partSize
			}
			completeParts = append(completeParts, minio.CompletePart{
				PartNumber: idx + 1,
				ETag:       v,
			})
		}
	}
	if len(completeParts) != totalChunks {
		return nil, fmt.Errorf("分片不完整: 已上传 %d/%d", len(completeParts), totalChunks)
	}
	for idx := 0; idx < totalChunks; idx++ {
		if !seenParts[idx] {
			return nil, fmt.Errorf("缺少分片: %d", idx)
		}
	}
	if uploadedSize > 0 && uploadedSize != fileSize {
		return nil, fmt.Errorf("分片大小校验失败: got=%d expected=%d", uploadedSize, fileSize)
	}

	// 按 PartNumber 升序
	sort.Slice(completeParts, func(i, j int) bool {
		return completeParts[i].PartNumber < completeParts[j].PartNumber
	})

	// 2.调用 MinIO 合并
	fileURL, thumbnailURL, err := s.minio.CompleteMultipartUpload(ctx, objectKey, uploadId, completeParts)
	if err != nil {
		return nil, fmt.Errorf("OSS 合并失败: %w", err)
	}
	if objectSize, err := s.minio.GetObjectInfo(ctx, objectKey); err != nil {
		_ = s.minio.DeleteFile(ctx, objectKey)
		return nil, fmt.Errorf("获取合并对象信息失败: %w", err)
	} else if objectSize != fileSize {
		_ = s.minio.DeleteFile(ctx, objectKey)
		return nil, fmt.Errorf("合并对象大小校验失败: got=%d expected=%d", objectSize, fileSize)
	}
	if len(fileHash) == 64 {
		computedHash, err := s.minio.ComputeObjectSHA256(ctx, objectKey)
		if err != nil {
			_ = s.minio.DeleteFile(ctx, objectKey)
			return nil, fmt.Errorf("计算合并对象hash失败: %w", err)
		}
		if !strings.EqualFold(computedHash, fileHash) {
			_ = s.minio.DeleteFile(ctx, objectKey)
			return nil, errors.New("合并对象hash校验失败")
		}
	}

	// 3.写入数据库
	ext := strings.TrimPrefix(filepath.Ext(fileName), ".")

	newFile := &models.File{
		Id:            utils.NewUUID(),
		UserId:        userId,
		Name:          fileName,
		ParentId:      nullableParentID(parentId),
		OssObjectKey:  objectKey,
		FileHash:      fileHash,
		FileURL:       fileURL,
		ThumbnailURL:  thumbnailURL,
		Size:          fileSize,
		SizeStr:       utils.FormatFileSize(fileSize),
		FileExtension: ext,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(newFile).Error; err != nil {
			return err
		}
		return s.storageQuotaRepo.UpdateUsedSpace(tx, userId, fileSize)
	})
	if err != nil {
		if cleanupErr := s.minio.DeleteObjectWithThumbnail(context.Background(), objectKey); cleanupErr != nil {
			slog.Error("cleanup merged object after db failure failed", "objectKey", objectKey, "error", cleanupErr)
		}
		s.deleteChunkUploadSession(context.Background(), sessionKey)
		return nil, err
	}
	s.generateThumbnailAsync(newFile.Id, newFile.OssObjectKey)

	// 4.清理：一次 DEL 清掉整个 Hash
	s.deleteChunkUploadSession(ctx, sessionKey)

	return newFile, nil
}

// CancelChunkUpload 取消上传
func (s *fileService) CancelChunkUpload(ctx context.Context, userId int, fileHash string) error {
	sessionKey := fmt.Sprintf("upload:%d:%s", userId, fileHash)

	uploadId, err := s.redis.HGet(ctx, sessionKey, "id").Result()
	objectKey, _ := s.redis.HGet(ctx, sessionKey, "key").Result()
	if err == nil && uploadId != "" && objectKey != "" {
		_ = s.minio.AbortMultipartUpload(ctx, objectKey, uploadId)
	}

	// 一次 DEL 清理整个会话
	s.deleteChunkUploadSession(ctx, sessionKey)
	return nil
}

func (s *fileService) StartChunkUploadCleanup(ctx context.Context) {
	if s.redis == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(chunkUploadCleanupInterval)
		defer ticker.Stop()

		s.cleanupExpiredChunkUploads(ctx, chunkUploadCleanupBatchSize)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.cleanupExpiredChunkUploads(ctx, chunkUploadCleanupBatchSize)
			}
		}
	}()
}

func (s *fileService) cleanupExpiredChunkUploads(ctx context.Context, limit int64) {
	now := time.Now().Unix()
	sessions, err := s.redis.ZRangeByScore(ctx, chunkUploadSessionSet, &redis.ZRangeBy{
		Min:    "-inf",
		Max:    strconv.FormatInt(now, 10),
		Offset: 0,
		Count:  limit,
	}).Result()
	if err != nil {
		slog.Error("scan expired chunk upload sessions failed", "error", err)
		return
	}

	for _, sessionKey := range sessions {
		uploadId, _ := s.redis.HGet(ctx, sessionKey, "id").Result()
		objectKey, _ := s.redis.HGet(ctx, sessionKey, "key").Result()
		if uploadId != "" && objectKey != "" {
			if err := s.minio.AbortMultipartUpload(ctx, objectKey, uploadId); err != nil {
				slog.Error("abort expired chunk upload failed", "sessionKey", sessionKey, "objectKey", objectKey, "error", err)
			}
		}
		s.deleteChunkUploadSession(ctx, sessionKey)
	}
}

func (s *fileService) refreshChunkUploadSession(ctx context.Context, sessionKey string) error {
	expiresAt := time.Now().Add(chunkUploadSessionTTL).Unix()
	if err := s.redis.HSet(ctx, sessionKey, "expiresAt", strconv.FormatInt(expiresAt, 10)).Err(); err != nil {
		return err
	}
	if err := s.redis.Expire(ctx, sessionKey, chunkUploadMetadataTTL).Err(); err != nil {
		return err
	}
	return s.redis.ZAdd(ctx, chunkUploadSessionSet, &redis.Z{
		Score:  float64(expiresAt),
		Member: sessionKey,
	}).Err()
}

func (s *fileService) deleteChunkUploadSession(ctx context.Context, sessionKey string) {
	s.redis.Del(ctx, sessionKey)
	s.redis.ZRem(ctx, chunkUploadSessionSet, sessionKey)
}

func (s *fileService) isChunkUploadSessionExpired(ctx context.Context, sessionKey string) (bool, error) {
	expiresAtStr, err := s.redis.HGet(ctx, sessionKey, "expiresAt").Result()
	if err != nil || expiresAtStr == "" {
		return false, err
	}
	expiresAt, err := strconv.ParseInt(expiresAtStr, 10, 64)
	if err != nil {
		return false, err
	}
	return expiresAt <= time.Now().Unix(), nil
}

func (s *fileService) getChunkUploadMetadata(ctx context.Context, sessionKey string) (int64, int64, int, error) {
	fields, err := s.redis.HGetAll(ctx, sessionKey).Result()
	if err != nil {
		return 0, 0, 0, err
	}
	return parseChunkUploadMetadata(fields)
}

func parseChunkUploadMetadata(fields map[string]string) (int64, int64, int, error) {
	fileSize, err := strconv.ParseInt(fields["fileSize"], 10, 64)
	if err != nil || fileSize <= 0 {
		return 0, 0, 0, errors.New("文件大小元数据无效")
	}
	chunkSize, err := strconv.ParseInt(fields["chunkSize"], 10, 64)
	if err != nil || chunkSize <= 0 {
		return 0, 0, 0, errors.New("分片大小元数据无效")
	}
	totalChunks, err := strconv.Atoi(fields["totalChunks"])
	if err != nil || totalChunks <= 0 {
		return 0, 0, 0, errors.New("分片数量元数据无效")
	}
	expectedTotalChunks := int((fileSize + chunkSize - 1) / chunkSize)
	if totalChunks != expectedTotalChunks {
		return 0, 0, 0, fmt.Errorf("分片数量元数据不一致: got=%d expected=%d", totalChunks, expectedTotalChunks)
	}
	return fileSize, chunkSize, totalChunks, nil
}

func isChunkUploadMetadataField(key string) bool {
	switch key {
	case "id", "key", "fileName", "parentId", "fileSize", "chunkSize", "totalChunks", "expiresAt":
		return true
	default:
		return strings.HasSuffix(key, "_hash") || strings.HasSuffix(key, "_size")
	}
}

func (s *fileService) generateThumbnailAsync(fileID, objectKey string) {
	if fileID == "" || objectKey == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		thumbURL, err := s.minio.GenerateThumbnailForObject(ctx, objectKey)
		if err != nil {
			slog.Warn("generate thumbnail failed", "fileId", fileID, "objectKey", objectKey, "error", err)
			return
		}
		if thumbURL == "" {
			return
		}
		result := s.db.WithContext(ctx).
			Model(&models.File{}).
			Where("id = ? AND is_deleted = ?", fileID, false).
			Update("thumbnail_url", thumbURL)
		if result.Error != nil || result.RowsAffected == 0 {
			if result.Error != nil {
				slog.Error("update async thumbnail failed", "fileId", fileID, "error", result.Error)
			}
			_ = s.minio.DeleteThumbnailForObject(context.Background(), objectKey)
		}
	}()
}

// SearchFiles 搜索文件和文件夹
