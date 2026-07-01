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
			if k == "id" || k == "key" || strings.HasSuffix(k, "_hash") {
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
		midChunkSize       int64 = 5 * 1024 * 1024   // 中等文件 5MB/块
		largeChunkSize     int64 = 10 * 1024 * 1024  // 大文件 10MB/块
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

	// Get all files belonging to user
	files, err := s.fileRepo.GetFileByIds(fileIds)
	if err != nil {
		return nil, "", fmt.Errorf("获取文件信息失败: %w", err)
	}

	// Collect non-directory files owned by user
	var toDownload []*models.File
	var names = make(map[string]int)
	for i := range files {
		f := &files[i]
		if f.IsDir {
			continue
		}
		if f.UserId != userId {
			continue
		}
		// Deduplicate names
		base := f.Name
		names[base]++
		toDownload = append(toDownload, f)
	}

	if len(toDownload) == 0 {
		return nil, "", errors.New("没有可下载的文件")
	}

	reader, writer := io.Pipe()

	go func() {
		defer writer.Close()

		zw := zip.NewWriter(writer)
		defer zw.Close()

		// Reset name counter for actual naming
		used := make(map[string]int)

		for _, f := range toDownload {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Generate unique name in ZIP
			zipName := f.Name
			if cnt := used[f.Name]; cnt > 0 {
				ext := filepath.Ext(f.Name)
				base := strings.TrimSuffix(f.Name, ext)
				zipName = fmt.Sprintf("%s_%d%s", base, cnt, ext)
			}
			used[f.Name]++

			header := &zip.FileHeader{
				Name:     zipName,
				Method:   zip.Deflate,
				Modified: f.UpdatedAt,
			}
			header.SetMode(0644)

			w, err := zw.CreateHeader(header)
			if err != nil {
				slog.Error("create zip entry failed", "file", f.Name, "error", err)
				continue
			}

			obj, err := s.minio.DownloadFile(ctx, f.OssObjectKey)
			if err != nil {
				slog.Error("download from minio failed", "key", f.OssObjectKey, "error", err)
				continue
			}

			_, copyErr := io.Copy(w, obj)
			obj.Close()
			if copyErr != nil {
				slog.Error("zip copy failed", "file", f.Name, "error", copyErr)
			}
		}
	}()

	zipFileName := "files.zip"
	if len(toDownload) == 1 {
		zipFileName = strings.TrimSuffix(toDownload[0].Name, filepath.Ext(toDownload[0].Name)) + ".zip"
	}

	return reader, zipFileName, nil
}

