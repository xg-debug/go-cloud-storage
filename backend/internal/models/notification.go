package models

import (
	"time"
)

// Notification 通知模型
type Notification struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"not null;index"`
	Title     string    `json:"title" gorm:"size:255;not null"`
	Message   string    `json:"message" gorm:"type:text;not null"`
	Type      string    `json:"type" gorm:"size:50;not null;default:'info'"` // info, success, warning, error
	IsRead    bool      `json:"is_read" gorm:"default:false"`
	Link      string    `json:"link" gorm:"size:500"`                        // 可选的跳转链接
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NotificationCreateRequest 创建通知请求
type NotificationCreateRequest struct {
	UserID  uint   `json:"user_id" binding:"required"`
	Title   string `json:"title" binding:"required,max=255"`
	Message string `json:"message" binding:"required"`
	Type    string `json:"type" binding:"omitempty,oneof=info success warning error"`
	Link    string `json:"link" binding:"omitempty,max=500"`
}

// NotificationResponse 通知响应
type NotificationResponse struct {
	ID        uint      `json:"id"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Type      string    `json:"type"`
	IsRead    bool      `json:"is_read"`
	Link      string    `json:"link"`
	CreatedAt time.Time `json:"created_at"`
}

// NotificationListResponse 通知列表响应
type NotificationListResponse struct {
	Notifications []NotificationResponse `json:"notifications"`
	UnreadCount   int64                  `json:"unread_count"`
	Total         int64                  `json:"total"`
}