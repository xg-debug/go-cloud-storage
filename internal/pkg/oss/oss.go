package oss

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
	"go-cloud-storage/internal/pkg/config"
	"go-cloud-storage/utils"
	"io"
	"path/filepath"
	"strings"
	"time"
)

// FileInfo 返回给业务层/前端的结构
type FileInfo struct {
	Id            string    `json:"id"`
	UserId        int       `json:"user_id"`
	Name          string    `json:"name"`
	Size          uint64    `json:"size"`
	IsDir         bool      `json:"is_dir"`
	FileExtension string    `json:"file_extension"`
	OssObjectKey  string    `json:"oss_object_key"`
	FileHash      string    `json:"file_hash"`
	ParentId      string    `json:"parent_id"`
	IsDeleted     bool      `json:"is_deleted"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Thumbnail     string    `json:"thumbnail"`
	URL           string    `json:"url"`
}

type OSSService struct {
	client   *oss.Client
	bucket   string
	endpoint string
	// 可添加 max size 等配置
}

func NewOSSService(cfg *config.AliyunOssConfig) (*OSSService, error) {
	if cfg.AccessId == "" || cfg.AccessSecret == "" {
		return nil, errors.New("阿里云OSS凭证缺失")
	}
	if cfg.Bucket == "" || cfg.EndPoint == "" {
		return nil, errors.New("阿里云OSS配置不完整")
	}

	credProvider := credentials.NewStaticCredentialsProvider(cfg.AccessId, cfg.AccessSecret)
	ossCfg := oss.LoadDefaultConfig().WithCredentialsProvider(credProvider).WithRegion("cn-beijing").WithEndpoint(cfg.EndPoint)
	client := oss.NewClient(ossCfg)

	return &OSSService{
		client:   client,
		bucket:   cfg.Bucket,
		endpoint: cfg.EndPoint,
	}, nil
}

// UploadFromStream 从 reader 上传文件，返回 FileInfo
// 注意：这里会把文件读入内存（io.ReadAll）。如果需要支持超大文件，后续可改成分片上传（MultipartUpload）。
func (s *OSSService) UploadFromStream(ctx context.Context, r io.Reader, fileName string, userId int, parentId string, maxSize int64) (*FileInfo, error) {
	if fileName == "" {
		return nil, errors.New("文件名不能为空")
	}

	// 扩展名（不含点）
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fileName), "."))

	// 读取全部内容（二进制安全）
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("读取文件内容失败: %w", err)
	}
	if maxSize > 0 && int64(len(data)) > maxSize {
		return nil, fmt.Errorf("文件大小超过限制: %d 字节", maxSize)
	}

	// 计算 sha256
	sum := sha256.Sum256(data)
	fileHash := fmt.Sprintf("%x", sum[:])

	// 生成文件 ID 和 OSS 路径
	fileId := utils.NewUUID()
	ossPath := fmt.Sprintf("files/%d", userId)
	if parentId != "" {
		ossPath = ossPath + "/" + parentId
	}
	// 保留原始扩展名（含点）
	ossPath = fmt.Sprintf("%s/%s%s", ossPath, fileId, filepath.Ext(fileName))

	// 使用 bytes.Reader 上传（不会破坏二进制）
	body := bytes.NewReader(data)

	request := &oss.PutObjectRequest{
		Bucket: oss.Ptr(s.bucket),
		Key:    oss.Ptr(ossPath),
		Body:   body,
	}

	_, err = s.client.PutObject(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("文件上传至 OSS 失败: %w", err)
	}

	fileURL := s.generateObjectURL(ossPath)

	fi := &FileInfo{
		Id:            fileId,
		UserId:        userId,
		Name:          fileName,
		Size:          uint64(len(data)),
		IsDir:         false,
		FileExtension: ext,
		OssObjectKey:  ossPath,
		FileHash:      fileHash,
		ParentId:      parentId,
		IsDeleted:     false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Thumbnail:     fileURL,
		URL:           fileURL,
	}
	return fi, nil
}

func (s *OSSService) generateObjectURL(objectKey string) string {
	return fmt.Sprintf("https://%s.%s/%s", s.bucket, s.endpoint, objectKey)
}

// 下载OSS文件

// 删除OSS文件
