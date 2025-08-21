package services

import (
	"context"
	"fmt"
	"go-cloud-storage/internal/models"
	"go-cloud-storage/internal/pkg/aliyunoss"
	"go-cloud-storage/internal/repositories"
	"sort"
	"sync"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
)

type FileChunkService interface {
	UploadChunk(fileId string, chunkIndex int, chunkData []byte, chunkHash, uploadId, ObjectKey string) error
	UploadChunksConcurrently(fileId, uploadId, objectKey string, chunks map[int][]byte) error
	GetUploadedChunks(fileId string) ([]int, error)
	MergeChunks(fileId string, objectKey string, totalChunks int) error
	CheckFileExists(fileHash string) (*models.File, error)
}

type fileChunkService struct {
	fileRepo      repositories.FileRepository
	fileChunkRepo repositories.FileChunkRepository
	ossService    *aliyunoss.OSSService
}

func NewFileChunkService(fcr repositories.FileChunkRepository, fr repositories.FileRepository, oss *aliyunoss.OSSService) FileChunkService {
	return &fileChunkService{
		fileRepo:      fr,
		fileChunkRepo: fcr,
		ossService:    oss,
	}
}

// UploadChunk 上传分片到OSS并记录到数据库
func (s *fileChunkService) UploadChunk(fileId string, chunkIndex int, chunkData []byte, chunkHash, uploadId, ObjectKey string) error {
	fmt.Printf("UploadChunk - fileId: %s, chunkIndex: %d, chunkHash: %s, uploadId: %s, ObjectKey: %s, dataSize: %d\n",
		fileId, chunkIndex, chunkHash, uploadId, ObjectKey, len(chunkData))

	// 检查分片是否已存在
	exists, err := s.fileChunkRepo.CheckChunkExists(fileId, chunkIndex)
	if err != nil {
		return fmt.Errorf("检查分片存在性失败: %v", err)
	}
	if exists {
		return nil // 分片已存在，跳过上传
	}

	// 上传分片到 OSS
	partNumber := chunkIndex
	fmt.Printf("UploadChunk - 调用 UploadPart，partNumber: %d\n", partNumber)
	etag, err := s.ossService.UploadPart(context.Background(), ObjectKey, uploadId, partNumber, chunkData)
	if err != nil {
		return fmt.Errorf("上传分片失败: %v", err)
	}

	// 创建分片记录
	chunk := &models.FileChunk{
		FileId:     fileId,
		ChunkIndex: chunkIndex,
		ChunkHash:  chunkHash,
		Size:       len(chunkData),
		OssEtag:    etag,
		UploadId:   uploadId,
	}

	if err := s.fileChunkRepo.CreateChunk(chunk); err != nil {
		return fmt.Errorf("创建分片记录失败: %v", err)
	}

	return nil
}

// UploadChunksConcurrently 并发上传多个分片
func (s *fileChunkService) UploadChunksConcurrently(fileId, uploadId, objectKey string, chunks map[int][]byte) error {
	var wg sync.WaitGroup
	var mutex sync.Mutex
	var errs []error

	for idx, data := range chunks {
		chunkIndex := idx
		chunkData := data
		wg.Add(1)
		go func() {
			defer wg.Done()
			chunkHash := fmt.Sprintf("%x", chunkData) // 可根据需要改成真实 hash
			err := s.UploadChunk(fileId, chunkIndex, chunkData, chunkHash, uploadId, objectKey)
			if err != nil {
				mutex.Lock()
				errs = append(errs, err)
				mutex.Unlock()
			}
		}()
	}

	wg.Wait()
	if len(errs) > 0 {
		return fmt.Errorf("部分分片上传失败: %v", errs)
	}
	return nil
}

// GetUploadedChunks 获取已上传的分片索引
func (s *fileChunkService) GetUploadedChunks(fileId string) ([]int, error) {
	return s.fileChunkRepo.GetUploadedChunks(fileId)
}

// MergeChunks 合并所有分片为最终文件
func (s *fileChunkService) MergeChunks(fileId string, objectKey string, totalChunks int) error {
	// 获取所有分片
	chunks, err := s.fileChunkRepo.GetChunksByFileId(fileId)
	if err != nil {
		return fmt.Errorf("获取分片列表失败: %v", err)
	}

	if len(chunks) != totalChunks {
		return fmt.Errorf("分片数量不匹配，期望 %d，实际 %d", totalChunks, len(chunks))
	}

	// 按 chunkIndex 排序
	sort.Slice(chunks, func(i, j int) bool {
		return chunks[i].ChunkIndex < chunks[j].ChunkIndex
	})

	var parts []oss.UploadPart
	for _, c := range chunks {
		parts = append(parts, oss.UploadPart{
			PartNumber: int32(c.ChunkIndex),
			ETag:       &c.OssEtag,
		})
	}

	// 使用任意分片的 uploadId 完成分片合并
	uploadId := chunks[0].UploadId
	if err := s.ossService.CompleteMultipartUpload(context.Background(), objectKey, uploadId, parts); err != nil {
		return fmt.Errorf("合并分片失败: %v", err)
	}

	// 异步清理数据库分片记录
	go s.cleanupChunks(fileId)

	return nil
}

// cleanupChunks 清理分片数据（异步执行）
func (s *fileChunkService) cleanupChunks(fileId string) {
	// 删除数据库中的分片记录
	_ = s.fileChunkRepo.DeleteChunksByFileId(fileId)
}

// CheckFileExists 检查文件是否已完全上传（通过MD5秒传）
func (s *fileChunkService) CheckFileExists(fileHash string) (*models.File, error) {
	file, err := s.fileRepo.FindByHash(fileHash)
	if err != nil {
		return nil, fmt.Errorf("查询文件失败: %v", err)
	}
	if file == nil {
		return nil, nil // 文件不存在
	}
	return file, nil
}
