package controller

import (
	"go-cloud-storage/internal/pkg/utils"
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
	userId := ctx.GetInt("userId")

	var req struct {
		FileId         string `json:"file_id" binding:"required"`
		ExtractionCode string `json:"extraction_code"`
		ExpireDays     int    `json:"expire_days"` // 0表示永久有效
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	share, err := c.shareService.CreateShare(userId, req.FileId, req.ExtractionCode, req.ExpireDays)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	utils.Success(ctx, share)
}

// GetUserShares 获取用户的分享列表
func (c *ShareController) GetUserShares(ctx *gin.Context) {
	userId := ctx.GetInt("userId")

	shares, err := c.shareService.GetUserShares(userId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	utils.Success(ctx, shares)
}

// GetShareDetail 获取分享详情
func (c *ShareController) GetShareDetail(ctx *gin.Context) {
	shareId, err := strconv.Atoi(ctx.Param("shareId"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的分享ID"})
		return
	}

	userId := ctx.GetInt("userId")

	share, err := c.shareService.GetShareDetail(shareId, userId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	utils.Success(ctx, share)
}

// CancelShare 取消分享
func (c *ShareController) CancelShare(ctx *gin.Context) {
	shareId, err := strconv.Atoi(ctx.Param("shareId"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的分享ID"})
		return
	}

	userId := ctx.GetInt("userId")

	err = c.shareService.CancelShare(shareId, userId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	utils.Success(ctx, nil)
}

// DeleteShare 删除分享记录
func (c *ShareController) DeleteShare(ctx *gin.Context) {
	shareID, err := strconv.Atoi(ctx.Param("shareId"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的分享ID"})
		return
	}

	userId := ctx.GetInt("userId")

	err = c.shareService.DeleteShare(shareID, userId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	utils.Success(ctx, nil)
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
	utils.Success(ctx, shareInfo)
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

	utils.Success(ctx, downloadURL)
}
