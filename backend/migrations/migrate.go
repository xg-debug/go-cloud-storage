package migrations

import (
	"go-cloud-storage/backend/internal/models"

	"gorm.io/gorm"
)

// AutoMigrate 执行数据库迁移
func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&models.ObjectDeletion{},
		&models.User{},
		&models.File{},
		&models.Favorite{},
		&models.Share{},
		&models.RecycleBin{},
		&models.StorageQuota{},
		&models.Notification{},       // 添加通知模型
		&models.PasswordResetToken{}, // 密码重置令牌（原遗漏）
	); err != nil {
		return err
	}
	return EnsureFileConstraints(db)
}
