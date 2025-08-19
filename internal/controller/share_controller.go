package controller

import (
	"net/http"
	"strconv"

	"go-cloud-storage/internal/services"

	"github.com/gin-gonic/gin"
)

type ShareController struct {
	shareService services.ShareService
}

func NewShareController(shareService services.ShareService) *ShareController {
	return &ShareController{
		shareService: shareService,
	}
}

// CreateShare 创建分享
func (c *ShareController) CreateShare(ctx *gin.Context) {
	userID := ctx.GetInt("user_id")

	var req struct {
		FileID         string `json:"file_id" binding:"required"`
		ExtractionCode string `json:"extraction_code"`
		ExpireDays     int    `json:"expire_days"` // 0表示永久有效
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	share, err := c.shareService.CreateShare(userID, req.FileID, req.ExtractionCode, req.ExpireDays)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"data":    share,
		"message": "分享创建成功",
	})
}

// GetUserShares 获取用户的分享列表
func (c *ShareController) GetUserShares(ctx *gin.Context) {
	userID := ctx.GetInt("user_id")

	shares, err := c.shareService.GetUserShares(userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"data":    shares,
		"message": "获取分享列表成功",
	})
}

// GetShareDetail 获取分享详情
func (c *ShareController) GetShareDetail(ctx *gin.Context) {
	shareID, err := strconv.Atoi(ctx.Param("shareId"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的分享ID"})
		return
	}

	userID := ctx.GetInt("user_id")

	share, err := c.shareService.GetShareDetail(shareID, userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"data":    share,
		"message": "获取分享详情成功",
	})
}

// CancelShare 取消分享
func (c *ShareController) CancelShare(ctx *gin.Context) {
	shareID, err := strconv.Atoi(ctx.Param("shareId"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的分享ID"})
		return
	}

	userID := ctx.GetInt("user_id")

	err = c.shareService.CancelShare(shareID, userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "分享已取消",
	})
}

// DeleteShare 删除分享记录
func (c *ShareController) DeleteShare(ctx *gin.Context) {
	shareID, err := strconv.Atoi(ctx.Param("shareId"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的分享ID"})
		return
	}

	userID := ctx.GetInt("user_id")

	err = c.shareService.DeleteShare(shareID, userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "分享记录已删除",
	})
}

// AccessShare 访问分享（通过分享链接）
func (c *ShareController) AccessShare(ctx *gin.Context) {
	shareToken := ctx.Param("token")
	extractionCode := ctx.Query("code")

	shareInfo, err := c.shareService.AccessShare(shareToken, extractionCode)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"data":    shareInfo,
		"message": "访问分享成功",
	})
}

// DownloadSharedFile 下载分享的文件
func (c *ShareController) DownloadSharedFile(ctx *gin.Context) {
	shareToken := ctx.Param("token")
	extractionCode := ctx.Query("code")

	downloadURL, err := c.shareService.DownloadSharedFile(shareToken, extractionCode)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"data":    gin.H{"download_url": downloadURL},
		"message": "获取下载链接成功",
	})
}
