package migrations

import (
	"go-cloud-storage/internal/models"

	"gorm.io/gorm"
)

// AutoMigrate 执行数据库迁移
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.File{},
		&models.Favorite{},
		&models.Share{},
		&models.RecycleBin{},
		&models.StorageQuota{},
		&models.Notification{}, // 添加通知模型
	)
}
