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
	"go-cloud-storage/backend/internal/ports"
	"go-cloud-storage/backend/pkg/utils"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// UploadApplication is the transport-neutral application boundary for uploads.
// Gin/HTTP request binding belongs in controller; object-storage SDK types belong
// in infrastructure adapters.
type UploadApplication interface {
	UploadFile(ctx context.Context, r io.Reader, userID int, fileName string, fileSize int64, fileHash, parentID string) (*models.File, error)
	InitChunkUpload(ctx context.Context, input InitChunkUploadInput) (*InitChunkUploadResult, error)
	UploadChunk(ctx context.Context, userID int, fileHash string, chunkIndex int, r io.Reader, chunkSize int64, expectedChunkHash string) error
	MergeChunks(ctx context.Context, userID int, fileHash, fileName, parentID string, fileSize, chunkSize int64, totalChunks int) (*models.File, error)
	CancelChunkUpload(ctx context.Context, userID int, fileHash string) error
	GetChunkUploadProgress(ctx context.Context, userID int, fileHash string) (*ChunkUploadProgress, error)
	StartChunkUploadCleanup(ctx context.Context)
}

// FileUploadedEvent is intentionally transport-neutral. It can be serialized by
// a RabbitMQ/Outbox adapter later without changing upload business logic.
type FileUploadedEvent struct {
	FileID    string `json:"fileId"`
	FileName  string `json:"fileName"`
	ObjectKey string `json:"objectKey"`
	Size      int64  `json:"size"`
}

type uploadApplication struct {
	*fileService
	storage ports.Storage
	events  ports.EventBus
}

func NewUploadApplication(legacy FileService, storage ports.Storage, events ports.EventBus) UploadApplication {
	base, ok := legacy.(*fileService)
	if !ok {
		panic("NewUploadApplication requires the legacy *fileService during phase-1 migration")
	}
	if storage == nil {
		panic("NewUploadApplication requires a Storage implementation")
	}
	return &uploadApplication{fileService: base, storage: storage, events: events}
}

func (s *uploadApplication) UploadFile(ctx context.Context, r io.Reader, userID int, fileName string, fileSize int64, fileHash, parentID string) (*models.File, error) {
	fileName = strings.TrimSpace(fileName)
	if err := validateFileName(fileName); err != nil {
		return nil, err
	}
	if err := s.ensureTargetFolder(ctx, userID, parentID); err != nil {
		return nil, err
	}
	if exists, err := s.fileRepo.CheckDuplicateName(userID, parentID, fileName); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.New("该目录下已有同名的文件")
	}

	// 秒传仅复用当前用户已有对象，避免通过已知 hash 探测其他用户文件。
	if existing, err := s.fileRepo.GetFileByMD5(userID, fileHash); err == nil && existing != nil && !existing.IsDeleted {
		file := cloneFileRecord(existing, userID, fileName, fileHash, parentID)
		if err := s.persistUploadedFile(ctx, file); err != nil {
			return nil, err
		}
		s.publishFileUploaded(ctx, file)
		return file, nil
	}

	available, err := s.storageQuotaRepo.GetAvailableSpace(userID)
	if err != nil {
		return nil, fmt.Errorf("获取可用空间失败: %w", err)
	}
	if fileSize > available {
		return nil, errors.New("存储空间不足，请升级存储配额")
	}

	object, err := s.storage.UploadFromStream(ctx, userID, r, fileName, fileSize, fileHash, parentID)
	if err != nil {
		return nil, fmt.Errorf("对象存储上传失败: %w", err)
	}

	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fileName), "."))
	file := &models.File{
		Id:            utils.NewUUID(),
		UserId:        userID,
		Name:          fileName,
		Size:          object.Size,
		SizeStr:       utils.FormatFileSize(object.Size),
		IsDir:         false,
		FileExtension: ext,
		OssObjectKey:  object.ObjectKey,
		FileHash:      fileHash,
		ParentId:      nullableParentID(parentID),
		IsDeleted:     false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		FileURL:       object.URL,
		ThumbnailURL:  object.ThumbnailURL,
	}

	if err := s.persistUploadedFile(ctx, file); err != nil {
		s.cleanupObject(ctx, object.ObjectKey)
		return nil, err
	}

	s.generateThumbnailAsync(file.Id, file.OssObjectKey)
	s.publishFileUploaded(ctx, file)
	return file, nil
}

func (s *uploadApplication) InitChunkUpload(ctx context.Context, input InitChunkUploadInput) (*InitChunkUploadResult, error) {
	fileName := strings.TrimSpace(input.FileName)
	if err := validateFileName(fileName); err != nil {
		return nil, err
	}
	if err := s.ensureTargetFolder(ctx, input.UserID, input.ParentID); err != nil {
		return nil, err
	}
	if exists, err := s.fileRepo.CheckDuplicateName(input.UserID, input.ParentID, fileName); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.New("该目录下已有同名的文件")
	}

	chunkSize := input.ChunkSize
	if chunkSize <= 0 {
		chunkSize = defaultUploadChunkSize
	}
	const (
		minChunkSize int64 = 1 * 1024 * 1024
		maxChunkSize int64 = 50 * 1024 * 1024
		maxChunks          = 10000
	)
	if chunkSize < minChunkSize {
		return nil, fmt.Errorf("分片大小过小（最小 %dMB）", minChunkSize/1024/1024)
	}
	if chunkSize > maxChunkSize {
		return nil, fmt.Errorf("分片大小过大（最大 %dMB）", maxChunkSize/1024/1024)
	}
	totalChunks := input.TotalChunks
	if totalChunks <= 0 && input.FileSize > 0 {
		totalChunks = int((input.FileSize + chunkSize - 1) / chunkSize)
	}
	if input.FileSize <= 0 || totalChunks <= 0 {
		return nil, errors.New("文件大小或分片数量无效")
	}
	if totalChunks > maxChunks {
		return nil, fmt.Errorf("分片数量过多（最多 %d 片），请增大分片大小", maxChunks)
	}

	if existing, err := s.fileRepo.GetFileByMD5(input.UserID, input.FileHash); err == nil && existing != nil && !existing.IsDeleted {
		file := cloneFileRecord(existing, input.UserID, fileName, input.FileHash, input.ParentID)
		if err := s.persistUploadedFile(ctx, file); err != nil {
			return nil, err
		}
		s.publishFileUploaded(ctx, file)
		return &InitChunkUploadResult{Finished: true, File: file, URL: file.FileURL}, nil
	}

	remaining, err := s.storageQuotaRepo.GetAvailableSpace(input.UserID)
	if err != nil {
		return nil, fmt.Errorf("获取可用空间失败: %w", err)
	}
	if input.FileSize > remaining {
		return nil, errors.New("存储空间不足，请升级存储配额")
	}

	sessionKey := fmt.Sprintf("upload:%d:%s", input.UserID, input.FileHash)
	var uploadID, objectKey string
	createSession := func() error {
		objectKey = s.storage.GenerateObjectKey(input.UserID, input.ParentID, fileName)
		var initErr error
		uploadID, initErr = s.storage.InitiateMultipartUpload(ctx, objectKey)
		if initErr != nil {
			return fmt.Errorf("初始化对象存储分片上传失败: %w", initErr)
		}
		if err := s.redis.HSet(ctx, sessionKey,
			"id", uploadID,
			"key", objectKey,
			"fileName", fileName,
			"parentId", input.ParentID,
			"fileSize", strconv.FormatInt(input.FileSize, 10),
			"chunkSize", strconv.FormatInt(chunkSize, 10),
			"totalChunks", strconv.Itoa(totalChunks),
		).Err(); err != nil {
			_ = s.storage.AbortMultipartUpload(ctx, objectKey, uploadID)
			return err
		}
		return s.refreshChunkUploadSession(ctx, sessionKey)
	}

	sessionExists, err := s.redis.Exists(ctx, sessionKey).Result()
	if err != nil || sessionExists == 0 {
		if err := createSession(); err != nil {
			return nil, err
		}
	} else {
		uploadID, _ = s.redis.HGet(ctx, sessionKey, "id").Result()
		objectKey, _ = s.redis.HGet(ctx, sessionKey, "key").Result()
		if uploadID == "" || objectKey == "" {
			return nil, errors.New("上传任务状态异常，请取消后重新上传")
		}
		if expired, _ := s.isChunkUploadSessionExpired(ctx, sessionKey); expired {
			_ = s.storage.AbortMultipartUpload(ctx, objectKey, uploadID)
			s.deleteChunkUploadSession(ctx, sessionKey)
			if err := createSession(); err != nil {
				return nil, err
			}
		} else {
			storedName, _ := s.redis.HGet(ctx, sessionKey, "fileName").Result()
			storedParent, _ := s.redis.HGet(ctx, sessionKey, "parentId").Result()
			if storedName != "" && (storedName != fileName || storedParent != input.ParentID) {
				return nil, errors.New("相同内容的文件正在上传到其他位置，请稍后重试或先取消该上传")
			}
			if err := s.redis.HSet(ctx, sessionKey,
				"fileName", fileName,
				"parentId", input.ParentID,
				"fileSize", strconv.FormatInt(input.FileSize, 10),
				"chunkSize", strconv.FormatInt(chunkSize, 10),
				"totalChunks", strconv.Itoa(totalChunks),
			).Err(); err != nil {
				return nil, err
			}
			if err := s.refreshChunkUploadSession(ctx, sessionKey); err != nil {
				return nil, err
			}
		}
	}

	fields, err := s.redis.HGetAll(ctx, sessionKey).Result()
	uploaded := make([]int, 0)
	if err == nil {
		for key := range fields {
			if isChunkUploadMetadataField(key) {
				continue
			}
			idx, convErr := strconv.Atoi(key)
			if convErr == nil {
				uploaded = append(uploaded, idx)
			}
		}
	}
	sort.Ints(uploaded)

	return &InitChunkUploadResult{
		Finished:       false,
		FileHash:       input.FileHash,
		UploadID:       uploadID,
		UploadedChunks: uploaded,
		ChunkSize:      chunkSize,
		TotalChunks:    totalChunks,
	}, nil
}

func (s *uploadApplication) UploadChunk(ctx context.Context, userID int, fileHash string, chunkIndex int, r io.Reader, chunkSize int64, expectedChunkHash string) error {
	sessionKey := fmt.Sprintf("upload:%d:%s", userID, fileHash)
	uploadID, err := s.redis.HGet(ctx, sessionKey, "id").Result()
	if err != nil || uploadID == "" {
		return errors.New("上传任务不存在或已过期，请重新初始化")
	}
	objectKey, err := s.redis.HGet(ctx, sessionKey, "key").Result()
	if err != nil || objectKey == "" {
		return errors.New("文件路径丢失")
	}
	if expired, _ := s.isChunkUploadSessionExpired(ctx, sessionKey); expired {
		_ = s.storage.AbortMultipartUpload(ctx, objectKey, uploadID)
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
	if expectedSize <= 0 || chunkSize != expectedSize {
		return fmt.Errorf("分片大小校验失败: index=%d got=%d expected=%d", chunkIndex, chunkSize, expectedSize)
	}

	part, computedHash, err := s.storage.UploadPart(ctx, objectKey, uploadID, chunkIndex+1, r, chunkSize, expectedChunkHash)
	if err != nil {
		return fmt.Errorf("对象存储分片上传失败: %w", err)
	}
	if err := s.redis.HSet(ctx, sessionKey,
		strconv.Itoa(chunkIndex), part.ETag,
		strconv.Itoa(chunkIndex)+"_hash", computedHash,
		strconv.Itoa(chunkIndex)+"_size", strconv.FormatInt(chunkSize, 10),
	).Err(); err != nil {
		return err
	}
	return s.refreshChunkUploadSession(ctx, sessionKey)
}

func (s *uploadApplication) MergeChunks(ctx context.Context, userID int, fileHash, fileName, parentID string, fileSize, chunkSize int64, totalChunks int) (*models.File, error) {
	fileName = strings.TrimSpace(fileName)
	if err := validateFileName(fileName); err != nil {
		return nil, err
	}
	if err := s.ensureTargetFolder(ctx, userID, parentID); err != nil {
		return nil, err
	}

	sessionKey := fmt.Sprintf("upload:%d:%s", userID, fileHash)
	lockKey := fmt.Sprintf("upload:%d:%s:lock", userID, fileHash)
	locked, err := s.redis.SetNX(ctx, lockKey, "1", 10*time.Minute).Result()
	if err != nil || !locked {
		return nil, errors.New("合并正在进行中，请稍后重试")
	}
	defer s.redis.Del(ctx, lockKey)

	uploadID, err := s.redis.HGet(ctx, sessionKey, "id").Result()
	if err != nil || uploadID == "" {
		return nil, errors.New("上传任务失败")
	}
	objectKey, err := s.redis.HGet(ctx, sessionKey, "key").Result()
	if err != nil || objectKey == "" {
		return nil, errors.New("文件路径丢失")
	}
	if expired, _ := s.isChunkUploadSessionExpired(ctx, sessionKey); expired {
		_ = s.storage.AbortMultipartUpload(ctx, objectKey, uploadID)
		s.deleteChunkUploadSession(ctx, sessionKey)
		return nil, errors.New("上传任务已过期，请重新初始化")
	}

	fields, err := s.redis.HGetAll(ctx, sessionKey).Result()
	if err != nil || len(fields) <= 2 {
		return nil, errors.New("未找到已上传的分片数据")
	}
	storedFileSize, storedChunkSize, storedTotalChunks, err := parseChunkUploadMetadata(fields)
	if err != nil {
		return nil, err
	}
	if storedName := fields["fileName"]; storedName != "" && storedName != fileName {
		return nil, errors.New("文件名与上传会话不一致")
	}
	if storedParent, ok := fields["parentId"]; ok && storedParent != parentID {
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
	if totalChunks <= 0 {
		totalChunks = storedTotalChunks
	} else if storedTotalChunks > 0 && totalChunks != storedTotalChunks {
		return nil, fmt.Errorf("分片数量与上传会话不一致: got=%d expected=%d", totalChunks, storedTotalChunks)
	}
	if exists, err := s.fileRepo.CheckDuplicateName(userID, parentID, fileName); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.New("该目录下已有同名的文件")
	}

	parts := make([]ports.CompletedPart, 0, totalChunks)
	seen := make(map[int]bool, totalChunks)
	var uploadedSize int64
	for key, etag := range fields {
		if isChunkUploadMetadataField(key) {
			continue
		}
		idx, convErr := strconv.Atoi(key)
		if convErr != nil {
			continue
		}
		if idx < 0 || idx >= totalChunks {
			return nil, fmt.Errorf("分片索引越界: %d", idx)
		}
		seen[idx] = true
		if partSize, sizeErr := strconv.ParseInt(fields[strconv.Itoa(idx)+"_size"], 10, 64); sizeErr == nil {
			uploadedSize += partSize
		}
		parts = append(parts, ports.CompletedPart{PartNumber: idx + 1, ETag: etag})
	}
	if len(parts) != totalChunks {
		return nil, fmt.Errorf("分片不完整: 已上传 %d/%d", len(parts), totalChunks)
	}
	for idx := 0; idx < totalChunks; idx++ {
		if !seen[idx] {
			return nil, fmt.Errorf("缺少分片: %d", idx)
		}
	}
	if uploadedSize > 0 && uploadedSize != fileSize {
		return nil, fmt.Errorf("分片大小校验失败: got=%d expected=%d", uploadedSize, fileSize)
	}
	sort.Slice(parts, func(i, j int) bool { return parts[i].PartNumber < parts[j].PartNumber })

	object, err := s.storage.CompleteMultipartUpload(ctx, objectKey, uploadID, parts)
	if err != nil {
		return nil, fmt.Errorf("对象存储合并失败: %w", err)
	}
	if object.Size != fileSize {
		_ = s.storage.DeleteFile(ctx, objectKey)
		return nil, fmt.Errorf("合并对象大小校验失败: got=%d expected=%d", object.Size, fileSize)
	}
	if len(fileHash) == 64 {
		computedHash, hashErr := s.storage.ComputeObjectSHA256(ctx, objectKey)
		if hashErr != nil {
			_ = s.storage.DeleteFile(ctx, objectKey)
			return nil, fmt.Errorf("计算合并对象hash失败: %w", hashErr)
		}
		if !strings.EqualFold(computedHash, fileHash) {
			_ = s.storage.DeleteFile(ctx, objectKey)
			return nil, errors.New("合并对象hash校验失败")
		}
	}

	ext := strings.TrimPrefix(filepath.Ext(fileName), ".")
	file := &models.File{
		Id:            utils.NewUUID(),
		UserId:        userID,
		Name:          fileName,
		ParentId:      nullableParentID(parentID),
		OssObjectKey:  objectKey,
		FileHash:      fileHash,
		FileURL:       object.URL,
		ThumbnailURL:  object.ThumbnailURL,
		Size:          fileSize,
		SizeStr:       utils.FormatFileSize(fileSize),
		FileExtension: ext,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := s.persistUploadedFile(ctx, file); err != nil {
		s.cleanupObject(ctx, objectKey)
		s.deleteChunkUploadSession(cleanupContext(ctx), sessionKey)
		return nil, err
	}

	s.generateThumbnailAsync(file.Id, file.OssObjectKey)
	s.deleteChunkUploadSession(ctx, sessionKey)
	s.publishFileUploaded(ctx, file)
	return file, nil
}

func (s *uploadApplication) CancelChunkUpload(ctx context.Context, userID int, fileHash string) error {
	sessionKey := fmt.Sprintf("upload:%d:%s", userID, fileHash)
	uploadID, err := s.redis.HGet(ctx, sessionKey, "id").Result()
	objectKey, _ := s.redis.HGet(ctx, sessionKey, "key").Result()
	if err == nil && uploadID != "" && objectKey != "" {
		_ = s.storage.AbortMultipartUpload(ctx, objectKey, uploadID)
	}
	s.deleteChunkUploadSession(ctx, sessionKey)
	return nil
}

func (s *uploadApplication) GetChunkUploadProgress(ctx context.Context, userID int, fileHash string) (*ChunkUploadProgress, error) {
	sessionKey := fmt.Sprintf("upload:%d:%s", userID, fileHash)
	uploadID, err := s.redis.HGet(ctx, sessionKey, "id").Result()
	if err != nil || uploadID == "" {
		return &ChunkUploadProgress{Status: "not_found", UploadedChunks: []int{}}, nil
	}
	fields, err := s.redis.HGetAll(ctx, sessionKey).Result()
	if err != nil {
		return nil, err
	}
	uploaded := make([]int, 0)
	for key := range fields {
		if isChunkUploadMetadataField(key) {
			continue
		}
		idx, convErr := strconv.Atoi(key)
		if convErr == nil {
			uploaded = append(uploaded, idx)
		}
	}
	sort.Ints(uploaded)
	return &ChunkUploadProgress{Status: "in_progress", UploadID: uploadID, UploadedChunks: uploaded, UploadedCount: len(uploaded)}, nil
}

func (s *uploadApplication) StartChunkUploadCleanup(ctx context.Context) {
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

func (s *uploadApplication) cleanupExpiredChunkUploads(ctx context.Context, limit int64) {
	sessions, err := s.redis.ZRangeByScore(ctx, chunkUploadSessionSet, &redis.ZRangeBy{
		Min: "-inf", Max: strconv.FormatInt(time.Now().Unix(), 10), Offset: 0, Count: limit,
	}).Result()
	if err != nil {
		slog.Error("scan expired chunk upload sessions failed", "error", err)
		return
	}
	for _, sessionKey := range sessions {
		uploadID, _ := s.redis.HGet(ctx, sessionKey, "id").Result()
		objectKey, _ := s.redis.HGet(ctx, sessionKey, "key").Result()
		if uploadID != "" && objectKey != "" {
			if err := s.storage.AbortMultipartUpload(ctx, objectKey, uploadID); err != nil {
				slog.Error("abort expired chunk upload failed", "sessionKey", sessionKey, "objectKey", objectKey, "error", err)
			}
		}
		s.deleteChunkUploadSession(ctx, sessionKey)
	}
}

func (s *uploadApplication) persistUploadedFile(ctx context.Context, file *models.File) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(file).Error; err != nil {
			return fmt.Errorf("保存文件记录失败: %w", err)
		}
		return s.storageQuotaRepo.UpdateUsedSpace(tx, file.UserId, file.Size)
	})
}

func cloneFileRecord(existing *models.File, userID int, fileName, fileHash, parentID string) *models.File {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fileName), "."))
	return &models.File{
		Id:            utils.NewUUID(),
		UserId:        userID,
		Name:          fileName,
		Size:          existing.Size,
		SizeStr:       existing.SizeStr,
		IsDir:         false,
		FileExtension: ext,
		OssObjectKey:  existing.OssObjectKey,
		FileHash:      fileHash,
		ParentId:      nullableParentID(parentID),
		IsDeleted:     false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		FileURL:       existing.FileURL,
		ThumbnailURL:  existing.ThumbnailURL,
	}
}

func (s *uploadApplication) publishFileUploaded(ctx context.Context, file *models.File) {
	if s.events == nil || file == nil {
		return
	}
	event := ports.Event{
		ID:          utils.NewUUID(),
		Type:        "file.uploaded.v1",
		AggregateID: file.Id,
		UserID:      file.UserId,
		OccurredAt:  time.Now(),
		Data: FileUploadedEvent{
			FileID: file.Id, FileName: file.Name, ObjectKey: file.OssObjectKey, Size: file.Size,
		},
	}
	if err := s.events.Publish(ctx, event); err != nil {
		// 单体阶段事件失败不回滚已经完成的上传；跨服务后由 Outbox 保证可靠投递。
		slog.Warn("publish file.uploaded.v1 failed", "fileId", file.Id, "error", err)
	}
}

func (s *uploadApplication) generateThumbnailAsync(fileID, objectKey string) {
	if fileID == "" || objectKey == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		thumbURL, err := s.storage.GenerateThumbnailForObject(ctx, objectKey)
		if err != nil {
			slog.Warn("generate thumbnail failed", "fileId", fileID, "objectKey", objectKey, "error", err)
			return
		}
		if thumbURL == "" {
			return
		}
		result := s.db.WithContext(ctx).Model(&models.File{}).
			Where("id = ? AND is_deleted = ?", fileID, false).
			Update("thumbnail_url", thumbURL)
		if result.Error != nil || result.RowsAffected == 0 {
			if result.Error != nil {
				slog.Error("update async thumbnail failed", "fileId", fileID, "error", result.Error)
			}
			_ = s.storage.DeleteThumbnailForObject(context.Background(), objectKey)
		}
	}()
}

func (s *uploadApplication) cleanupObject(ctx context.Context, objectKey string) {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	if err := s.storage.DeleteObjectWithThumbnail(cleanupCtx, objectKey); err != nil {
		slog.Error("cleanup uploaded object after persistence failure failed", "objectKey", objectKey, "error", err)
	}
}

func cleanupContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return context.WithoutCancel(ctx)
}

var _ UploadApplication = (*uploadApplication)(nil)
