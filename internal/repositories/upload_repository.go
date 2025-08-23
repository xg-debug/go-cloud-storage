package repositories

import (
	"go-cloud-storage/internal/models"

	"gorm.io/gorm"
)

type UploadRepository interface {
	Create(task *models.UploadTask) error
	Update(task *models.UploadTask) error
	GetById(id string) (*models.UploadTask, error)
	FindByUserAndHash(userId int, fileHash string) (*models.UploadTask, error)
	GetIncompleteByUserId(userId int) ([]models.UploadTask, error)
	Delete(id string) error
}

type uploadRepo struct {
	db *gorm.DB
}

func NewUploadRepository(db *gorm.DB) UploadRepository {
	return &uploadRepo{db: db}
}

func (r *uploadRepo) Create(task *models.UploadTask) error {
	return r.db.Create(task).Error
}

func (r *uploadRepo) Update(task *models.UploadTask) error {
	return r.db.Save(task).Error
}

func (r *uploadRepo) GetById(id string) (*models.UploadTask, error) {
	var task models.UploadTask
	err := r.db.First(&task, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	if task.UploadedChunks == nil {
		task.UploadedChunks = []models.UploadedChunk{}
	}
	return &task, nil
}

func (r *uploadRepo) FindByUserAndHash(userId int, fileHash string) (*models.UploadTask, error) {
	var task models.UploadTask
	err := r.db.First(&task, "user_id = ? AND file_hash = ?", userId, fileHash).Error
	if err != nil {
		return nil, err
	}
	if task.UploadedChunks == nil {
		task.UploadedChunks = []models.UploadedChunk{}
	}
	return &task, nil
}

func (r *uploadRepo) GetIncompleteByUserId(userId int) ([]models.UploadTask, error) {
	var tasks []models.UploadTask
	err := r.db.Where("user_id = ? AND status = ?", userId, 0).Find(&tasks).Error
	if err != nil {
		return nil, err
	}

	// 确保每个任务的UploadedChunks不为nil
	for i := range tasks {
		if tasks[i].UploadedChunks == nil {
			tasks[i].UploadedChunks = []models.UploadedChunk{}
		}
	}

	return tasks, nil
}

func (r *uploadRepo) Delete(id string) error {
	return r.db.Delete(&models.UploadTask{}, "id = ?", id).Error
}
