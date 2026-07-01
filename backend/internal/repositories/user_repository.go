package repositories

import (
	"go-cloud-storage/backend/internal/models"
	"gorm.io/gorm"
)

type UserRepository interface {
	GetUserInfoByAccount(account string) (*models.User, error)
	GetUserInfoByEmail(email string) (*models.User, error)
	Insert(user *models.User) error
	Update(user *models.User) error
	EmailExists(email string) (bool, error)
	GetUserInfoById(userId int) (*models.User, error)
	UpdateAvatarURL(userId int, avatarURL string) error
	UpdatePassword(userId int, hashedPassword string) error
	CreatePasswordResetToken(token *models.PasswordResetToken) error
	GetPasswordResetToken(token string) (*models.PasswordResetToken, error)
	MarkResetTokenUsed(tokenId int) error
}

type userRepo struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) GetUserInfoByAccount(account string) (*models.User, error) {
	var user models.User
	err := r.db.Where("email = ?", account).Or("phone = ?", account).First(&user).Error
	return &user, err
}

func (r *userRepo) Insert(user *models.User) error {
	err := r.db.Create(user).Error
	return err
}

func (r *userRepo) Update(user *models.User) error {
	err := r.db.Save(user).Error
	return err
}

func (r *userRepo) EmailExists(email string) (bool, error) {
	var count int64
	err := r.db.Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

func (r *userRepo) GetUserInfoById(userId int) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, userId).Error
	return &user, err
}

func (r *userRepo) UpdateAvatarURL(userId int, avatarURL string) error {
	return r.db.Model(&models.User{}).Where("id = ?", userId).Update("avatar", avatarURL).Error
}

func (r *userRepo) GetUserInfoByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.Where("email = ?", email).First(&user).Error
	return &user, err
}

func (r *userRepo) UpdatePassword(userId int, hashedPassword string) error {
	return r.db.Model(&models.User{}).Where("id = ?", userId).Update("password", hashedPassword).Error
}

func (r *userRepo) CreatePasswordResetToken(token *models.PasswordResetToken) error {
	return r.db.Create(token).Error
}

func (r *userRepo) GetPasswordResetToken(token string) (*models.PasswordResetToken, error) {
	var t models.PasswordResetToken
	err := r.db.Where("token = ? AND used = ?", token, false).First(&t).Error
	return &t, err
}

func (r *userRepo) MarkResetTokenUsed(tokenId int) error {
	return r.db.Model(&models.PasswordResetToken{}).Where("id = ?", tokenId).Update("used", true).Error
}
