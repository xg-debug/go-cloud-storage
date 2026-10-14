package application

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	ports "go-cloud-storage/backend/pkg/ports"
	domain "go-cloud-storage/backend/services/file-service/internal/domain"
)

func enqueueObjectDeletion(tx *gorm.DB, keys []string) error {
	seen := map[string]bool{}
	for _, key := range keys {
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		if err := tx.Create(&domain.ObjectDeletion{ID: uuid.NewString(), ObjectKey: key, NextAttemptAt: time.Now()}).Error; err != nil {
			return err
		}
	}
	return nil
}

// RunObjectCleanup runs in the file owner (including legacy monolith mode).
func RunObjectCleanup(ctx context.Context, db *gorm.DB, storage ports.Storage) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		for i := 0; i < 100; i++ {
			processed, err := ProcessObjectDeletion(ctx, db, storage)
			if err != nil {
				if ctx.Err() == nil {
					slog.Error("object cleanup failed", "error", err)
				}
				break
			}
			if !processed {
				break
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// A row lock prevents replicas from working on the same job. Failed deletes get
// bounded exponential backoff; a crash after deletion safely repeats the delete.
func ProcessObjectDeletion(ctx context.Context, db *gorm.DB, storage ports.Storage) (bool, error) {
	processed := false
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var job domain.ObjectDeletion
		err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Where("next_attempt_at <= ?", time.Now()).Order("next_attempt_at, id").First(&job).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		processed = true
		// Never delete an object still referenced by any live or recycled metadata.
		var refs int64
		if err = tx.Model(&domain.File{}).Where("oss_object_key = ?", job.ObjectKey).Count(&refs).Error; err != nil {
			return err
		}
		if refs == 0 {
			callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			err = storage.DeleteObjectWithThumbnail(callCtx, job.ObjectKey)
			cancel()
			if err != nil {
				delay := time.Second * time.Duration(1<<min(job.Attempts+1, 8))
				slog.Warn("object deletion deferred", "jobId", job.ID, "attempt", job.Attempts+1)
				return tx.Model(&job).Updates(map[string]any{"attempts": job.Attempts + 1, "next_attempt_at": time.Now().Add(delay)}).Error
			}
		}
		return tx.Delete(&job).Error
	})
	return processed, err
}
