package transport

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"go-cloud-storage/backend/pkg/contracts"
	cache "go-cloud-storage/backend/pkg/platform/cache"
	"time"
)

// Both tokens identify the same server-side session. Revocation also invalidates
// access tokens minted by a refresh request that races with logout.
type sessionStore interface {
	start(context.Context, int, string, string, time.Duration) error
	validate(context.Context, int, string, string) error
	revoke(context.Context, int, string) error
}
type redisSessions struct{ client *redis.Client }

func (s redisSessions) connection() *redis.Client {
	if s.client != nil {
		return s.client
	}
	return cache.GetClient()
}

func sessionKeys(id int) []string {
	return []string{fmt.Sprintf("user:%d:session", id), fmt.Sprintf("user:%d:refresh_token", id)}
}

var errSessionUnavailable = errors.New("session store unavailable")

func (s redisSessions) start(ctx context.Context, id int, sid, refresh string, ttl time.Duration) error {
	r := s.connection()
	if r == nil {
		return errSessionUnavailable
	}
	keys := sessionKeys(id)
	p := r.TxPipeline()
	p.Set(ctx, keys[0], sid, ttl)
	p.Set(ctx, keys[1], refresh, ttl)
	_, err := p.Exec(ctx)
	return err
}
func (s redisSessions) validate(ctx context.Context, id int, sid, refresh string) error {
	r := s.connection()
	if r == nil {
		return errSessionUnavailable
	}
	if sid == "" {
		return contracts.ErrSessionInvalid
	}
	n, err := r.Eval(ctx, `if redis.call('GET',KEYS[1]) ~= ARGV[1] then return 0 end
 if ARGV[2] ~= '' and redis.call('GET',KEYS[2]) ~= ARGV[2] then return 0 end
 return 1`, sessionKeys(id), sid, refresh).Int()
	if err != nil {
		return err
	}
	if n != 1 {
		return contracts.ErrSessionInvalid
	}
	return nil
}
func (s redisSessions) revoke(ctx context.Context, id int, sid string) error {
	r := s.connection()
	if r == nil {
		return errSessionUnavailable
	}
	// A delayed logout from an older login must not revoke a newer session.
	return r.Eval(ctx, `if redis.call('GET',KEYS[1]) == ARGV[1] then return redis.call('DEL',KEYS[1],KEYS[2]) end return 0`, sessionKeys(id), sid).Err()
}
