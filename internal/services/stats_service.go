package services

import (
	"go-cloud-storage/internal/models"
	"go-cloud-storage/internal/repositories"
	"math"
)

// StatsService 统计服务接口
type StatsService interface {
	GetUserDashboardStats(userId int) (*models.UserDashboardStatsResponse, error)
}

type statsService struct {
	fileRepo         repositories.FileRepository
	storageQuotaRepo repositories.StorageQuotaRepository
	shareRepo        repositories.ShareRepository
}

// NewStatsService 创建统计服务实例
func NewStatsService(fileRepo repositories.FileRepository, storageQuotaRepo repositories.StorageQuotaRepository, shareRepo repositories.ShareRepository) StatsService {
	return &statsService{
		fileRepo:         fileRepo,
		storageQuotaRepo: storageQuotaRepo,
		shareRepo:        shareRepo,
	}
}

// GetUserDashboardStats 获取用户仪表板统计信息
func (s *statsService) GetUserDashboardStats(userId int) (*models.UserDashboardStatsResponse, error) {
	// 1. 获取存储配额信息
	quota, err := s.storageQuotaRepo.GetByUserID(userId)
	if err != nil {
		// 如果不存在配额记录，创建默认配额
		if err := s.storageQuotaRepo.EnsureUserQuota(userId); err != nil {
			return nil, err
		}
		quota, err = s.storageQuotaRepo.GetByUserID(userId)
		if err != nil {
			return nil, err
		}
	}

	// 计算存储配额信息
	usedPercent := float64(0)
	if quota.Total > 0 {
		usedPercent = float64(quota.Used) / float64(quota.Total) * 100
	}

	// 转换为GB
	totalGB := float64(quota.Total) / (1024 * 1024 * 1024)
	usedGB := float64(quota.Used) / (1024 * 1024 * 1024)

	storageQuotaInfo := models.StorageQuotaInfo{
		Total:       quota.Total,
		Used:        quota.Used,
		UsedPercent: math.Round(usedPercent*100) / 100, // 保留两位小数
		TotalGB:     math.Round(totalGB*100) / 100,     // 保留两位小数
		UsedGB:      math.Round(usedGB*100) / 100,      // 保留两位小数
	}

	// 2. 获取文件统计信息
	// 获取用户所有文件（不包括已删除的）
	files, err := s.fileRepo.GetAllUserFiles(userId)
	if err != nil {
		return nil, err
	}

	// 统计文件信息
	var totalFiles, folders, sharedFiles int64

	// 文件类型统计
	fileTypeMap := make(map[string]*models.FileTypeStat)

	// 初始化文件类型统计
	fileTypeMap["image"] = &models.FileTypeStat{Type: "image", Name: "图片", Count: 0}
	fileTypeMap["video"] = &models.FileTypeStat{Type: "video", Name: "视频", Count: 0}
	fileTypeMap["audio"] = &models.FileTypeStat{Type: "audio", Name: "音频", Count: 0}
	fileTypeMap["document"] = &models.FileTypeStat{Type: "document", Name: "文档", Count: 0}
	fileTypeMap["other"] = &models.FileTypeStat{Type: "other", Name: "其他", Count: 0}

	for _, file := range files {
		if file.IsDir {
			folders++
		} else {
			totalFiles++
			// 判断文件类型
			fileType := getFileType(file.FileExtension)
			if stat, exists := fileTypeMap[fileType]; exists {
				stat.Count++
				//stat.Size += file.Size
			}
		}
	}
	// 文件夹数量需要减1，因为用户注册的时候系统默认创建了一个根目录"/"
	folders--
	// 查询共享文件的数量
	sharedFiles, err = s.shareRepo.CountSharedFiles(userId)

	// 计算文件类型百分比和GB大小
	var fileTypeStats []models.FileTypeStat
	for _, stat := range fileTypeMap {
		if totalFiles > 0 {
			stat.Percentage = math.Round(float64(stat.Count)/float64(totalFiles)*100*100) / 100
		}
		//stat.SizeGB = math.Round(float64(stat.Size)/(1024*1024*1024)*100) / 100
		fileTypeStats = append(fileTypeStats, *stat)
	}

	fileStatsInfo := models.FileStatsInfo{
		TotalFiles:  totalFiles,
		Folders:     folders,
		SharedFiles: sharedFiles,
	}

	// 3. 组装返回结果
	return &models.UserDashboardStatsResponse{
		StorageQuota:  storageQuotaInfo,
		FileStats:     fileStatsInfo,
		FileTypeStats: fileTypeStats,
	}, nil
}

// getFileType 根据文件扩展名判断文件类型
func getFileType(extension string) string {
	// 去掉扩展名前面的点
	if len(extension) > 0 && extension[0] == '.' {
		extension = extension[1:]
	}

	// 转换为小写
	ext := extension

	// 图片类型
	imageExts := []string{"jpg", "jpeg", "png", "gif", "bmp", "webp", "svg"}
	for _, imgExt := range imageExts {
		if ext == imgExt {
			return "image"
		}
	}

	// 视频类型
	videoExts := []string{"mp4", "avi", "mov", "wmv", "flv", "webm", "mkv"}
	for _, vidExt := range videoExts {
		if ext == vidExt {
			return "video"
		}
	}

	// 音频类型
	audioExts := []string{"mp3", "wav", "flac", "aac", "ogg", "m4a"}
	for _, audExt := range audioExts {
		if ext == audExt {
			return "audio"
		}
	}

	// 文档类型
	docExts := []string{"txt", "md", "pdf", "doc", "docx", "xls", "xlsx", "ppt", "pptx", "csv", "json", "xml"}
	for _, docExt := range docExts {
		if ext == docExt {
			return "document"
		}
	}

	// 其他类型
	return "other"
}
