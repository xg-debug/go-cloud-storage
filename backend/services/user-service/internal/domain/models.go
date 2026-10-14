package domain

import (
	"time"
)

// User 用户模型
type User struct {
	SessionVersion string    `gorm:"-" json:"-"`
	Id             int       `gorm:"primaryKey;autoIncrement" json:"id" form:"id"`
	Username       string    `gorm:"size:50;unique;not null" json:"username" form:"username"` // 用户名
	Email          string    `gorm:"size:100;unique;not null" json:"email" form:"email"`
	Phone          *string   `gorm:"size:20;unique" json:"phone" form:"phone"`
	Password       string    `gorm:"size:100;not null" json:"password" form:"password"`
	Avatar         string    `gorm:"size:255" json:"avatar" form:"avatar"`
	OpenId         string    `gorm:"size:255" json:"openId"`
	RegisterTime   time.Time `json:"registerTime" form:"registerTime"`
	RootFolderId   string    `json:"rootFolderId" form:"rootFolderId"`
}

// PasswordResetToken 密码重置令牌
type PasswordResetToken struct {
	Id        int       `gorm:"primaryKey;autoIncrement" json:"id"`
	UserId    int       `gorm:"not null" json:"user_id"`
	Token     string    `gorm:"size:128;unique;not null" json:"token"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	Used      bool      `gorm:"default:false" json:"used"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}
