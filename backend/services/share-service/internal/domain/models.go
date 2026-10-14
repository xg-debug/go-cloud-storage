package domain

import (
	"time"
)

// Share 分享表
type Share struct {
	FileVersion    int64   `gorm:"not null;default:0" json:"-"`
	Id             int     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserId         int     `gorm:"not null" json:"user_id"`                                            // 分享者ID
	FileId         string  `gorm:"type:varchar(40);not null;uniqueIndex:uk_share_file" json:"file_id"` // 分享的文件/文件夹
	ShareToken     string  `gorm:"type:varchar(50);unique;not null" json:"share_token"`                // 分享标识
	ExtractionCode *string `gorm:"type:varchar(20)" json:"extraction_code,omitempty"`                  // 提取码，可为空
	AccessCount    int     `gorm:"default:0" json:"access_count"`
	DownloadCount  int     `gorm:"default:0" json:"download_count"`
	IsDeleted      int     `gorm:"type:tinyint(1);default:0" json:"is_deleted"`

	ExpireTime *time.Time `gorm:"type:datetime" json:"expire_time,omitempty"` // 过期时间，可为空
	CreatedAt  time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (s *Share) GetExtractionCode() string {
	if s.ExtractionCode == nil {
		return ""
	}
	return *s.ExtractionCode
}
