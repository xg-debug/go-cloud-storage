package runtime

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"go-cloud-storage/backend/infrastructure/cache"
	"go-cloud-storage/backend/infrastructure/eventing"
	"go-cloud-storage/backend/infrastructure/mysql"
	"go-cloud-storage/backend/infrastructure/storagehttp"
	"go-cloud-storage/backend/internal/distributed"
	"go-cloud-storage/backend/internal/models"
	"go-cloud-storage/backend/internal/ports"
	"go-cloud-storage/backend/internal/router"
	"go-cloud-storage/backend/migrations"
	"go-cloud-storage/backend/pkg/config"
	"go-cloud-storage/backend/pkg/logger"
	"go-cloud-storage/backend/pkg/server"
	"go-cloud-storage/backend/pkg/utils"
	"gorm.io/gorm"
)

func Main(role string) {
	if err := Run(role); err != nil {
		slog.Error("service stopped", "service", role, "error", err)
		os.Exit(1)
	}
}
func Run(role string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	logger.Init(cfg.Server.Env)
	token := os.Getenv("GCS_SERVICE_TOKEN")
	if len(token) < 32 {
		return fmt.Errorf("GCS_SERVICE_TOKEN requires at least 32 bytes")
	}
	if role == "auth" || role == "user" {
		if len(cfg.JWT.Secret) < 32 {
			return fmt.Errorf("identity services require a stable JWT secret")
		}
		utils.InitJWTSecret(cfg.JWT.Secret)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var db *gorm.DB
	if role != "auth" {
		if err = mysql.InitDB(&cfg.Database); err != nil {
			return err
		}
		defer mysql.Close()
		db = mysql.GormDB
		if os.Getenv("GCS_AUTO_MIGRATE") == "true" {
			if err = Migrate(db, role); err != nil {
				return err
			}
		}
	}
	if role == "auth" || role == "file" || role == "notification" {
		if !cfg.Redis.Enabled {
			return fmt.Errorf("%s requires Redis", role)
		}
		if err = cache.InitRedis(&cfg.Redis); err != nil {
			return err
		}
		defer cache.Close()
	}
	if role == "file" || role == "share" || role == "notification" {
		if cfg.RabbitMQ.URL == "" {
			return fmt.Errorf("RabbitMQ URL required")
		}
	}
	var storage ports.Storage
	if role == "user" || role == "file" || role == "share" {
		storage, err = storagehttp.NewClient(cfg.StorageService.URL, cfg.StorageService.Token)
		if err != nil {
			return err
		}
	}
	clients := router.ServiceClients{Token: token}
	endpoints := map[string]**distributed.Client{}
	switch role {
	case "auth":
		endpoints["GCS_USER_URL"] = &clients.User
	case "user", "notification":
		endpoints["GCS_AUTH_URL"] = &clients.Auth
	case "file":
		endpoints["GCS_AUTH_URL"] = &clients.Auth
		endpoints["GCS_SHARE_URL"] = &clients.Share
	case "share":
		endpoints["GCS_AUTH_URL"] = &clients.Auth
		endpoints["GCS_FILE_URL"] = &clients.File
	default:
		return fmt.Errorf("unknown role %s", role)
	}
	for key, target := range endpoints {
		*target, err = distributed.NewClient(os.Getenv(key), token)
		if err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
	}
	handler, err := router.SetUpService(ctx, role, db, storage, cfg, clients)
	if err != nil {
		return err
	}
	return server.Run(ctx, fmt.Sprintf(":%d", cfg.Server.Port), handler)
}

// Migrate is opt-in and creates only tables owned by this service. Production
// deployments should run reviewed SQL migrations with a separate DDL account.
func Migrate(db *gorm.DB, role string) error {
	switch role {
	case "user":
		return db.AutoMigrate(&models.User{}, &models.PasswordResetToken{})
	case "file":
		if err := db.AutoMigrate(&models.File{}, &models.StorageQuota{}, &models.Favorite{}, &models.RecycleBin{}, &models.ObjectDeletion{}, &eventing.Outbox{}); err != nil {
			return err
		}
		return migrations.EnsureFileConstraints(db)
	case "share":
		return db.AutoMigrate(&models.Share{}, &eventing.Outbox{}, &eventing.Inbox{})
	case "notification":
		return db.AutoMigrate(&models.Notification{}, &eventing.Inbox{})
	}
	return fmt.Errorf("unknown schema owner %s", role)
}
