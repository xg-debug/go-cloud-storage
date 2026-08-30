package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"go-cloud-storage/backend/infrastructure/minio"
	"go-cloud-storage/backend/internal/models"
	"go-cloud-storage/backend/internal/models/vo"
	"go-cloud-storage/backend/internal/repositories"
	"go-cloud-storage/backend/pkg/utils"
	"log/slog"
	"math/rand"
	"mime/multipart"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService interface {
	AuthenticateUser(account, password string) (*models.User, error)
	RegisterUser(email, pwd, pwdConfirm string) error
	GetProfile(userId int) (*vo.UserProfileResponse, error)
	UpdateUserInfo(userId int, username, phone string) error
	ChangePassword(userId int, oldPassword, newPassword string) error
	UploadAvatar(ctx context.Context, userId int, file multipart.File, header *multipart.FileHeader) (string, error)
	ForgotPassword(email string) error
	ResetPassword(token, newPassword string) error
}

type userService struct {
	db           *gorm.DB
	userRepo     repositories.UserRepository
	fileRepo     repositories.FileRepository
	quotaRepo    repositories.StorageQuotaRepository
	minio        *minio.MinioService
	emailService EmailSender
	resetBaseURL string // 对外访问地址，用于生成密码重置链接
}

type EmailSender interface {
	SendResetPasswordEmail(to, resetLink string) error
}

func NewUserService(db *gorm.DB, userRepo repositories.UserRepository, fileRepo repositories.FileRepository,
	quotaRepo repositories.StorageQuotaRepository, minio *minio.MinioService, emailService EmailSender, resetBaseURL string) UserService {
	return &userService{db: db, userRepo: userRepo, fileRepo: fileRepo, quotaRepo: quotaRepo, minio: minio, emailService: emailService, resetBaseURL: resetBaseURL}
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
	if err := validatePassword(pwd); err != nil {
		return err
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

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return errors.New("注册失败")
		}

		// 给用户创建独立根目录
		rootId := utils.NewUUID()
		rootFolder := &models.File{
			Id:        rootId,
			UserId:    user.Id,
			Name:      "/",
			IsDir:     true,
			ParentId:  sql.NullString{},
			IsDeleted: false,
			CreatedAt: time.Now(),
		}
		if err := tx.Create(rootFolder).Error; err != nil {
			return errors.New("初始化根目录失败")
		}

		// 回写 user 表中的 root_folder_id
		user.RootFolderId = rootId
		if err := tx.Model(&user).Update("root_folder_id", rootId).Error; err != nil {
			return errors.New("回写root_folder_id失败")
		}

		// 给用户分配 10GB 的物理空间
		storage := &models.StorageQuota{
			UserID:    user.Id,
			Total:     10 * 1024 * 1024 * 1024,
			Used:      0,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := tx.Create(storage).Error; err != nil {
			return err
		}
		return nil
	})
}

func (s *userService) GetProfile(userId int) (*vo.UserProfileResponse, error) {
	user, err := s.userRepo.GetUserInfoById(userId)
	if err != nil {
		return nil, errors.New("获取当前用户信息失败")
	}
	// 私有桶下头像也需要预签名 URL（24h 有效，每次请求重新生成）
	avatar := user.Avatar
	if s.minio != nil {
		avatar = s.minio.PresignAvatarURL(context.Background(), user.Avatar, 24*time.Hour)
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
		Avatar:       avatar,
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
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("密码加密失败")
	}
	user.Password = string(hashedPassword)
	return s.userRepo.Update(user)
}

func (s *userService) ForgotPassword(email string) error {
	user, err := s.userRepo.GetUserInfoByEmail(email)
	if err == nil {
		token, tokenErr := utils.GenerateResetToken(user.Id, 30*time.Minute)
		if tokenErr == nil {
			resetToken := &models.PasswordResetToken{
				UserId:    user.Id,
				Token:     token,
				ExpiresAt: time.Now().Add(30 * time.Minute),
			}
			if saveErr := s.userRepo.CreatePasswordResetToken(resetToken); saveErr == nil {
				base := strings.TrimRight(s.resetBaseURL, "/")
				if base == "" {
					base = "http://localhost:8080"
				}
				resetLink := base + "/reset-password?token=" + token
				if sendErr := s.emailService.SendResetPasswordEmail(email, resetLink); sendErr != nil {
					// 只记录错误，不记录含 token 的链接（防日志泄露）
					slog.Warn("failed to send reset email", "email", email, "error", sendErr)
				}
			}
		}
	}

	// 无论邮箱是否存在，都返回成功，防止邮箱枚举攻击
	return nil
}

func (s *userService) ResetPassword(token, newPassword string) error {
	resetToken, err := s.userRepo.GetPasswordResetToken(token)
	if err != nil {
		return errors.New("无效的重置链接")
	}

	if time.Now().After(resetToken.ExpiresAt) {
		return errors.New("重置链接已过期")
	}

	if err := validatePassword(newPassword); err != nil {
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("密码加密失败")
	}

	if err := s.userRepo.UpdatePassword(resetToken.UserId, string(hashedPassword)); err != nil {
		return errors.New("更新密码失败")
	}

	return s.userRepo.MarkResetTokenUsed(resetToken.Id)
}

// validatePassword 密码强度校验：至少 8 位且同时包含大小写字母和数字
func validatePassword(pwd string) error {
	if len(pwd) < 8 {
		return errors.New("密码至少8个字符")
	}
	hasLower := false
	hasUpper := false
	hasDigit := false
	for _, c := range pwd {
		switch {
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= '0' && c <= '9':
			hasDigit = true
		}
	}
	if !hasLower || !hasUpper || !hasDigit {
		return errors.New("密码必须同时包含大小写字母和数字")
	}
	return nil
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
	// 私有桶下返回预签名 URL 供前端立即展示
	return s.minio.PresignAvatarURL(ctx, avatarURL, 24*time.Hour), nil
}

// generateUsername 生成用户名
func generateUsername(email string) string {
	prefix := strings.Split(email, "@")[0]
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("User_%s%04d", prefix, rng.Intn(10000))
}
