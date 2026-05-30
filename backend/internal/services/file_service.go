package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	miniosrv "go-cloud-storage/backend/infrastructure/minio"
	"go-cloud-storage/backend/internal/models"
	"go-cloud-storage/backend/internal/repositories"
	"go-cloud-storage/backend/pkg/utils"
	"io"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"

	"github.com/minio/minio-go/v7"

	"gorm.io/gorm"
)

type FileItem struct {
	Id           string `json:"id"`
	Name         string `json:"name"`
	ParentId     string `json:"parent_id"`
	IsDir        bool   `json:"is_dir"`
	Size         int64  `json:"size"`
	SizeStr      string `json:"size_str"`
	Extension    string `json:"extension"`
	CreatedAt    string `json:"created_at"`
	Modified     string `json:"modified"`
	FileURL      string `json:"file_url"`
	ThumbnailURL string `json:"thumbnail_url"`
}

type RecentFile struct {
	Date  string      `json:"date"`  // 例如 "2025-08-01"
	Range string      `json:"range"` // today / week / month
	Files []FileBrief `json:"files"`
}

type FileBrief struct {
	Name     string `json:"name"`
	IsDir    bool   `json:"is_dir"`
	Modified string `json:"modified"`
	SizeStr  string `json:"size_str"`
}

type FilePreview struct {
	Id               string `json:"id"`
	Name             string `json:"name"`
	Size             int64  `json:"size"`
	SizeStr          string `json:"size_str"`
	Extension        string `json:"extension"`
	FileURL          string `json:"file_url"`
	ThumbnailURL     string `json:"thumbnail_url"`
	CanPreview       bool   `json:"can_preview"`
	PreviewType      string `json:"preview_type"` // image, video, audio, text, pdf, office, other
	OfficePreviewURL string `json:"office_preview_url,omitempty"`
	Modified         string `json:"modified"`
	FilePath         string `json:"file_path"`
}

type FolderNode struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	ParentID string        `json:"parentId"`
	Children []*FolderNode `json:"children"`
}

type FileService interface {
	GetFileById(fileId string) (*models.File, error)
	GetFiles(ctx context.Context, userId int, parentId string, page, pageSize int, sortBy, sortOrder string) ([]FileItem, int64, error)
	CreateFolder(userId int, folderName string, parentId string) (*models.File, error)
	Rename(userId int, fileId, newName string) error
	Delete(fileId string, userId int) error
	CreateFileInfo(file *models.File) error
	GetRecentFiles(userId int, timeRange string) ([]*RecentFile, error)
	GetFilePath(file *models.File) (string, error)
	PreviewFile(userId int, fileId string) (*FilePreview, error)
	PreviewStream(ctx context.Context, userId int, fileId string) (io.ReadCloser, *models.File, error)
	SearchFiles(userId int, keyword, parentId string, page, pageSize int) ([]FileItem, int64, error)

	UploadFile(ctx context.Context, r io.Reader, userId int, fileName string, fileSize int64, fileHash string, parentId string) (*models.File, error)
	InitChunkUpload(ctx context.Context, userId int, filename, fileMd5 string, parentId string, fileSize int64) (gin.H, error)
	UploadChunk(ctx context.Context, userId int, fileHash string, chunkIndex int, r io.Reader, chunkSize int64, expectedChunkHash string) error
	MergeChunks(ctx context.Context, userId int, fileHash, fileName, parentId string, fileSize int64) (*models.File, error)
	CancelChunkUpload(ctx context.Context, userId int, fileHash string) error
	GetChunkUploadProgress(ctx context.Context, userId int, fileHash string) (map[string]interface{}, error)

	GetFolderTree(ctx context.Context, userId int) ([]FolderNode, error)
	MoveFile(ctx context.Context, userId int, fileId, targetFolderId string) error
	CopyFile(ctx context.Context, userId int, fileId, targetFolderId string) error

	Download(ctx context.Context, userId int, fileId string) (io.ReadCloser, *models.File, error)
	DownloadRange(ctx context.Context, userId int, fileId string, start, end int64) (io.ReadCloser, *models.File, int64, error)
	GetObjectSize(ctx context.Context, userId int, fileId string) (int64, error)
	GetPresignedDownloadURL(ctx context.Context, userId int, fileId string) (string, *models.File, error)
	GetDownloadInfo(ctx context.Context, userId int, fileId string) (map[string]interface{}, error)
}

type fileService struct {
	db               *gorm.DB
	redis            *redis.Client
	fileRepo         repositories.FileRepository
	storageQuotaRepo repositories.StorageQuotaRepository
	minio            *miniosrv.MinioService
}

func NewFileService(db *gorm.DB, redis *redis.Client, repo repositories.FileRepository, storageQuotaRepo repositories.StorageQuotaRepository, minio *miniosrv.MinioService) FileService {
	return &fileService{db: db, redis: redis, fileRepo: repo, storageQuotaRepo: storageQuotaRepo, minio: minio}
}

func (s *fileService) GetFileById(fileId string) (*models.File, error) {
	return s.fileRepo.GetFileById(fileId)
}

func (s *fileService) GetFiles(ctx context.Context, userId int, parentId string, page, pageSize int, sortBy, sortOrder string) ([]FileItem, int64, error) {
	files, total, err := s.fileRepo.GetFiles(ctx, userId, parentId, page, pageSize, sortBy, sortOrder)
	if err != nil {
		return nil, 0, err
	}
	var fileList []FileItem
	for _, file := range files {
		parentId := ""
		if file.ParentId.Valid {
			parentId = file.ParentId.String
		}
		fileList = append(fileList, FileItem{
			Id:           file.Id,
			Name:         file.Name,
			ParentId:     parentId,
			IsDir:        file.IsDir,
			Size:         file.Size,
			SizeStr:      file.SizeStr,
			Extension:    file.FileExtension,
			Modified:     file.UpdatedAt.Format("2006-01-02 15:04:05"),
			FileURL:      file.FileURL,
			ThumbnailURL: file.ThumbnailURL,
		})
	}
	return fileList, total, nil
}

func (s *fileService) CreateFolder(userId int, folderName string, parentId string) (*models.File, error) {
	folder, err := s.fileRepo.CreateFolder(userId, folderName, parentId)
	if err != nil {
		return nil, err
	}
	return folder, nil
}

func (s *fileService) Rename(userId int, fileId, newName string) error {
	// 检查是否存在
	file, err := s.fileRepo.GetFileById(fileId)
	if err != nil {
		return err
	}
	// 重名检查
	exists, err := s.fileRepo.CheckDuplicateName(userId, file.ParentId.String, newName)
	if err != nil {
		return err
	}
	typeStr := "文件夹"
	if file.IsDir == false {
		typeStr = "文件"
	}
	if exists {
		return errors.New("该目录下已有同名的" + typeStr)
	}
	return s.fileRepo.UpdateFileNameById(fileId, newName)
}

func (s *fileService) Delete(fileId string, userId int) error {
	file, err := s.fileRepo.GetFileById(fileId)
	if err != nil {
		return err
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if file.IsDir {
			deletedIds, err := s.fileRepo.SoftDeleteFolder(tx, userId, fileId)
			if err != nil {
				return err
			}

			// 累加所有被删除文件的大小（不含文件夹本身，文件夹 size=0）
			var totalSize int64
			if len(deletedIds) > 0 {
				deletedFiles, err := s.fileRepo.GetFileByIds(deletedIds)
				if err != nil {
					return err
				}
				for _, f := range deletedFiles {
					if !f.IsDir {
						totalSize += f.Size
					}
				}
			}

			for _, id := range deletedIds {
				if err := s.fileRepo.AddToRecycle(tx, &models.RecycleBin{
					FileId:    id,
					UserId:    userId,
					DeletedAt: time.Now(),
					ExpireAt:  time.Now().Add(7 * 24 * time.Hour),
				}); err != nil {
					return err
				}
			}

			if totalSize > 0 {
				if err := s.storageQuotaRepo.UpdateUsedSpace(tx, userId, -totalSize); err != nil {
					return err
				}
			}
		} else {
			if err := s.fileRepo.SoftDeleteFile(tx, userId, fileId); err != nil {
				return err
			}

			if err := s.fileRepo.AddToRecycle(tx, &models.RecycleBin{
				FileId:    fileId,
				UserId:    userId,
				DeletedAt: time.Now(),
				ExpireAt:  time.Now().Add(7 * 24 * time.Hour),
			}); err != nil {
				return err
			}

			if file.Size > 0 {
				if err := s.storageQuotaRepo.UpdateUsedSpace(tx, userId, -file.Size); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func (s *fileService) CreateFileInfo(file *models.File) error {
	return s.fileRepo.CreateFile(file)
}

func (s *fileService) GetRecentFiles(userId int, timeRange string) ([]*RecentFile, error) {
	var since time.Time
	now := time.Now()
	switch timeRange {
	case "today":
		since = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "week":
		weekday := int(now.Weekday())
		if weekday == 0 { // 周日
			weekday = 7
		}
		// 计算本周一的日期（weekday-1 天前）
		daysToSubtract := weekday - 1
		since = now.AddDate(0, 0, -daysToSubtract)
		since = time.Date(since.Year(), since.Month(), since.Day(), 0, 0, 0, 0, since.Location())
	case "month":
		since = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	default:
		since = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()) // 默认读取今天的
	}
	files, err := s.fileRepo.GetRecentFiles(userId, since)
	if err != nil {
		return nil, err
	}

	resultMap := make(map[string]*RecentFile)
	// result 存储最终按日期分组的结果。
	var result []*RecentFile

	//将查询到的文件按照 日期分组，每一天生成一个 RecentFile 对象，包含该天的文件列表。
	for _, f := range files {
		day := f.UpdatedAt.Format("2006-01-02")
		// 如果这个day还没有在 resultMap 中出现，就新建一个 RecentFile 并加入 result。
		if _, exist := resultMap[day]; !exist {
			resultMap[day] = &RecentFile{
				Date:  day,
				Range: timeRange,
				Files: []FileBrief{},
			}
			result = append(result, resultMap[day])
		}
		// 对已经创建的RecentFile对象修改（返回值是指针类型）: 把文件信息封装成 FileBrief，追加到 Files 列表。
		resultMap[day].Files = append(resultMap[day].Files, FileBrief{
			Name:     f.Name,
			IsDir:    f.IsDir,
			Modified: f.UpdatedAt.Format("15:04"),
			SizeStr:  f.SizeStr,
		})
	}
	return result, nil
}

func (s *fileService) GetFilePath(file *models.File) (string, error) {
	if !file.ParentId.Valid || file.ParentId.String == "" {
		return "/" + file.Name, nil
	}

	// 用递归 CTE 一次查询所有祖先目录名（从根到直接父节点）
	names, err := s.fileRepo.GetAncestorNames(context.Background(), file.Id)
	if err != nil {
		return "", err
	}

	// CTE 返回的是从文件自底向上的祖先链，需要反转以得到从根向下的路径
	for i, j := 0, len(names)-1; i < j; i, j = i+1, j-1 {
		names[i], names[j] = names[j], names[i]
	}

	return "/" + strings.Join(names, "/"), nil
}

// 判断文件类型是否可预览
func getPreviewType(extension string) (bool, string) {
	ext := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(extension)), ".")
	if ext == "" {
		return false, "other"
	}

	// 图片类型
	imageExts := []string{"jpg", "jpeg", "png", "gif", "bmp", "webp", "svg"}
	for _, imgExt := range imageExts {
		if ext == imgExt {
			return true, "image"
		}
	}

	// 视频类型
	videoExts := []string{"mp4", "avi", "mov", "wmv", "flv", "webm", "mkv"}
	for _, vidExt := range videoExts {
		if ext == vidExt {
			return true, "video"
		}
	}

	// 音频类型
	audioExts := []string{"mp3", "wav", "flac", "aac", "ogg", "m4a"}
	for _, audExt := range audioExts {
		if ext == audExt {
			return true, "audio"
		}
	}

	// Markdown
	if ext == "md" {
		return true, "markdown"
	}

	// 文本类型
	textExts := []string{"txt", "json", "xml", "csv", "log", "js", "css", "html", "go", "java", "py", "c", "cpp"}
	for _, txtExt := range textExts {
		if ext == txtExt {
			return true, "text"
		}
	}

	// PDF类型
	if ext == "pdf" {
		return true, "pdf"
	}

	// Office 类型：暂不支持在线预览（需 MinIO 公网可访问，微软 Office Online 才能抓取文件）
	officeExts := []string{"doc", "docx", "xls", "xlsx", "ppt", "pptx"}
	for _, offExt := range officeExts {
		if ext == offExt {
			return false, "office"
		}
	}

	return false, "other"
}

func (s *fileService) PreviewFile(userId int, fileId string) (*FilePreview, error) {
	// 获取文件信息
	file, err := s.fileRepo.GetFileById(fileId)
	if err != nil {
		return nil, errors.New("文件不存在")
	}

	// 检查文件所有权
	if file.UserId != userId {
		return nil, errors.New("无权限访问该文件")
	}

	// 检查是否为文件夹
	if file.IsDir {
		return nil, errors.New("文件夹无法预览")
	}

	// 检查文件是否已删除
	if file.IsDeleted {
		return nil, errors.New("文件已删除")
	}

	// 获取文件路径
	filePath, err := s.GetFilePath(file)
	if err != nil {
		filePath = "/" + file.Name
	}

	// 判断文件类型和是否可预览
	canPreview, previewType := getPreviewType(file.FileExtension)

	// 为可预览类型生成 inline 预签名 URL（防止浏览器弹出下载）
	previewFileURL := file.FileURL
	if canPreview {
		if u, err := s.minio.PresignedGetPreviewURL(context.Background(), file.OssObjectKey, 30*time.Minute); err == nil {
			previewFileURL = u
		}
	}

	// Office 文档：用预签名 URL 构建 Office Online 查看链接
	// 注意：需要 MinIO 能被公网访问，否则微软服务器无法获取文件
	officePreviewURL := ""
	if previewType == "office" {
		officePreviewURL = buildOfficePreviewURL(previewFileURL)
	}

	return &FilePreview{
		Id:               file.Id,
		Name:             file.Name,
		Size:             file.Size,
		SizeStr:          file.SizeStr,
		Extension:        file.FileExtension,
		FileURL:          previewFileURL,
		ThumbnailURL:     file.ThumbnailURL,
		CanPreview:       canPreview,
		PreviewType:      previewType,
		OfficePreviewURL: officePreviewURL,
		Modified:         file.UpdatedAt.Format("2006-01-02 15:04:05"),
		FilePath:         filePath,
	}, nil
}

func buildOfficePreviewURL(fileURL string) string {
	if strings.TrimSpace(fileURL) == "" {
		return ""
	}
	return "https://view.officeapps.live.com/op/view.aspx?src=" + url.QueryEscape(fileURL) + "&wdAr=1.3333333333333333"
}

// UploadFile 小文件上传
func (s *fileService) UploadFile(ctx context.Context, r io.Reader, userId int, fileName string, fileSize int64, fileHash string, parentId string) (*models.File, error) {
	// 秒传检查：基于文件内容哈希跨用户匹配
	existingFile, err := s.fileRepo.GetFileByMD5(userId, fileHash)
	if err == nil && existingFile != nil && !existingFile.IsDeleted {
		// 秒传成功：为当前用户创建新文件记录，复用已有的 MinIO 对象
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fileName), "."))
		pid := sql.NullString{String: parentId, Valid: parentId != ""}
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
			ParentId:      pid,
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
		return nil, err
	}

	return uploadFile, nil
}

// InitChunkUpload 初始化分片上传
// 逻辑：秒传检查 -> 断点续传检查 -> 新建上传任务
func (s *fileService) InitChunkUpload(ctx context.Context, userId int, fileName, fileHash string, parentId string, fileSize int64) (gin.H, error) {
	// 1.秒传检查：基于文件内容哈希跨用户匹配
	existingFile, err := s.fileRepo.GetFileByMD5(userId, fileHash)
	if err == nil && existingFile != nil && !existingFile.IsDeleted {
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fileName), "."))
		pid := sql.NullString{String: parentId, Valid: parentId != ""}
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
			ParentId:      pid,
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

	// 检查是否有进行中的上传会话
	sessionExists, err := s.redis.Exists(ctx, sessionKey).Result()
	uploadId := ""
	objectKey := ""

	if err != nil || sessionExists == 0 {
		// 3.1 无会话：初始化新的 MinIO 分片上传
		objectKey = s.minio.GenerateObjectKey(userId, parentId, fileName)
		uploadId, err = s.minio.InitiateMultipartUpload(ctx, objectKey)
		if err != nil {
			return nil, fmt.Errorf("初始化 OSS 上传失败: %w", err)
		}

		// 存入 Redis Hash，所有字段共享 24h TTL
		err = s.redis.HSet(ctx, sessionKey,
			"id", uploadId,
			"key", objectKey,
		).Err()
		if err != nil {
			return nil, err
		}
		s.redis.Expire(ctx, sessionKey, 24*time.Hour)
	} else {
		// 3.2 会话存在：从 Hash 中读取 uploadId 和 objectKey
		uploadId, _ = s.redis.HGet(ctx, sessionKey, "id").Result()
		objectKey, _ = s.redis.HGet(ctx, sessionKey, "key").Result()
	}

	// 4.获取已上传的分片列表（从 Hash 中读取 ETag 字段，排除 id/key/锁字段）
	allFields, err := s.redis.HGetAll(ctx, sessionKey).Result()

	uploadedChunks := make([]int, 0)
	if err == nil {
		for k := range allFields {
			// 跳过元数据字段和 hash 字段，只提取纯数字 key（分片索引）
			if k == "id" || k == "key" {
				continue
			}
			if strings.HasSuffix(k, "_hash") {
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

	partNumber := chunkIndex + 1
	partInfo, computedHash, err := s.minio.UploadPart(ctx, objectKey, uploadId, partNumber, r, chunkSize, expectedChunkHash)
	if err != nil {
		return fmt.Errorf("OSS 分片上传失败: %w", err)
	}

	// 幂等存储：ETag + 分片 hash 写入同一个 Hash
	err = s.redis.HSet(ctx, sessionKey,
		strconv.Itoa(chunkIndex), partInfo.ETag,
		strconv.Itoa(chunkIndex)+"_hash", computedHash,
	).Err()
	if err != nil {
		return err
	}

	// 单次 Expire 刷新整个会话的 TTL
	s.redis.Expire(ctx, sessionKey, 24*time.Hour)
	return nil
}

// MergeChunks 合并分片
func (s *fileService) MergeChunks(ctx context.Context, userId int, fileHash, fileName, parentId string, fileSize int64) (*models.File, error) {
	sessionKey := fmt.Sprintf("upload:%d:%s", userId, fileHash)

	// 分布式锁
	lockKey := fmt.Sprintf("upload:%d:%s:lock", userId, fileHash)
	locked, err := s.redis.SetNX(ctx, lockKey, "1", 30*time.Second).Result()
	if err != nil || !locked {
		return nil, errors.New("合并正在进行中，请稍后重试")
	}
	defer s.redis.Del(ctx, lockKey)

	uploadId, err := s.redis.HGet(ctx, sessionKey, "id").Result()
	if err != nil || uploadId == "" {
		return nil, errors.New("上传任务失败")
	}
	objectKey, err := s.redis.HGet(ctx, sessionKey, "key").Result()

	// 1.获取所有分片 ETag（过滤 id/key 和非数字字段）
	allFields, err := s.redis.HGetAll(ctx, sessionKey).Result()
	if err != nil || len(allFields) <= 2 { // 只有 id 和 key，没有分片
		return nil, errors.New("未找到已上传的分片数据")
	}

	var completeParts []minio.CompletePart
	for k, v := range allFields {
		if k == "id" || k == "key" || strings.HasSuffix(k, "_hash") {
			continue
		}
		idx, convErr := strconv.Atoi(k)
		if convErr == nil {
			completeParts = append(completeParts, minio.CompletePart{
				PartNumber: idx + 1,
				ETag:       v,
			})
		}
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

	// 3.写入数据库
	ext := strings.TrimPrefix(filepath.Ext(fileName), ".")
	pid := sql.NullString{String: parentId, Valid: parentId != ""}

	newFile := &models.File{
		Id:            utils.NewUUID(),
		UserId:        userId,
		Name:          fileName,
		ParentId:      pid,
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
		if err := s.fileRepo.CreateFile(newFile); err != nil {
			return err
		}
		return s.storageQuotaRepo.UpdateUsedSpace(tx, userId, fileSize)
	})
	if err != nil {
		return nil, err
	}

	// 4.清理：一次 DEL 清掉整个 Hash
	s.redis.Del(ctx, sessionKey)

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
	s.redis.Del(ctx, sessionKey)
	return nil
}

// SearchFiles 搜索文件和文件夹
func (s *fileService) SearchFiles(userId int, keyword, parentId string, page, pageSize int) ([]FileItem, int64, error) {
	// 计算偏移量
	offset := (page - 1) * pageSize

	// 构建查询条件
	query := s.db.Model(&models.File{}).Where("user_id = ? AND is_deleted = ?", userId, false)

	// 如果指定了父目录，则在该目录下搜索
	if parentId != "" {
		query = query.Where("parent_id = ?", parentId)
	}

	// ngram FULLTEXT 搜索（索引需 WITH PARSER ngram）
	if keyword != "" {
		// 清除 FULLTEXT 布尔操作符，防止用户输入被误解析
		safe := strings.NewReplacer(
			"+", "", "-", "", ">", "", "<", "", "(", "", ")", "",
			"~", "", "*", "", "\"", "", "@", "",
		).Replace(keyword)
		if safe != "" {
			query = query.Where("MATCH(name) AGAINST(? IN BOOLEAN MODE)", safe)
		}
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取文件列表
	var files []models.File
	if err := query.Order("is_dir DESC, name ASC").Offset(offset).Limit(pageSize).Find(&files).Error; err != nil {
		return nil, 0, err
	}

	// 转换为返回格式
	var fileItems []FileItem
	for _, file := range files {
		parentId := ""
		if file.ParentId.Valid {
			parentId = file.ParentId.String
		}
		fileItems = append(fileItems, FileItem{
			Id:           file.Id,
			Name:         file.Name,
			ParentId:     parentId,
			IsDir:        file.IsDir,
			Size:         file.Size,
			SizeStr:      file.SizeStr,
			Extension:    file.FileExtension,
			CreatedAt:    file.CreatedAt.Format("2006-01-02 15:04:05"),
			Modified:     file.UpdatedAt.Format("2006-01-02 15:04:05"),
			FileURL:      file.FileURL,
			ThumbnailURL: file.ThumbnailURL,
		})
	}

	return fileItems, total, nil
}

// GetFolderTree  获取文件夹树结构
func (s *fileService) GetFolderTree(ctx context.Context, userId int) ([]FolderNode, error) {
	folders, err := s.fileRepo.GetAllFolders(ctx, userId)
	if err != nil {
		return nil, err
	}

	// 构建 map: id -> node
	nodeMap := make(map[string]*FolderNode)

	var rootId string

	for _, f := range folders {
		if f.Name == "/" && !f.ParentId.Valid {
			rootId = f.Id
		}
		nodeMap[f.Id] = &FolderNode{
			ID:       f.Id,
			Name:     f.Name,
			ParentID: nullToString(f.ParentId),
			Children: []*FolderNode{},
		}
	}

	root := nodeMap[rootId]

	if root == nil {
		return nil, errors.New("未找到根目录")
	}

	// 组装目录树
	for _, node := range nodeMap {
		if node.ID == root.ID {
			continue
		}

		parentNode, ok := nodeMap[node.ParentID]
		if ok {
			parentNode.Children = append(parentNode.Children, node)
		}
	}
	return []FolderNode{*root}, nil
}

func (s *fileService) MoveFile(ctx context.Context, userId int, fileId, targetFolderId string) error {
	//file, _ := s.fileRepo.GetFileById(fileId)

	if fileId == targetFolderId {
		return errors.New("不能移动到自身")
	}

	// 若是目录，不能移动到子目录
	file, err := s.fileRepo.GetFileById(fileId)
	if err != nil {
		return err
	}
	if file.IsDir {
		isSub, err := s.fileRepo.IsSubFolder(ctx, userId, fileId, targetFolderId)
		if err != nil {
			return err
		}
		if isSub {
			return errors.New("不能移动到子文件夹")
		}
	}
	// 更新 parentId
	return s.fileRepo.UpdateParent(ctx, fileId, targetFolderId)
}

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
	if err := tx.Model(&models.File{}).Create(&models.File{
		Id: uuid.New().String(), UserId: userId, Name: newName, Size: src.Size, SizeStr: src.SizeStr,
		IsDir: false, FileExtension: src.FileExtension, FileHash: src.FileHash, FileURL: src.FileURL,
		ThumbnailURL: src.ThumbnailURL, OssObjectKey: src.OssObjectKey, ParentId: sql.NullString{String: targetParentId, Valid: true},
	}).Error; err != nil {
		return err
	}
	return s.storageQuotaRepo.UpdateUsedSpace(tx, userId, src.Size)
}

func (s *fileService) PreviewStream(ctx context.Context, userId int, fileId string) (io.ReadCloser, *models.File, error) {
	return s.Download(ctx, userId, fileId)
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

func nullToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}
