package controller

import (
	"go-cloud-storage/internal/models"
	"go-cloud-storage/internal/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type NotificationController struct {
	notificationService *services.NotificationService
}

func NewNotificationController(notificationService *services.NotificationService) *NotificationController {
	return &NotificationController{
		notificationService: notificationService,
	}
}

// GetNotifications 获取用户通知列表
func (c *NotificationController) GetNotifications(ctx *gin.Context) {
	userId := ctx.GetUint("userId")

	// 获取分页参数
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	result, err := c.notificationService.GetUserNotifications(userId, page, pageSize)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取通知失败",
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取成功",
		"data":    result,
	})
}

// GetUnreadCount 获取未读通知数量
func (c *NotificationController) GetUnreadCount(ctx *gin.Context) {
	userId := ctx.GetUint("userId")

	count, err := c.notificationService.GetUnreadCount(userId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取未读数量失败",
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取成功",
		"data": gin.H{
			"unread_count": count,
		},
	})
}

// MarkAsRead 标记通知为已读
func (c *NotificationController) MarkAsRead(ctx *gin.Context) {
	userId := ctx.GetUint("userId")

	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的通知ID",
		})
		return
	}

	err = c.notificationService.MarkAsRead(uint(id), userId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "标记已读失败",
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "标记成功",
	})
}

// MarkAllAsRead 标记所有通知为已读
func (c *NotificationController) MarkAllAsRead(ctx *gin.Context) {
	userId := ctx.GetUint("userId")

	err := c.notificationService.MarkAllAsRead(userId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "标记失败",
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "全部标记成功",
	})
}

// DeleteNotification 删除通知
func (c *NotificationController) DeleteNotification(ctx *gin.Context) {
	userId := ctx.GetUint("user_id")

	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的通知ID",
		})
		return
	}

	err = c.notificationService.DeleteNotification(uint(id), userId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "删除失败",
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "删除成功",
	})
}

// DeleteAllNotifications 删除所有通知
func (c *NotificationController) DeleteAllNotifications(ctx *gin.Context) {
	userId := ctx.GetUint("userId")

	err := c.notificationService.DeleteAllNotifications(userId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "删除失败",
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "全部删除成功",
	})
}

// CreateNotification 创建通知（管理员接口）
func (c *NotificationController) CreateNotification(ctx *gin.Context) {
	var req models.NotificationCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误",
			"error":   err.Error(),
		})
		return
	}

	err := c.notificationService.CreateNotification(&req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "创建通知失败",
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "创建成功",
	})
}
