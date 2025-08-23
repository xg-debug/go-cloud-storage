package aliyunoss

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"go-cloud-storage/internal/models"
	"go-cloud-storage/internal/pkg/config"
	"go-cloud-storage/internal/pkg/utils"
	"io"
	"log"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)

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
	ossCfg := oss.LoadDefaultConfig().WithCredentialsProvider(credProvider).WithRegion(cfg.Region).WithEndpoint(cfg.EndPoint)
	client := oss.NewClient(ossCfg)

	return &OSSService{
		client:   client,
		bucket:   cfg.Bucket,
		endpoint: cfg.EndPoint,
	}, nil
}

// UploadFromStream 上传文件
// 注意：这里会把文件读入内存（io.ReadAll）。（适合小文件）
func (s *OSSService) UploadFromStream(ctx context.Context, r io.Reader, fileName string, userId int, parentId string, maxSize int64) (*models.File, error) {
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

	pId := sql.NullString{
		String: parentId,
		Valid:  parentId != "",
	}

	file := &models.File{
		Id:            fileId,
		UserId:        userId,
		Name:          fileName,
		Size:          int64(len(data)),
		SizeStr:       utils.FormatFileSize(int64(len(data))),
		IsDir:         false,
		FileExtension: ext,
		OssObjectKey:  ossPath,
		FileHash:      fileHash,
		ParentId:      pId,
		IsDeleted:     false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		FileURL:       fileURL,
		ThumbnailURL:  fileURL,
	}
	return file, nil
}

func (s *OSSService) generateObjectURL(objectKey string) string {
	return fmt.Sprintf("https://%s.%s/%s", s.bucket, s.endpoint, objectKey)
}

// GenerateObjectURL 生成对象URL（公开方法）
func (s *OSSService) GenerateObjectURL(objectKey string) string {
	return s.generateObjectURL(objectKey)
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

// DownloadFile 下载 OSS文件
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

// DeleteFile 删除 OSS文件
func (s *OSSService) DeleteFile(ctx context.Context, objectKey string) error {
	request := &oss.DeleteObjectRequest{
		Bucket: oss.Ptr(s.bucket),
		Key:    oss.Ptr(objectKey),
	}
	result, err := s.client.DeleteObject(ctx, request)
	if err != nil {
		return fmt.Errorf("删除 OSS 文件失败: %w", err)
	}
	// 打印删除对象的结果
	log.Printf("delete objects result:%#v", result)
	return nil
}

// DeleteFiles 删除指定的多个 OSS 文件。
// 接收文件的 ObjectKey 列表，构造批量删除请求并调用 OSS 客户端执行删除。
func (s *OSSService) DeleteFiles(ctx context.Context, objectKeys []string) error {
	if len(objectKeys) == 0 {
		return nil
	}

	var deleteObjects []oss.DeleteObject
	for _, key := range objectKeys {
		deleteObjects = append(deleteObjects, oss.DeleteObject{Key: oss.Ptr(key)})
	}

	// 创建删除多个对象的请求
	request := &oss.DeleteMultipleObjectsRequest{
		Bucket:  oss.Ptr(s.bucket),
		Objects: deleteObjects, // 要删除的对象列表
	}

	// 执行删除多个对象的操作并处理结果
	_, err := s.client.DeleteMultipleObjects(ctx, request)
	if err != nil {
		return fmt.Errorf("删除多个 OSS 文件失败: %v", err)
	}

	return nil
}

// InitiateMultipartUpload 初始化分片上传，返回 uploadId
func (s *OSSService) InitiateMultipartUpload(ctx context.Context, objectKey string) (string, error) {
	req := &oss.InitiateMultipartUploadRequest{
		Bucket: oss.Ptr(s.bucket),
		Key:    oss.Ptr(objectKey),
	}
	resp, err := s.client.InitiateMultipartUpload(ctx, req)
	if err != nil {
		return "", fmt.Errorf("初始化分片上传失败: %w", err)
	}
	return *resp.UploadId, nil
}

// GeneratePresignedURL 生成某个分片的预签名
func (s *OSSService) GeneratePresignedURL(ctx context.Context, objectKey, uploadId string, partNumber int, expire time.Duration) (string, error) {
	signedResult, err := s.client.Presign(ctx, &oss.UploadPartRequest{
		Bucket:     &s.bucket,
		Key:        &objectKey,
		UploadId:   &uploadId,
		PartNumber: int32(partNumber),
	}, oss.PresignExpiration(time.Now().Add(expire)))
	if err != nil {
		return "", err
	}
	return signedResult.URL, nil
}

// CompleteMultipartUpload 完成分片上传
func (s *OSSService) CompleteMultipartUpload(ctx context.Context, objectKey, uploadId string, parts []oss.UploadPart) error {
	_, err := s.client.CompleteMultipartUpload(ctx, &oss.CompleteMultipartUploadRequest{
		Bucket:   &s.bucket,
		Key:      &objectKey,
		UploadId: &uploadId,
		CompleteMultipartUpload: &oss.CompleteMultipartUpload{
			Parts: parts,
		},
	})
	return err
}

// 下面方法弃用--不再采用后端上传分片，而是前端上传，后端生成预签名URL这种方式。
// UploadPart 上传单个分片
//func (s *OSSService) UploadPart(ctx context.Context, objectKey, uploadId string, partNumber int, data []byte) (string, error) {
//	fmt.Printf("UploadPart - objectKey: %s, uploadId: %s, partNumber: %d, dataSize: %d\n", objectKey, uploadId, partNumber, len(data))
//
//	req := &oss.UploadPartRequest{
//		Bucket:     oss.Ptr(s.bucket),
//		Key:        oss.Ptr(objectKey),
//		UploadId:   oss.Ptr(uploadId),     // 上传Id
//		PartNumber: int32(partNumber),     // 分片编号
//		Body:       bytes.NewReader(data), // 上传数据
//		ProgressFn: func(increment, transferred, total int64) {
//			fmt.Printf("increment:%v, transferred:%v, total:%v\n", increment, transferred, total)
//		}, // 进度回调函数，用于显示上传进度
//	}
//
//	fmt.Printf("UploadPart - Request: Bucket=%s, Key=%s, UploadId=%s, PartNumber=%d\n",
//		*req.Bucket, *req.Key, *req.UploadId, req.PartNumber)
//
//	resp, err := s.client.UploadPart(ctx, req)
//	if err != nil {
//		return "", fmt.Errorf("上传分片失败: %w", err)
//	}
//
//	etag := strings.Trim(*resp.ETag, `"`)
//	return etag, nil
//}

// CompleteMultipartUpload 合并所有分片
//func (s *OSSService) CompleteMultipartUpload(ctx context.Context, objectKey, uploadId string, parts []oss.UploadPart) error {
//	req := &oss.CompleteMultipartUploadRequest{
//		Bucket:   oss.Ptr(s.bucket),
//		Key:      oss.Ptr(objectKey),
//		UploadId: oss.Ptr(uploadId),
//		CompleteMultipartUpload: &oss.CompleteMultipartUpload{
//			Parts: parts,
//		},
//	}
//	_, err := s.client.CompleteMultipartUpload(ctx, req)
//	if err != nil {
//		return fmt.Errorf("完成分片上传失败: %w", err)
//	}
//	return nil
//}
