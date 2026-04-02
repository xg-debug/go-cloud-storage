package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go-cloud-storage/backend/pkg/config"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	defaultExchange   = "recycle.exchange"
	defaultQueue      = "recycle.expired.queue"
	defaultRoutingKey = "recycle.expired"
)

type RecycleExpiredMessage struct {
	FileID    string `json:"fileId"`
	Timestamp int64  `json:"timestamp"`
}

type RabbitMQClient struct {
	conn       *amqp.Connection
	channel    *amqp.Channel
	exchange   string
	queue      string
	routingKey string
	consumer   string
}

func NewRabbitMQClient(cfg *config.RabbitMQConfig) (*RabbitMQClient, error) {
	if cfg == nil || !cfg.Enabled {
		return nil, nil
	}
	if cfg.URL == "" {
		return nil, fmt.Errorf("rabbitmq.url is required when enabled")
	}

	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("connect rabbitmq failed: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("open rabbitmq channel failed: %w", err)
	}

	client := &RabbitMQClient{
		conn:       conn,
		channel:    ch,
		exchange:   valueOrDefault(cfg.Exchange, defaultExchange),
		queue:      valueOrDefault(cfg.Queue, defaultQueue),
		routingKey: valueOrDefault(cfg.RoutingKey, defaultRoutingKey),
		consumer:   valueOrDefault(cfg.ConsumerTag, "recycle-cleanup-worker"),
	}

	if err := client.setup(); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}
	return client, nil
}

func (c *RabbitMQClient) setup() error {
	if err := c.channel.ExchangeDeclare(
		c.exchange,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("declare exchange failed: %w", err)
	}

	queue, err := c.channel.QueueDeclare(
		c.queue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("declare queue failed: %w", err)
	}

	if err := c.channel.QueueBind(
		queue.Name,
		c.routingKey,
		c.exchange,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("bind queue failed: %w", err)
	}

	return c.channel.Qos(8, 0, false)
}

func (c *RabbitMQClient) PublishExpiredFilePurge(ctx context.Context, fileID string) error {
	if c == nil {
		return nil
	}
	if fileID == "" {
		return fmt.Errorf("file id is required")
	}

	body, err := json.Marshal(RecycleExpiredMessage{
		FileID:    fileID,
		Timestamp: time.Now().Unix(),
	})
	if err != nil {
		return err
	}

	return c.channel.PublishWithContext(ctx,
		c.exchange,
		c.routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
			Timestamp:    time.Now(),
		},
	)
}

func (c *RabbitMQClient) ConsumeExpiredFilePurge(ctx context.Context, handler func(context.Context, string) error) error {
	if c == nil {
		return nil
	}
	deliveries, err := c.channel.Consume(
		c.queue,
		c.consumer,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-deliveries:
			if !ok {
				return fmt.Errorf("rabbitmq delivery channel closed")
			}

			payload := RecycleExpiredMessage{}
			if err := json.Unmarshal(msg.Body, &payload); err != nil {
				_ = msg.Nack(false, false)
				continue
			}
			if payload.FileID == "" {
				_ = msg.Nack(false, false)
				continue
			}

			if err := handler(ctx, payload.FileID); err != nil {
				_ = msg.Nack(false, true)
				continue
			}
			_ = msg.Ack(false)
		}
	}
}

func (c *RabbitMQClient) Close() error {
	if c == nil {
		return nil
	}
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func valueOrDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
