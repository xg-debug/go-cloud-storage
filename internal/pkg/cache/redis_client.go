package cache

import (
	"context"
	"encoding/json"
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

// 分片上传信息结构
type ChunkUploadInfo struct {
	FileHash       string `json:"fileHash"`
	FileName       string `json:"fileName"`
	TotalChunks    int    `json:"totalChunks"`
	UploadedChunks []int  `json:"uploadedChunks"`
	Status         string `json:"status"`    // uploading, completed
	UploadId       string `json:"uploadId"`  // OSS分片上传ID
	ObjectKey      string `json:"objectKey"` // OSS对象键
	CreatedAt      int64  `json:"createdAt"`
	UpdatedAt      int64  `json:"updatedAt"`
}

// 保存分片上传信息
func SaveChunkUploadInfo(ctx context.Context, fileHash string, info *ChunkUploadInfo) error {
	if globalClient == nil {
		return errors.New("Redis client not initialized")
	}

	key := fmt.Sprintf("chunk_upload:%s", fileHash)
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}

	// 设置过期时间为24小时
	return globalClient.Set(ctx, key, data, 24*time.Hour).Err()
}

// 获取分片上传信息
func GetChunkUploadInfo(ctx context.Context, fileHash string) (*ChunkUploadInfo, error) {
	if globalClient == nil {
		return nil, errors.New("Redis client not initialized")
	}

	key := fmt.Sprintf("chunk_upload:%s", fileHash)
	data, err := globalClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 不存在
		}
		return nil, err
	}

	var info ChunkUploadInfo
	err = json.Unmarshal([]byte(data), &info)
	if err != nil {
		return nil, err
	}

	return &info, nil
}

// 更新已上传的分片
func UpdateUploadedChunk(ctx context.Context, fileHash string, chunkIndex int) error {
	if globalClient == nil {
		return errors.New("Redis client not initialized")
	}

	info, err := GetChunkUploadInfo(ctx, fileHash)
	if err != nil {
		return err
	}
	if info == nil {
		return fmt.Errorf("chunk upload info not found for fileHash: %s", fileHash)
	}

	// 检查分片是否已存在
	for _, chunk := range info.UploadedChunks {
		if chunk == chunkIndex {
			return nil // 已存在，不需要重复添加
		}
	}

	// 添加新的分片
	info.UploadedChunks = append(info.UploadedChunks, chunkIndex)
	info.UpdatedAt = time.Now().Unix()

	// 检查是否所有分片都已上传
	if len(info.UploadedChunks) == info.TotalChunks {
		info.Status = "completed"
	}

	return SaveChunkUploadInfo(ctx, fileHash, info)
}

// 删除分片上传信息
func DeleteChunkUploadInfo(ctx context.Context, fileHash string) error {
	if globalClient == nil {
		return errors.New("Redis client not initialized")
	}

	key := fmt.Sprintf("chunk_upload:%s", fileHash)
	return globalClient.Del(ctx, key).Err()
}
