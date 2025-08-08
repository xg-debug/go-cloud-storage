package services

import (
	"errors"
	"github.com/google/uuid"
	"go-cloud-storage/internal/models"
	"go-cloud-storage/internal/repositories"
	"time"
)

type ShareService interface {
	CreateShare(share *models.ShareRecord) error
	GetUserShares(userId int) ([]models.ShareRecord, error)
	DeleteShare(shareId, userId int) error
	ValidateShare(link, password string) (*models.ShareRecord, error)
}

type shareService struct {
	shareRepo repositories.ShareRepository
}

func NewShareService(repo repositories.ShareRepository) ShareService {
	return &shareService{shareRepo: repo}
}

func (s *shareService) CreateShare(share *models.ShareRecord) error {
	// 设置默认过期时间（7天）
	if share.ExpiresAt.IsZero() {
		share.ExpiresAt = time.Now().Add(7 * 24 * time.Hour)
	}
	// 生成唯一分享链接（实际项目中应该使用更安全的生成方式）
	share.ShareLink = generateShareLink()

	return s.shareRepo.CreateShare(share)
}

func (s *shareService) GetUserShares(userId int) ([]models.ShareRecord, error) {
	return s.shareRepo.GetShareByUser(userId)
}

func (s *shareService) DeleteShare(shareId, userId int) error {
	share, err := s.shareRepo.GetShareById(shareId)
	if err != nil {
		return err
	}
	if share.OwnerID != userId {
		return errors.New("未授权的操作")
	}
	return s.shareRepo.DeleteShare(shareId)
}

func (s *shareService) ValidateShare(link, password string) (*models.ShareRecord, error) {
	share, err := s.shareRepo.GetShareByLink(link)
	if err != nil {
		return nil, err
	}
	// 检查过期时间
	if time.Now().After(share.ExpiresAt) {
		return nil, errors.New("分享链接已过期")
	}
	// 检查密码
	if share.Password != "" && share.Password != password {
		return nil, errors.New("密码错误")
	}
	return nil, err
}

// 辅助函数：生成分享链接（示例）
func generateShareLink() string {
	// 实际项目中应使用安全的随机字符串生成
	return "https://yourcloud.com/share/" + uuid.New().String()
}
