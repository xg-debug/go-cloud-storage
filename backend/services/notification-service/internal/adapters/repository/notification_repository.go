package repository

import (
	"gorm.io/gorm"

	domain "go-cloud-storage/backend/services/notification-service/internal/domain"
)

type NotificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// Create 创建通知
func (r *NotificationRepository) Create(notification *domain.Notification) error {
	return r.db.Create(notification).Error
}

// GetByUserID 获取用户的通知列表
func (r *NotificationRepository) GetByUserID(userID uint, limit, offset int) ([]domain.Notification, error) {
	var notifications []domain.Notification
	err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&notifications).Error
	return notifications, err
}

// GetUnreadCountByUserID 获取用户未读通知数量
func (r *NotificationRepository) GetUnreadCountByUserID(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&domain.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&count).Error
	return count, err
}

// GetTotalCountByUserID 获取用户通知总数
func (r *NotificationRepository) GetTotalCountByUserID(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&domain.Notification{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count, err
}

// MarkAsRead 标记通知为已读
func (r *NotificationRepository) MarkAsRead(id, userID uint) error {
	return r.db.Model(&domain.Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("is_read", true).Error
}

// MarkAllAsRead 标记用户所有通知为已读
func (r *NotificationRepository) MarkAllAsRead(userID uint) error {
	return r.db.Model(&domain.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Update("is_read", true).Error
}

// Delete 删除通知
func (r *NotificationRepository) Delete(id, userID uint) error {
	return r.db.Where("id = ? AND user_id = ?", id, userID).
		Delete(&domain.Notification{}).Error
}

// DeleteAll 删除用户所有通知
func (r *NotificationRepository) DeleteAll(userID uint) error {
	return r.db.Where("user_id = ?", userID).
		Delete(&domain.Notification{}).Error
}

// GetByID 根据ID获取通知
func (r *NotificationRepository) GetByID(id, userID uint) (*domain.Notification, error) {
	var notification domain.Notification
	err := r.db.Where("id = ? AND user_id = ?", id, userID).
		First(&notification).Error
	if err != nil {
		return nil, err
	}
	return &notification, nil
}
