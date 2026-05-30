package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type visitor struct {
	lastSeen time.Time
	tokens   int
}

// RateLimiter 简单的基于用户ID的令牌桶限流器
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[int]*visitor
	rate     int           // 每秒令牌数
	burst    int           // 突发容量
	cleanup  time.Duration // 清理间隔
}

var defaultLimiter *RateLimiter

func InitRateLimiter(rate, burst int) {
	if rate <= 0 {
		rate = 50
	}
	if burst <= 0 {
		burst = 100
	}
	defaultLimiter = &RateLimiter{
		visitors: make(map[int]*visitor),
		rate:     rate,
		burst:    burst,
		cleanup:  5 * time.Minute,
	}
	go defaultLimiter.cleanupVisitors()
}

func (rl *RateLimiter) allow(userId int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[userId]
	now := time.Now()

	if !exists {
		rl.visitors[userId] = &visitor{lastSeen: now, tokens: rl.burst - 1}
		return true
	}

	elapsed := now.Sub(v.lastSeen)
	tokensToAdd := int(elapsed.Seconds()) * rl.rate
	if tokensToAdd > 0 {
		v.tokens += tokensToAdd
		if v.tokens > rl.burst {
			v.tokens = rl.burst
		}
		v.lastSeen = now
	}

	if v.tokens > 0 {
		v.tokens--
		return true
	}

	// 尝试补充至少1个token（防止浮点精度问题）
	if elapsed >= time.Second {
		v.tokens = rl.rate - 1
		v.lastSeen = now
		return true
	}

	return false
}

func (rl *RateLimiter) cleanupVisitors() {
	for {
		time.Sleep(rl.cleanup)
		rl.mu.Lock()
		cutoff := time.Now().Add(-rl.cleanup)
		for id, v := range rl.visitors {
			if v.lastSeen.Before(cutoff) {
				delete(rl.visitors, id)
			}
		}
		rl.mu.Unlock()
	}
}

func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if defaultLimiter == nil {
			c.Next()
			return
		}
		userId := c.GetInt("userId")
		// 未认证用户不限制（登录/注册已由业务逻辑控制）
		if userId == 0 {
			c.Next()
			return
		}
		if !defaultLimiter.allow(userId) {
			c.AbortWithStatusJSON(429, gin.H{
				"code":      30009,
				"message":   "请求过于频繁，请稍后再试",
				"requestId": c.GetString("requestId"),
			})
			return
		}
		c.Next()
	}
}
