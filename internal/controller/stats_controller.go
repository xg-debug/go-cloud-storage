package controller

import (
	"go-cloud-storage/internal/models"
	"go-cloud-storage/internal/pkg/utils"
	"go-cloud-storage/internal/services"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
)

// StatsController 统计控制器
type StatsController struct {
	statsService   services.StatsService
	storageService services.StorageQuotaService
}

// NewStatsController 创建统计控制器实例
func NewStatsController(statsService services.StatsService, storageService services.StorageQuotaService) *StatsController {
	return &StatsController{statsService: statsService, storageService: storageService}
}

// GetUserDashboardStats 获取用户仪表板统计信息
func (c *StatsController) GetUserDashboardStats(ctx *gin.Context) {
	userId := ctx.GetInt("userId")
	stats, err := c.statsService.GetUserDashboardStats(userId)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "获取用户统计信息失败: "+err.Error())
		return
	}
	utils.Success(ctx, stats)
}

// GetUserStorage 获取侧边栏用户存储配额
func (c *StatsController) GetUserStorage(ctx *gin.Context) {
	userId := ctx.GetInt("userId")
	quota, err := c.storageService.GetUserQuota(userId)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, err.Error())
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

	storage := &models.StorageQuotaInfo{
		Total:       quota.Total,
		Used:        quota.Used,
		UsedPercent: math.Round(usedPercent*100) / 100, // 保留两位小数
		TotalGB:     math.Round(totalGB*100) / 100,     // 保留两位小数
		UsedGB:      math.Round(usedGB*100) / 100,      // 保留两位小数
	}

	utils.Success(ctx, storage)
}
