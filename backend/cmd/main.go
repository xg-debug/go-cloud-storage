package main

import (
	"fmt"
	"go-cloud-storage/backend/infrastructure/cache"
	"go-cloud-storage/backend/pkg/config"
	"go-cloud-storage/backend/infrastructure/minio"
	"go-cloud-storage/backend/infrastructure/mq"
	"go-cloud-storage/backend/infrastructure/mysql"
	"go-cloud-storage/backend/internal/router"
	"log"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("加载配置文件时出现错误：%v", err)
	}
	// 初始化数据库
	if err := mysql.InitDB(&cfg.Database); err != nil {
		log.Fatalf("数据库初始化失败：%v", err)
	}
	// defer确保程序退出时自动关闭数据库连接
	defer mysql.Close()

	// 初始化Redis
	if err := cache.InitRedis(&cfg.Redis); err != nil {
		log.Fatalf("Redis初始化失败: %v", err)
	}
	defer cache.Close() // 确保程序退出时关闭Redis连接

	// 初始化 Minio
	minioService, err := minio.NewMinioService(&cfg.Minio)
	if err != nil {
		log.Fatalf("MinIO 初始化失败: %v", err)
	}

	rabbitClient, err := mq.NewRabbitMQClient(&cfg.RabbitMQ)
	if err != nil {
		log.Fatalf("RabbitMQ 初始化失败: %v", err)
	}
	defer func() {
		if rabbitClient != nil {
			_ = rabbitClient.Close()
		}
	}()

	// 初始化其他组件（Redis、HTTP服务器等）

	r := router.SetUpRouter(mysql.GormDB, minioService, rabbitClient, &cfg.RabbitMQ)

	port := fmt.Sprintf(":%d", cfg.Server.Port)
	if err := r.Run(port); err != nil {
		log.Fatal("服务器启动失败...")
	}
}
