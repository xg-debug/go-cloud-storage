package minio

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"log"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go-cloud-storage/internal/models"
	"go-cloud-storage/internal/pkg/config"
	"go-cloud-storage/internal/pkg/utils"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioService struct {
	client   *minio.Client
	core     *minio.Core // Core 用于底层分片上传操作
	bucket   string
	endpoint string
	useSSL   bool
}

func NewMinioService(cfg *config.MinioConfig) (*MinioService, error) {
	if cfg.Endpoint == "" || cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return nil, errors.New("MinIO 配置不完整")
	}

	// 提取 Options 以便复用
	minioOpts := &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	}

	// 1.初始化标准 MinIO 客户端（用于普通上传、下载、管理）
	minioClient, err := minio.New(cfg.Endpoint, minioOpts)
	if err != nil {
		return nil, err
	}

	// 2.初始化 MinIO Core 客户端（用于分片上传）
	minioCore, err := minio.NewCore(cfg.Endpoint, minioOpts)
	if err != nil {
		return nil, fmt.Errorf("初始化 MinIOn Core 失败: %w", err)
	}

	// 自动检查并创建 Bucket
	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("检查 Bucket 失败: %w", err)
	}
	if !exists {
		err = minioClient.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{Region: cfg.Region})
		if err != nil {
			return nil, fmt.Errorf("创建 Bucket 失败: %w", err)
		}
	}

	// 设置 Bucket 策略为公开只读 (public-read)
	// 这一步是必须的，否则外部无法直接通过 URL 访问图片(403 Forbidden)
	policy := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {
					"AWS": ["*"]
				},
				"Action": ["s3:GetObject"],
				"Resource": ["arn:aws:s3:::%s/*"]
			}
		]
	}`, cfg.Bucket)

	if err := minioClient.SetBucketPolicy(ctx, cfg.Bucket, policy); err != nil {
		return nil, fmt.Errorf("设置 Bucket 策略失败: %w", err)
	}

	return &MinioService{
		client:   minioClient,
		core:     minioCore,
		bucket:   cfg.Bucket,
		endpoint: cfg.Endpoint,
		useSSL:   cfg.UseSSL,
	}, nil
}

func (s *MinioService) UploadFromStream(ctx context.Context, userId int, r io.Reader, fileName string, fileSize int64, parentId string) (*models.File, error) {
	if fileName == "" {
		return nil, errors.New("文件名不能为空")
	}
	// 获取文件扩展名(exe, txt等)
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fileName), "."))
	data, err := io.ReadAll(r)

	if err != nil {
		return nil, fmt.Errorf("读取文件失败：%w", err)
	}

	sum := sha256.Sum256(data)
	fileHash := fmt.Sprintf("%x", sum[:])

	objectKey := s.GenerateObjectKey(userId, parentId, fileName)
	body := bytes.NewReader(data)

	_, err = s.client.PutObject(ctx, s.bucket, objectKey, body, int64(len(data)), minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return nil, fmt.Errorf("上传到 MinIO 失败: %w", err)
	}

	fileURL := s.GenerateObjectURL(objectKey)

	pId := sql.NullString{
		String: parentId,
		Valid:  parentId != "",
	}

	newFile := &models.File{
		Id:            utils.NewUUID(),
		UserId:        userId,
		Name:          fileName,
		Size:          int64(len(data)),
		SizeStr:       utils.FormatFileSize(fileSize),
		IsDir:         false,
		FileExtension: ext,
		OssObjectKey:  objectKey,
		FileHash:      fileHash,
		ParentId:      pId,
		IsDeleted:     false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		FileURL:       fileURL,
		ThumbnailURL:  fileURL,
	}

	if thumbURL, err := s.generateThumbnailFromBytes(ctx, objectKey, ext, data); err == nil && thumbURL != "" {
		newFile.ThumbnailURL = thumbURL
	} else if err != nil {
		log.Printf("generate thumbnail failed (small upload): %v\n", err)
	}

	return newFile, nil
}

func (s *MinioService) UploadAvatarFromStream(ctx context.Context, r io.Reader, userId int, header *multipart.FileHeader) (string, error) {
	if header.Size > 5*1024*1024 {
		return "", fmt.Errorf("头像大小不能超过5MB")
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		return "", fmt.Errorf("不支持的头像格式")
	}

	avatarPath := fmt.Sprintf("avatars/%d%s", userId, ext)

	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	body := bytes.NewReader(data)

	_, err = s.client.PutObject(ctx, s.bucket, avatarPath, body, int64(len(data)), minio.PutObjectOptions{
		ContentType: header.Header.Get("Content-Type"),
	})

	if err != nil {
		return "", fmt.Errorf("上传头像失败: %w", err)
	}

	// 加时间戳放缓存
	return fmt.Sprintf("%s?t=%d", s.GenerateObjectURL(avatarPath), time.Now().Unix()), nil
}

func (s *MinioService) DownloadFile(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (s *MinioService) DeleteFile(ctx context.Context, objectKey string) error {
	return s.client.RemoveObject(ctx, s.bucket, objectKey, minio.RemoveObjectOptions{})
}

func (s *MinioService) DeleteFiles(ctx context.Context, objectKeys []string) error {
	objectsCh := make(chan minio.ObjectInfo)
	go func() {
		defer close(objectsCh)
		for _, key := range objectKeys {
			objectsCh <- minio.ObjectInfo{Key: key}
		}
	}()

	for err := range s.client.RemoveObjects(ctx, s.bucket, objectsCh, minio.RemoveObjectsOptions{}) {
		if err.Err != nil {
			return fmt.Errorf("删除对象失败: %w", &err.Err)
		}
	}
	return nil
}

// InitiateMultipartUpload 初始化分片上传，返回 UploadID
func (s *MinioService) InitiateMultipartUpload(ctx context.Context, objectKey string) (string, error) {
	// 使用 Core API
	uploadId, err := s.core.NewMultipartUpload(ctx, s.bucket, objectKey, minio.PutObjectOptions{
		// 可以根据文件名后缀自动推断 Content-Type，或者默认为 application/octet-stream
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return "", fmt.Errorf("初始化分片上传失败: %w", err)
	}
	return uploadId, nil
}

// UploadPart 上传单个分片，返回 minio.PartInfo，其中包含 ETag，这是完成上传所必须的
func (s *MinioService) UploadPart(ctx context.Context, objectKey, uploadId string, partNumber int, data []byte) (minio.ObjectPart, error) {
	// 限制分片大小，MinIO/S3 要求除最后一块外，每块至少 5MB
	// 这里不做强制校验，交由上层业务逻辑控制，但在实际调用 core 时如果太小可能会报错

	reader := bytes.NewReader(data)
	size := int64(len(data))

	part, err := s.core.PutObjectPart(ctx, s.bucket, objectKey, uploadId, partNumber, reader, size, minio.PutObjectPartOptions{})
	if err != nil {
		return minio.ObjectPart{}, fmt.Errorf("上传分片 %d 失败: %w", partNumber, err)
	}
	return part, nil
}

// CompleteMultipartUpload 完成分片上传，parts 参数必须包含所有分片的 PartNumber 和 ETag，且通常需要按 PartNumber 排序
func (s *MinioService) CompleteMultipartUpload(ctx context.Context, objectKey, uploadId string, parts []minio.CompletePart) (string, string, error) {
	// 执行合并
	uploadInfo, err := s.core.CompleteMultipartUpload(ctx, s.bucket, objectKey, uploadId, parts, minio.PutObjectOptions{})
	if err != nil {
		return "", "", fmt.Errorf("合并分片失败: %w", err)
	}
	// 生成最终的文件 URL
	fileURL := s.GenerateObjectURL(uploadInfo.Key)

	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(objectKey), "."))

	thumbnailURL, err := s.generateThumbnailFromObject(ctx, objectKey, ext)
	if err != nil {
		return fileURL, "", err
	}

	return fileURL, thumbnailURL, nil
}

// AbortMultipartUpload 取消分片上传
func (s *MinioService) AbortMultipartUpload(ctx context.Context, objectKey, uploadId string) error {
	err := s.core.AbortMultipartUpload(ctx, s.bucket, objectKey, uploadId)
	if err != nil {
		return fmt.Errorf("取消分片上传失败: %w", err)
	}
	return nil
}

func (s *MinioService) GenerateObjectKey(userId int, parentId, fileName string) string {
	fileId := utils.NewUUID()
	ossPath := fmt.Sprintf("files/%d", userId)
	if parentId != "" {
		ossPath = ossPath + "/" + parentId
	}
	ext := filepath.Ext(fileName)
	return fmt.Sprintf("%s/%s%s", ossPath, fileId, ext)
}

func (s *MinioService) GenerateObjectURL(objectKey string) string {
	protocol := "http"
	if s.useSSL {
		protocol = "https"
	}
	return fmt.Sprintf("%s://%s/%s/%s", protocol, s.endpoint, s.bucket, objectKey)
}

// generateThumbnailFromBytes 适用于小文件，直接复用内存中的data []byte
func (s *MinioService) generateThumbnailFromBytes(ctx context.Context, objectKey, ext string, data []byte) (string, error) {
	if isImageExtension(ext) {
		return s.generateImageThumbnail(ctx, objectKey, data)
	}
	if isVideoExtension(ext) {
		return s.generateVideoThumbnailFromBytes(ctx, objectKey, data)
	}
	return "", nil
}

// generateThumbnailFromObject 适用大文件，需要先下载
func (s *MinioService) generateThumbnailFromObject(ctx context.Context, objectKey, ext string) (string, error) {
	if isImageExtension(ext) {
		obj, err := s.client.GetObject(ctx, s.bucket, objectKey, minio.GetObjectOptions{})
		if err != nil {
			return "", err
		}
		defer obj.Close()
		data, err := io.ReadAll(obj)
		if err != nil {
			return "", err
		}
		return s.generateImageThumbnail(ctx, objectKey, data)
	}
	if isVideoExtension(ext) {
		return s.generateVideoThumbnailFromObject(ctx, objectKey)
	}
	return "", nil
}

func (s *MinioService) generateImageThumbnail(ctx context.Context, objectKey string, data []byte) (string, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	resized := resizeImage(img, 360)
	var buf bytes.Buffer
	if err := png.Encode(&buf, resized); err != nil {
		return "", err
	}
	return s.uploadThumbnail(ctx, objectKey, buf.Bytes(), "image/png")
}

func (s *MinioService) generateVideoThumbnailFromBytes(ctx context.Context, objectKey string, data []byte) (string, error) {
	tmpFile, err := os.CreateTemp("", "upload-video-*"+filepath.Ext(objectKey))
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return "", err
	}
	tmpFile.Close()
	return s.generateVideoThumbnailFromPath(ctx, objectKey, tmpFile.Name())
}

func (s *MinioService) generateVideoThumbnailFromObject(ctx context.Context, objectKey string) (string, error) {
	tmpFile, err := os.CreateTemp("", "merged-video-*"+filepath.Ext(objectKey))
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpFile.Name())
	obj, err := s.client.GetObject(ctx, s.bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		tmpFile.Close()
		return "", err
	}
	defer obj.Close()
	if _, err := io.Copy(tmpFile, obj); err != nil {
		tmpFile.Close()
		return "", err
	}
	tmpFile.Close()
	return s.generateVideoThumbnailFromPath(ctx, objectKey, tmpFile.Name())
}

func (s *MinioService) generateVideoThumbnailFromPath(ctx context.Context, objectKey, videoPath string) (string, error) {
	frameFile, err := os.CreateTemp("", "video-thumb-*.jpg")
	if err != nil {
		return "", err
	}
	frameFile.Close()
	defer os.Remove(frameFile.Name())

	cmd := exec.CommandContext(ctx, "ffmpeg", "-y", "-i", videoPath, "-ss", "00:00:01", "-frames:v", "1", "-vf", "scale=360:-1", frameFile.Name())
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("ffmpeg thumbnail error: %w, output: %s", err, string(output))
	}

	data, err := os.ReadFile(frameFile.Name())
	if err != nil {
		return "", err
	}
	return s.uploadThumbnail(ctx, objectKey, data, "image/jpeg")
}

func (s *MinioService) uploadThumbnail(ctx context.Context, objectKey string, data []byte, contentType string) (string, error) {
	thumbKey := thumbnailObjectKey(objectKey)
	reader := bytes.NewReader(data)
	_, err := s.client.PutObject(ctx, s.bucket, thumbKey, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}
	return s.GenerateObjectURL(thumbKey), nil
}

func thumbnailObjectKey(objectKey string) string {
	ext := filepath.Ext(objectKey)
	base := strings.TrimSuffix(objectKey, ext)
	return fmt.Sprintf("%s_thumb.jpg", base)
}

func resizeImage(src image.Image, maxWidth int) image.Image {
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= maxWidth {
		return src
	}
	scale := float64(maxWidth) / float64(width)
	newHeight := int(float64(height) * scale)
	if newHeight <= 0 {
		newHeight = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, maxWidth, newHeight))
	for y := 0; y < newHeight; y++ {
		for x := 0; x < maxWidth; x++ {
			srcX := int(float64(x) / scale)
			srcY := int(float64(y) / scale)
			if srcX >= width {
				srcX = width - 1
			}
			if srcY >= height {
				srcY = height - 1
			}
			dst.Set(x, y, src.At(bounds.Min.X+srcX, bounds.Min.Y+srcY))
		}
	}
	return dst
}

func isImageExtension(ext string) bool {
	switch strings.ToLower(ext) {
	case "jpg", "jpeg", "png", "gif", "bmp", "webp":
		return true
	}
	return false
}

func isVideoExtension(ext string) bool {
	switch strings.ToLower(ext) {
	case "mp4", "mov", "avi", "mkv", "flv", "wmv", "webm":
		return true
	}
	return false
}
