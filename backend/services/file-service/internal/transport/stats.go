package transport

import (
	"log/slog"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"

	"go-cloud-storage/backend/pkg/utils"
	application "go-cloud-storage/backend/services/file-service/internal/application"
	domain "go-cloud-storage/backend/services/file-service/internal/domain"
)

// StatsController 统计控制器
type StatsController struct {
	statsService   application.StatsService
	storageService application.StorageQuotaService
}

// NewStatsController 创建统计控制器实例
func NewStatsController(statsService application.StatsService, storageService application.StorageQuotaService) *StatsController {
	return &StatsController{statsService: statsService, storageService: storageService}
}

// GetUserDashboardStats 获取用户仪表板统计信息
func (c *StatsController) GetUserDashboardStats(ctx *gin.Context) {
	userId := ctx.GetInt("userId")
	stats, err := c.statsService.GetUserDashboardStats(userId)
	if err != nil {
		slog.Error("获取用户统计信息失败", "error", err)
		utils.Fail(ctx, http.StatusInternalServerError, "获取用户统计信息失败")
		return
	}
	utils.Success(ctx, stats)
}

// GetUserStorage 获取侧边栏用户存储配额
func (c *StatsController) GetUserStorage(ctx *gin.Context) {
	userId := ctx.GetInt("userId")
	quota, err := c.storageService.GetUserQuota(userId)
	if err != nil {
		slog.Error("获取存储配额失败", "error", err)
		utils.Fail(ctx, http.StatusInternalServerError, "获取存储配额失败")
		return
	}
	// 计算存储配额信息
	usedPercent := float64(0)
	if quota.Total > 0 {
		usedPercent = float64(quota.Used) / float64(quota.Total) * 100
	}
	// 转换为GB
	totalGB := float64(quota.Total) / (1024 * 1024 * 1024)
	usedGB := float64(quota.Used) / (1024 * 1024 * 1024)

	storage := &domain.StorageQuotaInfo{
		Total:       quota.Total,
		Used:        quota.Used,
		UsedPercent: math.Round(usedPercent*100) / 100, // 保留两位小数
		TotalGB:     math.Round(totalGB*100) / 100,     // 保留两位小数
		UsedGB:      math.Round(usedGB*100) / 100,      // 保留两位小数
	}

	utils.Success(ctx, storage)
}
