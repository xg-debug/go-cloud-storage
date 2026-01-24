package services

import (
	"errors"
	"fmt"
	"go-cloud-storage/internal/models"
	"go-cloud-storage/internal/repositories"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ShareService interface {
	CreateShare(userId int, fileId string, expireDays int, extractionCode string) (*models.Share, error)
	GetUserShares(userId int) ([]*ShareItem, error)
	GetShareDetail(userId int, shareId int) (*ShareDetail, error)
	CancelShare(userId int, shareId int) error
	DeleteShare(userId int, shareId int) error
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
	// 1. 检查文件是否存在且属于该用户
	file, err := s.fileRepo.GetFile(fileId)
	if err != nil {
		return nil, errors.New("文件不存在")
	}
	if file.UserId != userId {
		return nil, errors.New("无权分享此文件")
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
		ExtractionCode: extractionCode,
		ExpireTime:     expireTime,
		ClickCount:     0,
		DownloadCount:  0,
		SaveCount:      0,
	}

	return s.shareRepo.CreateShare(share)
}

type ShareItem struct {
	Id            int        `json:"id"`
	FileName      string     `json:"fileName"`
	FileSize      int64      `json:"fileSize"`
	FileType      string     `json:"fileType"`
	ShareToken    string     `json:"shareToken"`
	ShareUrl      string     `json:"shareUrl"`
	ExtractCode   string     `json:"extractCode"`
	ExpireAt      *time.Time `json:"expireAt"`
	CreatedAt     time.Time  `json:"createdAt"`
	ClickCount    int        `json:"clickCount"`
	DownloadCount int        `json:"downloadCount"`
	SaveCount     int        `json:"saveCount"`
	Status        string     `json:"status"` // active, expired
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

		extractCode := ""
		if share.ExtractionCode != "" {
			extractCode = share.ExtractionCode
		}

		result = append(result, &ShareItem{
			Id:            share.Id,
			FileName:      file.FileName,
			FileSize:      file.Size,
			FileType:      getFileTypeFromExtension(file.FileExtension),
			ShareToken:    share.ShareToken,
			ShareUrl:      fmt.Sprintf("http://localhost:8080/s/%s", share.ShareToken),
			ExtractCode:   extractCode,
			ExpireAt:      share.ExpireTime,
			CreatedAt:     share.CreatedAt,
			ClickCount:    share.ClickCount,
			DownloadCount: share.DownloadCount,
			SaveCount:     share.SaveCount,
			Status:        status,
		})
	}

	return result, nil
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
	ClickCount    int        `json:"clickCount"`
	SaveCount     int        `json:"saveCount"`
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

	extractCode := ""
	if share.ExtractionCode != "" {
		extractCode = share.ExtractionCode
	}

	return &ShareDetail{
		Id:            share.Id,
		FileName:      file.FileName,
		FileSize:      file.Size,
		FileType:      getFileTypeFromExtension(file.FileExtension),
		ShareToken:    share.ShareToken,
		ShareUrl:      fmt.Sprintf("http://localhost:8080/s/%s", share.ShareToken),
		ExtractCode:   extractCode,
		CreatedAt:     share.CreatedAt,
		ExpireAt:      share.ExpireTime,
		ClickCount:    share.ClickCount,
		SaveCount:     share.SaveCount,
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

func (s *shareService) DeleteShare(userId int, shareId int) error {
	return s.CancelShare(userId, shareId)
}

type ShareAccessResponse struct {
	ShareToken  string     `json:"shareToken"`
	FileName    string     `json:"fileName"`
	FileSize    int64      `json:"fileSize"`
	FileType    string     `json:"fileType"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	ExpireAt    *time.Time `json:"expireAt"`
	DownloadUrl string     `json:"downloadUrl"` // Only if code verified or no code
	NeedCode    bool       `json:"needCode"`
}

func (s *shareService) AccessShare(shareToken string, extractionCode string) (*ShareAccessResponse, error) {
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

	// Check code
	if share.ExtractionCode != "" {
		if extractionCode == "" {
			// Return basic info but indicate code needed
			file, _ := s.fileRepo.GetFileById(share.FileId)
			return &ShareAccessResponse{
				ShareToken: shareToken,
				FileName:   file.FileName,
				FileSize:   file.Size,
				FileType:   getFileTypeFromExtension(file.FileExtension),
				UpdatedAt:  share.UpdatedAt,
				ExpireAt:   share.ExpireTime,
				NeedCode:   true,
			}, errors.New("请输入提取码") // Or handled by caller as 403
		}
		if share.ExtractionCode != extractionCode {
			return nil, errors.New("提取码错误")
		}
	}

	file, err := s.fileRepo.GetFileById(share.FileId)
	if err != nil {
		return nil, errors.New("文件不存在")
	}

	// Update stats (ClickCount) - maybe async?
	// s.shareRepo.IncrementClick(share.Id)

	downloadUrl := file.FileURL

	return &ShareAccessResponse{
		ShareToken:  shareToken,
		FileName:    file.FileName,
		FileSize:    file.Size,
		FileType:    getFileTypeFromExtension(file.FileExtension),
		UpdatedAt:   share.UpdatedAt,
		ExpireAt:    share.ExpireTime,
		DownloadUrl: downloadUrl,
		NeedCode:    false,
	}, nil
}

func (s *shareService) DownloadSharedFile(shareToken string, extractionCode string) (string, error) {
	// Re-verify
	share, err := s.shareRepo.GetShareByToken(shareToken)
	if err != nil {
		return "", errors.New("分享不存在")
	}
	if share.ExpireTime != nil && share.ExpireTime.Before(time.Now()) {
		return "", errors.New("分享已过期")
	}
	if share.ExtractionCode != "" && share.ExtractionCode != extractionCode {
		return "", errors.New("提取码错误")
	}

	file, err := s.fileRepo.GetFileById(share.FileId)
	if err != nil {
		return "", errors.New("文件不存在")
	}

	// Increment download count
	// s.shareRepo.IncrementDownload(share.Id)

	// Return real download URL (e.g. MinIO signed URL or backend proxy URL)
	// Return the FileURL which should be a public-read MinIO/OSS URL
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
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif":
		return "image"
	case ".mp4", ".avi", ".mov":
		return "video"
	case ".mp3", ".wav":
		return "audio"
	case ".doc", ".docx", ".pdf", ".txt":
		return "document"
	default:
		return "other"
	}
}
