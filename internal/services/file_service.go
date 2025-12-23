package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"go-cloud-storage/internal/models"
	miniosrv "go-cloud-storage/internal/pkg/minio"
	"go-cloud-storage/internal/pkg/utils"
	"go-cloud-storage/internal/repositories"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"

	"github.com/minio/minio-go/v7"

	"gorm.io/gorm"
)

type FileItem struct {
	Id           string `json:"id"`
	Name         string `json:"name"`
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
	Id           string `json:"id"`
	Name         string `json:"name"`
	Size         int64  `json:"size"`
	SizeStr      string `json:"size_str"`
	Extension    string `json:"extension"`
	FileURL      string `json:"file_url"`
	ThumbnailURL string `json:"thumbnail_url"`
	CanPreview   bool   `json:"can_preview"`
	PreviewType  string `json:"preview_type"` // image, video, audio, text, pdf, office, other
	Modified     string `json:"modified"`
	FilePath     string `json:"file_path"`
}

type FolderNode struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	ParentID string        `json:"parentId"`
	Children []*FolderNode `json:"children"`
}

type FileService interface {
	GetFileById(fileId string) (*models.File, error)
	GetFiles(ctx context.Context, userId int, parentId string) ([]FileItem, int64, error)
	CreateFolder(userId int, folderName string, parentId string) (*models.File, error)
	Rename(userId int, fileId, newName string) error
	Delete(fileId string, userId int) error
	CreateFileInfo(file *models.File) error
	GetRecentFiles(userId int, timeRange string) ([]*RecentFile, error)
	GetFilePath(file *models.File) (string, error)
	PreviewFile(userId int, fileId string) (*FilePreview, error)
	SearchFiles(userId int, keyword, parentId string, page, pageSize int) ([]FileItem, int64, error)

	UploadFile(ctx context.Context, r io.Reader, userId int, fileName string, fileSize int64, fileHash string, parentId string) (*models.File, error)
	InitChunkUpload(ctx context.Context, userId int, filename, fileMd5 string, parentId string, fileSize int64) (gin.H, error)
	UploadChunk(ctx context.Context, userId int, fileHash string, chunkIndex int, data []byte) error
	MergeChunks(ctx context.Context, userId int, fileHash, fileName, parentId string, fileSize int64) (*models.File, error)
	CancelChunkUpload(ctx context.Context, userId int, fileHash string) error

	GetFolderTree(ctx context.Context, userId int) ([]FolderNode, error)
	MoveFile(ctx context.Context, userId int, fileId, targetFolderId string) error

	Download(ctx context.Context, fileId string) (io.ReadCloser, *models.File, error)
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

func (s *fileService) GetFiles(ctx context.Context, userId int, parentId string) ([]FileItem, int64, error) {
	files, total, err := s.fileRepo.GetFiles(ctx, userId, parentId)
	if err != nil {
		return nil, 0, err
	}
	var fileList []FileItem
	for _, file := range files {
		fileList = append(fileList, FileItem{
			Id:           file.Id,
			Name:         file.Name,
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
	// 先获取文件信息，以便后续更新存储配额
	file, err := s.fileRepo.GetFileById(fileId)
	if err != nil {
		return err
	}

	// 如果不是文件夹，需要更新存储配额
	var fileSize int64 = 0
	if !file.IsDir {
		fileSize = file.Size
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1.软删除文件
		if err := s.fileRepo.SoftDeleteFile(tx, userId, fileId); err != nil {
			return err
		}

		// 2.构造回收站记录
		recycleEntry := &models.RecycleBin{
			FileId:    fileId,
			UserId:    userId,
			DeletedAt: time.Now(),
			ExpireAt:  time.Now().Add(10 * 24 * time.Hour),
		}
		if err := s.fileRepo.AddToRecycle(tx, recycleEntry); err != nil {
			return err
		}

		// 3.如果是文件（非文件夹），更新存储配额
		if !file.IsDir && fileSize > 0 {
			// 减少已使用空间（传入负数表示减少）
			if err := s.storageQuotaRepo.UpdateUsedSpace(tx, userId, -fileSize); err != nil {
				return err
			}
		}

		// 如果到这里都没报错，事务会自动提交
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
	if file.ParentId.Valid == false || file.ParentId.String == "" {
		// 已经是根目录
		return "/" + file.Name, nil
	}

	var pathParts []string
	currentParentId := file.ParentId.String
	for currentParentId != "" {
		parent, err := s.fileRepo.GetFileById(currentParentId)
		if err != nil {
			return "", err
		}

		// 跳过根目录(name = "/")
		if parent.Name != "/" {
			pathParts = append(pathParts, parent.Name)
		}

		if parent.ParentId.Valid && parent.ParentId.String != "" { // 父级的 parentId 不为 NULL
			currentParentId = parent.ParentId.String
		} else {
			break
		}
	}
	// 反转 pathParts
	for i, j := 0, len(pathParts)-1; i < j; i, j = i+1, j-1 {
		pathParts[i], pathParts[j] = pathParts[j], pathParts[i]
	}

	return "/" + strings.Join(pathParts, "/"), nil
}

// 判断文件类型是否可预览
func getPreviewType(extension string) (bool, string) {
	ext := strings.ToLower(extension)

	// 图片类型
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg"}
	for _, imgExt := range imageExts {
		if ext == imgExt {
			return true, "image"
		}
	}

	// 视频类型
	videoExts := []string{".mp4", ".avi", ".mov", ".wmv", ".flv", ".webm", ".mkv"}
	for _, vidExt := range videoExts {
		if ext == vidExt {
			return true, "video"
		}
	}

	// 音频类型
	audioExts := []string{".mp3", ".wav", ".flac", ".aac", ".ogg", ".m4a"}
	for _, audExt := range audioExts {
		if ext == audExt {
			return true, "audio"
		}
	}

	// 文本类型
	textExts := []string{".txt", ".md", ".json", ".xml", ".csv", ".log", ".js", ".css", ".html", ".go", ".java", ".py", ".c", ".cpp"}
	for _, txtExt := range textExts {
		if ext == txtExt {
			return true, "text"
		}
	}

	// PDF类型
	if ext == ".pdf" {
		return true, "pdf"
	}

	// Office类型
	officeExts := []string{".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx"}
	for _, offExt := range officeExts {
		if ext == offExt {
			return true, "office"
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

	return &FilePreview{
		Id:           file.Id,
		Name:         file.Name,
		Size:         file.Size,
		SizeStr:      file.SizeStr,
		Extension:    file.FileExtension,
		FileURL:      file.FileURL,
		ThumbnailURL: file.ThumbnailURL,
		CanPreview:   canPreview,
		PreviewType:  previewType,
		Modified:     file.UpdatedAt.Format("2006-01-02 15:04:05"),
		FilePath:     filePath,
	}, nil
}

// UploadFile 小文件上传
func (s *fileService) UploadFile(ctx context.Context, r io.Reader, userId int, fileName string, fileSize int64, fileHash string, parentId string) (*models.File, error) {
	// 秒传检查：检查数据库中是否已经有该 Hash 的文件
	existingFile, err := s.fileRepo.GetFileByMD5(userId, fileHash)
	if err == nil && existingFile != nil && !existingFile.IsDeleted {
		// 秒传成功：直接返回文件信息返回
		return existingFile, nil
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
	uploadFile, err := s.minio.UploadFromStream(ctx, userId, r, fileName, fileSize, parentId)
	if err != nil {
		return nil, fmt.Errorf("MinIO 上传失败: %w", err)
	}

	// 事务处理：入库 + 扣减配额
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 保存文件记录
		if err := tx.Create(uploadFile).Error; err != nil {
			return fmt.Errorf("保存文件记录失败: %w", err)
		}
		// 更新已使用空间
		if err := s.storageQuotaRepo.UpdateUsedSpace(tx, userId, fileSize); err != nil {
			return fmt.Errorf("更新存储空间失败: %w", err)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return uploadFile, nil
}

// InitChunkUpload 初始化分片上传
// 逻辑：秒传检查 -> 断点续传检查 -> 新建上传任务
func (s *fileService) InitChunkUpload(ctx context.Context, userId int, fileName, fileHash string, parentId string, fileSize int64) (gin.H, error) {
	// 1.秒传检查：检查数据库中是否已经有该 Hash 的文件
	existingFile, err := s.fileRepo.GetFileByMD5(userId, fileHash)
	if err == nil && existingFile != nil && !existingFile.IsDeleted {
		// 秒传成功：直接复制旧文件信息返回
		return gin.H{
			"finished": true,
			"file":     existingFile,
			"url":      existingFile.FileURL,
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
	uploadIdKey := fmt.Sprintf("upload:%d:%s:id", userId, fileHash)
	objectKeyKey := fmt.Sprintf("upload:%d:%s:key", userId, fileHash)

	uploadId, err := s.redis.Get(ctx, uploadIdKey).Result()
	objectKey := ""

	if err == redis.Nil || uploadId == "" {
		// 3.1 没有正在进行的上传，初始化一个新的
		objectKey = s.minio.GenerateObjectKey(userId, parentId, fileName)
		uploadId, err := s.minio.InitiateMultipartUpload(ctx, objectKey)
		if err != nil {
			return nil, fmt.Errorf("初始化 OSS 上传失败: %w", err)
		}

		// 存入 Redis，有效期 24 小时
		pipe := s.redis.Pipeline()
		pipe.Set(ctx, uploadIdKey, uploadId, 24*time.Hour)
		pipe.Set(ctx, objectKeyKey, objectKey, 24*time.Hour)
		_, err = pipe.Exec(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		// 3.2 存在上传任务，获取 ObjectKey
		objectKey, _ = s.redis.Get(ctx, objectKeyKey).Result()
	}

	// 4.获取已上传的分片列表
	// Redis Key: upload:parts:{fileHash} -> Hash结构 { "0": "etag1", "1": "etag2" }
	uploadedPartsKey := fmt.Sprintf("upload:%d:%s:parts", userId, fileHash)
	uploadedMap, err := s.redis.HGetAll(ctx, uploadedPartsKey).Result()

	uploadedChunks := make([]int, 0)
	if err != nil {
		for k := range uploadedMap {
			idx, _ := strconv.Atoi(k) // 分片 Id
			uploadedChunks = append(uploadedChunks, idx)
		}
	}

	// 排序，方便前端处理
	sort.Ints(uploadedChunks)

	return gin.H{
		"finished":       false,
		"fileHash":       fileHash,
		"uploadId":       uploadId, // 前端不一定需要 uploadId，后端存着就行，前端只需要 fileHash
		"uploadedChunks": uploadedChunks,
	}, nil
}

// UploadChunk 上传单个分片
func (s *fileService) UploadChunk(ctx context.Context, userId int, fileHash string, chunkIndex int, data []byte) error {
	// 1.获取上下文信息
	uploadIdKey := fmt.Sprintf("upload:%d:%s:id", userId, fileHash)
	objectKeyKey := fmt.Sprintf("upload:%d:%s:key", userId, fileHash)

	uploadId, err := s.redis.Get(ctx, uploadIdKey).Result()
	if err != nil || uploadId == "" {
		return errors.New("上传任务不存在或已过期，请重新初始化")
	}
	objectKey, err := s.redis.Get(ctx, objectKeyKey).Result()
	if err != nil {
		return errors.New("文件路径丢失")
	}

	// 2.调用 OSS 上传分片
	// MinIO/S3 partNumber 从 1 开始，所以 +1
	partNumber := chunkIndex + 1
	partInfo, err := s.minio.UploadPart(ctx, objectKey, uploadId, partNumber, data)
	if err != nil {
		return fmt.Errorf("OSS 分片上传失败: %w", err)
	}

	// 3.保存分片信息到 Redis
	// 使用 Hset 保证幂等性（同一个分片传多次只会覆盖，不会重复添加）
	// Key: upload:parts:{fileHash}, Field: {chunkIndex}, Value: {ETag}
	uploadedPartsKey := fmt.Sprintf("upload:%d:%s:parts", userId, fileHash)

	// 这里只存 ETag 也可以，因为 field 是 index。
	// 为了方便后续 Complete，只存 ETag 字符串。
	err = s.redis.HSet(ctx, uploadedPartsKey, strconv.Itoa(chunkIndex), partInfo.ETag).Err()
	if err != nil {
		return err
	}

	// 刷新过期时间
	s.redis.Expire(ctx, uploadIdKey, 24*time.Hour)
	s.redis.Expire(ctx, uploadedPartsKey, 24*time.Hour)
	return nil
}

// MergeChunks 合并分片
func (s *fileService) MergeChunks(ctx context.Context, userId int, fileHash, fileName, parentId string, fileSize int64) (*models.File, error) {
	uploadIdKey := fmt.Sprintf("upload:%d:%s:id", userId, fileHash)
	objectKeyKey := fmt.Sprintf("upload:%d:%s:key", userId, fileHash)
	uploadedPartsKey := fmt.Sprintf("upload:%d:%s:parts", userId, fileHash)

	uploadId, err := s.redis.Get(ctx, uploadIdKey).Result()
	if err != nil || uploadId == "" {
		return nil, errors.New("上传任务失败")
	}
	objectKey, err := s.redis.Get(ctx, objectKeyKey).Result()

	// 1.获取所有分片 ETag
	partsMap, err := s.redis.HGetAll(ctx, uploadedPartsKey).Result()
	if err != nil || len(partsMap) == 0 {
		return nil, errors.New("未找到已上传的分片数据")
	}

	// 2.构造 MinIO 需要的 CompletePart 数组，并按 PartNumber 排序
	var completeParts []minio.CompletePart
	for chunkIdxStr, etag := range partsMap {
		idx, _ := strconv.Atoi(chunkIdxStr)
		completeParts = append(completeParts, minio.CompletePart{
			PartNumber: idx + 1, // 还原为 S3 的 1-based index
			ETag:       etag,
		})
	}

	// 排序，MinIO 要求 PartNumber 升序
	sort.Slice(completeParts, func(i, j int) bool {
		return completeParts[i].PartNumber < completeParts[j].PartNumber
	})

	// 3.调用 MinIO 合并分片，成功后产生 fileURL 与缩略图
	fileURL, thumbnailURL, err := s.minio.CompleteMultipartUpload(ctx, objectKey, uploadId, completeParts)
	if err != nil {
		return nil, fmt.Errorf("OSS 合并失败: %w", err)
	}

	// 4.写入数据库
	// 使用 filepath.Ext 获取带点的扩展名 (例如 ".txt")
	extWithDot := filepath.Ext(fileName)
	ext := strings.TrimPrefix(extWithDot, ".")

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

	// 事务处理：保存文件 + 更新配额
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.fileRepo.CreateFile(newFile); err != nil {
			return err
		}
		// 更新用户已用空间
		return s.storageQuotaRepo.UpdateUsedSpace(tx, userId, fileSize)
	})
	if err != nil {
		return nil, err
	}

	// 写入秒传缓存
	//fileInfoJSON, _ := json.Marshal(newFile)
	//s.redis.Set(ctx, "file:hash:"+fileHash, fileInfoJSON, 24*time.Hour)

	// 5.清理 Redis 缓存
	s.redis.Del(ctx, uploadIdKey, objectKeyKey, uploadedPartsKey)

	return newFile, nil
}

// CancelChunkUpload 取消上传
func (s *fileService) CancelChunkUpload(ctx context.Context, userId int, fileHash string) error {
	uploadIdKey := fmt.Sprintf("upload:%d:%s:id", userId, fileHash)
	objectKeyKey := fmt.Sprintf("upload:%d:%s:key", userId, fileHash)
	uploadedPartsKey := fmt.Sprintf("upload:%d:%s:parts", userId, fileHash)

	uploadId, err := s.redis.Get(ctx, uploadIdKey).Result()
	objectKey, _ := s.redis.Get(ctx, objectKeyKey).Result()
	if err == nil && uploadId != "" && objectKey != "" {
		// 通知 MinIO 取消
		_ = s.minio.AbortMultipartUpload(ctx, objectKey, uploadId)
	}

	// 清理 Redis
	s.redis.Del(ctx, uploadIdKey, objectKey, uploadedPartsKey)
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

	// 添加关键词搜索条件（模糊匹配文件名）
	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
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
		fileItems = append(fileItems, FileItem{
			Id:           file.Id,
			Name:         file.Name,
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
	if fileId == targetFolderId {
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

func (s *fileService) Download(ctx context.Context, fileId string) (io.ReadCloser, *models.File, error) {
	// 1.查询文件是否存在
	file, err := s.fileRepo.GetFileById(fileId)
	if err != nil {
		return nil, nil, errors.New("要下载的文件不存在")
	}
	// 2.从 MinIO 下载
	reader, err := s.minio.DownloadFile(ctx, file.OssObjectKey)
	if err != nil {
		return nil, nil, err
	}
	// 返回 reader + 文件信息
	return reader, file, nil
}

func nullToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}
