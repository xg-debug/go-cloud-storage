package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go-cloud-storage/backend/infrastructure/cache"
)

// ipRateLimiter 基于客户端 IP 的固定窗口限流器。
// 用途：登录/注册/找回密码等公开接口的防暴力破解与滥用防护。
// 优先使用 Redis（多实例部署下计数一致），Redis 不可用时降级为进程内存实现。
type ipRateLimiter struct {
	limit  int
	window time.Duration

	mu      sync.Mutex
	memHits map[string]*memEntry // key -> 窗口计数（内存降级用）
}

type memEntry struct {
	windowStart int64
	count       int
}

// NewIPRateLimiter 返回按 IP 限流的 Gin 中间件：每个 IP 在 window 内最多 limit 次请求。
// 超过限制返回 429。
func NewIPRateLimiter(limit int, window time.Duration) gin.HandlerFunc {
	rl := &ipRateLimiter{limit: limit, window: window, memHits: make(map[string]*memEntry)}
	go rl.cleanupLoop()
	return rl.handle
}

func (rl *ipRateLimiter) handle(c *gin.Context) {
	ip := c.ClientIP()
	route := c.FullPath()
	if route == "" {
		route = c.Request.URL.Path
	}
	if !rl.allow(route, ip) {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
			"code":    30009,
			"message": "请求过于频繁，请稍后再试",
		})
		return
	}
	c.Next()
}

func (rl *ipRateLimiter) allow(route, ip string) bool {
	rdb := cache.GetClient()
	key := fmt.Sprintf("rl:ip:%s:%s", route, ip)

	if rdb != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		// 固定窗口：首次 INCR 时设置过期时间
		n, err := rdb.Incr(ctx, key).Result()
		if err == nil {
			if n == 1 {
				rdb.Expire(ctx, key, rl.window)
			}
			return n <= int64(rl.limit)
		}
		// Redis 异常时降级到内存计数
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	windowStart := now.Unix() / int64(rl.window.Seconds()) * int64(rl.window.Seconds())
	entry, ok := rl.memHits[key]
	if !ok || entry.windowStart != windowStart {
		rl.memHits[key] = &memEntry{windowStart: windowStart, count: 1}
		return 1 <= rl.limit
	}
	entry.count++
	return entry.count <= rl.limit
}

func (rl *ipRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()
	cutoffWindow := time.Now().Unix() - 2*int64(rl.window.Seconds())
	for range ticker.C {
		rl.mu.Lock()
		for k, e := range rl.memHits {
			if e.windowStart < cutoffWindow {
				delete(rl.memHits, k)
			}
		}
		rl.mu.Unlock()
		cutoffWindow = time.Now().Unix() - 2*int64(rl.window.Seconds())
	}
}
