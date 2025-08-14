package repositories

import (
	"go-cloud-storage/internal/models"
	"gorm.io/gorm"
)

type UserRepository interface {
	GetUserInfoByAccount(account string) (*models.User, error)
	Insert(user *models.User) error
	Update(user *models.User) error
	EmailExists(email string) (bool, error)
	GetUserInfoById(userId int) (*models.User, error)
	UpdateAvatarURL(userId int, avatarURL string) error
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
	return r.db.Model(&models.File{}).Where("user_id = ?", userId).Update("avatar", avatarURL).Error
}
