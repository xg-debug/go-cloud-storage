package repositories

import (
	"go-cloud-storage/internal/models"
	"gorm.io/gorm"
	"time"
)

// RecycleRepository 1.接口定义：在go中，接口本身就是引用类型，返回接口值已经包含了指向具体实现的指针
type RecycleRepository interface {
	AddToRecycle(record *models.RecycleBin) error
	GetByUser(userId int) ([]models.RecycleBin, error)
	GetByFileId(fileId string) (*models.RecycleBin, error)
	PermanentDelete(fileId string) error
	CleanExpiredRecords() (int64, error)
	MarkAsRestored(fileId string) error
}

/*
recycleRepo 结构体是私有的（小写开头），这意味着它只能在当前包内被使用，外部包只能通过接口来访问其功能。
这样避免了外部直接操作数据库对象，保证了数据访问的安全性。
*/
// recycleRepo 2.实现结构体
type recycleRepo struct {
	db *gorm.DB
}

// NewRecycleRepository 3.构造函数 所以这里返回值类型不是 *RecycleRepository
func NewRecycleRepository(db *gorm.DB) RecycleRepository {
	return &recycleRepo{db: db}
}

func (r *recycleRepo) AddToRecycle(record *models.RecycleBin) error {
	return r.db.Create(record).Error
}

func (r *recycleRepo) GetByUser(userId int) ([]models.RecycleBin, error) {
	var records []models.RecycleBin
	err := r.db.Where("user_id = ?", userId).Order("deleted_at DESC").Find(&records).Error
	return records, err
}

func (r *recycleRepo) GetByFileId(fileId string) (*models.RecycleBin, error) {
	var record models.RecycleBin
	err := r.db.First(&record, "file_id = ?", fileId).Error
	return &record, err
}

func (r *recycleRepo) PermanentDelete(fileId string) error {
	return r.db.Where("file_id = ?", fileId).Delete(&models.RecycleBin{}).Error
}

func (r *recycleRepo) CleanExpiredRecords() (int64, error) {
	result := r.db.Where("expire_at < ?", time.Now()).Delete(&models.RecycleBin{})
	return result.RowsAffected, result.Error
}

func (r *recycleRepo) MarkAsRestored(fileId string) error {
	return r.db.Where("file_id = ?", fileId).Update("restored_at", time.Now()).Error
}
