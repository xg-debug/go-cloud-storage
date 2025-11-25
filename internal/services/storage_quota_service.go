package services

import (
	"go-cloud-storage/internal/models"
	"go-cloud-storage/internal/repositories"
)

// StorageQuotaService 存储配额服务接口
type StorageQuotaService interface {
	GetUserQuota(userId int) (*models.StorageQuota, error)
	UpdateUsedSpace(userId int, deltaSize int64) error
	EnsureUserQuota(userId int) error
}

type storageQuotaService struct {
	storageQuotaRepo repositories.StorageQuotaRepository
}

// NewStorageQuotaService 创建存储配额服务实例
func NewStorageQuotaService(repo repositories.StorageQuotaRepository) StorageQuotaService {
	return &storageQuotaService{
		storageQuotaRepo: repo,
	}
}

// GetUserQuota 获取用户存储配额
func (s *storageQuotaService) GetUserQuota(userId int) (*models.StorageQuota, error) {
	// 确保用户有存储配额记录
	if err := s.EnsureUserQuota(userId); err != nil {
		return nil, err
	}

	return s.storageQuotaRepo.GetByUserID(userId)
}

// UpdateUsedSpace 更新已使用空间
// deltaSize 为正数表示增加已使用空间（上传文件）
// deltaSize 为负数表示减少已使用空间（删除文件）
func (s *storageQuotaService) UpdateUsedSpace(userId int, deltaSize int64) error {
	// 确保用户有存储配额记录
	if err := s.EnsureUserQuota(userId); err != nil {
		return err
	}

	return s.storageQuotaRepo.UpdateUsedSpace(nil, userId, deltaSize)
}

// EnsureUserQuota 确保用户有存储配额记录
func (s *storageQuotaService) EnsureUserQuota(userId int) error {
	return s.storageQuotaRepo.EnsureUserQuota(userId)
}
