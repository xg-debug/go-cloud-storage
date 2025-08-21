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
	Phone        *string   `gorm:"size:20;unique" json:"phone" form:"phone"`
	Password     string    `gorm:"size:100;not null" json:"password" form:"password"`
	Avatar       string    `gorm:"size:255" json:"avatar" form:"avatar"`
	OpenId       string    `gorm:"size:255" json:"openId"`
	RegisterTime time.Time `json:"registerTime" form:"registerTime"`
	RootFolderId string    `json:"rootFolderId" form:"rootFolderId"`
}

// File 文件模型
type File struct {
	Id            string `gorm:"type:varchar(40);primaryKey" json:"id"`                     // UUID或OSS标识
	UserId        int    `gorm:"not null" json:"user_id"`                                   // 用户ID
	Name          string `gorm:"size:255;not null" json:"name"`                             // 原始文件名
	Size          int64  `gorm:"not null" json:"size"`                                      // 字节大小
	SizeStr       string `gorm:"column:size_str;type:varchar(20);not null" json:"size_str"` // 可读大小，如2.8MB
	IsDir         bool   `gorm:"default:false;not null" json:"is_dir"`                      // 是否为目录
	FileExtension string `gorm:"size:20;not null" json:"file_extension"`                    // 文件扩展名
	FileURL       string `gorm:"column:file_url;not null" json:"file_url"`
	ThumbnailURL  string `gorm:"column:thumbnail_url" json:"thumbnail_url"`

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
	UserId    int       `gorm:"not null" json:"user_id"`
	FileId    string    `gorm:"type:varchar(40);not null" json:"file_id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// Share 分享表
type Share struct {
	Id             int        `gorm:"primaryKey;autoIncrement" json:"id"`
	UserId         int        `gorm:"not null" json:"user_id"`                           // 分享者ID
	FileId         string     `gorm:"type:varchar(40);not null" json:"file_id"`          // 分享的文件/文件夹
	ShareToken     string     `gorm:"type:varchar(100);not null" json:"share_token"`     // 分享标识
	ExtractionCode *string    `gorm:"type:varchar(20)" json:"extraction_code,omitempty"` // 提取码，可为空
	ExpireTime     *time.Time `gorm:"type:timestamp" json:"expire_time,omitempty"`       // 过期时间，可为空
	CreatedAt      time.Time  `gorm:"autoCreateTime" json:"created_at"`
}

// RecycleBin 回收站模型
type RecycleBin struct {
	Id        uint64    `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	FileId    string    `gorm:"type:varchar(40);not null;column:file_id" json:"file_id"`
	UserId    int       `gorm:"not null;column:user_id" json:"user_id"`
	DeletedAt time.Time `gorm:"not null;column:deleted_at;default:CURRENT_TIMESTAMP" json:"deleted_at"`
	ExpireAt  time.Time `gorm:"not null;column:expire_at" json:"expire_at"`
}

// StorageQuota 用户存储配额模型
type StorageQuota struct {
	UserID int   `gorm:"primaryKey" json:"user_id"`
	Total  int64 `gorm:"type:bigint;default:10737418240" json:"total"` // 默认10GB (10 * 1024^3)
	Used   int64 `gorm:"type:bigint;default:0" json:"used"`
	//UsedPercent float32   `gorm:"type:float;default:0.00" json:"used_percent"` // 该字段自动计算，不需要
	CreatedAt time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// FileChunk 文件分片模型
type FileChunk struct {
	Id         int64     `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	FileId     string    `gorm:"type:varchar(40);not null;column:file_id" json:"file_id"`
	ChunkIndex int       `gorm:"not null;column:chunk_index" json:"chunk_index"`
	ChunkHash  string    `gorm:"type:varchar(64);not null;column:chunk_hash" json:"chunk_hash"`
	Size       int       `gorm:"not null;column:size" json:"size"`
	OssEtag    string    `gorm:"type:varchar(64);not null;column:oss_etag" json:"oss_etag"`
	UploadId   string    `gorm:"type:varchar(100);not null;column:upload_id" json:"upload_id"` // 对应 OSS 上传任务的 uploadId
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}
