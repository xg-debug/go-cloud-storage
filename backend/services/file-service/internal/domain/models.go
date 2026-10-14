package domain

import (
	"database/sql"
	"time"
)

// File 文件模型
type File struct {
	ShareVersion  int64  `gorm:"not null;default:0" json:"shareVersion"`
	Id            string `gorm:"type:varchar(40);primaryKey" json:"id"`                                                    // UUID或OSS标识
	UserId        int    `gorm:"not null;index:idx_user_hash,priority:1;index:idx_user_deleted,priority:1" json:"user_id"` // 用户ID
	Name          string `gorm:"size:255;not null" json:"name"`                                                            // 原始文件名
	Size          int64  `gorm:"not null" json:"size"`                                                                     // 字节大小
	SizeStr       string `gorm:"column:size_str;type:varchar(20);not null" json:"size_str"`                                // 可读大小，如2.8MB
	IsDir         bool   `gorm:"default:false;not null" json:"is_dir"`                                                     // 是否为目录
	FileExtension string `gorm:"size:20;not null" json:"file_extension"`                                                   // 文件扩展名
	FileURL       string `gorm:"column:file_url;not null" json:"file_url"`
	ThumbnailURL  string `gorm:"column:thumbnail_url" json:"thumbnail_url"`

	// 存储信息
	OssObjectKey string `gorm:"size:1024;not null" json:"-"`                                                          // OSS对象键（不暴露给前端）
	FileHash     string `gorm:"size:64;not null;index:idx_file_hash;index:idx_user_hash,priority:2" json:"file_hash"` // SHA256哈希值

	ParentId  sql.NullString `gorm:"type:varchar(40)" json:"parent_id"`                                 // 父目录ID（指针类型允许NULL）
	IsDeleted bool           `gorm:"default:false;index:idx_user_deleted,priority:2" json:"is_deleted"` // 软删除标志

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// Favorite 收藏夹模型
type Favorite struct {
	Id        int       `gorm:"primaryKey;autoIncrement" json:"id"`
	UserId    int       `gorm:"not null;uniqueIndex:uniq_fav,priority:1" json:"user_id"`
	FileId    string    `gorm:"type:varchar(40);not null;uniqueIndex:uniq_fav,priority:2" json:"file_id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// RecycleBin 回收站模型
type RecycleBin struct {
	Id        uint64    `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	FileId    string    `gorm:"type:varchar(40);not null;column:file_id;uniqueIndex:uniq_user_file,priority:2" json:"file_id"`
	UserId    int       `gorm:"not null;column:user_id;uniqueIndex:uniq_user_file,priority:1" json:"user_id"`
	DeletedAt time.Time `gorm:"not null;column:deleted_at;autoCreateTime" json:"deleted_at"`
	ExpireAt  time.Time `gorm:"not null;column:expire_at" json:"expire_at"`
}

// StorageQuota 用户存储配额模型
type StorageQuota struct {
	UserID int   `gorm:"primaryKey" json:"user_id"`
	Total  int64 `gorm:"type:bigint;default:10737418240" json:"total"` // 默认10GB (10 * 1024^3)
	Used   int64 `gorm:"type:bigint;default:0" json:"used"`
	//UsedPercent float32   `gorm:"type:float;default:0.00" json:"used_percent"` // 该字段自动计算，不需要
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}
