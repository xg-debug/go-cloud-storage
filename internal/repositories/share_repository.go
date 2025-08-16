package repositories

import (
	"gorm.io/gorm"
)

type ShareRepository interface {
}

type shareRepo struct {
	db *gorm.DB
}

func NewShareRepository(db *gorm.DB) ShareRepository {
	return &shareRepo{db: db}
}
