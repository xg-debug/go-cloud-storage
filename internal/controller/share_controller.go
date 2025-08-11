package controller

import (
	"github.com/gin-gonic/gin"
	"go-cloud-storage/internal/models"
	"go-cloud-storage/internal/services"
	"net/http"
	"strconv"
)

type ShareController struct {
	shareService services.ShareService
}

func NewShareController(service services.ShareService) *ShareController {
	return &ShareController{shareService: service}
}

// CreateShare 创建分享
func (c *ShareController) CreateShare(ctx *gin.Context) {
	var share models.ShareRecord
	if err := ctx.ShouldBindJSON(&share); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userId := ctx.GetInt("userId")
	share.OwnerID = userId

	if err := c.shareService.CreateShare(&share); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, share)
}

// ListUserShares 获取用户分享列表
func (c *ShareController) ListUserShares(ctx *gin.Context) {
	userId := ctx.GetInt("userId")
	shares, err := c.shareService.GetUserShares(userId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, shares)
}

// DeleteShare 删除分享
func (c *ShareController) DeleteShare(ctx *gin.Context) {
	shareId, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid share ID"})
		return
	}
	userId := ctx.GetInt("userId")
	if err := c.shareService.DeleteShare(shareId, userId); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.Status(http.StatusNoContent)
}

// AccessShare 访问分享内容
func (c *ShareController) AccessShare(ctx *gin.Context) {
	link := ctx.Param("link")
	password := ctx.Query("password")

	share, err := c.shareService.ValidateShare(link, password)
	if err != nil {
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	// 获取分享的文件/文件夹详情（需实现文件服务）
	// file, err := fileService.GetFile(share.TargetID)
	// ...

	ctx.JSON(http.StatusOK, gin.H{
		"share": share,
		// "file": file,
	})
}
