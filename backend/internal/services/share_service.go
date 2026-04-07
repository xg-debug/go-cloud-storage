package services

import (
	"errors"
	"fmt"
	"go-cloud-storage/backend/internal/models"
	"go-cloud-storage/backend/internal/repositories"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ShareService interface {
	CreateShare(userId int, fileId string, expireDays int, extractionCode string) (*models.Share, error)
	GetUserShares(userId int) ([]*ShareItem, error)
	GetShareDetail(userId int, shareId int) (*ShareDetail, error)
	CancelShare(userId int, shareId int) error
	AccessShare(shareToken string, extractionCode string) (*ShareAccessResponse, error)
	DownloadSharedFile(shareToken string, extractionCode string) (string, error)
	UpdateShare(shareID int, userID int, extractionCode string, expireDays int) error
}

type shareService struct {
	shareRepo repositories.ShareRepository
	fileRepo  repositories.FileRepository
}

func NewShareService(shareRepo repositories.ShareRepository, fileRepo repositories.FileRepository) ShareService {
	return &shareService{
		shareRepo: shareRepo,
		fileRepo:  fileRepo,
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

func (s *shareService) GetUserShares(userId int) ([]*ShareItem, error) {
	shares, err := s.shareRepo.GetUserShares(userId)
	if err != nil {
		return nil, err
	}

	var result []*ShareItem
	for _, share := range shares {
		file, err := s.fileRepo.GetFileById(share.FileId)
		if err != nil {
			continue // Skip if file deleted
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
			FileType:      getFileTypeFromExtension(file.FileExtension),
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

	return result, nil
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

	file, err := s.fileRepo.GetFileById(share.FileId)
	if err != nil {
		return nil, errors.New("文件不存在")
	}

	extractCode := share.GetExtractionCode()

	return &ShareDetail{
		Id:            share.Id,
		FileName:      file.Name,
		FileSize:      file.Size,
		FileType:      getFileTypeFromExtension(file.FileExtension),
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

	canPreview, previewType := getSharePreviewType(file.FileExtension)
	officePreviewURL := buildShareOfficePreviewURL(file.FileURL)
	downloadURL := fmt.Sprintf("/s/%s/download", shareToken)

	if share.GetExtractionCode() != "" && inputCode == "" {
		return &ShareAccessResponse{
			ShareToken:       shareToken,
			FileName:         file.Name,
			FileSize:         file.Size,
			FileType:         getFileTypeFromExtension(file.FileExtension),
			UpdatedAt:        share.UpdatedAt,
			ExpireAt:         share.ExpireTime,
			FileURL:          file.FileURL,
			ThumbnailURL:     file.ThumbnailURL,
			CanPreview:       canPreview,
			PreviewType:      previewType,
			OfficePreviewURL: officePreviewURL,
			DownloadUrl:      downloadURL,
			NeedCode:         true,
		}, nil
	}

	if share.GetExtractionCode() != "" && share.GetExtractionCode() != inputCode {
		return nil, errors.New("提取码错误")
	}

	_ = s.shareRepo.IncrementAccessCount(share.Id)

	return &ShareAccessResponse{
		ShareToken:       shareToken,
		FileName:         file.Name,
		FileSize:         file.Size,
		FileType:         getFileTypeFromExtension(file.FileExtension),
		UpdatedAt:        share.UpdatedAt,
		ExpireAt:         share.ExpireTime,
		DownloadUrl:      downloadURL,
		FileURL:          file.FileURL,
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
		return "", errors.New("提取码错误")
	}

	file, err := s.fileRepo.GetUserFileByID(share.UserId, share.FileId)
	if err != nil {
		return "", errors.New("文件不存在")
	}

	_ = s.shareRepo.IncrementDownloadCount(share.Id)

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

func getFileTypeFromExtension(ext string) string {
	switch normalizeExt(ext) {
	case "jpg", "jpeg", "png", "gif", "bmp", "webp", "svg":
		return "image"
	case "mp4", "avi", "mov", "wmv", "flv", "webm", "mkv":
		return "video"
	case "mp3", "wav", "flac", "aac", "ogg", "m4a":
		return "audio"
	case "doc", "docx", "xls", "xlsx", "ppt", "pptx", "pdf", "txt", "md":
		return "document"
	default:
		return "other"
	}
}

func getSharePreviewType(extension string) (bool, string) {
	ext := normalizeExt(extension)
	if ext == "" {
		return false, "other"
	}

	imageExts := map[string]bool{"jpg": true, "jpeg": true, "png": true, "gif": true, "bmp": true, "webp": true, "svg": true}
	videoExts := map[string]bool{"mp4": true, "avi": true, "mov": true, "wmv": true, "flv": true, "webm": true, "mkv": true}
	audioExts := map[string]bool{"mp3": true, "wav": true, "flac": true, "aac": true, "ogg": true, "m4a": true}
	textExts := map[string]bool{"txt": true, "md": true, "json": true, "xml": true, "csv": true, "log": true, "js": true, "css": true, "html": true, "go": true, "java": true, "py": true, "c": true, "cpp": true}
	officeExts := map[string]bool{"doc": true, "docx": true, "xls": true, "xlsx": true, "ppt": true, "pptx": true}

	if imageExts[ext] {
		return true, "image"
	}
	if videoExts[ext] {
		return true, "video"
	}
	if audioExts[ext] {
		return true, "audio"
	}
	if textExts[ext] {
		return true, "text"
	}
	if ext == "pdf" {
		return true, "pdf"
	}
	if officeExts[ext] {
		return true, "office"
	}
	return false, "other"
}

func normalizeExt(ext string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(ext)), ".")
}

func buildShareOfficePreviewURL(fileURL string) string {
	if strings.TrimSpace(fileURL) == "" {
		return ""
	}
	encoded := url.QueryEscape(fileURL)
	return "https://view.officeapps.live.com/op/view.aspx?src=" + encoded + "&wdAr=1.3333333333333333"
}
