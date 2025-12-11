package services

import (
	"go-cloud-storage/internal/models"
	"go-cloud-storage/internal/repositories"
)

type NotificationService struct {
	notificationRepo *repositories.NotificationRepository
}

func NewNotificationService(notificationRepo *repositories.NotificationRepository) *NotificationService {
	return &NotificationService{
		notificationRepo: notificationRepo,
	}
}

// CreateNotification 创建通知
func (s *NotificationService) CreateNotification(req *models.NotificationCreateRequest) error {
	notification := &models.Notification{
		UserID:  req.UserID,
		Title:   req.Title,
		Message: req.Message,
		Type:    req.Type,
		Link:    req.Link,
		IsRead:  false,
	}

	// 设置默认类型
	if notification.Type == "" {
		notification.Type = "info"
	}

	return s.notificationRepo.Create(notification)
}

// GetUserNotifications 获取用户通知列表
func (s *NotificationService) GetUserNotifications(userID uint, page, pageSize int) (*models.NotificationListResponse, error) {
	offset := (page - 1) * pageSize

	// 获取通知列表
	notifications, err := s.notificationRepo.GetByUserID(userID, pageSize, offset)
	if err != nil {
		return nil, err
	}

	// 获取未读数量
	unreadCount, err := s.notificationRepo.GetUnreadCountByUserID(userID)
	if err != nil {
		return nil, err
	}

	// 获取总数
	total, err := s.notificationRepo.GetTotalCountByUserID(userID)
	if err != nil {
		return nil, err
	}

	// 转换为响应格式
	var notificationResponses []models.NotificationResponse
	for _, notification := range notifications {
		notificationResponses = append(notificationResponses, models.NotificationResponse{
			ID:        notification.ID,
			Title:     notification.Title,
			Message:   notification.Message,
			Type:      notification.Type,
			IsRead:    notification.IsRead,
			Link:      notification.Link,
			CreatedAt: notification.CreatedAt,
		})
	}

	return &models.NotificationListResponse{
		Notifications: notificationResponses,
		UnreadCount:   unreadCount,
		Total:         total,
	}, nil
}

// MarkAsRead 标记通知为已读
func (s *NotificationService) MarkAsRead(id, userID uint) error {
	return s.notificationRepo.MarkAsRead(id, userID)
}

// MarkAllAsRead 标记所有通知为已读
func (s *NotificationService) MarkAllAsRead(userID uint) error {
	return s.notificationRepo.MarkAllAsRead(userID)
}

// DeleteNotification 删除通知
func (s *NotificationService) DeleteNotification(id, userID uint) error {
	return s.notificationRepo.Delete(id, userID)
}

// DeleteAllNotifications 删除所有通知
func (s *NotificationService) DeleteAllNotifications(userID uint) error {
	return s.notificationRepo.DeleteAll(userID)
}

// GetUnreadCount 获取未读通知数量
func (s *NotificationService) GetUnreadCount(userID uint) (int64, error) {
	return s.notificationRepo.GetUnreadCountByUserID(userID)
}

// CreateSystemNotification 创建系统通知（便捷方法）
func (s *NotificationService) CreateSystemNotification(userID uint, title, message string, notificationType string) error {
	req := &models.NotificationCreateRequest{
		UserID:  userID,
		Title:   title,
		Message: message,
		Type:    notificationType,
	}
	return s.CreateNotification(req)
}

// CreateFileShareNotification 创建文件分享通知（便捷方法）
func (s *NotificationService) CreateFileShareNotification(userID uint, fileName, shareLink string) error {
	req := &models.NotificationCreateRequest{
		UserID:  userID,
		Title:   "文件分享成功",
		Message: "您的文件 \"" + fileName + "\" 已成功创建分享链接",
		Type:    "success",
		Link:    "/shared-files",
	}
	return s.CreateNotification(req)
}

// CreateUploadCompleteNotification 创建上传完成通知（便捷方法）
func (s *NotificationService) CreateUploadCompleteNotification(userID uint, fileName string) error {
	req := &models.NotificationCreateRequest{
		UserID:  userID,
		Title:   "文件上传完成",
		Message: "文件 \"" + fileName + "\" 已成功上传",
		Type:    "success",
		Link:    "/my-drive",
	}
	return s.CreateNotification(req)
}
