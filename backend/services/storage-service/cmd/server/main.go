package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"go-cloud-storage/backend/infrastructure/minio"
	"go-cloud-storage/backend/infrastructure/storage"
	"go-cloud-storage/backend/pkg/config"
	"go-cloud-storage/backend/pkg/server"
	"go-cloud-storage/backend/services/storage-service/internal/transport"
)

func main() {
	if err := run(); err != nil {
		slog.Error("storage-service stopped", "error", err)
		os.Exit(1)
	}
}
func run() error {
	cfg := config.MinioConfig{
		Endpoint: os.Getenv("GCS_MINIO_ENDPOINT"), PublicEndpoint: os.Getenv("GCS_MINIO_PUBLIC_ENDPOINT"),
		AccessKeyID: os.Getenv("GCS_MINIO_ACCESS_KEY"), SecretAccessKey: os.Getenv("GCS_MINIO_SECRET_KEY"),
		Bucket: os.Getenv("GCS_MINIO_BUCKET"), Region: os.Getenv("GCS_MINIO_REGION"), UseSSL: os.Getenv("GCS_MINIO_SSL") == "true",
	}
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	token := os.Getenv("GCS_STORAGE_TOKEN")
	if len(token) < 32 {
		return errors.New("GCS_STORAGE_TOKEN must contain at least 32 bytes")
	}
	limit := int64(10 << 30)
	if value := os.Getenv("GCS_STORAGE_MAX_BODY_BYTES"); value != "" {
		var err error
		limit, err = strconv.ParseInt(value, 10, 64)
		if err != nil || limit <= 0 {
			return errors.New("invalid GCS_STORAGE_MAX_BODY_BYTES")
		}
	}
	service, err := minio.NewMinioService(&cfg)
	if err != nil {
		return err
	}
	handler, err := transport.NewHandler(storage.NewMinIOStorageV2(service), token, limit)
	if err != nil {
		return err
	}
	addr := os.Getenv("GCS_STORAGE_LISTEN")
	if addr == "" {
		addr = ":8082"
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	slog.Info("storage-service listening", "addr", addr)
	return server.Run(ctx, addr, handler)
}
