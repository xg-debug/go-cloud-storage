package cache

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"go-cloud-storage/backend/pkg/config"

	"github.com/go-redis/redis/v8"
)

var (
	globalClient *redis.Client // 全局唯一连接池
	once         sync.Once
	initErr      error
)

func InitRedis(cfg *config.RedisConfig) error {
	// sync.Once 保证闭包内的代码只会执行一次，但闭包本身没有返回值。闭包内的 return 只是退出当前闭包的执行，而不是退出外层函数
	once.Do(func() {
		if !cfg.Enabled {
			// Redis 关闭不等于启动失败：服务可降级运行。
			// 依赖 Redis 的功能（refresh token 持久化、分片上传会话、限流计数）将不可用，
			// 各调用方需对 GetClient() 判空。
			globalClient = nil
			initErr = nil
			return
		}
		globalClient = redis.NewClient(&redis.Options{
			Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
			Password:     cfg.Password,
			DB:           cfg.DB,
			PoolSize:     cfg.PoolSize,
			DialTimeout:  3 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
			MaxRetries:   2,
		})

		// 测试连接
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second) // 超时控制：3秒测试超时
		defer cancel()
		if _, err := globalClient.Ping(ctx).Result(); err != nil {
			initErr = fmt.Errorf("redis连接测试失败：%w", err)
			globalClient = nil
		}
	})
	return initErr
}

// GetClient 获取全局 Redis 客户端
func GetClient() *redis.Client {
	return globalClient // 始终返回同一实例
}

// defer client.Close() 确保程序退出时关闭。只需要在main.go中调用Close(),其他地方获取redis连接后不需要关闭

// Close 关闭Redis连接
func Close() {
	if globalClient != nil {
		if err := globalClient.Close(); err != nil {
			log.Printf("关闭Redis连接发生错误: %v", err)
		} else {
			log.Println("Redis连接已关闭!")
		}
	}
}
