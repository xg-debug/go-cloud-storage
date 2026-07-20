package services

import (
	"context"
	"database/sql"
	"errors"
	miniosrv "go-cloud-storage/backend/infrastructure/minio"
	"go-cloud-storage/backend/internal/models"
	"go-cloud-storage/backend/internal/repositories"
	"go-cloud-storage/backend/pkg/filetypes"
	"go-cloud-storage/backend/pkg/utils"
	"io"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"

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
	FileCount    int64  `json:"file_count"`
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

type DuplicateGroup struct {
	FileHash    string     `json:"fileHash"`
	FileSize    int64      `json:"fileSize"`
	SizeStr     string     `json:"sizeStr"`
	Count       int        `json:"count"`
	WastedSpace int64      `json:"wastedSpace"`
	WastedStr   string     `json:"wastedStr"`
	Files       []FileItem `json:"files"`
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
	InitChunkUpload(ctx context.Context, userId int, filename, fileMd5 string, parentId string, fileSize int64, chunkSize int64, totalChunks int) (gin.H, error)
	UploadChunk(ctx context.Context, userId int, fileHash string, chunkIndex int, r io.Reader, chunkSize int64, expectedChunkHash string) error
	MergeChunks(ctx context.Context, userId int, fileHash, fileName, parentId string, fileSize int64, chunkSize int64, totalChunks int) (*models.File, error)
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
	DownloadBatchZip(ctx context.Context, userId int, fileIds []string) (io.ReadCloser, string, error)
	GetDuplicateFiles(ctx context.Context, userId int) ([]DuplicateGroup, int64, error)
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
		var fileCount int64
		var cntErr error
		if file.IsDir {
			fileCount, cntErr = s.fileRepo.CountFilesInFolder(ctx, userId, file.Id)
			if cntErr != nil {
				slog.Warn("CountFilesInFolder failed", "folderId", file.Id, "error", cntErr)
			}
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
			FileCount:    fileCount,
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
	if file.UserId != userId {
		return errors.New("无权限操作该文件")
	}
	if file.IsDeleted {
		return errors.New("文件已删除")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if file.IsDir {
			deletedIds, err := s.fileRepo.SoftDeleteFolder(tx, userId, fileId)
			if err != nil {
				return err
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
	return filetypes.Previewable(extension)
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
func (s *fileService) SearchFiles(userId int, keyword, parentId string, page, pageSize int) ([]FileItem, int64, error) {
	offset := (page - 1) * pageSize

	query := s.db.Model(&models.File{}).Where("user_id = ? AND is_deleted = ?", userId, false)

	if parentId != "" {
		query = query.Where("parent_id = ?", parentId)
	}

	if keyword != "" {
		kw := strings.TrimSpace(keyword)
		if kw != "" {
			ext := strings.TrimPrefix(strings.ToLower(kw), ".")
			extTypes := []string{"jpg", "jpeg", "png", "gif", "bmp", "webp", "svg",
				"mp4", "avi", "mov", "wmv", "flv", "mkv", "webm",
				"mp3", "wav", "flac", "aac", "ogg", "m4a",
				"pdf", "doc", "docx", "xls", "xlsx", "ppt", "pptx",
				"txt", "md", "csv", "json", "xml", "zip", "rar", "7z"}
			isExt := false
			for _, e := range extTypes {
				if ext == e {
					isExt = true
					break
				}
			}
			if isExt {
				query = query.Where("file_extension = ?", ext)
			} else {
				query = query.Where("name LIKE ?", "%"+kw+"%")
			}
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var files []models.File
	if err := query.Order("is_dir DESC, name ASC").Offset(offset).Limit(pageSize).Find(&files).Error; err != nil {
		return nil, 0, err
	}

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

func (s *fileService) PreviewStream(ctx context.Context, userId int, fileId string) (io.ReadCloser, *models.File, error) {
	return s.Download(ctx, userId, fileId)
}

// GetDuplicateFiles finds duplicate files based on SHA-256 hash for the user.
func (s *fileService) GetDuplicateFiles(ctx context.Context, userId int) ([]DuplicateGroup, int64, error) {
	type hashGroup struct {
		FileHash string
		Cnt      int
	}
	var groups []hashGroup
	err := s.db.Model(&models.File{}).
		Select("file_hash, COUNT(*) as cnt").
		Where("user_id = ? AND is_deleted = ? AND is_dir = ?", userId, false, false).
		Where("file_hash IS NOT NULL AND file_hash != ''").
		Group("file_hash").
		Having("cnt > 1").
		Find(&groups).Error
	if err != nil {
		return nil, 0, err
	}
	if len(groups) == 0 {
		return nil, 0, nil
	}

	var hashes []string
	for _, g := range groups {
		hashes = append(hashes, g.FileHash)
	}

	var dupFiles []models.File
	err = s.db.Where("user_id = ? AND is_deleted = ? AND file_hash IN ?", userId, false, hashes).
		Order("file_hash, created_at ASC").
		Find(&dupFiles).Error
	if err != nil {
		return nil, 0, err
	}

	// Group files by hash
	fileMap := make(map[string][]models.File)
	for _, f := range dupFiles {
		fileMap[f.FileHash] = append(fileMap[f.FileHash], f)
	}

	var result []DuplicateGroup
	var totalWasted int64
	for _, hash := range hashes {
		files := fileMap[hash]
		if len(files) < 2 {
			continue
		}
		var items []FileItem
		for _, f := range files {
			pid := ""
			if f.ParentId.Valid {
				pid = f.ParentId.String
			}
			items = append(items, FileItem{
				Id:        f.Id,
				Name:      f.Name,
				ParentId:  pid,
				IsDir:     f.IsDir,
				Size:      f.Size,
				SizeStr:   f.SizeStr,
				Extension: f.FileExtension,
				CreatedAt: f.CreatedAt.Format("2006-01-02 15:04:05"),
				Modified:  f.UpdatedAt.Format("2006-01-02 15:04:05"),
			})
		}
		wasted := files[0].Size * int64(len(files)-1)
		totalWasted += wasted
		result = append(result, DuplicateGroup{
			FileHash:    hash,
			FileSize:    files[0].Size,
			SizeStr:     files[0].SizeStr,
			Count:       len(files),
			WastedSpace: wasted,
			WastedStr:   utils.FormatFileSize(wasted),
			Files:       items,
		})
	}

	return result, totalWasted, nil
}

func nullToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}
