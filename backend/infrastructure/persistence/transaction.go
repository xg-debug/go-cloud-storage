package persistence

import (
	"context"

	"go-cloud-storage/backend/internal/ports"

	"gorm.io/gorm"
)

type transactionContextKey struct{}

type GormTransactionManager struct {
	db *gorm.DB
}

func NewGormTransactionManager(db *gorm.DB) *GormTransactionManager {
	return &GormTransactionManager{db: db}
}

func (m *GormTransactionManager) WithinTransaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	if fn == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if existing := dbFromContext(ctx, nil); existing != nil {
		return fn(ctx)
	}
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, transactionContextKey{}, tx)
		return fn(txCtx)
	})
}

func dbFromContext(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if ctx != nil {
		if tx, ok := ctx.Value(transactionContextKey{}).(*gorm.DB); ok && tx != nil {
			return tx.WithContext(ctx)
		}
	}
	if fallback == nil {
		return nil
	}
	return fallback.WithContext(ctx)
}

var _ ports.TransactionManager = (*GormTransactionManager)(nil)
