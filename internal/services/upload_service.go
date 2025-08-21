package services

import (
	"context"
	"errors"
	"fmt"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"go-cloud-storage/internal/models"
	"go-cloud-storage/internal/pkg/aliyunoss"
	"go-cloud-storage/internal/pkg/utils"
	"go-cloud-storage/internal/repositories"
	"path/filepath"
)

type UploadService interface {
	InitUpload(userId int, fileName string, fileSize int64, chunkCount int) (*models.UploadTask, error)
	UploadChunk(taskId string, chunkIndex int, data []byte) error
	CompleteUpload(taskId string) error
}

type uploadService struct {
	repo       repositories.UploadRepository
	ossService *aliyunoss.OSSService
}

func NewUploadService(repo repositories.UploadRepository, oss *aliyunoss.OSSService) UploadService {
	return &uploadService{repo: repo, ossService: oss}
}

// InitUpload 初始化上传
func (s *uploadService) InitUpload(userId int, fileName string, fileSize int64, chunkCount int) (*models.UploadTask, error) {
	taskId := utils.NewUUID()
	objectKey := fmt.Sprintf("files/%d/%s", userId, taskId+filepath.Ext(fileName))

	// 1.请求 oss 初始化分片
	uploadId, err := s.ossService.InitiateMultipartUpload(context.Background(), objectKey)
	if err != nil {
		return nil, err
	}
	task := &models.UploadTask{
		Id:         taskId,
		UserId:     userId,
		FileName:   fileName,
		FileSize:   fileSize,
		ChunkCount: chunkCount,
		UploadId:   uploadId,
		Status:     0,
	}
	if err := s.repo.CreateTask(task); err != nil {
		return nil, err
	}

	// 创建 FileChunk 记录
	var chunks []models.FileChunk
	for i := 1; i <= chunkCount; i++ {
		chunks = append(chunks, models.FileChunk{
			TaskId:     taskId,
			ChunkIndex: i,
			Status:     0,
		})
	}
	err = s.repo.CreateChunks(chunks)
	if err != nil {
		return nil, err
	}

	return task, nil
}

// UploadChunk 上传单个分片
func (s *uploadService) UploadChunk(taskId string, chunkIndex int, data []byte) error {
	task, _ := s.repo.GetTask(taskId)
	if task == nil {
		return errors.New("upload task not found")
	}

	objectKey := fmt.Sprintf("files/%d/%s", task.UserId, task.Id+filepath.Ext(task.FileName))
	etag, err := s.ossService.UploadPart(context.Background(), objectKey, task.UploadId, chunkIndex, data)
	if err != nil {
		return err
	}

	return s.repo.UpdateChunk(taskId, chunkIndex, etag, int64(len(data)))
}

// CompleteUpload 完成上传
func (s *uploadService) CompleteUpload(taskId string) error {
	task, _ := s.repo.GetTask(taskId)
	chunks, _ := s.repo.GetChunks(taskId)

	var parts []oss.UploadPart
	for _, c := range chunks {
		if c.Status != 1 {
			return errors.New("some chunks not uploaded")
		}
		parts = append(parts, oss.UploadPart{
			PartNumber: int32(c.ChunkIndex),
			ETag:       &c.ETag,
		})
	}

	objectKey := fmt.Sprintf("files/%d/%s", task.UserId, task.Id+filepath.Ext(task.FileName))
	if err := s.ossService.CompleteMultipartUpload(context.Background(), objectKey, task.UploadId, parts); err != nil {
		return err
	}

	return s.repo.UpdateTaskStatus(taskId, 1)
}
