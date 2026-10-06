package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-cloud-storage/backend/infrastructure/cache"
	"go-cloud-storage/backend/infrastructure/minio"
	"go-cloud-storage/backend/infrastructure/mq"
	"go-cloud-storage/backend/infrastructure/mysql"
	"go-cloud-storage/backend/internal/router"
	"go-cloud-storage/backend/pkg/config"
	"go-cloud-storage/backend/pkg/logger"
	"go-cloud-storage/backend/pkg/utils"
	"log/slog"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("加载配置文件失败", "error", err)
		panic(err)
	}

	logger.Init(cfg.Server.Env)
	utils.InitJWTSecret(cfg.JWT.Secret)

	if err := mysql.InitDB(&cfg.Database); err != nil {
		slog.Error("数据库初始化失败", "error", err)
		panic(err)
	}
	defer mysql.Close()

	if err := cache.InitRedis(&cfg.Redis); err != nil {
		slog.Error("Redis初始化失败", "error", err)
		panic(err)
	}
	defer cache.Close()

	minioService, err := minio.NewMinioService(&cfg.Minio)
	if err != nil {
		slog.Error("MinIO 初始化失败", "error", err)
		panic(err)
	}

	// MQ 不可用时不阻塞启动，回收站清理自动降级为定时扫描。
	rabbitClient, err := mq.NewRabbitMQClient(&cfg.RabbitMQ)
	if err != nil {
		slog.Warn("RabbitMQ 不可用，回收站过期清理将降级为定时扫描", "error", err)
		rabbitClient = nil
	}
	defer func() {
		if rabbitClient != nil {
			if closeErr := rabbitClient.Close(); closeErr != nil {
				slog.Error("RabbitMQ 连接关闭失败", "error", closeErr)
			}
		}
	}()

	// One lifecycle context owns HTTP and all long-running workers. SIGINT/SIGTERM
	// cancels it so Docker/Kubernetes shutdown stops background jobs as well.
	appCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	r := router.SetUpRouter(appCtx, mysql.GormDB, minioService, rabbitClient, cfg)
	port := fmt.Sprintf(":%d", cfg.Server.Port)
	server := &http.Server{Addr: port, Handler: r}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("服务器启动", "port", port, "env", cfg.Server.Env)
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("服务器启动失败", "error", err)
			panic(err)
		}
		return
	case <-appCtx.Done():
		slog.Info("收到退出信号，开始优雅关闭")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP 服务优雅关闭失败", "error", err)
		_ = server.Close()
	}
}
