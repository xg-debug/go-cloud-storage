package eventing

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"go-cloud-storage/backend/pkg/platform/persistence"
	ports "go-cloud-storage/backend/pkg/ports"
)

type Outbox struct {
	ID          string `gorm:"primaryKey;size:64"`
	Payload     []byte `gorm:"type:longblob"`
	CreatedAt   time.Time
	DeliveredAt *time.Time `gorm:"index"`
}

func (Outbox) TableName() string { return "event_outbox" }

type Inbox struct {
	ID        string `gorm:"primaryKey;size:64"`
	CreatedAt time.Time
}

func (Inbox) TableName() string { return "event_inbox" }

type Publisher struct{ DB *gorm.DB }

func (*Publisher) Transactional() bool { return true }
func (p *Publisher) Publish(ctx context.Context, event ports.Event) error {
	return Enqueue(persistence.TransactionDB(ctx, p.DB), event)
}
func Enqueue(tx *gorm.DB, event ports.Event) error {
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now()
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return tx.Create(&Outbox{ID: event.ID, Payload: data, CreatedAt: time.Now()}).Error
}

// Apply acknowledges duplicate delivery only after the original transaction committed.
func Apply(db *gorm.DB, id string, fn func(*gorm.DB) error) error {
	if id == "" {
		return errors.New("missing event ID")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&Inbox{ID: id, CreatedAt: time.Now()})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		return fn(tx)
	})
}
