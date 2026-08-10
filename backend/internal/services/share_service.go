package services

import (
	"context"
	"errors"
	"fmt"
	miniosrv "go-cloud-storage/backend/infrastructure/minio"
	"log/slog"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"go-cloud-storage/backend/internal/models"
	"go-cloud-storage/backend/internal/repositories"
	"go-cloud-storage/backend/pkg/filetypes"
)

var (
	ErrExtractCodeWrong = errors.New("提取码错误")
	ErrShareExpired     = errors.New("分享已过期")
	ErrShareNotFound    = errors.New("分享不存在")
)

type ShareService interface {
	CreateShare(userId int, fileId string, expireDays int, extractionCode string) (*models.Share, error)
	GetUserShares(userId int, page, pageSize int) ([]*ShareItem, int64, error)
	GetShareDetail(userId int, shareId int) (*ShareDetail, error)
	CancelShare(userId int, shareId int) error
	AccessShare(shareToken string, extractionCode string) (*ShareAccessResponse, error)
	DownloadSharedFile(shareToken string, extractionCode string) (string, error)
	UpdateShare(shareID int, userID int, extractionCode string, expireDays int) error
}

type shareService struct {
	shareRepo repositories.ShareRepository
	fileRepo  repositories.FileRepository
	minio     *miniosrv.MinioService
}

func NewShareService(shareRepo repositories.ShareRepository, fileRepo repositories.FileRepository, minio *miniosrv.MinioService) ShareService {
	return &shareService{
		shareRepo: shareRepo,
		fileRepo:  fileRepo,
		minio:     minio,
	}
}

func (s *shareService) CreateShare(userId int, fileId string, expireDays int, extractionCode string) (*models.Share, error) {
	// 1. 检查文件是否存在
	file, err := s.fileRepo.GetUserFileByID(userId, fileId)
	if err != nil || file == nil {
		return nil, errors.New("文件不存在")
	}

	// 2. 检查是否已经分享过
	isShared, existShare := s.shareRepo.IsShared(fileId)
	if isShared {
		return existShare, nil
	}

	// 3. 生成分享Token
	shareToken := uuid.New().String()

	// 4. 计算过期时间
	var expireTime *time.Time
	if expireDays > 0 {
		t := time.Now().AddDate(0, 0, expireDays)
		expireTime = &t
	}

	// 5. 创建分享记录
	share := &models.Share{
		UserId:         userId,
		FileId:         fileId,
		ShareToken:     shareToken,
		ExtractionCode: &extractionCode,
		ExpireTime:     expireTime,
		AccessCount:    0,
		DownloadCount:  0,
	}

	return s.shareRepo.CreateShare(share)
}

type ShareItem struct {
	Id            int       `json:"id"`
	FileName      string    `json:"fileName"`
	FileSize      int64     `json:"fileSize"`
	FileType      string    `json:"fileType"`
	ShareToken    string    `json:"shareToken"`
	ShareUrl      string    `json:"shareUrl"`
	ExtractCode   string    `json:"extractCode"`
	ExpireAt      string    `json:"expireAt"`
	CreatedAt     time.Time `json:"createdAt"`
	AccessCount   int       `json:"accessCount"`
	DownloadCount int       `json:"downloadCount"`
	Status        string    `json:"status"` // active, expired
}

func (s *shareService) GetUserShares(userId int, page, pageSize int) ([]*ShareItem, int64, error) {
	shares, total, err := s.shareRepo.GetUserShares(userId, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	if len(shares) == 0 {
		return []*ShareItem{}, total, nil
	}

	// 批量查询：收集所有 fileId，一次查询获取所有文件
	fileIds := make([]string, len(shares))
	for i, share := range shares {
		fileIds[i] = share.FileId
	}

	files, err := s.fileRepo.GetFileByIds(fileIds)
	if err != nil {
		return nil, 0, err
	}

	fileMap := make(map[string]models.File, len(files))
	for _, f := range files {
		fileMap[f.Id] = f
	}

	var result []*ShareItem
	for _, share := range shares {
		file, ok := fileMap[share.FileId]
		if !ok || file.UserId != userId || file.IsDeleted {
			continue
		}

		status := "active"
		if share.ExpireTime != nil && share.ExpireTime.Before(time.Now()) {
			status = "expired"
		}

		extractCode := share.GetExtractionCode()

		result = append(result, &ShareItem{
			Id:            share.Id,
			FileName:      file.Name,
			FileSize:      file.Size,
			FileType:      filetypes.Category(file.FileExtension),
			ShareToken:    share.ShareToken,
			ShareUrl:      fmt.Sprintf("/s/%s", share.ShareToken),
			ExtractCode:   extractCode,
			ExpireAt:      StatusText(share.ExpireTime),
			CreatedAt:     share.CreatedAt,
			AccessCount:   share.AccessCount,
			DownloadCount: share.DownloadCount,
			Status:        status,
		})
	}

	return result, total, nil
}

func StatusText(ExpireAt *time.Time) string {

	if ExpireAt == nil {
		return "永久有效"
	}

	diff := time.Until(*ExpireAt).Hours() / 24

	if diff > 0 {
		return fmt.Sprintf("%d天后过期", int(math.Ceil(diff)))
	}

	return "已过期"
}

type ShareDetail struct {
	Id            int        `json:"id"`
	FileName      string     `json:"fileName"`
	FileSize      int64      `json:"fileSize"`
	FileType      string     `json:"fileType"`
	ShareToken    string     `json:"shareToken"`
	ShareUrl      string     `json:"shareUrl"`
	ExtractCode   string     `json:"extractCode"`
	CreatedAt     time.Time  `json:"createdAt"`
	ExpireAt      *time.Time `json:"expireAt"`
	AccessCount   int        `json:"accessCount"`
	DownloadCount int        `json:"downloadCount"`
}

func (s *shareService) GetShareDetail(userId int, shareId int) (*ShareDetail, error) {
	share, err := s.shareRepo.GetShareByID(shareId)
	if err != nil {
		return nil, err
	}

	if share.UserId != userId {
		return nil, errors.New("无权查看此分享")
	}

	file, err := s.fileRepo.GetUserFileByID(userId, share.FileId)
	if err != nil {
		return nil, errors.New("文件不存在")
	}

	extractCode := share.GetExtractionCode()

	return &ShareDetail{
		Id:            share.Id,
		FileName:      file.Name,
		FileSize:      file.Size,
		FileType:      filetypes.Category(file.FileExtension),
		ShareToken:    share.ShareToken,
		ShareUrl:      fmt.Sprintf("/s/%s", share.ShareToken),
		ExtractCode:   extractCode,
		CreatedAt:     share.CreatedAt,
		ExpireAt:      share.ExpireTime,
		AccessCount:   share.AccessCount,
		DownloadCount: share.DownloadCount,
	}, nil
}

func (s *shareService) CancelShare(userId int, shareId int) error {
	share, err := s.shareRepo.GetShareByID(shareId)
	if err != nil {
		return err
	}

	if share.UserId != userId {
		return errors.New("无权取消此分享")
	}

	return s.shareRepo.Delete(nil, shareId)
}

type ShareAccessResponse struct {
	ShareToken       string     `json:"shareToken"`
	FileName         string     `json:"fileName"`
	FileSize         int64      `json:"fileSize"`
	FileType         string     `json:"fileType"`
	UpdatedAt        time.Time  `json:"updatedAt"`
	ExpireAt         *time.Time `json:"expireAt"`
	DownloadUrl      string     `json:"downloadUrl"`
	FileURL          string     `json:"fileUrl"`
	ThumbnailURL     string     `json:"thumbnailUrl"`
	CanPreview       bool       `json:"canPreview"`
	PreviewType      string     `json:"previewType"`
	OfficePreviewURL string     `json:"officePreviewUrl"`
	NeedCode         bool       `json:"needCode"`
}

func (s *shareService) AccessShare(shareToken string, inputCode string) (*ShareAccessResponse, error) {
	share, err := s.shareRepo.GetShareByToken(shareToken)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("分享链接不存在或已失效")
		}
		return nil, err
	}

	if share.ExpireTime != nil && share.ExpireTime.Before(time.Now()) {
		return nil, errors.New("分享链接已过期")
	}

	file, err := s.fileRepo.GetUserFileByID(share.UserId, share.FileId)
	if err != nil {
		return nil, errors.New("文件不存在")
	}

	canPreview, previewType := filetypes.Previewable(file.FileExtension)
	downloadURL := fmt.Sprintf("/s/%s/download", shareToken)

	if share.GetExtractionCode() != "" && inputCode == "" {
		return &ShareAccessResponse{
			ShareToken:  shareToken,
			FileName:    file.Name,
			FileSize:    file.Size,
			FileType:    filetypes.Category(file.FileExtension),
			UpdatedAt:   share.UpdatedAt,
			ExpireAt:    share.ExpireTime,
			CanPreview:  canPreview,
			PreviewType: previewType,
			NeedCode:    true,
		}, nil
	}

	if share.GetExtractionCode() != "" && share.GetExtractionCode() != inputCode {
		return nil, ErrExtractCodeWrong
	}

	if err := s.shareRepo.IncrementAccessCount(share.Id); err != nil {
		slog.Error("更新访问计数失败", "error", err, "shareId", share.Id)
	}

	previewURL := file.FileURL
	if canPreview && s.minio != nil {
		if u, err := s.minio.PresignedGetPreviewURL(context.Background(), file.OssObjectKey, 30*time.Minute); err == nil {
			previewURL = u
		}
	}
	officePreviewURL := buildShareOfficePreviewURL(previewURL)

	return &ShareAccessResponse{
		ShareToken:       shareToken,
		FileName:         file.Name,
		FileSize:         file.Size,
		FileType:         filetypes.Category(file.FileExtension),
		UpdatedAt:        share.UpdatedAt,
		ExpireAt:         share.ExpireTime,
		DownloadUrl:      downloadURL,
		FileURL:          previewURL,
		ThumbnailURL:     file.ThumbnailURL,
		CanPreview:       canPreview,
		PreviewType:      previewType,
		OfficePreviewURL: officePreviewURL,
		NeedCode:         false,
	}, nil
}

func (s *shareService) DownloadSharedFile(shareToken string, inputCode string) (string, error) {
	// Re-verify
	share, err := s.shareRepo.GetShareByToken(shareToken)
	if err != nil {
		return "", errors.New("分享不存在")
	}
	if share.ExpireTime != nil && share.ExpireTime.Before(time.Now()) {
		return "", errors.New("分享已过期")
	}
	if share.GetExtractionCode() != "" && share.GetExtractionCode() != inputCode {
		return "", ErrExtractCodeWrong
	}

	file, err := s.fileRepo.GetUserFileByID(share.UserId, share.FileId)
	if err != nil {
		return "", errors.New("文件不存在")
	}

	if err := s.shareRepo.IncrementDownloadCount(share.Id); err != nil {
		slog.Error("更新下载计数失败", "error", err, "shareId", share.Id)
	}

	if s.minio != nil {
		return s.minio.PresignedDownloadURL(context.Background(), file.OssObjectKey, file.Name, 10*time.Minute)
	}
	return file.FileURL, nil
}

func (s *shareService) UpdateShare(shareID int, userID int, extractionCode string, expireDays int) error {
	share, err := s.shareRepo.GetShareByID(shareID)
	if err != nil {
		return errors.New("分享不存在")
	}

	if share.UserId != userID {
		return errors.New("无权限操作此分享")
	}

	// Calculate expireTime
	var expireTime *time.Time
	if expireDays > 0 {
		expire := time.Now().AddDate(0, 0, expireDays)
		expireTime = &expire
	}
	// If expireDays == 0, expireTime is nil (permanent)

	// Handle extractionCode
	var code *string
	if extractionCode != "" {
		code = &extractionCode
	} else {
		// If empty string, pass pointer to empty string? Or nil?
		// Logic: If user wants to remove code, they send empty string.
		// Repo uses *string. If nil, it might ignore update?
		// Wait, repo: updates := map... "extraction_code": code.
		// If code is nil, GORM map update with nil value -> sets to NULL.
		// So if extractionCode is "", code should be nil?
		// No, if I want to set it to NULL (no code), I should pass nil?
		// Or if I pass pointer to "", does it set empty string?
		// DB column is likely varchar. Empty string is valid.
		// But usually we treat empty string as "no code".
		// Let's assume we pass pointer to empty string if we want empty string.
		// If we want NULL, we pass nil.
		// Let's assume empty string means "no code" in logic, so in DB it can be NULL or "".
		// GORM: if I pass `nil` to map, it updates column to NULL.
		// If I pass `&""`, it updates to `""`.
		// Let's stick to `nil` for no code.
		code = nil
	}

	// Wait, if extractionCode is provided (not empty), we use it.
	if extractionCode != "" {
		code = &extractionCode
	}

	return s.shareRepo.UpdateShareInfo(shareID, code, expireTime)
}

func buildShareOfficePreviewURL(fileURL string) string {
	if strings.TrimSpace(fileURL) == "" {
		return ""
	}
	encoded := url.QueryEscape(fileURL)
	return "https://view.officeapps.live.com/op/view.aspx?src=" + encoded + "&wdAr=1.3333333333333333"
}
