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
	"mime"
	"mime/multipart"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go-cloud-storage/backend/internal/models"
	"go-cloud-storage/backend/pkg/config"
	"go-cloud-storage/backend/pkg/utils"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
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

	// 安全加固：Bucket 必须为私有（移除历史遗留的 public-read 策略）。
	// 所有对外访问（预览/下载/缩略图/分享）一律通过预签名 URL，杜绝永久直链外泄。
	if err := minioClient.SetBucketPolicy(ctx, cfg.Bucket, ""); err != nil {
		return nil, fmt.Errorf("清除 Bucket 公开策略失败: %w", err)
	}

	// 生命周期规则：自动中止超过 1 天未完成的 Multipart Upload，
	// 防止 Redis 会话丢失/进程崩溃后产生孤儿分片占用存储。
	if err := applyAbortMultipartLifecycle(ctx, minioClient, cfg.Bucket); err != nil {
		// 部分对象存储不支持该规则，失败仅告警不阻断启动
		log.Printf("warn: failed to set abort-multipart lifecycle rule: %v\n", err)
	}

	return &MinioService{
		client:   minioClient,
		core:     minioCore,
		bucket:   cfg.Bucket,
		endpoint: cfg.Endpoint,
		useSSL:   cfg.UseSSL,
	}, nil
}

// applyAbortMultipartLifecycle 配置生命周期规则，自动清理未完成的 Multipart Upload。
func applyAbortMultipartLifecycle(ctx context.Context, client *minio.Client, bucket string) error {
	cfg := lifecycle.NewConfiguration()
	cfg.Rules = []lifecycle.Rule{
		{
			ID:     "abort-incomplete-multipart-uploads",
			Status: "Enabled",
			AbortIncompleteMultipartUpload: lifecycle.AbortIncompleteMultipartUpload{
				DaysAfterInitiation: 1,
			},
		},
	}
	return client.SetBucketLifecycle(ctx, bucket, cfg)
}

// UploadFromStream 小文件上传 (流式)
// fileHash 由前端计算传入（SHA-256），后端不做重复计算，确保秒传一致性
func (s *MinioService) UploadFromStream(ctx context.Context, userId int, r io.Reader, fileName string, fileSize int64, fileHash, parentId string) (*models.File, error) {
	if fileName == "" {
		return nil, errors.New("文件名不能为空")
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fileName), "."))

	objectKey := s.GenerateObjectKey(userId, parentId, fileName)

	_, err := s.client.PutObject(ctx, s.bucket, objectKey, r, fileSize, minio.PutObjectOptions{
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
		Size:          fileSize,
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

func (s *MinioService) GetObjectInfo(ctx context.Context, objectKey string) (int64, error) {
	info, err := s.client.StatObject(ctx, s.bucket, objectKey, minio.StatObjectOptions{})
	if err != nil {
		return 0, err
	}
	return info.Size, nil
}

func (s *MinioService) ComputeObjectSHA256(ctx context.Context, objectKey string) (string, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return "", err
	}
	defer obj.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, obj); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func (s *MinioService) DownloadFile(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (s *MinioService) DownloadFileRange(ctx context.Context, objectKey string, start, end int64) (io.ReadCloser, error) {
	opts := minio.GetObjectOptions{}
	if err := opts.SetRange(start, end); err != nil {
		return nil, err
	}
	obj, err := s.client.GetObject(ctx, s.bucket, objectKey, opts)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (s *MinioService) PresignedGetURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	reqParams := make(url.Values)
	u, err := s.client.PresignedGetObject(ctx, s.bucket, objectKey, expiry, reqParams)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (s *MinioService) PresignedDownloadURL(ctx context.Context, objectKey, fileName string, expiry time.Duration) (string, error) {
	reqParams := make(url.Values)
	if fileName != "" {
		reqParams.Set("response-content-disposition", mime.FormatMediaType("attachment", map[string]string{"filename": fileName}))
	}
	u, err := s.client.PresignedGetObject(ctx, s.bucket, objectKey, expiry, reqParams)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (s *MinioService) PutObjectStream(ctx context.Context, objectKey string, r io.Reader, size int64, contentType string) (int64, error) {
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}
	opts := minio.PutObjectOptions{ContentType: contentType}
	if size < 0 {
		opts.PartSize = 10 * 1024 * 1024
	}
	info, err := s.client.PutObject(ctx, s.bucket, objectKey, r, size, opts)
	if err != nil {
		return 0, err
	}
	return info.Size, nil
}

func (s *MinioService) DeleteFile(ctx context.Context, objectKey string) error {
	return s.client.RemoveObject(ctx, s.bucket, objectKey, minio.RemoveObjectOptions{})
}

func (s *MinioService) DeleteObjectWithThumbnail(ctx context.Context, objectKey string) error {
	if objectKey == "" {
		return nil
	}
	if err := s.DeleteFile(ctx, objectKey); err != nil {
		return err
	}
	_ = s.DeleteThumbnailForObject(ctx, objectKey)
	return nil
}

func (s *MinioService) DeleteThumbnailForObject(ctx context.Context, objectKey string) error {
	if objectKey == "" {
		return nil
	}
	return s.DeleteFile(ctx, ThumbnailObjectKey(objectKey))
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
			return fmt.Errorf("删除对象失败: %w", err.Err)
		}
	}
	return nil
}

// CopyObject 复制 MinIO 对象（服务端复制，不下载）
func (s *MinioService) CopyObject(ctx context.Context, srcKey, dstKey string) error {
	src := minio.CopySrcOptions{Bucket: s.bucket, Object: srcKey}
	dst := minio.CopyDestOptions{Bucket: s.bucket, Object: dstKey}
	_, err := s.client.CopyObject(ctx, dst, src)
	return err
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

// UploadPart 上传单个分片，流式上传同时计算 SHA-256 用于完整性校验
// expectedHash 为空则跳过校验；chunkSize 为分片实际字节数
func (s *MinioService) UploadPart(ctx context.Context, objectKey, uploadId string, partNumber int, r io.Reader, chunkSize int64, expectedHash string) (minio.ObjectPart, string, error) {
	hash := sha256.New()
	tee := io.TeeReader(r, hash)

	part, err := s.core.PutObjectPart(ctx, s.bucket, objectKey, uploadId, partNumber, tee, chunkSize, minio.PutObjectPartOptions{})
	if err != nil {
		return minio.ObjectPart{}, "", fmt.Errorf("上传分片 %d 失败: %w", partNumber, err)
	}

	computedHash := fmt.Sprintf("%x", hash.Sum(nil))

	// 如果前端传了 expectedHash 就校验
	if expectedHash != "" && computedHash != expectedHash {
		return minio.ObjectPart{}, computedHash, fmt.Errorf(
			"分片 %d hash 校验失败: got=%s expected=%s", partNumber, shortHash(computedHash), shortHash(expectedHash))
	}

	return part, computedHash, nil
}

func shortHash(hash string) string {
	if len(hash) <= 16 {
		return hash
	}
	return hash[:16]
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
	return fileURL, fileURL, nil
}

func (s *MinioService) GenerateThumbnailForObject(ctx context.Context, objectKey string) (string, error) {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(objectKey), "."))
	if thumb, err := s.generateThumbnailFromObject(ctx, objectKey, ext); err == nil && thumb != "" {
		return thumb, nil
	} else if err != nil {
		log.Printf("generate thumbnail failed (chunk upload): %v\n", err)
		return "", err
	}
	return "", nil
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

// PresignedGetPreviewURL 生成带 inline 处置的预签名预览 URL
func (s *MinioService) PresignedGetPreviewURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	reqParams := make(url.Values)
	reqParams.Set("response-content-disposition", "inline")
	u, err := s.client.PresignedGetObject(ctx, s.bucket, objectKey, expiry, reqParams)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

// PresignThumbnailURL 生成缩略图对象的预签名 URL（私有桶下缩略图同样需要签名访问）。
// 缩略图可能尚未生成，返回的 URL 可能 404，由前端兜底显示占位图。
func (s *MinioService) PresignThumbnailURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	if objectKey == "" {
		return "", nil
	}
	u, err := s.client.PresignedGetObject(ctx, s.bucket, ThumbnailObjectKey(objectKey), expiry, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

// PresignAvatarURL 从数据库中存储的头像 URL 解析对象键并生成预签名 URL。
// 解析失败（例如历史遗留的第三方存储 URL）时原样返回，保证兼容。
func (s *MinioService) PresignAvatarURL(ctx context.Context, storedURL string, expiry time.Duration) string {
	if storedURL == "" {
		return ""
	}
	key, ok := s.parseObjectKeyFromURL(storedURL)
	if !ok {
		return storedURL
	}
	u, err := s.client.PresignedGetObject(ctx, s.bucket, key, expiry, nil)
	if err != nil {
		return storedURL
	}
	return u.String()
}

// parseObjectKeyFromURL 从 `http(s)://endpoint/bucket/objectKey?...` 中提取 objectKey。
func (s *MinioService) parseObjectKeyFromURL(storedURL string) (string, bool) {
	u, err := url.Parse(storedURL)
	if err != nil || u.Host == "" {
		return "", false
	}
	trimmed := strings.TrimPrefix(u.Path, "/")
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) != 2 || parts[0] != s.bucket || parts[1] == "" {
		return "", false
	}
	return parts[1], true
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
		// 限制最多读取 32MB 用于缩略图生成
		lr := io.LimitReader(obj, 32*1024*1024)
		data, err := io.ReadAll(lr)
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
	data, err := s.extractVideoThumbnailData(ctx, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	return s.uploadThumbnail(ctx, objectKey, data, "image/jpeg")
}

func (s *MinioService) generateVideoThumbnailFromObject(ctx context.Context, objectKey string) (string, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return "", err
	}
	defer obj.Close()

	data, err := s.extractVideoThumbnailData(ctx, obj)
	if err != nil {
		return "", err
	}
	return s.uploadThumbnail(ctx, objectKey, data, "image/jpeg")
}

func (s *MinioService) extractVideoThumbnailData(ctx context.Context, r io.Reader) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-i", "pipe:0",
		"-ss", "00:00:01",
		"-frames:v", "1",
		"-vf", "scale=360:-1",
		"-f", "image2",
		"pipe:1",
	)
	cmd.Stdin = r

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg thumbnail error: %w, output: %s", err, stderr.String())
	}

	if stdout.Len() == 0 {
		return nil, errors.New("ffmpeg produced no thumbnail output")
	}

	return stdout.Bytes(), nil
}

func (s *MinioService) uploadThumbnail(ctx context.Context, objectKey string, data []byte, contentType string) (string, error) {
	thumbKey := ThumbnailObjectKey(objectKey)
	reader := bytes.NewReader(data)
	_, err := s.client.PutObject(ctx, s.bucket, thumbKey, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}
	return s.GenerateObjectURL(thumbKey), nil
}

func ThumbnailObjectKey(objectKey string) string {
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
