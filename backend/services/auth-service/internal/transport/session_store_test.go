package transport

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"go-cloud-storage/backend/pkg/contracts"
	"go-cloud-storage/backend/pkg/utils"
	"os"
	"testing"
	"time"
)

func TestRedisLogoutAndRefreshSessionOwnership(t *testing.T) {
	addr := os.Getenv("GCS_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("set GCS_TEST_REDIS_ADDR to run real Redis regression")
	}
	r := redis.NewClient(&redis.Options{Addr: addr})
	defer r.Close()
	ctx := context.Background()
	id := int(time.Now().UnixNano()%1000000000) + 100000
	s := redisSessions{client: r}
	defer r.Del(ctx, sessionKeys(id)...)
	old, newer := utils.NewUUID(), utils.NewUUID()
	if err := s.start(ctx, id, old, "refresh-old", time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := s.validate(ctx, id, old, "refresh-old"); err != nil {
		t.Fatal(err)
	}
	if err := s.revoke(ctx, id, old); err != nil {
		t.Fatal(err)
	}
	for _, refresh := range []string{"", "refresh-old"} {
		if err := s.validate(ctx, id, old, refresh); !errors.Is(err, contracts.ErrSessionInvalid) {
			t.Fatalf("revoked session accepted: %v", err)
		}
	}
	if err := s.start(ctx, id, newer, "refresh-new", time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := s.revoke(ctx, id, old); err != nil {
		t.Fatal(err)
	}
	if err := s.validate(ctx, id, newer, "refresh-new"); err != nil {
		t.Fatalf("delayed logout revoked new login: %v", err)
	}
	if err := s.validate(ctx, id, "", ""); !errors.Is(err, contracts.ErrSessionInvalid) {
		t.Fatal("legacy token accepted")
	}
}
