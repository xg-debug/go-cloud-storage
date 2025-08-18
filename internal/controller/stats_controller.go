package controller

import (
	"go-cloud-storage/internal/pkg/utils"
	"go-cloud-storage/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

// StatsController 统计控制器
type StatsController struct {
	statsService services.StatsService
}

// NewStatsController 创建统计控制器实例
func NewStatsController(service services.StatsService) *StatsController {
	return &StatsController{statsService: service}
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
