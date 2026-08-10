package services

import (
	"archive/zip"
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
)

const (
	batchZipPresignTTL = 30 * time.Minute
	batchZipObjectTTL  = 2 * time.Hour
	batchZipPrefix     = "tmp/batch-downloads"
)

type BatchDownloadResult struct {
	FileName         string `json:"fileName"`
	DownloadURL      string `json:"downloadUrl"`
	ExpiresInSeconds int64  `json:"expiresInSeconds"`
	Size             int64  `json:"size"`
}

func (s *fileService) Download(ctx context.Context, userId int, fileId string) (io.ReadCloser, *models.File, error) {
	file, err := s.fileRepo.GetUserFileByID(userId, fileId)
	if err != nil {
		return nil, nil, errors.New("要下载的文件不存在")
	}

	reader, err := s.minio.DownloadFile(ctx, file.OssObjectKey)
	if err != nil {
		return nil, nil, err
	}
	return reader, file, nil
}

func (s *fileService) DownloadRange(ctx context.Context, userId int, fileId string, start, end int64) (io.ReadCloser, *models.File, int64, error) {
	file, err := s.fileRepo.GetUserFileByID(userId, fileId)
	if err != nil {
		return nil, nil, 0, errors.New("要下载的文件不存在")
	}

	infoSize, err := s.minio.GetObjectInfo(ctx, file.OssObjectKey)
	if err != nil {
		return nil, nil, 0, err
	}

	reader, err := s.minio.DownloadFileRange(ctx, file.OssObjectKey, start, end)
	if err != nil {
		return nil, nil, 0, err
	}
	return reader, file, infoSize, nil
}

func (s *fileService) GetObjectSize(ctx context.Context, userId int, fileId string) (int64, error) {
	file, err := s.fileRepo.GetUserFileByID(userId, fileId)
	if err != nil {
		return 0, errors.New("要下载的文件不存在")
	}
	return s.minio.GetObjectInfo(ctx, file.OssObjectKey)
}

func (s *fileService) GetPresignedDownloadURL(ctx context.Context, userId int, fileId string) (string, *models.File, error) {
	file, err := s.fileRepo.GetUserFileByID(userId, fileId)
	if err != nil {
		return "", nil, errors.New("要下载的文件不存在")
	}

	u, err := s.minio.PresignedDownloadURL(ctx, file.OssObjectKey, file.Name, 10*time.Minute)
	if err != nil {
		return "", nil, err
	}
	return u, file, nil
}

// GetDownloadInfo 返回下载策略信息，告诉客户端如何最优地分段下载
// GetChunkUploadProgress 查询服务端上传进度
func (s *fileService) GetChunkUploadProgress(ctx context.Context, userId int, fileHash string) (map[string]interface{}, error) {
	sessionKey := fmt.Sprintf("upload:%d:%s", userId, fileHash)

	uploadId, err := s.redis.HGet(ctx, sessionKey, "id").Result()
	if err != nil || uploadId == "" {
		return map[string]interface{}{
			"status":         "not_found",
			"uploadedChunks": []int{},
		}, nil
	}

	allFields, err := s.redis.HGetAll(ctx, sessionKey).Result()
	uploadedChunks := make([]int, 0)
	if err == nil {
		for k := range allFields {
			if k == "id" || k == "key" || k == "fileSize" || k == "chunkSize" || k == "totalChunks" || strings.HasSuffix(k, "_hash") || strings.HasSuffix(k, "_size") {
				continue
			}
			idx, convErr := strconv.Atoi(k)
			if convErr == nil {
				uploadedChunks = append(uploadedChunks, idx)
			}
		}
		sort.Ints(uploadedChunks)
	}

	return map[string]interface{}{
		"status":         "in_progress",
		"uploadId":       uploadId,
		"uploadedChunks": uploadedChunks,
		"uploadedCount":  len(uploadedChunks),
	}, nil
}

func (s *fileService) GetDownloadInfo(ctx context.Context, userId int, fileId string) (map[string]interface{}, error) {
	file, err := s.fileRepo.GetUserFileByID(userId, fileId)
	if err != nil {
		return nil, errors.New("文件不存在")
	}

	const (
		midChunkSize   int64 = 5 * 1024 * 1024  // 中等文件 5MB/块
		largeChunkSize int64 = 10 * 1024 * 1024 // 大文件 10MB/块
	)

	var chunkSize int64
	switch {
	case file.Size <= 10*1024*1024:
		chunkSize = 0 // 不需要分块
	case file.Size <= 100*1024*1024:
		chunkSize = midChunkSize
	default:
		chunkSize = largeChunkSize
	}

	chunks := int64(0)
	if chunkSize > 0 {
		chunks = (file.Size + chunkSize - 1) / chunkSize
	}

	directURL, _ := s.minio.PresignedDownloadURL(ctx, file.OssObjectKey, file.Name, 10*time.Minute)

	return map[string]interface{}{
		"fileId":            file.Id,
		"fileName":          file.Name,
		"fileSize":          file.Size,
		"chunkSize":         chunkSize,
		"chunks":            chunks,
		"supportsRange":     true,
		"preferredStrategy": "presigned",
		"expiresInSeconds":  600,
		"directDownloadUrl": directURL,
	}, nil
}

// DownloadBatchZip creates a streaming ZIP archive of the given files and returns a reader.
func (s *fileService) DownloadBatchZip(ctx context.Context, userId int, fileIds []string) (io.ReadCloser, string, error) {
	if len(fileIds) == 0 {
		return nil, "", errors.New("未选择文件")
	}

	files, err := s.getBatchDownloadFiles(ctx, userId, fileIds)
	if err != nil {
		return nil, "", fmt.Errorf("获取文件信息失败: %w", err)
	}

	entries := buildZipEntries(files, fileIds)
	if len(entries) == 0 {
		return nil, "", errors.New("没有可下载的文件")
	}

	reader, writer := io.Pipe()

	go func() {
		if err := s.writeBatchZip(ctx, writer, entries); err != nil {
			_ = writer.CloseWithError(err)
			return
		}
		_ = writer.Close()
	}()

	zipFileName := batchZipFileName(files, fileIds)

	return reader, zipFileName, nil
}

func (s *fileService) CreateBatchDownload(ctx context.Context, userId int, fileIds []string) (*BatchDownloadResult, error) {
	if len(fileIds) == 0 {
		return nil, errors.New("未选择文件")
	}

	files, err := s.getBatchDownloadFiles(ctx, userId, fileIds)
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}

	entries := buildZipEntries(files, fileIds)
	if len(entries) == 0 {
		return nil, errors.New("没有可下载的文件")
	}

	zipFileName := batchZipFileName(files, fileIds)
	objectKey := fmt.Sprintf("%s/%d/%s.zip", batchZipPrefix, userId, utils.NewUUID())

	reader, writer := io.Pipe()
	zipErrCh := make(chan error, 1)
	go func() {
		err := s.writeBatchZip(ctx, writer, entries)
		if err != nil {
			_ = writer.CloseWithError(err)
		} else {
			_ = writer.Close()
		}
		zipErrCh <- err
	}()

	size, uploadErr := s.minio.PutObjectStream(ctx, objectKey, reader, -1, "application/zip")
	if uploadErr != nil {
		_ = reader.CloseWithError(uploadErr)
		if zipErr := <-zipErrCh; zipErr != nil {
			slog.Error("batch zip writer failed after upload error", "error", zipErr)
		}
		return nil, fmt.Errorf("生成批量下载文件失败: %w", uploadErr)
	}
	if zipErr := <-zipErrCh; zipErr != nil {
		_ = s.minio.DeleteFile(context.Background(), objectKey)
		return nil, fmt.Errorf("生成批量下载文件失败: %w", zipErr)
	}

	downloadURL, err := s.minio.PresignedDownloadURL(ctx, objectKey, zipFileName, batchZipPresignTTL)
	if err != nil {
		_ = s.minio.DeleteFile(context.Background(), objectKey)
		return nil, err
	}

	time.AfterFunc(batchZipObjectTTL, func() {
		if err := s.minio.DeleteFile(context.Background(), objectKey); err != nil {
			slog.Warn("cleanup temporary batch zip failed", "objectKey", objectKey, "error", err)
		}
	})

	return &BatchDownloadResult{
		FileName:         zipFileName,
		DownloadURL:      downloadURL,
		ExpiresInSeconds: int64(batchZipPresignTTL.Seconds()),
		Size:             size,
	}, nil
}

func (s *fileService) writeBatchZip(ctx context.Context, w io.Writer, entries []zipEntry) error {
	zw := zip.NewWriter(w)
	closed := false
	defer func() {
		if !closed {
			_ = zw.Close()
		}
	}()

	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if entry.file.IsDir {
			if _, err := zw.Create(entry.zipPath + "/"); err != nil {
				return fmt.Errorf("创建ZIP目录失败: %w", err)
			}
			continue
		}

		header := &zip.FileHeader{
			Name:     entry.zipPath,
			Method:   zip.Deflate,
			Modified: entry.file.UpdatedAt,
		}
		header.SetMode(0644)

		entryWriter, err := zw.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("创建ZIP条目失败: %w", err)
		}

		obj, err := s.minio.DownloadFile(ctx, entry.file.OssObjectKey)
		if err != nil {
			return fmt.Errorf("读取文件对象失败: %w", err)
		}

		_, copyErr := io.Copy(entryWriter, obj)
		closeErr := obj.Close()
		if copyErr != nil {
			return fmt.Errorf("写入ZIP条目失败: %w", copyErr)
		}
		if closeErr != nil {
			return fmt.Errorf("关闭文件对象失败: %w", closeErr)
		}
	}

	if err := zw.Close(); err != nil {
		return fmt.Errorf("关闭ZIP失败: %w", err)
	}
	closed = true
	return nil
}

func batchZipFileName(files []models.File, selectedIDs []string) string {
	zipFileName := "files.zip"
	if len(selectedIDs) == 1 {
		for _, file := range files {
			if file.Id == selectedIDs[0] {
				name := strings.TrimSpace(strings.TrimSuffix(file.Name, filepath.Ext(file.Name)))
				if name == "" {
					name = "download"
				}
				return name + ".zip"
			}
		}
	}
	return zipFileName
}

type zipEntry struct {
	file    *models.File
	zipPath string
}

func (s *fileService) getBatchDownloadFiles(ctx context.Context, userId int, fileIds []string) ([]models.File, error) {
	var files []models.File
	err := s.db.WithContext(ctx).Raw(`
		WITH RECURSIVE selected AS (
			SELECT * FROM file
			WHERE user_id = ? AND is_deleted = 0 AND id IN ?
			UNION ALL
			SELECT f.* FROM file f
			INNER JOIN selected s ON f.parent_id = s.id
			WHERE f.user_id = ? AND f.is_deleted = 0
		)
		SELECT DISTINCT * FROM selected
	`, userId, fileIds, userId).Scan(&files).Error
	return files, err
}

func buildZipEntries(files []models.File, selectedIDs []string) []zipEntry {
	if len(files) == 0 {
		return nil
	}

	fileMap := make(map[string]*models.File, len(files))
	for i := range files {
		fileMap[files[i].Id] = &files[i]
	}

	selectedSet := make(map[string]bool, len(selectedIDs))
	for _, id := range selectedIDs {
		selectedSet[id] = true
	}

	usedPaths := make(map[string]int)
	entries := make([]zipEntry, 0, len(files))
	for i := range files {
		file := &files[i]
		path := batchZipPath(file, fileMap, selectedSet)
		if path == "" {
			continue
		}
		path = uniqueZipPath(safeZipPath(path), usedPaths, file.IsDir)
		entries = append(entries, zipEntry{file: file, zipPath: path})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].file.IsDir != entries[j].file.IsDir {
			return entries[i].file.IsDir
		}
		return entries[i].zipPath < entries[j].zipPath
	})
	return entries
}

func batchZipPath(file *models.File, fileMap map[string]*models.File, selectedSet map[string]bool) string {
	names := []string{file.Name}
	current := file
	root := file
	for current.ParentId.Valid {
		parent, ok := fileMap[current.ParentId.String]
		if !ok {
			break
		}
		names = append(names, parent.Name)
		current = parent
		if selectedSet[parent.Id] {
			root = parent
		}
	}

	if selectedSet[file.Id] && root.Id == file.Id {
		return file.Name
	}

	for len(names) > 0 && names[len(names)-1] != root.Name {
		names = names[:len(names)-1]
	}
	for i, j := 0, len(names)-1; i < j; i, j = i+1, j-1 {
		names[i], names[j] = names[j], names[i]
	}
	return strings.Join(names, "/")
}

func safeZipPath(path string) string {
	parts := strings.Split(strings.ReplaceAll(path, "\\", "/"), "/")
	safe := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "." || part == ".." {
			continue
		}
		safe = append(safe, part)
	}
	return strings.Join(safe, "/")
}

func uniqueZipPath(path string, used map[string]int, isDir bool) string {
	if path == "" {
		path = "unnamed"
	}
	key := path
	if isDir {
		key += "/"
	}
	if used[key] == 0 {
		used[key] = 1
		return path
	}

	used[key]++
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	if isDir {
		return fmt.Sprintf("%s_%d", path, used[key]-1)
	}
	return fmt.Sprintf("%s_%d%s", base, used[key]-1, ext)
}
