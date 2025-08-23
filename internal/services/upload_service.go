package services

import (
	"context"
	"errors"
	"fmt"
	"go-cloud-storage/internal/models"
	"go-cloud-storage/internal/pkg/aliyunoss"
	"go-cloud-storage/internal/repositories"
	"strings"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/google/uuid"
)

type UploadService interface {
	InitUpload(ctx context.Context, userId int, fileName string, fileSize, chunkSize int64, fileHash string) (*models.UploadTask, error)
	GetChunkPresignedURL(ctx context.Context, taskId string, partNumber int) (string, error)
	MarkChunkUploaded(ctx context.Context, taskId string, partNumber int, etag string) error
	GetTask(taskId string) (*models.UploadTask, error)
	CompleteUpload(ctx context.Context, taskId string) error
	GetIncompleteTasks(userId int) ([]models.UploadTask, error)
	DeleteTask(taskId string, userId int) error
}

type uploadService struct {
	repo       repositories.UploadRepository
	ossService *aliyunoss.OSSService
}

func NewUploadService(repo repositories.UploadRepository, oss *aliyunoss.OSSService) UploadService {
	return &uploadService{repo: repo, ossService: oss}
}

// 初始化上传任务
func (s *uploadService) InitUpload(ctx context.Context, userId int, fileName string, fileSize, chunkSize int64, fileHash string) (*models.UploadTask, error) {
	chunkCount := int((fileSize + chunkSize - 1) / chunkSize)
	objectKey := "files/" + uuid.New().String() + "_" + fileName
	uploadId, err := s.ossService.InitiateMultipartUpload(ctx, objectKey)
	if err != nil {
		return nil, err
	}

	task := &models.UploadTask{
		Id:             uuid.New().String(),
		UserId:         userId,
		FileName:       fileName,
		FileSize:       fileSize,
		FileHash:       fileHash,
		ChunkSize:      chunkSize,
		ChunkCount:     chunkCount,
		UploadedChunks: []models.UploadedChunk{},
		UploadId:       uploadId,
		ObjectKey:      objectKey,
		Status:         0,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	err = s.repo.Create(task)
	if err != nil {
		return nil, err
	}
	return task, nil
}

// 获取某个分片预签名 URL
func (s *uploadService) GetChunkPresignedURL(ctx context.Context, taskId string, partNumber int) (string, error) {
	task, err := s.repo.GetById(taskId)
	if err != nil {
		return "", err
	}
	if partNumber < 1 || partNumber > task.ChunkCount {
		return "", errors.New("invalid part number")
	}
	return s.ossService.GeneratePresignedURL(ctx, task.ObjectKey, task.UploadId, partNumber, 1*time.Hour)
}

// 标记分片已上传
func (s *uploadService) MarkChunkUploaded(ctx context.Context, taskId string, partNumber int, etag string) error {
	task, err := s.repo.GetById(taskId)
	if err != nil {
		return err
	}

	// 检查分片是否已存在，防止重复添加
	for _, c := range task.UploadedChunks {
		if c.Index == partNumber {
			return nil
		}
	}

	task.UploadedChunks = append(task.UploadedChunks, models.UploadedChunk{
		Index: partNumber,
		ETag:  etag,
	})

	// 如果所有分片都上传完成，则标记状态为完成
	if len(task.UploadedChunks) == task.ChunkCount {
		task.Status = 1
	}

	task.UpdatedAt = time.Now()
	return s.repo.Update(task)
}

func (s *uploadService) GetTask(taskId string) (*models.UploadTask, error) {
	return s.repo.GetById(taskId)
}

// 完成整个上传
func (s *uploadService) CompleteUpload(ctx context.Context, taskId string) error {
	task, err := s.repo.GetById(taskId)
	if err != nil {
		return err
	}
	if len(task.UploadedChunks) != task.ChunkCount {
		return errors.New("not all chunks uploaded")
	}

	var parts []oss.UploadPart
	for _, c := range task.UploadedChunks {
		parts = append(parts, oss.UploadPart{
			PartNumber: int32(c.Index),
			ETag:       &c.ETag,
		})
	}

	// 完成OSS分片上传
	err = s.ossService.CompleteMultipartUpload(ctx, task.ObjectKey, task.UploadId, parts)
	if err != nil {
		return err
	}

	// 标记任务完成
	task.Status = 1
	task.UpdatedAt = time.Now()
	err = s.repo.Update(task)
	if err != nil {
		return err
	}

	// 创建文件记录
	fileURL := s.ossService.GenerateObjectURL(task.ObjectKey)
	fileInfo := &models.File{
		Id:            uuid.New().String(),
		Name:          task.FileName,
		Size:          task.FileSize,
		SizeStr:       s.formatFileSize(task.FileSize),
		IsDir:         false,
		FileExtension: s.getFileExtension(task.FileName),
		FileHash:      task.FileHash,
		FileURL:       fileURL,
		UserId:        task.UserId,
		IsDeleted:     false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// TODO: 这里需要通过依赖注入获取文件服务来创建文件记录
	// 暂时先记录日志
	fmt.Printf("文件上传完成: %+v\n", fileInfo)

	return nil
}

// 辅助函数
func (s *uploadService) formatFileSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.2f KB", float64(size)/1024)
	} else if size < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MB", float64(size)/(1024*1024))
	} else {
		return fmt.Sprintf("%.2f GB", float64(size)/(1024*1024*1024))
	}
}

func (s *uploadService) getFileExtension(filename string) string {
	if idx := strings.LastIndex(filename, "."); idx != -1 {
		return filename[idx:]
	}
	return ""
}

// 获取未完成的上传任务
func (s *uploadService) GetIncompleteTasks(userId int) ([]models.UploadTask, error) {
	return s.repo.GetIncompleteByUserId(userId)
}

// 删除上传任务
func (s *uploadService) DeleteTask(taskId string, userId int) error {
	// 先验证任务是否属于该用户
	task, err := s.repo.GetById(taskId)
	if err != nil {
		return err
	}
	if task.UserId != userId {
		return errors.New("unauthorized to delete this task")
	}

	// 如果任务正在进行中，需要取消OSS的分片上传
	if task.Status == 0 && task.UploadId != "" {
		// 这里可以选择是否取消OSS的分片上传，或者保留以便后续恢复
		// s.ossService.AbortMultipartUpload(context.Background(), task.ObjectKey, task.UploadId)
	}

	return s.repo.Delete(taskId)
}
