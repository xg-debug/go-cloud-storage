package application

import (
	"bytes"
	"context"
	"crypto/hmac"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	contracts "go-cloud-storage/backend/pkg/contracts"
	ports "go-cloud-storage/backend/pkg/ports"
	"go-cloud-storage/backend/pkg/utils"
	repository "go-cloud-storage/backend/services/user-service/internal/adapters/repository"
	domain "go-cloud-storage/backend/services/user-service/internal/domain"
	vo "go-cloud-storage/backend/services/user-service/internal/domain/vo"
)

type UserService interface {
	ValidateSession(userId int, version string) error
	AuthenticateUser(account, password string) (*domain.User, error)
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
	userRepo     repository.UserRepository
	minio        ports.Storage
	emailService EmailSender
	resetBaseURL string // 对外访问地址，用于生成密码重置链接
}

type EmailSender interface {
	SendResetPasswordEmail(to, resetLink string) error
}

// NewIndependentUserService owns identity data only. File provisioning is idempotent in file-service.
type IdentityUserService interface {
	UserService
	RootFolderID(int) (string, error)
}

func NewIndependentUserService(db *gorm.DB, storage ports.Storage, email EmailSender, baseURL string) IdentityUserService {
	return &userService{db: db, userRepo: repository.NewUserRepository(db), minio: storage, emailService: email, resetBaseURL: baseURL}
}

func (s *userService) AuthenticateUser(account, password string) (*domain.User, error) {
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

var ErrSessionInvalid = contracts.ErrSessionInvalid

func (s *userService) ValidateSession(userId int, version string) error {
	user, err := s.userRepo.GetUserInfoById(userId)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err != nil || user == nil || version == "" || !hmac.Equal([]byte(version), []byte(utils.CredentialVersion(user.Password))) {
		return ErrSessionInvalid
	}
	return nil
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
	user := domain.User{
		Username:     username,
		Email:        email,
		Phone:        nil,
		Password:     string(hashedPassword),
		Avatar:       "https://cube.elemecdn.com/3/7c/3ea6beec64369c2642b92c6726f1epng.png",
		RegisterTime: time.Now(),
	}

	user.RootFolderId = utils.NewUUID()
	return s.db.Create(&user).Error
}

func (s *userService) GetProfile(userId int) (*vo.UserProfileResponse, error) {
	user, err := s.userRepo.GetUserInfoById(userId)
	if err != nil {
		return nil, fmt.Errorf("获取当前用户信息失败: %w", err)
	}
	// 私有桶下头像也需要预签名 URL（24h 有效，每次请求重新生成）
	avatar := user.Avatar
	if s.minio != nil {
		avatar = s.minio.PresignStoredObjectURL(context.Background(), user.Avatar, 24*time.Hour)
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
		RegisterTime: user.RegisterTime.Format(time.RFC3339),
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
	return s.db.Model(&domain.User{}).Where("id = ?", userId).Updates(map[string]interface{}{"username": user.Username, "phone": user.Phone}).Error
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
	return s.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&domain.User{}).Where("id = ? AND password = ?", userId, user.Password).Update("password", string(hashedPassword))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("密码已发生变更，请重新登录")
		}
		return tx.Model(&domain.PasswordResetToken{}).Where("user_id = ? AND used = ?", userId, false).Update("used", true).Error
	})
}

func (s *userService) ForgotPassword(email string) error {
	user, err := s.userRepo.GetUserInfoByEmail(email)
	if err == nil {
		token, tokenErr := utils.GenerateResetToken(user.Id, 30*time.Minute)
		if tokenErr == nil {
			resetToken := &domain.PasswordResetToken{
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

	return s.db.Transaction(func(tx *gorm.DB) error {
		var user domain.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, resetToken.UserId).Error; err != nil {
			return err
		}
		result := tx.Model(&domain.PasswordResetToken{}).Where("id = ? AND used = ? AND expires_at > ?", resetToken.Id, false, time.Now()).Update("used", true)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("重置链接已失效")
		}
		if err := tx.Model(&user).Update("password", string(hashedPassword)).Error; err != nil {
			return err
		}
		return tx.Model(&domain.PasswordResetToken{}).Where("user_id = ? AND used = ?", user.Id, false).Update("used", true).Error
	})
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
	// Validate at the application boundary; the storage service only sees objects.
	if header == nil || header.Size > 5*1024*1024 {
		return "", errors.New("头像大小不能超过5MB")
	}
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		return "", errors.New("不支持的头像格式")
	}
	data, err := io.ReadAll(io.LimitReader(file, 5*1024*1024+1))
	if err != nil {
		return "", err
	}
	if len(data) > 5*1024*1024 {
		return "", errors.New("头像大小不能超过5MB")
	}
	avatarKey := fmt.Sprintf("avatars/%d%s", userId, ext)
	_, err = s.minio.PutObjectStream(ctx, avatarKey, bytes.NewReader(data), int64(len(data)), header.Header.Get("Content-Type"))

	if err != nil {
		return "", err
	}
	avatarURL := s.minio.GenerateObjectURL(avatarKey)
	if avatarURL == "" {
		return "", errors.New("生成头像地址失败")
	}
	// 更新数据库
	if err = s.userRepo.UpdateAvatarURL(userId, avatarURL); err != nil {
		return "", fmt.Errorf("更新用户头像失败: %w", err)
	}
	// 私有桶下返回预签名 URL 供前端立即展示
	return s.minio.PresignStoredObjectURL(ctx, avatarURL, 24*time.Hour), nil
}

// generateUsername 生成用户名
func generateUsername(email string) string {
	prefix := strings.Split(email, "@")[0]
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("User_%s%04d", prefix, rng.Intn(10000))
}

func (s *userService) RootFolderID(userID int) (string, error) {
	user, err := s.userRepo.GetUserInfoById(userID)
	if err != nil {
		return "", err
	}
	return user.RootFolderId, nil
}
