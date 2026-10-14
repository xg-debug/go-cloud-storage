package distributed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"go-cloud-storage/backend/infrastructure/eventing"
	"go-cloud-storage/backend/internal/models"
	"go-cloud-storage/backend/internal/ports"
	"go-cloud-storage/backend/internal/services"
	"gorm.io/gorm"
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
				if err := tx.Where("file_id = ? AND file_version < ?", id, version).Delete(&models.Share{}).Error; err != nil {
					return err
				}
			}
			return nil
		})
	}
}

const notificationChannel = "cloudbox.notifications.v1"

func NotificationEventHandler(db *gorm.DB, rdb *redis.Client) func(context.Context, ports.Event) error {
	return func(ctx context.Context, event ports.Event) error {
		if event.Type != "file.uploaded.v1" && event.Type != "file.shared.v1" {
			return nil
		}
		if event.UserID <= 0 {
			return eventing.ErrInvalidEvent
		}
		data, err := json.Marshal(event.Data)
		if err != nil {
			return err
		}
		var info struct {
			FileName string `json:"fileName"`
		}
		if err = json.Unmarshal(data, &info); err != nil || info.FileName == "" {
			return eventing.ErrInvalidEvent
		}
		title, message, link := "文件上传完成", fmt.Sprintf("文件 %q 已成功上传", info.FileName), "/"
		if event.Type == "file.shared.v1" {
			title = "文件分享成功"
			message = fmt.Sprintf("文件 %q 已成功创建分享链接", info.FileName)
			link = "/shared"
		}
		err = eventing.Apply(db.WithContext(ctx), event.ID, func(tx *gorm.DB) error {
			return tx.Create(&models.Notification{EventID: &event.ID, UserID: uint(event.UserID), Title: title, Message: message, Type: "success", Link: link}).Error
		})
		if err != nil {
			return err
		}
		// Re-query on redelivery: a failed Redis publication can be retried without
		// inserting a second notification. Deleted notifications are never recreated.
		var n models.Notification
		err = db.WithContext(ctx).Where("event_id = ?", event.ID).First(&n).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		payload, err := json.Marshal(n)
		if err != nil {
			return err
		}
		return rdb.Publish(ctx, notificationChannel, payload).Err()
	}
}
func NotificationFanout(ctx context.Context, rdb *redis.Client, broker *services.SSEBroker) {
	subscription := rdb.Subscribe(ctx, notificationChannel)
	defer subscription.Close()
	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-subscription.Channel():
			if !ok {
				return
			}
			var n models.Notification
			if json.Unmarshal([]byte(message.Payload), &n) == nil {
				broker.SendToUser(int(n.UserID), "notification", map[string]any{"id": n.ID, "title": n.Title, "message": n.Message, "type": n.Type, "link": n.Link})
			}
		}
	}
}
