// Package contracts defines versioned HTTP wire data, never database entities.
package contracts

import (
	"errors"
	"time"
)

// User is the public identity returned by the user RPC. It cannot carry a password hash.
type User struct {
	SessionVersion string    `json:"-"`
	Id             int       `json:"id"`
	Username       string    `json:"username"` // 用户名
	Email          string    `json:"email"`
	Phone          *string   `json:"phone"`
	Avatar         string    `json:"avatar"`
	OpenId         string    `json:"openId"`
	RegisterTime   time.Time `json:"registerTime"`
	RootFolderId   string    `json:"rootFolderId"`
}

// FileMetadata is the versioned file RPC response, independent of the file ORM.
type FileMetadata struct {
	ShareVersion  int64  `json:"shareVersion"`
	Id            string `json:"id"`             // UUID或OSS标识
	UserId        int    `json:"user_id"`        // 用户ID
	Name          string `json:"name"`           // 原始文件名
	Size          int64  `json:"size"`           // 字节大小
	SizeStr       string `json:"size_str"`       // 可读大小，如2.8MB
	IsDir         bool   `json:"is_dir"`         // 是否为目录
	FileExtension string `json:"file_extension"` // 文件扩展名
	FileURL       string `json:"file_url"`
	ThumbnailURL  string `json:"thumbnail_url"`

	// 存储信息
	OssObjectKey string `json:"objectKey"` // OSS对象键（不暴露给前端）
	FileHash     string `json:"file_hash"` // SHA256哈希值

	ParentId  NullableString `json:"parent_id"`  // 父目录ID（指针类型允许NULL）
	IsDeleted bool           `json:"is_deleted"` // 软删除标志

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Identity struct {
	UserID int    `json:"userId"`
	RootID string `json:"rootId"`
}
type UserRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
	Confirm  string `json:"confirm"`
	UserID   int    `json:"userId"`
	Version  string `json:"version"`
	Token    string `json:"token"`
}
type Credentials struct {
	User    User   `json:"user"`
	Version string `json:"version"`
}

type FileRequest struct {
	UserID int      `json:"userId"`
	ID     string   `json:"id"`
	IDs    []string `json:"ids"`
}

var ErrSessionInvalid = errors.New("登录已失效，请重新登录")

// NullableString preserves the existing parent_id JSON wire shape.
type NullableString struct {
	String string
	Valid  bool
}
