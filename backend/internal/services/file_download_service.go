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
)

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

	u, err := s.minio.PresignedGetURL(ctx, file.OssObjectKey, 10*time.Minute)
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
		midChunkSize       int64 = 5 * 1024 * 1024  // 中等文件 5MB/块
		largeChunkSize     int64 = 10 * 1024 * 1024 // 大文件 10MB/块
		presignedThreshold int64 = 100 * 1024 * 1024
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

	directURL := ""
	if file.Size > presignedThreshold {
		directURL, _ = s.minio.PresignedGetURL(ctx, file.OssObjectKey, 10*time.Minute)
	}

	return map[string]interface{}{
		"fileId":            file.Id,
		"fileName":          file.Name,
		"fileSize":          file.Size,
		"chunkSize":         chunkSize,
		"chunks":            chunks,
		"supportsRange":     true,
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
		defer writer.Close()

		zw := zip.NewWriter(writer)
		defer zw.Close()

		for _, entry := range entries {
			select {
			case <-ctx.Done():
				return
			default:
			}

			if entry.file.IsDir {
				if _, err := zw.Create(entry.zipPath + "/"); err != nil {
					slog.Error("create zip folder failed", "folder", entry.zipPath, "error", err)
				}
				continue
			}

			header := &zip.FileHeader{
				Name:     entry.zipPath,
				Method:   zip.Deflate,
				Modified: entry.file.UpdatedAt,
			}
			header.SetMode(0644)

			w, err := zw.CreateHeader(header)
			if err != nil {
				slog.Error("create zip entry failed", "file", entry.file.Name, "error", err)
				continue
			}

			obj, err := s.minio.DownloadFile(ctx, entry.file.OssObjectKey)
			if err != nil {
				slog.Error("download from minio failed", "key", entry.file.OssObjectKey, "error", err)
				continue
			}

			_, copyErr := io.Copy(w, obj)
			obj.Close()
			if copyErr != nil {
				slog.Error("zip copy failed", "file", entry.file.Name, "error", copyErr)
			}
		}
	}()

	zipFileName := "files.zip"
	if len(fileIds) == 1 {
		for _, file := range files {
			if file.Id == fileIds[0] {
				zipFileName = strings.TrimSuffix(file.Name, filepath.Ext(file.Name)) + ".zip"
				break
			}
		}
	}

	return reader, zipFileName, nil
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
