package repositories

import (
	"go-cloud-storage/internal/models"
	"gorm.io/gorm"
)

type UploadRepository interface {
	CreateTask(task *models.UploadTask) error
	GetTask(taskId string) (*models.UploadTask, error)
	UpdateTaskStatus(taskId string, status int) error

	CreateChunks(chunks []models.FileChunk) error
	UpdateChunk(taskId string, index int, etag string, size int64) error
	GetChunks(taskId string) ([]models.FileChunk, error)
}

type uploadRepo struct {
	db *gorm.DB
}

func NewUploadRepository(db *gorm.DB) UploadRepository {
	return &uploadRepo{db: db}
}

func (r *uploadRepo) CreateTask(task *models.UploadTask) error {
	return r.db.Create(task).Error
}

func (r *uploadRepo) GetTask(taskId string) (*models.UploadTask, error) {
	var task *models.UploadTask
	err := r.db.Where("id = ?", taskId).First(&task).Error
	return task, err
}

func (r *uploadRepo) UpdateTaskStatus(taskId string, status int) error {
	return r.db.Model(&models.UploadTask{}).Where("id = ?", taskId).Update("status", status).Error
}

func (r *uploadRepo) CreateChunks(chunks []models.FileChunk) error {
	return r.db.Create(&chunks).Error
}

func (r *uploadRepo) UpdateChunk(taskId string, index int, etag string, size int64) error {
	return r.db.Model(&models.FileChunk{}).
		Where("task_id = ? AND chunk_index = ?", taskId, index).
		Updates(map[string]interface{}{"etag": etag, "size": size, "status": 1}).Error
}

func (r *uploadRepo) GetChunks(taskId string) ([]models.FileChunk, error) {
	var chunks []models.FileChunk
	err := r.db.Where("task_id = ?", taskId).Find(&chunks).Error
	return chunks, err
}
