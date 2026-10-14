package models

import "time"

// ObjectDeletion is committed with metadata removal. It survives process and
// storage failures; completion is removed only after an idempotent object delete.
type ObjectDeletion struct {
	ID            string    `gorm:"primaryKey;size:64"`
	ObjectKey     string    `gorm:"size:1024;not null"`
	Attempts      int       `gorm:"not null;default:0"`
	NextAttemptAt time.Time `gorm:"not null;index"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
}
