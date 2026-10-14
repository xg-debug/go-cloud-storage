package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"go-cloud-storage/backend/pkg/config"
	"go-cloud-storage/backend/pkg/logger"
	cache "go-cloud-storage/backend/pkg/platform/cache"
	rpc "go-cloud-storage/backend/pkg/rpc"
	"go-cloud-storage/backend/pkg/server"
	"go-cloud-storage/backend/pkg/utils"
)

func Main() {
	if err := Run(); err != nil {
		slog.Error("service stopped", "service", "auth", "error", err)
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

	if len(cfg.JWT.Secret) < 32 {
		return fmt.Errorf("identity services require a stable JWT secret")
	}
	utils.InitJWTSecret(cfg.JWT.Secret)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if !cfg.Redis.Enabled {
		return fmt.Errorf("Redis required")
	}
	if err = cache.InitRedis(&cfg.Redis); err != nil {
		return err
	}
	defer cache.Close()

	clients := ServiceClients{Token: token}
	clients.User, err = rpc.NewClient(os.Getenv("GCS_USER_URL"), token)
	if err != nil {
		return err
	}
	handler, err := setUp(ctx, cfg, clients)
	if err != nil {
		return err
	}
	return server.Run(ctx, fmt.Sprintf(":%d", cfg.Server.Port), handler)
}
