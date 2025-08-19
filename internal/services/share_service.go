package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"go-cloud-storage/internal/models"
	"go-cloud-storage/internal/repositories"
)

type ShareService interface {
	CreateShare(userID int, fileID string, extractionCode string, expireDays int) (*ShareResponse, error)
	GetUserShares(userID int) ([]*ShareResponse, error)
	GetShareDetail(shareID int, userID int) (*ShareResponse, error)
	CancelShare(shareID int, userID int) error
	DeleteShare(shareID int, userID int) error
	AccessShare(shareToken string, extractionCode string) (*ShareAccessResponse, error)
	DownloadSharedFile(shareToken string, extractionCode string) (string, error)
}

type shareService struct {
	shareRepo repositories.ShareRepository
	fileRepo  repositories.FileRepository
}

type ShareResponse struct {
	ID            int        `json:"id"`
	FileName      string     `json:"fileName"`
	FileType      string     `json:"fileType"`
	FileSize      int64      `json:"fileSize"`
	ShareUrl      string     `json:"shareUrl"`
	ExtractCode   string     `json:"extractCode,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	ExpiresAt     *time.Time `json:"expiresAt,omitempty"`
	Status        string     `json:"status"`
	DownloadCount int        `json:"downloadCount"`
	ViewCount     int        `json:"viewCount"`
}

type ShareAccessResponse struct {
	FileName    string `json:"fileName"`
	FileSize    int64  `json:"fileSize"`
	FileType    string `json:"fileType"`
	DownloadURL string `json:"downloadUrl"`
	PreviewURL  string `json:"previewUrl,omitempty"`
}

func NewShareService(shareRepo repositories.ShareRepository, fileRepo repositories.FileRepository) ShareService {
	return &shareService{
		shareRepo: shareRepo,
		fileRepo:  fileRepo,
	}
}

// CreateShare 创建分享
func (s *shareService) CreateShare(userID int, fileID string, extractionCode string, expireDays int) (*ShareResponse, error) {
	// 检查文件是否存在且属于用户
	file, err := s.fileRepo.GetFileById(fileID)
	if err != nil {
		return nil, errors.New("文件不存在")
	}

	if file.UserId != userID {
		return nil, errors.New("无权限分享此文件")
	}

	// 生成分享token
	shareToken, err := generateShareToken()
	if err != nil {
		return nil, errors.New("生成分享链接失败")
	}

	// 计算过期时间
	var expireTime *time.Time
	if expireDays > 0 {
		expire := time.Now().AddDate(0, 0, expireDays)
		expireTime = &expire
	}

	// 创建分享记录
	share := &models.Share{
		UserId:         userID,
		FileId:         fileID,
		ShareToken:     shareToken,
		ExtractionCode: &extractionCode,
		ExpireTime:     expireTime,
		CreatedAt:      time.Now(),
	}

	if extractionCode == "" {
		share.ExtractionCode = nil
	}

	createdShare, err := s.shareRepo.CreateShare(share)
	if err != nil {
		return nil, errors.New("创建分享失败")
	}

	// 构建响应
	response := &ShareResponse{
		ID:            createdShare.Id,
		FileName:      file.Name,
		FileType:      getFileTypeFromExtension(file.FileExtension),
		FileSize:      file.Size,
		ShareUrl:      fmt.Sprintf("http://localhost:8080/share/%s", shareToken),
		ExtractCode:   extractionCode,
		CreatedAt:     createdShare.CreatedAt,
		ExpiresAt:     expireTime,
		Status:        getShareStatus(expireTime),
		DownloadCount: 0,
		ViewCount:     0,
	}

	return response, nil
}

// GetUserShares 获取用户的分享列表
func (s *shareService) GetUserShares(userID int) ([]*ShareResponse, error) {
	shares, err := s.shareRepo.GetUserShares(userID)
	if err != nil {
		return nil, err
	}

	var responses []*ShareResponse
	for _, share := range shares {
		file, err := s.fileRepo.GetFileById(share.FileId)
		if err != nil {
			continue // 跳过已删除的文件
		}

		extractCode := ""
		if share.ExtractionCode != nil {
			extractCode = *share.ExtractionCode
		}

		response := &ShareResponse{
			ID:            share.Id,
			FileName:      file.Name,
			FileType:      getFileTypeFromExtension(file.FileExtension),
			FileSize:      file.Size,
			ShareUrl:      fmt.Sprintf("http://localhost:8080/share/%s", share.ShareToken),
			ExtractCode:   extractCode,
			CreatedAt:     share.CreatedAt,
			ExpiresAt:     share.ExpireTime,
			Status:        getShareStatus(share.ExpireTime),
			DownloadCount: 0, // TODO: 从统计表获取
			ViewCount:     0, // TODO: 从统计表获取
		}

		responses = append(responses, response)
	}

	return responses, nil
}

// GetShareDetail 获取分享详情
func (s *shareService) GetShareDetail(shareID int, userID int) (*ShareResponse, error) {
	share, err := s.shareRepo.GetShareByID(shareID)
	if err != nil {
		return nil, errors.New("分享不存在")
	}

	if share.UserId != userID {
		return nil, errors.New("无权限访问此分享")
	}

	file, err := s.fileRepo.GetFileById(share.FileId)
	if err != nil {
		return nil, errors.New("文件不存在")
	}

	extractCode := ""
	if share.ExtractionCode != nil {
		extractCode = *share.ExtractionCode
	}

	response := &ShareResponse{
		ID:            share.Id,
		FileName:      file.Name,
		FileType:      getFileTypeFromExtension(file.FileExtension),
		FileSize:      file.Size,
		ShareUrl:      fmt.Sprintf("http://localhost:8080/share/%s", share.ShareToken),
		ExtractCode:   extractCode,
		CreatedAt:     share.CreatedAt,
		ExpiresAt:     share.ExpireTime,
		Status:        getShareStatus(share.ExpireTime),
		DownloadCount: 0, // TODO: 从统计表获取
		ViewCount:     0, // TODO: 从统计表获取
	}

	return response, nil
}

// CancelShare 取消分享
func (s *shareService) CancelShare(shareID int, userID int) error {
	share, err := s.shareRepo.GetShareByID(shareID)
	if err != nil {
		return errors.New("分享不存在")
	}

	if share.UserId != userID {
		return errors.New("无权限操作此分享")
	}

	// 设置过期时间为当前时间，使分享失效
	now := time.Now()
	return s.shareRepo.UpdateShareExpireTime(shareID, &now)
}

// DeleteShare 删除分享记录
func (s *shareService) DeleteShare(shareID int, userID int) error {
	share, err := s.shareRepo.GetShareByID(shareID)
	if err != nil {
		return errors.New("分享不存在")
	}

	if share.UserId != userID {
		return errors.New("无权限操作此分享")
	}

	return s.shareRepo.DeleteShare(shareID)
}

// AccessShare 访问分享
func (s *shareService) AccessShare(shareToken string, extractionCode string) (*ShareAccessResponse, error) {
	share, err := s.shareRepo.GetShareByToken(shareToken)
	if err != nil {
		return nil, errors.New("分享链接不存在")
	}

	// 检查是否过期
	if share.ExpireTime != nil && time.Now().After(*share.ExpireTime) {
		return nil, errors.New("分享链接已过期")
	}

	// 检查提取码
	if share.ExtractionCode != nil && *share.ExtractionCode != extractionCode {
		return nil, errors.New("提取码错误")
	}

	file, err := s.fileRepo.GetFileById(share.FileId)
	if err != nil {
		return nil, errors.New("文件不存在")
	}

	// TODO: 增加访问统计

	response := &ShareAccessResponse{
		FileName:    file.Name,
		FileSize:    file.Size,
		FileType:    getFileTypeFromExtension(file.FileExtension),
		DownloadURL: file.FileURL,
		PreviewURL:  file.ThumbnailURL,
	}

	return response, nil
}

// DownloadSharedFile 下载分享的文件
func (s *shareService) DownloadSharedFile(shareToken string, extractionCode string) (string, error) {
	shareInfo, err := s.AccessShare(shareToken, extractionCode)
	if err != nil {
		return "", err
	}

	// TODO: 增加下载统计

	return shareInfo.DownloadURL, nil
}

// 生成分享token
func generateShareToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// 获取分享状态
func getShareStatus(expireTime *time.Time) string {
	if expireTime == nil {
		return "active"
	}

	if time.Now().After(*expireTime) {
		return "expired"
	}

	return "active"
}

// 根据文件扩展名获取文件类型
func getFileTypeFromExtension(extension string) string {
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp"}
	videoExts := []string{".mp4", ".avi", ".mov", ".wmv", ".flv", ".mkv"}
	audioExts := []string{".mp3", ".wav", ".flac", ".aac", ".ogg"}
	docExts := []string{".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".txt"}

	for _, ext := range imageExts {
		if extension == ext {
			return "image"
		}
	}

	for _, ext := range videoExts {
		if extension == ext {
			return "video"
		}
	}

	for _, ext := range audioExts {
		if extension == ext {
			return "audio"
		}
	}

	for _, ext := range docExts {
		if extension == ext {
			return "document"
		}
	}

	return "other"
}
