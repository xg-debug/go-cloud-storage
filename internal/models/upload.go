package models

import "time"

type UploadedChunk struct {
	Index int    `json:"index"`
	ETag  string `json:"etag"`
}

type UploadTask struct {
	Id             string          `gorm:"primaryKey;type:varchar(64)" json:"id"`
	UserId         int             `json:"user_id"`
	FileName       string          `json:"file_name"`
	FileSize       int64           `json:"file_size"`
	FileHash       string          `json:"file_hash"`
	ChunkSize      int64           `json:"chunk_size"`
	ChunkCount     int             `json:"chunk_count"`
	UploadedChunks []UploadedChunk `gorm:"type:json" json:"uploaded_chunks"`
	UploadId       string          `json:"upload_id"`
	ObjectKey      string          `json:"object_key"`
	Status         int             `json:"status"` // 0:进行中 1:完成 2:失败
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}
