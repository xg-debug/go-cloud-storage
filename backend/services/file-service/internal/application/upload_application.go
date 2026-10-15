package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
	"time"

	ports "go-cloud-storage/backend/pkg/ports"
	"go-cloud-storage/backend/pkg/utils"
	domain "go-cloud-storage/backend/services/file-service/internal/domain"
	fileports "go-cloud-storage/backend/services/file-service/internal/ports"
)

const (
	applicationDefaultUploadChunkSize int64 = 10 * 1024 * 1024
	applicationCleanupBatchSize             = 100
	applicationCleanupInterval              = 5 * time.Minute
	applicationMergeLockTTL                 = 10 * time.Minute
)

// UploadApplication is the transport-neutral application boundary for uploads.
// HTTP/Gin, Redis, GORM and object-storage SDK types are intentionally excluded.
type UploadApplication interface {
	UploadFile(ctx context.Context, r io.Reader, userID int, fileName string, fileSize int64, fileHash, parentID string) (*domain.File, error)
	InitChunkUpload(ctx context.Context, input InitChunkUploadInput) (*InitChunkUploadResult, error)
	UploadChunk(ctx context.Context, userID int, fileHash string, chunkIndex int, r io.Reader, chunkSize int64, expectedChunkHash string) error
	MergeChunks(ctx context.Context, userID int, fileHash, fileName, parentID string, fileSize, chunkSize int64, totalChunks int) (*domain.File, error)
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
	files    fileports.UploadFileRepository
	quotas   fileports.UploadQuotaRepository
	tx       ports.TransactionManager
	sessions fileports.UploadSessionStore
	storage  ports.Storage
	events   ports.EventBus
}

func NewUploadApplication(
	files fileports.UploadFileRepository,
	quotas fileports.UploadQuotaRepository,
	tx ports.TransactionManager,
	sessions fileports.UploadSessionStore,
	storage ports.Storage,
	events ports.EventBus,
) UploadApplication {
	if files == nil {
		panic("NewUploadApplication requires an UploadFileRepository")
	}
	if quotas == nil {
		panic("NewUploadApplication requires an UploadQuotaRepository")
	}
	if tx == nil {
		panic("NewUploadApplication requires a TransactionManager")
	}
	if sessions == nil {
		panic("NewUploadApplication requires an UploadSessionStore")
	}
	if storage == nil {
		panic("NewUploadApplication requires a Storage implementation")
	}
	return &uploadApplication{
		files:    files,
		quotas:   quotas,
		tx:       tx,
		sessions: sessions,
		storage:  storage,
		events:   events,
	}
}

func (s *uploadApplication) UploadFile(ctx context.Context, r io.Reader, userID int, fileName string, fileSize int64, fileHash, parentID string) (*domain.File, error) {
	// Normal uploads are capped at 10 MiB. Verify bytes before trusting the hash
	// for either metadata or deduplication, including requests that hit an existing object.
	data, err := io.ReadAll(io.LimitReader(r, (10<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > 10<<20 || int64(len(data)) != fileSize {
		return nil, errors.New("普通上传大小校验失败")
	}
	actual := fmt.Sprintf("%x", sha256.Sum256(data))
	if !strings.EqualFold(actual, fileHash) {
		return nil, errors.New("文件内容与SHA256不一致")
	}
	fileHash = actual
	r = bytes.NewReader(data)
	fileName = strings.TrimSpace(fileName)
	if err := validateFileName(fileName); err != nil {
		return nil, err
	}
	if err := s.ensureTargetFolder(ctx, userID, parentID); err != nil {
		return nil, err
	}
	if exists, err := s.files.CheckDuplicateName(ctx, userID, parentID, fileName); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.New("该目录下已有同名的文件")
	}

	// 秒传只复用当前用户已有对象，避免通过已知 hash 探测其他用户文件。
	existing, err := s.files.FindByHash(ctx, userID, fileHash)
	if err != nil {
		return nil, err
	}
	if existing != nil && !existing.IsDeleted {
		file := cloneFileRecord(existing, userID, fileName, fileHash, parentID)
		if err := s.persistUploadedFile(ctx, file, existing.Id); err != nil {
			return nil, err
		}
		s.publishFileUploaded(ctx, file)
		return file, nil
	}

	available, err := s.quotas.GetAvailableSpace(ctx, userID)
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
	file := &domain.File{
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
	if !s.sessions.Available() {
		return nil, fileports.ErrUploadSessionUnavailable
	}
	fileName := strings.TrimSpace(input.FileName)
	if err := validateFileName(fileName); err != nil {
		return nil, err
	}
	if err := s.ensureTargetFolder(ctx, input.UserID, input.ParentID); err != nil {
		return nil, err
	}
	if exists, err := s.files.CheckDuplicateName(ctx, input.UserID, input.ParentID, fileName); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.New("该目录下已有同名的文件")
	}

	chunkSize := input.ChunkSize
	if chunkSize <= 0 {
		chunkSize = applicationDefaultUploadChunkSize
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

	existing, err := s.files.FindByHash(ctx, input.UserID, input.FileHash)
	if err != nil {
		return nil, err
	}
	if existing != nil && !existing.IsDeleted {
		file := cloneFileRecord(existing, input.UserID, fileName, input.FileHash, input.ParentID)
		if err := s.persistUploadedFile(ctx, file, existing.Id); err != nil {
			return nil, err
		}
		s.publishFileUploaded(ctx, file)
		return &InitChunkUploadResult{Finished: true, File: file, URL: file.FileURL}, nil
	}

	remaining, err := s.quotas.GetAvailableSpace(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("获取可用空间失败: %w", err)
	}
	if input.FileSize > remaining {
		return nil, errors.New("存储空间不足，请升级存储配额")
	}

	ctx, release, err := s.lockSession(ctx, input.UserID, input.FileHash)
	if err != nil {
		return nil, err
	}
	defer release()
	createSession := func() (*fileports.UploadSession, error) {
		objectKey := s.storage.GenerateObjectKey(input.UserID, input.ParentID, fileName)
		uploadID, initErr := s.storage.InitiateMultipartUpload(ctx, objectKey)
		if initErr != nil {
			return nil, fmt.Errorf("初始化对象存储分片上传失败: %w", initErr)
		}
		session := &fileports.UploadSession{
			UserID:      input.UserID,
			FileHash:    input.FileHash,
			UploadID:    uploadID,
			ObjectKey:   objectKey,
			FileName:    fileName,
			ParentID:    input.ParentID,
			FileSize:    input.FileSize,
			ChunkSize:   chunkSize,
			TotalChunks: totalChunks,
			Parts:       make(map[int]fileports.UploadPartState),
		}
		if saveErr := s.sessions.Save(ctx, session); saveErr != nil {
			_ = s.storage.AbortMultipartUpload(cleanupContext(ctx), objectKey, uploadID)
			return nil, saveErr
		}
		return session, nil
	}

	session, err := s.sessions.Get(ctx, input.UserID, input.FileHash)
	if errors.Is(err, fileports.ErrUploadSessionNotFound) {
		session, err = createSession()
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		if session.UploadID == "" || session.ObjectKey == "" {
			return nil, errors.New("上传任务状态异常，请取消后重新上传")
		}
		if session.Expired(time.Now()) {
			_ = s.storage.AbortMultipartUpload(cleanupContext(ctx), session.ObjectKey, session.UploadID)
			_ = s.sessions.Delete(cleanupContext(ctx), input.UserID, input.FileHash, session.UploadID)
			session, err = createSession()
			if err != nil {
				return nil, err
			}
		} else {
			if session.FileName != "" && (session.FileName != fileName || session.ParentID != input.ParentID) {
				return nil, errors.New("相同内容的文件正在上传到其他位置，请稍后重试或先取消该上传")
			}
			if len(session.Parts) > 0 && (session.FileSize != input.FileSize || session.ChunkSize != chunkSize || session.TotalChunks != totalChunks) {
				return nil, errors.New("上传参数与已有断点不一致，请先取消该上传任务")
			}
			session.FileName = fileName
			session.ParentID = input.ParentID
			session.FileSize = input.FileSize
			session.ChunkSize = chunkSize
			session.TotalChunks = totalChunks
			if err := s.sessions.Save(ctx, session); err != nil {
				return nil, err
			}
		}
	}

	uploaded := uploadedChunkIndexes(session.Parts)
	return &InitChunkUploadResult{
		Finished:       false,
		FileHash:       input.FileHash,
		UploadID:       session.UploadID,
		UploadedChunks: uploaded,
		ChunkSize:      chunkSize,
		TotalChunks:    totalChunks,
	}, nil
}

func (s *uploadApplication) UploadChunk(ctx context.Context, userID int, fileHash string, chunkIndex int, r io.Reader, chunkSize int64, expectedChunkHash string) error {
	if !s.sessions.Available() {
		return fileports.ErrUploadSessionUnavailable
	}
	session, err := s.sessions.Get(ctx, userID, fileHash)
	if errors.Is(err, fileports.ErrUploadSessionNotFound) {
		return errors.New("上传任务不存在或已过期，请重新初始化")
	}
	if err != nil {
		return err
	}
	if session.UploadID == "" || session.ObjectKey == "" {
		return errors.New("上传任务状态异常")
	}
	if session.Expired(time.Now()) {
		_ = s.storage.AbortMultipartUpload(cleanupContext(ctx), session.ObjectKey, session.UploadID)
		_ = s.sessions.Delete(cleanupContext(ctx), userID, fileHash, session.UploadID)
		return errors.New("上传任务已过期，请重新初始化")
	}
	if chunkIndex < 0 || chunkIndex >= session.TotalChunks {
		return fmt.Errorf("分片索引越界: %d", chunkIndex)
	}

	expectedSize := session.ChunkSize
	if chunkIndex == session.TotalChunks-1 {
		expectedSize = session.FileSize - int64(session.TotalChunks-1)*session.ChunkSize
	}
	if expectedSize <= 0 || chunkSize != expectedSize {
		return fmt.Errorf("分片大小校验失败: index=%d got=%d expected=%d", chunkIndex, chunkSize, expectedSize)
	}

	part, computedHash, err := s.storage.UploadPart(ctx, session.ObjectKey, session.UploadID, chunkIndex+1, r, chunkSize, expectedChunkHash)
	if err != nil {
		return fmt.Errorf("对象存储分片上传失败: %w", err)
	}
	return s.sessions.SavePart(ctx, userID, fileHash, fileports.UploadPartState{
		UploadID: session.UploadID,
		Index:    chunkIndex,
		ETag:     part.ETag,
		Hash:     computedHash,
		Size:     chunkSize,
	})
}

func (s *uploadApplication) MergeChunks(ctx context.Context, userID int, fileHash, fileName, parentID string, fileSize, chunkSize int64, totalChunks int) (*domain.File, error) {
	if !s.sessions.Available() {
		return nil, fileports.ErrUploadSessionUnavailable
	}
	fileName = strings.TrimSpace(fileName)
	if err := validateFileName(fileName); err != nil {
		return nil, err
	}
	if err := s.ensureTargetFolder(ctx, userID, parentID); err != nil {
		return nil, err
	}

	ctx, release, err := s.lockSession(ctx, userID, fileHash)
	if err != nil {
		return nil, err
	}
	defer release()

	session, err := s.sessions.Get(ctx, userID, fileHash)
	if errors.Is(err, fileports.ErrUploadSessionNotFound) {
		return nil, errors.New("上传任务失败")
	}
	if err != nil {
		return nil, err
	}
	if session.UploadID == "" || session.ObjectKey == "" {
		return nil, errors.New("上传任务状态异常")
	}
	if session.Expired(time.Now()) {
		_ = s.storage.AbortMultipartUpload(cleanupContext(ctx), session.ObjectKey, session.UploadID)
		_ = s.sessions.Delete(cleanupContext(ctx), userID, fileHash, session.UploadID)
		return nil, errors.New("上传任务已过期，请重新初始化")
	}
	if session.FileName != "" && session.FileName != fileName {
		return nil, errors.New("文件名与上传会话不一致")
	}
	if session.ParentID != parentID {
		return nil, errors.New("父目录与上传会话不一致")
	}

	if fileSize <= 0 {
		fileSize = session.FileSize
	} else if session.FileSize > 0 && fileSize != session.FileSize {
		return nil, fmt.Errorf("文件大小与上传会话不一致: got=%d expected=%d", fileSize, session.FileSize)
	}
	if chunkSize <= 0 {
		chunkSize = session.ChunkSize
	} else if session.ChunkSize > 0 && chunkSize != session.ChunkSize {
		return nil, fmt.Errorf("分片大小与上传会话不一致: got=%d expected=%d", chunkSize, session.ChunkSize)
	}
	if totalChunks <= 0 {
		totalChunks = session.TotalChunks
	} else if session.TotalChunks > 0 && totalChunks != session.TotalChunks {
		return nil, fmt.Errorf("分片数量与上传会话不一致: got=%d expected=%d", totalChunks, session.TotalChunks)
	}
	if totalChunks <= 0 || fileSize <= 0 || chunkSize <= 0 {
		return nil, errors.New("上传会话元数据无效")
	}
	if exists, err := s.files.CheckDuplicateName(ctx, userID, parentID, fileName); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.New("该目录下已有同名的文件")
	}

	parts := make([]ports.CompletedPart, 0, totalChunks)
	var uploadedSize int64
	for idx := 0; idx < totalChunks; idx++ {
		part, ok := session.Parts[idx]
		if !ok || part.ETag == "" {
			return nil, fmt.Errorf("缺少分片: %d", idx)
		}
		uploadedSize += part.Size
		parts = append(parts, ports.CompletedPart{PartNumber: idx + 1, ETag: part.ETag})
	}
	if len(parts) != totalChunks {
		return nil, fmt.Errorf("分片不完整: 已上传 %d/%d", len(parts), totalChunks)
	}
	if uploadedSize > 0 && uploadedSize != fileSize {
		return nil, fmt.Errorf("分片大小校验失败: got=%d expected=%d", uploadedSize, fileSize)
	}

	object, err := s.storage.CompleteMultipartUpload(ctx, session.ObjectKey, session.UploadID, parts)
	if err != nil {
		return nil, fmt.Errorf("对象存储合并失败: %w", err)
	}
	if object.Size != fileSize {
		_ = s.storage.DeleteFile(cleanupContext(ctx), session.ObjectKey)
		return nil, fmt.Errorf("合并对象大小校验失败: got=%d expected=%d", object.Size, fileSize)
	}
	if len(fileHash) == 64 {
		computedHash, hashErr := s.storage.ComputeObjectSHA256(ctx, session.ObjectKey)
		if hashErr != nil {
			_ = s.storage.DeleteFile(cleanupContext(ctx), session.ObjectKey)
			return nil, fmt.Errorf("计算合并对象hash失败: %w", hashErr)
		}
		if !strings.EqualFold(computedHash, fileHash) {
			_ = s.storage.DeleteFile(cleanupContext(ctx), session.ObjectKey)
			return nil, errors.New("合并对象hash校验失败")
		}
	}

	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fileName), "."))
	file := &domain.File{
		Id:            utils.NewUUID(),
		UserId:        userID,
		Name:          fileName,
		ParentId:      nullableParentID(parentID),
		OssObjectKey:  session.ObjectKey,
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
		s.cleanupObject(ctx, session.ObjectKey)
		_ = s.sessions.Delete(cleanupContext(ctx), userID, fileHash, session.UploadID)
		return nil, err
	}

	s.generateThumbnailAsync(file.Id, file.OssObjectKey)
	_ = s.sessions.Delete(ctx, userID, fileHash, sessionID(session))
	s.publishFileUploaded(ctx, file)
	return file, nil
}

func (s *uploadApplication) CancelChunkUpload(ctx context.Context, userID int, fileHash string) error {
	if !s.sessions.Available() {
		return fileports.ErrUploadSessionUnavailable
	}
	ctx, release, err := s.lockSession(ctx, userID, fileHash)
	if err != nil {
		return err
	}
	defer release()
	session, err := s.sessions.Get(ctx, userID, fileHash)
	if err != nil && !errors.Is(err, fileports.ErrUploadSessionNotFound) {
		return err
	}
	if session != nil && session.UploadID != "" && session.ObjectKey != "" {
		_ = s.storage.AbortMultipartUpload(cleanupContext(ctx), session.ObjectKey, session.UploadID)
	}
	if err := s.sessions.Delete(ctx, userID, fileHash, sessionID(session)); err != nil && !errors.Is(err, fileports.ErrUploadSessionNotFound) {
		return err
	}
	return nil
}

func (s *uploadApplication) GetChunkUploadProgress(ctx context.Context, userID int, fileHash string) (*ChunkUploadProgress, error) {
	if !s.sessions.Available() {
		return nil, fileports.ErrUploadSessionUnavailable
	}
	session, err := s.sessions.Get(ctx, userID, fileHash)
	if errors.Is(err, fileports.ErrUploadSessionNotFound) {
		return &ChunkUploadProgress{Status: "not_found", UploadedChunks: []int{}}, nil
	}
	if err != nil {
		return nil, err
	}
	uploaded := uploadedChunkIndexes(session.Parts)
	return &ChunkUploadProgress{
		Status:         "in_progress",
		UploadID:       session.UploadID,
		UploadedChunks: uploaded,
		UploadedCount:  len(uploaded),
	}, nil
}

func (s *uploadApplication) StartChunkUploadCleanup(ctx context.Context) {
	if s.sessions == nil || !s.sessions.Available() {
		return
	}
	go func() {
		ticker := time.NewTicker(applicationCleanupInterval)
		defer ticker.Stop()
		s.cleanupExpiredChunkUploads(ctx, applicationCleanupBatchSize)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.cleanupExpiredChunkUploads(ctx, applicationCleanupBatchSize)
			}
		}
	}()
}

func (s *uploadApplication) cleanupExpiredChunkUploads(ctx context.Context, limit int64) {
	refs, err := s.sessions.ListExpired(ctx, time.Now(), limit)
	if err != nil {
		slog.Error("scan expired chunk upload sessions failed", "error", err)
		return
	}
	for _, ref := range refs {
		s.cleanupExpiredSession(ctx, ref)
	}
}

func (s *uploadApplication) persistUploadedFile(ctx context.Context, file *domain.File, sourceIDs ...string) error {
	return s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.quotas.LockUser(txCtx, file.UserId); err != nil {
			return err
		}
		if err := s.ensureTargetFolder(txCtx, file.UserId, nullToString(file.ParentId)); err != nil {
			return err
		}
		if len(sourceIDs) > 0 {
			// Lock and recheck the dedup source in this transaction: a stale
			// lookup must not resurrect a reference after permanent deletion.
			source, err := s.files.GetUserFile(txCtx, file.UserId, sourceIDs[0])
			if err != nil {
				return err
			}
			if source == nil || source.IsDeleted || source.OssObjectKey != file.OssObjectKey {
				return errors.New("秒传源文件已变化，请重试上传")
			}
		}
		if err := s.files.Create(txCtx, file); err != nil {
			return fmt.Errorf("保存文件记录失败: %w", err)
		}
		if err := s.quotas.AddUsedSpace(txCtx, file.UserId, file.Size); err != nil {
			return err
		}
		if bus, ok := s.events.(interface{ Transactional() bool }); ok && bus.Transactional() {
			return s.emitFileUploaded(txCtx, file)
		}
		return nil
	})
}

func (s *uploadApplication) ensureTargetFolder(ctx context.Context, userID int, parentID string) error {
	if parentID == "" {
		return nil
	}
	folder, err := s.files.GetUserFile(ctx, userID, parentID)
	if err != nil {
		return fmt.Errorf("查询目标文件夹失败: %w", err)
	}
	if folder == nil || !folder.IsDir {
		return errors.New("目标文件夹不存在或无权限")
	}
	return nil
}

func cloneFileRecord(existing *domain.File, userID int, fileName, fileHash, parentID string) *domain.File {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fileName), "."))
	return &domain.File{
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

func uploadedChunkIndexes(parts map[int]fileports.UploadPartState) []int {
	uploaded := make([]int, 0, len(parts))
	for idx := range parts {
		uploaded = append(uploaded, idx)
	}
	sort.Ints(uploaded)
	return uploaded
}

func (s *uploadApplication) publishFileUploaded(ctx context.Context, file *domain.File) {
	if s.events == nil || file == nil {
		return
	}
	if bus, ok := s.events.(interface{ Transactional() bool }); ok && bus.Transactional() {
		return
	}
	if err := s.emitFileUploaded(ctx, file); err != nil {
		slog.Warn("publish file.uploaded.v1 failed", "fileId", file.Id, "error", err)
	}
}

func (s *uploadApplication) emitFileUploaded(ctx context.Context, file *domain.File) error {
	event := ports.Event{
		ID:          utils.NewUUID(),
		Type:        "file.uploaded.v1",
		AggregateID: file.Id,
		UserID:      file.UserId,
		OccurredAt:  time.Now(),
		Data: FileUploadedEvent{
			FileID:    file.Id,
			FileName:  file.Name,
			ObjectKey: file.OssObjectKey,
			Size:      file.Size,
		},
	}
	return s.events.Publish(ctx, event)
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
		updated, updateErr := s.files.UpdateThumbnail(ctx, fileID, thumbURL)
		if updateErr != nil || !updated {
			if updateErr != nil {
				slog.Error("update async thumbnail failed", "fileId", fileID, "error", updateErr)
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

func sessionID(s *fileports.UploadSession) string {
	if s == nil {
		return ""
	}
	return s.UploadID
}

// Operations are bounded to less than the lease lifetime. Owner tokens prevent
// an expired request from releasing a subsequent request's lock.
func (s *uploadApplication) lockSession(ctx context.Context, userID int, hash string) (context.Context, func(), error) {
	owner := utils.NewUUID()
	locked, err := s.sessions.AcquireMergeLock(ctx, userID, hash, owner, applicationMergeLockTTL)
	if err != nil {
		return ctx, nil, err
	}
	if !locked {
		return ctx, nil, errors.New("上传任务正在处理中，请稍后重试")
	}
	bounded, cancel := context.WithTimeout(ctx, applicationMergeLockTTL/2)
	return bounded, func() { cancel(); _ = s.sessions.ReleaseMergeLock(cleanupContext(ctx), userID, hash, owner) }, nil
}
func (s *uploadApplication) cleanupExpiredSession(ctx context.Context, ref fileports.UploadSessionRef) {
	ctx, release, err := s.lockSession(ctx, ref.UserID, ref.FileHash)
	if err != nil {
		return
	}
	defer release()
	session, err := s.sessions.Get(ctx, ref.UserID, ref.FileHash)
	if errors.Is(err, fileports.ErrUploadSessionNotFound) {
		_ = s.sessions.Delete(ctx, ref.UserID, ref.FileHash, "")
		return
	}
	if err != nil || !session.Expired(time.Now()) {
		return
	}
	if session.UploadID != "" && session.ObjectKey != "" {
		if err = s.storage.AbortMultipartUpload(ctx, session.ObjectKey, session.UploadID); err != nil {
			return
		}
	}
	_ = s.sessions.Delete(ctx, ref.UserID, ref.FileHash, session.UploadID)
}
