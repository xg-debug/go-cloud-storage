package services

import (
	"context"
	"database/sql"
	"errors"
	"go-cloud-storage/internal/models"
	"go-cloud-storage/internal/pkg/utils"
	"go-cloud-storage/internal/repositories"
	"strings"
	"time"

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

type FileService interface {
	GetFileById(fileId string) (*models.File, error)
	GetFiles(ctx context.Context, userId int, parentId string, page int, pageSize int) ([]FileItem, int64, error)
	CreateFolder(userId int, folderName string, parentId string) (*models.File, error)
	UploadFile(fileName, extension string, size int64, parentId string) (*models.File, error)
	Rename(userId int, fileId, newName string) error
	Delete(fileId string, userId int) error
	CreateFileInfo(file *models.File) error
	GetRecentFiles(userId int, timeRange string) ([]*RecentFile, error)
	GetFilePath(file *models.File) (string, error)
	PreviewFile(ctx context.Context, userId int, fileId string) (*FilePreview, error)
	CheckFileExistsByMD5(userId int, fileMD5 string) (bool, *models.File, error)
	SearchFiles(ctx context.Context, userId int, keyword, parentId string, page, pageSize int) ([]FileItem, int64, error)
}

type fileService struct {
	db               *gorm.DB
	fileRepo         repositories.FileRepository
	storageQuotaRepo repositories.StorageQuotaRepository
}

func NewFileService(db *gorm.DB, repo repositories.FileRepository, storageQuotaRepo repositories.StorageQuotaRepository) FileService {
	return &fileService{db: db, fileRepo: repo, storageQuotaRepo: storageQuotaRepo}
}

func (s *fileService) GetFileById(fileId string) (*models.File, error) {
	return s.fileRepo.GetFileById(fileId)
}

func (s *fileService) GetFiles(ctx context.Context, userId int, parentId string, page int, pageSize int) ([]FileItem, int64, error) {
	files, total, err := s.fileRepo.GetFiles(ctx, userId, parentId, page, pageSize)
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

func (s *fileService) UploadFile(fileName, extension string, size int64, parentId string) (*models.File, error) {
	var pId sql.NullString
	if parentId == "" {
		pId = sql.NullString{Valid: false} // NULL
	} else {
		pId = sql.NullString{String: parentId, Valid: true} // 有值
	}

	file := &models.File{
		Id:            utils.NewUUID(),
		Name:          fileName,
		Size:          size,
		IsDir:         false,
		FileExtension: extension,
		FileHash:      "hash",
		ParentId:      pId,
		IsDeleted:     false,
		CreatedAt:     time.Now(),
	}
	err := s.fileRepo.CreateFile(file)
	return file, err
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
			if err := s.storageQuotaRepo.UpdateUsedSpace(userId, -fileSize); err != nil {
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

	pathParts := []string{}
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

func (s *fileService) PreviewFile(ctx context.Context, userId int, fileId string) (*FilePreview, error) {
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

// 检查文件是否存在（通过MD5）
func (s *fileService) CheckFileExistsByMD5(userId int, fileMD5 string) (bool, *models.File, error) {
	file, err := s.fileRepo.GetFileByMD5(userId, fileMD5)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil, nil
		}
		return false, nil, err
	}

	// 检查文件是否已删除
	if file.IsDeleted {
		return false, nil, nil
	}

	return true, file, nil
}

// 检查文件是否存在（通过Hash）
func (s *fileService) CheckFileExistsByHash(fileHash string, userId int) (bool, *models.File, error) {
	return s.CheckFileExistsByMD5(userId, fileHash)
}

// SearchFiles 搜索文件和文件夹
func (s *fileService) SearchFiles(ctx context.Context, userId int, keyword, parentId string, page, pageSize int) ([]FileItem, int64, error) {
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
