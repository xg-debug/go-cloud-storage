package cache

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"go-cloud-storage/internal/pkg/config"

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
			initErr = errors.New("Redis is disabled in config")
			return // // 只是退出闭包once.Do()，不是退出InitRedis函数
		}
		globalClient = redis.NewClient(&redis.Options{
			Addr: fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
			//Password: cfg.Password,
			DB:       cfg.DB,
			PoolSize: cfg.PoolSize,
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
