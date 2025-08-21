package models

import "time"

// 上传任务表
type UploadTask struct {
	Id         string    `gorm:"primaryKey;type:varchar(64)" json:"id"`
	UserId     int       `json:"user_id"`
	FileName   string    `json:"file_name"`
	FileSize   int64     `json:"file_size"`
	FileHash   string    `json:"file_hash"`
	ChunkCount int       `json:"chunk_count"`
	UploadId   string    `json:"upload_id"` // OSS 返回的 uploadId
	Status     int       `json:"status"`    // 0:进行中 1:完成 2:失败
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// 分片表
type FileChunk struct {
	Id         int       `gorm:"primaryKey" json:"id"`
	TaskId     string    `gorm:"index;type:varchar(64)" json:"task_id"`
	ChunkIndex int       `json:"chunk_index"`
	Size       int64     `json:"size"`
	ETag       string    `json:"etag"`
	Status     int       `json:"status"` // 0:未上传 1:已上传
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
