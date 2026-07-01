package controller

import (
	"errors"
	"go-cloud-storage/backend/internal/middleware"
	"go-cloud-storage/backend/pkg/utils"
	"log/slog"
	"strconv"

	"go-cloud-storage/backend/internal/services"

	"github.com/gin-gonic/gin"
)

type ShareController struct {
	shareService  services.ShareService
	bruteProtector *middleware.ShareBruteProtector
}

func NewShareController(shareService services.ShareService, protector *middleware.ShareBruteProtector) *ShareController {
	return &ShareController{
		shareService:  shareService,
		bruteProtector: protector,
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
		utils.Fail(ctx, 400, "参数错误")
		return
	}

	share, err := c.shareService.CreateShare(userId, req.FileId, req.ExpireDays, req.ExtractionCode)
	if err != nil {
		slog.Error("创建分享失败", "error", err)
		utils.Fail(ctx, 500, "创建分享失败")
		return
	}
	utils.Success(ctx, share)
}

// GetUserShares 获取用户的分享列表
func (c *ShareController) GetUserShares(ctx *gin.Context) {
	userId := ctx.GetInt("userId")
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))

	shares, total, err := c.shareService.GetUserShares(userId, page, pageSize)
	if err != nil {
		slog.Error("获取分享列表失败", "error", err)
		utils.Fail(ctx, 500, "获取分享列表失败")
		return
	}
	utils.Success(ctx, gin.H{"list": shares, "total": total})
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
		slog.Error("获取分享详情失败", "error", err)
		utils.Fail(ctx, 500, "获取分享详情失败")
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
		slog.Error("取消分享失败", "error", err)
		utils.Fail(ctx, 500, "取消分享失败")
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
		utils.Fail(ctx, 400, "参数错误")
		return
	}

	err = c.shareService.UpdateShare(shareId, userId, req.ExtractionCode, req.ExpireDays)
	if err != nil {
		slog.Error("更新分享失败", "error", err)
		utils.Fail(ctx, 500, "更新分享失败")
		return
	}
	utils.Success(ctx, nil)
}

// AccessShare 访问分享（通过分享链接）
func (c *ShareController) AccessShare(ctx *gin.Context) {
	shareToken := ctx.Param("token")
	extractionCode := ctx.Query("code")
	clientIP := ctx.ClientIP()

	// 暴力破解防护：检查是否被锁定
	if c.bruteProtector.IsLocked(shareToken, clientIP) {
		slog.Warn("share access locked due to brute force", "token", shareToken, "ip", clientIP)
		utils.Fail(ctx, 429, "尝试次数过多，请15分钟后再试")
		return
	}

	shareInfo, err := c.shareService.AccessShare(shareToken, extractionCode)
	if err != nil {
		// 提取码错误时记录失败尝试
		if errors.Is(err, services.ErrExtractCodeWrong) {
			if locked := c.bruteProtector.RecordFailed(shareToken, clientIP); locked {
				slog.Warn("share brute force lock triggered", "token", shareToken, "ip", clientIP)
			}
		}
		slog.Info("share access failed", "token", shareToken, "ip", clientIP, "error", err)
		// 仅暴露用户友好的错误，防止内部错误泄露
		if errors.Is(err, services.ErrExtractCodeWrong) || errors.Is(err, services.ErrShareExpired) || errors.Is(err, services.ErrShareNotFound) {
			utils.Fail(ctx, 400, err.Error())
		} else {
			utils.Fail(ctx, 400, "访问分享失败")
		}
		return
	}

	// 成功时重置失败计数
	c.bruteProtector.Reset(shareToken, clientIP)
	slog.Info("share accessed", "token", shareToken, "ip", clientIP, "file", shareInfo.FileName)
	utils.Success(ctx, shareInfo)
}

// DownloadSharedFile 下载分享的文件
func (c *ShareController) DownloadSharedFile(ctx *gin.Context) {
	shareToken := ctx.Param("token")
	extractionCode := ctx.Query("code")

	downloadURL, err := c.shareService.DownloadSharedFile(shareToken, extractionCode)
	if err != nil {
		slog.Error("下载分享文件失败", "error", err)
		utils.Fail(ctx, 400, "下载失败")
		return
	}

	utils.Success(ctx, downloadURL)
}
