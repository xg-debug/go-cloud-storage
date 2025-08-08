package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"gorm.io/gorm/schema"
	"log"
	"os"
	"time"

	"go-cloud-storage/internal/pkg/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	GormDB *gorm.DB
	SqlDB  *sql.DB
)

func InitDB(cfg *config.DatabaseConfig) error {
	// 构建DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.DBName,
	)

	// 创建GORM日志器
	newLogger := createGormLogger(cfg.LogLevel)

	// 初始化GORM
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
		Logger: newLogger,
	})
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}

	// 获取底层SQL连接
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取底层sql连接[sql.DB]失败: %w", err)
	}

	// 配置连接池
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// 赋值给全局变量
	GormDB = db
	SqlDB = sqlDB

	// 测试数据库连接
	if err := testConnection(); err != nil {
		return fmt.Errorf("数据库连接测试失败: %w", err)
	}

	log.Println("数据库初始化成功!")
	return nil
}

func createGormLogger(level string) logger.Interface {
	logLevel := logger.Silent
	switch level {
	case "error":
		logLevel = logger.Error
	case "warn":
		logLevel = logger.Warn
	case "info":
		logLevel = logger.Info
	}

	return logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)
}

func testConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := SqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("mysql ping failed: %w", err)
	}
	return nil
}

func Close() {
	if SqlDB != nil {
		if err := SqlDB.Close(); err != nil {
			log.Printf("关闭MySQL连接发生错误: %v", err)
		} else {
			log.Println("MySQL连接已关闭!")
		}
	}
}
