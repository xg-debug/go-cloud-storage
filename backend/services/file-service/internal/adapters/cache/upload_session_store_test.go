package cache

import (
	"context"
	"github.com/go-redis/redis/v8"
	"go-cloud-storage/backend/pkg/utils"
	fileports "go-cloud-storage/backend/services/file-service/internal/ports"
	"os"
	"testing"
	"time"
)

func TestRedisSessionOwnershipAndLockFencing(t *testing.T) {
	addr := os.Getenv("GCS_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("set GCS_TEST_REDIS_ADDR to run real Redis regression")
	}
	r := redis.NewClient(&redis.Options{Addr: addr})
	defer r.Close()
	ctx := context.Background()
	hash := utils.NewUUID()
	s := NewRedisUploadSessionStore(r, time.Hour, 2*time.Hour)
	key := uploadSessionKey(7, hash)
	defer r.Del(ctx, key, uploadSessionLockKey(7, hash))
	defer r.ZRem(ctx, uploadSessionSet, key)
	old := &fileports.UploadSession{UserID: 7, FileHash: hash, UploadID: "old", ObjectKey: "old-key", FileSize: 3, ChunkSize: 3, TotalChunks: 1}
	if err := s.Save(ctx, old); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, 7, hash, "old"); err != nil {
		t.Fatal(err)
	}
	newer := *old
	newer.UploadID = "new"
	newer.ObjectKey = "new-key"
	if err := s.Save(ctx, &newer); err != nil {
		t.Fatal(err)
	}
	if err := s.SavePart(ctx, 7, hash, fileports.UploadPartState{UploadID: "old", Index: 0, ETag: "stale"}); err != fileports.ErrUploadSessionNotFound {
		t.Fatalf("stale part accepted: %v", err)
	}
	for _, id := range []string{"old", ""} {
		if err := s.Delete(ctx, 7, hash, id); err != nil {
			t.Fatal(err)
		}
	}
	current, err := s.Get(ctx, 7, hash)
	if err != nil || current.UploadID != "new" || len(current.Parts) != 0 {
		t.Fatalf("new session damaged: %+v %v", current, err)
	}
	if err = s.SavePart(ctx, 7, hash, fileports.UploadPartState{UploadID: "new", Index: 0, ETag: "fresh", Size: 3}); err != nil {
		t.Fatal(err)
	}
	current, err = s.Get(ctx, 7, hash)
	if err != nil || current.Parts[0].ETag != "fresh" {
		t.Fatal("valid part not saved", err)
	}
	locked, err := s.AcquireMergeLock(ctx, 7, hash, "old-owner", time.Minute)
	if err != nil || !locked {
		t.Fatal("lock failed", err)
	}
	if locked, err = s.AcquireMergeLock(ctx, 7, hash, "competitor", time.Minute); err != nil || locked {
		t.Fatal("concurrent lock accepted", err)
	}
	// Simulate Redis lease expiry, then acquire a new lease before old cleanup.
	if err = r.Del(ctx, uploadSessionLockKey(7, hash)).Err(); err != nil {
		t.Fatal(err)
	}
	if locked, err = s.AcquireMergeLock(ctx, 7, hash, "new-owner", time.Minute); err != nil || !locked {
		t.Fatal("replacement lock failed", err)
	}
	if err = s.ReleaseMergeLock(ctx, 7, hash, "old-owner"); err != nil {
		t.Fatal(err)
	}
	if got := r.Get(ctx, uploadSessionLockKey(7, hash)).Val(); got != "new-owner" {
		t.Fatalf("old request released new lock: %q", got)
	}
}
