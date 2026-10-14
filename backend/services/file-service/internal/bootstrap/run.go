package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"gorm.io/gorm"

	"go-cloud-storage/backend/pkg/config"
	"go-cloud-storage/backend/pkg/logger"
	cache "go-cloud-storage/backend/pkg/platform/cache"
	eventing "go-cloud-storage/backend/pkg/platform/eventing"
	mysql "go-cloud-storage/backend/pkg/platform/mysql"
	ports "go-cloud-storage/backend/pkg/ports"
	rpc "go-cloud-storage/backend/pkg/rpc"
	"go-cloud-storage/backend/pkg/server"
	storagehttp "go-cloud-storage/backend/pkg/storagehttp"
	migrations "go-cloud-storage/backend/services/file-service/internal/adapters/migrations"
	domain "go-cloud-storage/backend/services/file-service/internal/domain"
)

func Main() {
	if err := Run(); err != nil {
		slog.Error("service stopped", "service", "file", "error", err)
		os.Exit(1)
	}
}
func Run() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	logger.Init(cfg.Server.Env)
	token := os.Getenv("GCS_SERVICE_TOKEN")
	if len(token) < 32 {
		return fmt.Errorf("GCS_SERVICE_TOKEN requires at least 32 bytes")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var db *gorm.DB

	if err = mysql.InitDB(&cfg.Database); err != nil {
		return err
	}
	defer mysql.Close()
	db = mysql.GormDB
	if os.Getenv("GCS_AUTO_MIGRATE") == "true" {
		if err = migrate(db); err != nil {
			return err
		}
	}

	if !cfg.Redis.Enabled {
		return fmt.Errorf("Redis required")
	}
	if err = cache.InitRedis(&cfg.Redis); err != nil {
		return err
	}
	defer cache.Close()

	if cfg.RabbitMQ.URL == "" {
		return fmt.Errorf("RabbitMQ URL required")
	}

	var storage ports.Storage

	storage, err = storagehttp.NewClient(cfg.StorageService.URL, cfg.StorageService.Token)
	if err != nil {
		return err
	}

	clients := ServiceClients{Token: token}
	clients.Auth, err = rpc.NewClient(os.Getenv("GCS_AUTH_URL"), token)
	if err != nil {
		return err
	}
	clients.Share, err = rpc.NewClient(os.Getenv("GCS_SHARE_URL"), token)
	if err != nil {
		return err
	}
	handler, err := setUp(ctx, db, storage, cfg, clients)
	if err != nil {
		return err
	}
	return server.Run(ctx, fmt.Sprintf(":%d", cfg.Server.Port), handler)
}
func migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&domain.File{}, &domain.StorageQuota{}, &domain.Favorite{}, &domain.RecycleBin{}, &domain.ObjectDeletion{}, &eventing.Outbox{}); err != nil {
		return err
	}
	return migrations.EnsureFileConstraints(db)
}
