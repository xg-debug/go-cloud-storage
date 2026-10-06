package services

import "go-cloud-storage/backend/internal/models"

// InitChunkUploadInput is the application-layer command for starting or resuming
// a multipart upload. HTTP/Gin request binding should be converted to this type
// by the controller instead of leaking gin.Context/gin.H into service APIs.
type InitChunkUploadInput struct {
	UserID      int
	FileName    string
	FileHash    string
	ParentID    string
	FileSize    int64
	ChunkSize   int64
	TotalChunks int
}

// InitChunkUploadResult is transport-neutral and can be returned by HTTP today
// and by gRPC later without changing the core upload logic.
type InitChunkUploadResult struct {
	Finished       bool         `json:"finished"`
	File           *models.File `json:"file,omitempty"`
	URL            string       `json:"url,omitempty"`
	FileHash       string       `json:"fileHash,omitempty"`
	UploadID       string       `json:"uploadId,omitempty"`
	UploadedChunks []int        `json:"uploadedChunks,omitempty"`
	ChunkSize      int64        `json:"chunkSize,omitempty"`
	TotalChunks    int          `json:"totalChunks,omitempty"`
}

// ChunkUploadProgress is the typed representation of resumable upload state.
type ChunkUploadProgress struct {
	Status         string `json:"status"`
	UploadID       string `json:"uploadId,omitempty"`
	UploadedChunks []int  `json:"uploadedChunks"`
	UploadedCount  int    `json:"uploadedCount,omitempty"`
}
