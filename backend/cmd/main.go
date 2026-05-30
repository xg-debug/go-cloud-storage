package main

import (
	"fmt"
	"go-cloud-storage/backend/infrastructure/cache"
	"go-cloud-storage/backend/pkg/config"
	"go-cloud-storage/backend/pkg/logger"
	"go-cloud-storage/backend/pkg/utils"
	"go-cloud-storage/backend/infrastructure/minio"
	"go-cloud-storage/backend/infrastructure/mq"
	"go-cloud-storage/backend/infrastructure/mysql"
	"go-cloud-storage/backend/internal/router"
	"log/slog"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("加载配置文件失败", "error", err)
		panic(err)
	}

	// 初始化结构化日志
	logger.Init(cfg.Server.Env)

	// 初始化 JWT 密钥
	utils.InitJWTSecret(cfg.JWT.Secret)

	// 初始化数据库
	if err := mysql.InitDB(&cfg.Database); err != nil {
		slog.Error("数据库初始化失败", "error", err)
		panic(err)
	}
	defer mysql.Close()

	// 初始化Redis
	if err := cache.InitRedis(&cfg.Redis); err != nil {
		slog.Error("Redis初始化失败", "error", err)
		panic(err)
	}
	defer cache.Close()

	// 初始化 Minio
	minioService, err := minio.NewMinioService(&cfg.Minio)
	if err != nil {
		slog.Error("MinIO 初始化失败", "error", err)
		panic(err)
	}

	// 初始化 RabbitMQ 客户端
	rabbitClient, err := mq.NewRabbitMQClient(&cfg.RabbitMQ)
	if err != nil {
		slog.Error("RabbitMQ 初始化失败", "error", err)
		panic(err)
	}
	defer func() {
		if rabbitClient != nil {
			if closeErr := rabbitClient.Close(); closeErr != nil {
				slog.Error("RabbitMQ 连接关闭失败", "error", closeErr)
			}
		}
	}()

	r := router.SetUpRouter(mysql.GormDB, minioService, rabbitClient, cfg)

	port := fmt.Sprintf(":%d", cfg.Server.Port)
	slog.Info("服务器启动", "port", port, "env", cfg.Server.Env)
	if err := r.Run(port); err != nil {
		slog.Error("服务器启动失败", "error", err)
		panic(err)
	}
}
