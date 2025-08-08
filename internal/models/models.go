package models

import (
	"database/sql"
	"time"
)

// User 用户模型
type User struct {
	Id           int       `gorm:"primaryKey;autoIncrement" json:"id" form:"id"`
	Username     string    `gorm:"size:50;unique;not null" json:"username" form:"username"` // 用户名
	Email        string    `gorm:"size:100;unique;not null" json:"email" form:"email"`
	Phone        string    `gorm:"size:20;unique" json:"phone" form:"phone"`
	Password     string    `gorm:"size:100;not null" json:"password" form:"password"`
	Avatar       string    `gorm:"size:255" json:"avatar" form:"avatar"`
	OpenId       string    `gorm:"size:255" json:"openId"`
	RegisterTime time.Time `json:"registerTime" form:"registerTime"`
	RootFolderId string    `json:"rootFolderId" form:"rootFolderId"`
}

// File 文件模型
type File struct {
	Id            string `gorm:"type:varchar(40);primaryKey" json:"id"`  // UUID或OSS标识
	UserId        int    `gorm:"not null" json:"user_id"`                // 用户ID
	Name          string `gorm:"size:255;not null" json:"name"`          // 原始文件名
	Size          int64  `gorm:"not null" json:"size"`                   // 字节大小
	IsDir         bool   `gorm:"default:false;not null" json:"is_dir"`   // 是否为目录
	FileExtension string `gorm:"size:20;not null" json:"file_extension"` // 文件扩展名

	// 存储信息
	OssObjectKey string `gorm:"size:1024;not null" json:"-"`       // OSS对象键（不暴露给前端）
	FileHash     string `gorm:"size:64;not null" json:"file_hash"` // SHA256哈希值

	ParentId  sql.NullString `gorm:"type:varchar(40)" json:"parent_id"` // 父目录ID（指针类型允许NULL）
	IsDeleted bool           `gorm:"default:false" json:"is_deleted"`   // 软删除标志

	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// Favorite 收藏夹模型
type Favorite struct {
	Id        int       `gorm:"primaryKey;autoIncrement" json:"id"`
	UserId    int       `gorm:"not null;index:idx_user_file,unique" json:"user_id"`
	FileId    string    `gorm:"type:varchar(40);not null;index:idx_user_file,unique" json:"file_id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// ShareRecord 文件分享模型
type ShareRecord struct {
	Id         int       `gorm:"primaryKey"`
	OwnerID    int       `gorm:"column:owner_id"`    // 分享者ID（int）
	TargetID   string    `gorm:"column:target_id"`   // 文件/文件夹ID（varchar(40)）
	SharedType string    `gorm:"column:shared_type"` // "FILE" 或 "FOLDER"
	SharedTo   string    `gorm:"column:shared_to"`   // 被分享者标识
	ShareLink  string    `gorm:"column:share_link"`  // 共享链接
	Password   string    `gorm:"column:password"`    // 访问密码
	Permission string    `gorm:"column:permissions"` // 权限集合
	ExpiresAt  time.Time `gorm:"column:expires_at"`  // 过期时间
	CreatedAt  time.Time `gorm:"column:created_at"`
}

// RecycleBin 回收站模型
type RecycleBin struct {
	FileId       string    `gorm:"type:varchar(40);primaryKey" json:"file_id"`
	UserId       int       `gorm:"not null" json:"user_id"`
	OriginalPath string    `gorm:"type:varchar(1000);not null" json:"original_path"`
	DeletedAt    time.Time `gorm:"autoCreateTime" json:"deleted_at"`
	ExpireAt     time.Time `gorm:"->" json:"expire_at"` // 只读字段，由数据库计算
}

// StorageQuota 用户存储配额模型
type StorageQuota struct {
	UserID    int       `gorm:"primaryKey" json:"user_id"`
	Total     int64     `gorm:"type:bigint;default:10737418240" json:"total"` // 默认10GB (10 * 1024^3)
	Used      int64     `gorm:"type:bigint;default:0" json:"used"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
