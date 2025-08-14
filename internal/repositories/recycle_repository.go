package repositories

import (
	"go-cloud-storage/internal/models"
	"gorm.io/gorm"
	"time"
)

type TrashItem struct {
	FileId    string
	Name      string
	IsDir     bool
	SizeStr   string
	DeletedAt time.Time
	ExpireAt  time.Time
}

// RecycleRepository 1.接口定义：在go中，接口本身就是引用类型，返回接口值已经包含了指向具体实现的指针
type RecycleRepository interface {
	GetFiles(userId int) ([]TrashItem, error)
	DeleteOne(db *gorm.DB, fileId string) error
	DeleteBatch(db *gorm.DB, fileIds []string) error
	DeleteAll(db *gorm.DB, userId int) error
	RestoreOne(fileId string) error
	RestoreBatch(fileIds []string) error
	RestoreAll(userId int) error

	CleanExpiredRecords() (int64, error)
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

func (r *recycleRepo) GetFiles(userId int) ([]TrashItem, error) {
	var res []TrashItem
	err := r.db.Table("recycle_bin AS rb").
		Select(`f.id AS file_id, f.name, f.is_dir, f.size_str, rb.deleted_at, rb.expire_at`).
		Joins(`LEFT JOIN file AS f ON rb.file_id = f.id`).
		Where(`rb.user_id = ?`, userId).Scan(&res).Error
	return res, err
}

func (r *recycleRepo) DeleteOne(db *gorm.DB, fileId string) error {
	return db.Where("file_id = ?", fileId).Delete(&models.RecycleBin{}).Error
}

func (r *recycleRepo) DeleteBatch(db *gorm.DB, fileIds []string) error {
	return db.Where("file_id IN ?", fileIds).Delete(&models.RecycleBin{}).Error
}

func (r *recycleRepo) DeleteAll(db *gorm.DB, userId int) error {
	return db.Where("user_id = ?", userId).Delete(&models.RecycleBin{}).Error
}

func (r *recycleRepo) RestoreOne(fileId string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1.删除回收站的记录
		if err := tx.Where("file_id = ?", fileId).Delete(&models.RecycleBin{}).Error; err != nil {
			return err
		}
		// 2.修改file表中软删除的标志
		if err := tx.Model(&models.File{}).Where("file_id = ?", fileId).Update("is_deleted", false).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *recycleRepo) RestoreBatch(fileIds []string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1.删除回收站的记录
		if err := tx.Where("file_id IN ?", fileIds).Delete(&models.RecycleBin{}).Error; err != nil {
			return err
		}
		// 2.修改file表中软删除的标志
		if err := tx.Model(&models.File{}).Where("file_id IN ?", fileIds).Updates(map[string]interface{}{"is_deleted": false}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *recycleRepo) RestoreAll(userId int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1.删除回收站的记录
		if err := tx.Where("user_id = ?", userId).Delete(&models.RecycleBin{}).Error; err != nil {
			return err
		}
		// 2.修改file表中软删除的标志
		if err := tx.Model(&models.File{}).Where("user_id = ?", userId).Updates(map[string]interface{}{"is_deleted": false}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *recycleRepo) CleanExpiredRecords() (int64, error) {
	result := r.db.Where("expire_at < ?", time.Now()).Delete(&models.RecycleBin{})
	return result.RowsAffected, result.Error
}
