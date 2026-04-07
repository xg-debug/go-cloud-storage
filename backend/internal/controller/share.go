package controller

import (
	"go-cloud-storage/backend/pkg/utils"
	"strconv"

	"go-cloud-storage/backend/internal/services"

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
		utils.Fail(ctx, 400, "参数错误: "+err.Error())
		return
	}

	share, err := c.shareService.CreateShare(userId, req.FileId, req.ExpireDays, req.ExtractionCode)
	if err != nil {
		utils.Fail(ctx, 500, err.Error())
		return
	}
	utils.Success(ctx, share)
}

// GetUserShares 获取用户的分享列表
func (c *ShareController) GetUserShares(ctx *gin.Context) {
	userId := ctx.GetInt("userId")

	shares, err := c.shareService.GetUserShares(userId)
	if err != nil {
		utils.Fail(ctx, 500, err.Error())
		return
	}
	utils.Success(ctx, shares)
}

// GetShareDetail 获取分享详情
func (c *ShareController) GetShareDetail(ctx *gin.Context) {
	shareId, err := strconv.Atoi(ctx.Param("shareId"))
	if err != nil {
		utils.Fail(ctx, 400, "无效的分享ID")
		return
	}

	userId := ctx.GetInt("userId")

	share, err := c.shareService.GetShareDetail(userId, shareId)
	if err != nil {
		utils.Fail(ctx, 500, err.Error())
		return
	}
	utils.Success(ctx, share)
}

// CancelShare 取消分享
func (c *ShareController) CancelShare(ctx *gin.Context) {
	shareId, err := strconv.Atoi(ctx.Param("shareId"))
	if err != nil {
		utils.Fail(ctx, 400, "无效的分享ID")
		return
	}

	userId := ctx.GetInt("userId")

	err = c.shareService.CancelShare(userId, shareId)
	if err != nil {
		utils.Fail(ctx, 500, err.Error())
		return
	}

	utils.Success(ctx, nil)
}

// UpdateShare 更新分享设置
func (c *ShareController) UpdateShare(ctx *gin.Context) {
	shareId, err := strconv.Atoi(ctx.Param("shareId"))
	if err != nil {
		utils.Fail(ctx, 400, "无效的分享ID")
		return
	}
	userId := ctx.GetInt("userId")

	var req struct {
		ExtractionCode string `json:"extraction_code"`
		ExpireDays     int    `json:"expire_days"` // 0表示永久有效
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Fail(ctx, 400, "参数错误: "+err.Error())
		return
	}

	err = c.shareService.UpdateShare(shareId, userId, req.ExtractionCode, req.ExpireDays)
	if err != nil {
		utils.Fail(ctx, 500, err.Error())
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
		utils.Fail(ctx, 400, err.Error())
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
		utils.Fail(ctx, 400, err.Error())
		return
	}

	utils.Success(ctx, downloadURL)
}
