package application

import (
	"fmt"
	"math"

	"gorm.io/gorm"

	"go-cloud-storage/backend/pkg/filetypes"
	repository "go-cloud-storage/backend/services/file-service/internal/adapters/repository"
	domain "go-cloud-storage/backend/services/file-service/internal/domain"
)

type ShareIndex interface {
	CountSharedFiles(int) (int64, error)
	DeleteBatch(*gorm.DB, []string) error
}

// StatsService 统计服务接口
type StatsService interface {
	GetUserDashboardStats(userId int) (*domain.UserDashboardStatsResponse, error)
}

type statsService struct {
	fileRepo         repository.FileRepository
	storageQuotaRepo repository.StorageQuotaRepository
	shareRepo        ShareIndex
}

// NewStatsService 创建统计服务实例
func NewStatsService(fileRepo repository.FileRepository, storageQuotaRepo repository.StorageQuotaRepository, shareRepo ShareIndex) StatsService {
	return &statsService{
		fileRepo:         fileRepo,
		storageQuotaRepo: storageQuotaRepo,
		shareRepo:        shareRepo,
	}
}

// GetUserDashboardStats 获取用户仪表板统计信息
func (s *statsService) GetUserDashboardStats(userId int) (*domain.UserDashboardStatsResponse, error) {
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

	storageQuotaInfo := domain.StorageQuotaInfo{
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
	fileTypeMap := make(map[string]*domain.FileTypeStat)

	// 初始化文件类型统计
	fileTypeMap["image"] = &domain.FileTypeStat{Type: "image", Name: "图片", Count: 0}
	fileTypeMap["video"] = &domain.FileTypeStat{Type: "video", Name: "视频", Count: 0}
	fileTypeMap["audio"] = &domain.FileTypeStat{Type: "audio", Name: "音频", Count: 0}
	fileTypeMap["document"] = &domain.FileTypeStat{Type: "document", Name: "文档", Count: 0}
	fileTypeMap["other"] = &domain.FileTypeStat{Type: "other", Name: "其他", Count: 0}

	for _, file := range files {
		if file.IsDir {
			folders++
		} else {
			totalFiles++
			// 判断文件类型
			fileType := filetypes.Category(file.FileExtension)
			if stat, exists := fileTypeMap[fileType]; exists {
				stat.Count++
				stat.Size += file.Size
			}
		}
	}
	// 文件夹数量需要减1，因为用户注册的时候系统默认创建了一个根目录"/"
	folders--
	// 查询共享文件的数量
	sharedFiles, err = s.shareRepo.CountSharedFiles(userId)
	if err != nil {
		return nil, err
	}

	// 计算文件类型百分比和GB大小
	var fileTypeStats []domain.FileTypeStat
	for _, stat := range fileTypeMap {
		if totalFiles > 0 {
			stat.Percentage = math.Round(float64(stat.Count)/float64(totalFiles)*100*100) / 100
		}
		stat.SizeGB = math.Round(float64(stat.Size)/(1024*1024*1024)*100) / 100
		stat.SizeStr = formatSize(stat.Size)
		fileTypeStats = append(fileTypeStats, *stat)
	}

	fileStatsInfo := domain.FileStatsInfo{
		TotalFiles:  totalFiles,
		Folders:     folders,
		SharedFiles: sharedFiles,
	}

	// 3. 组装返回结果
	return &domain.UserDashboardStatsResponse{
		StorageQuota:  storageQuotaInfo,
		FileStats:     fileStatsInfo,
		FileTypeStats: fileTypeStats,
	}, nil
}

func formatSize(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	k := float64(1024)
	size := float64(bytes)
	i := 0
	for size >= k && i < len(units)-1 {
		size /= k
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%d B", bytes)
	}
	return fmt.Sprintf("%.1f %s", size, units[i])
}
