package repository

import (
	"gorm.io/gorm"

	eventing "go-cloud-storage/backend/pkg/platform/eventing"
	ports "go-cloud-storage/backend/pkg/ports"
	domain "go-cloud-storage/backend/services/share-service/internal/domain"
)

type EventShares struct {
	ShareRepository
	DB *gorm.DB
}

func NewEventShares(db *gorm.DB) *EventShares {
	return &EventShares{ShareRepository: NewShareRepository(db), DB: db}
}
func (s *EventShares) CreateShareAndEvent(share *domain.Share, name string) (*domain.Share, error) {
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(share).Error; err != nil {
			return err
		}
		return eventing.Enqueue(tx, ports.Event{Type: "file.shared.v1", UserID: share.UserId, AggregateID: share.FileId, Data: map[string]string{"fileName": name}})
	})
	return share, err
}
