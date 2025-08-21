package repositories

import (
	"go-cloud-storage/internal/models"

	"gorm.io/gorm"
)

type FileChunkRepository interface {
	CreateChunk(chunk *models.FileChunk) error
	GetChunksByFileId(fileId string) ([]models.FileChunk, error)
	GetChunkByFileIdAndIndex(fileId string, chunkIndex int) (*models.FileChunk, error)
	DeleteChunksByFileId(fileId string) error
	CheckChunkExists(fileId string, chunkIndex int) (bool, error)
	GetUploadedChunks(fileId string) ([]int, error)
}

type fileChunkRepo struct {
	db *gorm.DB
}

func NewFileChunkRepository(db *gorm.DB) FileChunkRepository {
	return &fileChunkRepo{db: db}
}

// CreateChunk 创建分片记录
func (r *fileChunkRepo) CreateChunk(chunk *models.FileChunk) error {
	return r.db.Create(chunk).Error
}

// GetChunksByFileId 获取文件的所有分片
func (r *fileChunkRepo) GetChunksByFileId(fileId string) ([]models.FileChunk, error) {
	var chunks []models.FileChunk
	err := r.db.Where("file_id = ?", fileId).Order("chunk_index").Find(&chunks).Error
	return chunks, err
}

// GetChunkByFileIdAndIndex 获取指定文件的指定分片
func (r *fileChunkRepo) GetChunkByFileIdAndIndex(fileId string, chunkIndex int) (*models.FileChunk, error) {
	var chunk models.FileChunk
	err := r.db.Where("file_id = ? AND chunk_index = ?", fileId, chunkIndex).First(&chunk).Error
	if err != nil {
		return nil, err
	}
	return &chunk, nil
}

// DeleteChunksByFileId 删除文件的所有分片记录
func (r *fileChunkRepo) DeleteChunksByFileId(fileId string) error {
	return r.db.Where("file_id = ?", fileId).Delete(&models.FileChunk{}).Error
}

// CheckChunkExists 检查分片是否存在
func (r *fileChunkRepo) CheckChunkExists(fileId string, chunkIndex int) (bool, error) {
	var count int64
	err := r.db.Model(&models.FileChunk{}).Where("file_id = ? AND chunk_index = ?", fileId, chunkIndex).Count(&count).Error
	return count > 0, err
}

// GetUploadedChunks 获取已上传的分片索引列表
func (r *fileChunkRepo) GetUploadedChunks(fileId string) ([]int, error) {
	var chunks []models.FileChunk
	err := r.db.Select("chunk_index").Where("file_id = ?", fileId).Order("chunk_index").Find(&chunks).Error
	if err != nil {
		return nil, err
	}

	var indexes []int
	for _, chunk := range chunks {
		indexes = append(indexes, chunk.ChunkIndex)
	}
	return indexes, nil
}
