package repository

import (
	"gorm.io/gorm"

	domain "go-cloud-storage/backend/services/user-service/internal/domain"
)

type UserRepository interface {
	GetUserInfoByAccount(account string) (*domain.User, error)
	GetUserInfoByEmail(email string) (*domain.User, error)
	Insert(user *domain.User) error
	Update(user *domain.User) error
	EmailExists(email string) (bool, error)
	GetUserInfoById(userId int) (*domain.User, error)
	UpdateAvatarURL(userId int, avatarURL string) error
	UpdatePassword(userId int, hashedPassword string) error
	CreatePasswordResetToken(token *domain.PasswordResetToken) error
	GetPasswordResetToken(token string) (*domain.PasswordResetToken, error)
	MarkResetTokenUsed(tokenId int) error
}

type userRepo struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) GetUserInfoByAccount(account string) (*domain.User, error) {
	var user domain.User
	err := r.db.Where("email = ?", account).Or("phone = ?", account).First(&user).Error
	return &user, err
}

func (r *userRepo) Insert(user *domain.User) error {
	err := r.db.Create(user).Error
	return err
}

func (r *userRepo) Update(user *domain.User) error {
	err := r.db.Save(user).Error
	return err
}

func (r *userRepo) EmailExists(email string) (bool, error) {
	var count int64
	err := r.db.Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

func (r *userRepo) GetUserInfoById(userId int) (*domain.User, error) {
	var user domain.User
	err := r.db.First(&user, userId).Error
	return &user, err
}

func (r *userRepo) UpdateAvatarURL(userId int, avatarURL string) error {
	return r.db.Model(&domain.User{}).Where("id = ?", userId).Update("avatar", avatarURL).Error
}

func (r *userRepo) GetUserInfoByEmail(email string) (*domain.User, error) {
	var user domain.User
	err := r.db.Where("email = ?", email).First(&user).Error
	return &user, err
}

func (r *userRepo) UpdatePassword(userId int, hashedPassword string) error {
	return r.db.Model(&domain.User{}).Where("id = ?", userId).Update("password", hashedPassword).Error
}

func (r *userRepo) CreatePasswordResetToken(token *domain.PasswordResetToken) error {
	return r.db.Create(token).Error
}

func (r *userRepo) GetPasswordResetToken(token string) (*domain.PasswordResetToken, error) {
	var t domain.PasswordResetToken
	err := r.db.Where("token = ? AND used = ?", token, false).First(&t).Error
	return &t, err
}

func (r *userRepo) MarkResetTokenUsed(tokenId int) error {
	return r.db.Model(&domain.PasswordResetToken{}).Where("id = ?", tokenId).Update("used", true).Error
}
