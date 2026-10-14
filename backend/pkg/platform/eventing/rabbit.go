package eventing

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"gorm.io/gorm"

	ports "go-cloud-storage/backend/pkg/ports"
)

const Exchange = "cloudbox.events.v1"

var ErrInvalidEvent = errors.New("invalid event payload")

func connect(ctx context.Context, address string) (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.DialConfig(address, amqp.Config{Dial: func(network, addr string) (net.Conn, error) {
		conn, err := (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, network, addr)
		if err == nil {
			err = conn.SetDeadline(time.Now().Add(10 * time.Second))
			if err != nil {
				conn.Close()
			}
		}
		return conn, err
	}})
	if err != nil {
		return nil, nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	if err = topology(ch); err != nil {
		ch.Close()
		conn.Close()
		return nil, nil, err
	}
	return conn, ch, nil
}
func topology(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(Exchange, "topic", true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.ExchangeDeclare(Exchange+".dead", "fanout", true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare("cloudbox.dead", true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind("cloudbox.dead", "", Exchange+".dead", false, nil); err != nil {
		return err
	}
	for queue, keys := range map[string][]string{"cloudbox.notification": {"file.uploaded.v1", "file.shared.v1"}, "cloudbox.share": {"file.deleted.v1"}} {
		if _, err := ch.QueueDeclare(queue, true, false, false, false, amqp.Table{"x-dead-letter-exchange": Exchange + ".dead"}); err != nil {
			return err
		}
		for _, key := range keys {
			if err := ch.QueueBind(queue, key, Exchange, false, nil); err != nil {
				return err
			}
		}
	}
	return nil
}
func pause(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

// Relay uses publisher confirms. A crash after publish but before marking the row
// delivered causes a duplicate, handled by the consumer's transactional Inbox.
func Relay(ctx context.Context, db *gorm.DB, address string) {
	for ctx.Err() == nil {
		if err := relayBatch(ctx, db, address); err != nil && ctx.Err() == nil {
			slog.Warn("outbox relay retrying", "error", err)
		}
		if !pause(ctx, time.Second) {
			return
		}
	}
}
func relayBatch(ctx context.Context, db *gorm.DB, address string) error {
	var rows []Outbox
	if err := db.WithContext(ctx).Where("delivered_at IS NULL").Order("created_at, id").Limit(100).Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	conn, ch, err := connect(ctx, address)
	if err != nil {
		return err
	}
	defer conn.Close()
	defer ch.Close()
	if err = ch.Confirm(false); err != nil {
		return err
	}
	for _, row := range rows {
		var event ports.Event
		if err = json.Unmarshal(row.Payload, &event); err != nil {
			return err
		}
		publishCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		confirmation, e := ch.PublishWithDeferredConfirmWithContext(publishCtx, Exchange, event.Type, false, false, amqp.Publishing{ContentType: "application/json", DeliveryMode: amqp.Persistent, MessageId: event.ID, Body: row.Payload})
		if e == nil {
			var ok bool
			ok, e = confirmation.WaitContext(publishCtx)
			if e == nil && !ok {
				e = amqp.ErrClosed
			}
		}
		cancel()
		if e != nil {
			return e
		}
		if err = db.WithContext(ctx).Model(&Outbox{}).Where("id = ?", row.ID).Update("delivered_at", time.Now()).Error; err != nil {
			return err
		}
	}
	return nil
}
func Consume(ctx context.Context, address, queue string, handle func(context.Context, ports.Event) error) {
	for ctx.Err() == nil {
		if err := consume(ctx, address, queue, handle); err != nil && ctx.Err() == nil {
			slog.Warn("event consumer reconnecting", "queue", queue, "error", err)
		}
		if !pause(ctx, time.Second) {
			return
		}
	}
}
func consume(ctx context.Context, address, queue string, handle func(context.Context, ports.Event) error) error {
	conn, ch, err := connect(ctx, address)
	if err != nil {
		return err
	}
	defer conn.Close()
	defer ch.Close()
	if err = ch.Qos(1, 0, false); err != nil {
		return err
	}
	deliveries, err := ch.Consume(queue, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-deliveries:
			if !ok {
				return amqp.ErrClosed
			}
			var event ports.Event
			if json.Unmarshal(d.Body, &event) != nil || event.ID == "" || event.Type == "" {
				_ = d.Nack(false, false)
				continue
			}
			if err = handle(ctx, event); err != nil {
				if errors.Is(err, ErrInvalidEvent) {
					_ = d.Nack(false, false)
					continue
				}
				_ = d.Nack(false, true)
				if !pause(ctx, time.Second) {
					return nil
				}
				continue
			}
			if err = d.Ack(false); err != nil {
				return err
			}
		}
	}
}
