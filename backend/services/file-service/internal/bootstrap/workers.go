package bootstrap

import (
	"context"
	"log/slog"
	"time"

	"go-cloud-storage/backend/pkg/config"
	mq "go-cloud-storage/backend/services/file-service/internal/adapters/mq"
	application "go-cloud-storage/backend/services/file-service/internal/application"
)

func startRecycleCleanupWorkers(ctx context.Context, recycleService application.RecycleService, rabbitClient *mq.RabbitMQClient, mqCfg *config.RabbitMQConfig) {
	interval := 60 * time.Second
	if mqCfg != nil && mqCfg.ScanIntervalSeconds > 0 {
		interval = time.Duration(mqCfg.ScanIntervalSeconds) * time.Second
	}

	go func() {
		if rabbitClient == nil {
			return
		}
		if err := rabbitClient.ConsumeExpiredFilePurge(ctx, func(ctx context.Context, fileID string) error {
			return recycleService.PurgeExpired(ctx, []string{fileID})
		}); err != nil && ctx.Err() == nil {
			slog.Error("recycle cleanup consumer exited", "error", err)
		}
	}()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		_, err := recycleService.DispatchExpiredPurgeJobs(ctx, 200)
		if err != nil && ctx.Err() == nil {
			slog.Error("dispatch recycle cleanup job failed", "error", err)
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				n, dispatchErr := recycleService.DispatchExpiredPurgeJobs(ctx, 200)
				if dispatchErr != nil {
					if ctx.Err() == nil {
						slog.Error("dispatch recycle cleanup job failed", "error", dispatchErr)
					}
					continue
				}
				if n > 0 {
					slog.Info("dispatched recycle cleanup jobs", "count", n)
				}
			}
		}
	}()
}
