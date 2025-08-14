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
	"go-cloud-storage/internal/pkg/utils"
	"io"
	"mime/multipart"
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

// 上传头像
func (s *OSSService) UploadAvatarFromStream(ctx context.Context, r io.Reader, userId int, header *multipart.FileHeader) (string, error) {
	// 限制文件大小
	if header.Size > 5*1024*1024 {
		return "", fmt.Errorf("头像大小不能超过5MB")
	}
	// 校验文件扩展名
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		return "", fmt.Errorf("不支持的头像格式")
	}

	// 固定路径，覆盖旧头像
	avatarPath := fmt.Sprintf("avatars/%d%s", userId, ext)

	// 读取文件数据
	data, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("读取头像失败: %w", err)
	}
	body := bytes.NewReader(data)
	request := &oss.PutObjectRequest{
		Bucket: oss.Ptr(s.bucket),
		Key:    oss.Ptr(avatarPath),
		Body:   body,
	}

	_, err = s.client.PutObject(ctx, request)
	if err != nil {
		return "", fmt.Errorf("上传头像失败: %w", err)
	}

	// 防缓存，加时间戳
	avatarURL := fmt.Sprintf("https://%s.%s/%s?t=%d", s.bucket, s.endpoint, avatarPath, time.Now().Unix())

	return avatarURL, nil
}

// DownloadFile 下载OSS文件
func (s *OSSService) DownloadFile(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	request := &oss.GetObjectRequest{
		Bucket: oss.Ptr(s.bucket),
		Key:    oss.Ptr(objectKey),
	}
	resp, err := s.client.GetObject(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("下载 OSS 文件失败: %w", err)
	}
	return resp.Body, nil
}

// 删除OSS文件
func (s *OSSService) DeleteFile(ctx context.Context, objectKey string) error {
	request := &oss.DeleteObjectRequest{
		Bucket: oss.Ptr(s.bucket),
		Key:    oss.Ptr(objectKey),
	}
	_, err := s.client.DeleteObject(ctx, request)
	if err != nil {
		return fmt.Errorf("删除 OSS 文件失败: %w", err)
	}
	return nil
}
