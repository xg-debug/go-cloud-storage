package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"go-cloud-storage/backend/internal/models"
	"go-cloud-storage/backend/internal/models/vo"
	"go-cloud-storage/backend/infrastructure/minio"
	"go-cloud-storage/backend/pkg/utils"
	"go-cloud-storage/backend/internal/repositories"
	"math/rand"
	"mime/multipart"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type UserService interface {
	AuthenticateUser(account, password string) (*models.User, error)
	RegisterUser(email, pwd, pwdConfirm string) error
	GetProfile(userId int) (*vo.UserProfileResponse, error)
	UpdateUserInfo(userId int, username, phone string) error
	ChangePassword(userId int, oldPassword, newPassword string) error
	UploadAvatar(ctx context.Context, userId int, file multipart.File, header *multipart.FileHeader) (string, error)
}

type userService struct {
	userRepo         repositories.UserRepository
	fileRepo         repositories.FileRepository
	storageQuotaRepo repositories.StorageQuotaRepository
	minio            *minio.MinioService
}

func NewUserService(userRepo repositories.UserRepository, fileRepo repositories.FileRepository,
	quotaRepo repositories.StorageQuotaRepository, minio *minio.MinioService) UserService {
	return &userService{userRepo: userRepo, fileRepo: fileRepo, storageQuotaRepo: quotaRepo, minio: minio}
}

func (s *userService) AuthenticateUser(account, password string) (*models.User, error) {
	// 1.查找用户：根据邮箱或者手机号
	user, err := s.userRepo.GetUserInfoByAccount(account)
	if err != nil {
		return nil, errors.New("用户不存在")
	}
	// 2.验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("密码错误")
	}

	// 3.返回用户ID
	return user, nil
}

func (s *userService) RegisterUser(email, pwd, pwdConfirm string) error {
	// 1.密码一致性验证
	if pwd != pwdConfirm {
		return errors.New("两次输入的密码不一致!")
	}
	// 2.检查邮箱是否已注册
	if exist, _ := s.userRepo.EmailExists(email); exist {
		return errors.New("邮箱已被注册")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("密码加密失败")
	}

	// 3.生成username (User+邮箱前缀+随机数)
	username := generateUsername(email)
	user := models.User{
		Username:     username,
		Email:        email,
		Phone:        nil,
		Password:     string(hashedPassword),
		Avatar:       "https://cube.elemecdn.com/3/7c/3ea6beec64369c2642b92c6726f1epng.png",
		RegisterTime: time.Now(),
	}
	// 4.密码加密：（暂时使用明文密码）

	if err := s.userRepo.Insert(&user); err != nil {
		return errors.New("注册失败")
	}

	// 5.给用户创建独立根目录
	rootId := utils.NewUUID()
	rootFolder := &models.File{
		Id:        rootId,
		UserId:    user.Id,
		Name:      "/", // 根目录名
		IsDir:     true,
		ParentId:  sql.NullString{}, // 表示根
		IsDeleted: false,
		CreatedAt: time.Now(),
	}
	if err := s.fileRepo.InitFolder(rootFolder); err != nil {
		return errors.New("初始化根目录失败")
	}

	// 回写 user 表中的root—_folder_id
	user.RootFolderId = rootId
	if err := s.userRepo.Update(&user); err != nil {
		return errors.New("回写root_folder_id失败")
	}

	// 6.给用户分配 10GB 的物理空间
	storage := &models.StorageQuota{
		UserID:    user.Id,
		Total:     10 * 1024 * 1024 * 1024,
		Used:      0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	s.storageQuotaRepo.Create(storage)
	return nil
}

func (s *userService) GetProfile(userId int) (*vo.UserProfileResponse, error) {
	user, err := s.userRepo.GetUserInfoById(userId)
	if err != nil {
		return nil, errors.New("获取当前用户信息失败")
	}
	profile := &vo.UserProfileResponse{
		Id:       user.Id,
		Username: user.Username,
		Email:    user.Email,
		Phone: func() string { // 避免空指针的问题
			if user.Phone != nil {
				return *user.Phone
			}
			return ""
		}(),
		Avatar:       user.Avatar,
		OpenId:       user.OpenId,
		RegisterTime: user.RegisterTime.Format("2006-01-02 15:04:05"),
		RootFolderId: user.RootFolderId,
	}
	return profile, nil
}

func (s *userService) UpdateUserInfo(userId int, username, phone string) error {
	user, err := s.userRepo.GetUserInfoById(userId)
	if err != nil {
		return err
	}
	user.Username = username
	user.Phone = &phone
	return s.userRepo.Update(user)
}

func (s *userService) ChangePassword(userId int, oldPassword, newPassword string) error {
	user, err := s.userRepo.GetUserInfoById(userId)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return errors.New("旧密码错误")
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("密码加密失败")
	}
	user.Password = string(hashedPassword)
	return s.userRepo.Update(user)
}

func (s *userService) UploadAvatar(ctx context.Context, userId int, file multipart.File, header *multipart.FileHeader) (string, error) {
	// 上传OSS
	avatarURL, err := s.minio.UploadAvatarFromStream(ctx, file, userId, header)
	if err != nil {
		return "", err
	}
	// 更新数据库
	if err = s.userRepo.UpdateAvatarURL(userId, avatarURL); err != nil {
		return "", fmt.Errorf("更新用户头像失败: %w", err)
	}
	return avatarURL, nil
}

// generateUsername 生成用户名
func generateUsername(email string) string {
	prefix := strings.Split(email, "@")[0]
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("User_%s%04d", prefix, rng.Intn(10000))
}
