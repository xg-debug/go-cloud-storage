package messaging

import (
	"context"
	"encoding/json"

	"gorm.io/gorm"

	eventing "go-cloud-storage/backend/pkg/platform/eventing"
	ports "go-cloud-storage/backend/pkg/ports"
	domain "go-cloud-storage/backend/services/share-service/internal/domain"
)

func ShareEventHandler(db *gorm.DB) func(context.Context, ports.Event) error {
	return func(ctx context.Context, event ports.Event) error {
		if event.Type != "file.deleted.v1" {
			return nil
		}
		data, err := json.Marshal(event.Data)
		if err != nil {
			return err
		}
		var versions map[string]int64
		if err = json.Unmarshal(data, &versions); err != nil {
			return eventing.ErrInvalidEvent
		}
		for id, version := range versions {
			if id == "" || version < 0 {
				return eventing.ErrInvalidEvent
			}
		}
		return eventing.Apply(db.WithContext(ctx), event.ID, func(tx *gorm.DB) error {
			for id, version := range versions {
				if err := tx.Where("file_id = ? AND file_version < ?", id, version).Delete(&domain.Share{}).Error; err != nil {
					return err
				}
			}
			return nil
		})
	}
}
