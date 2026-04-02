package models

type UploadedChunk struct {
	Index int    `json:"index"`
	ETag  string `json:"etag"`
}
