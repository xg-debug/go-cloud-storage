package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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
	url        string
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
		url:        cfg.URL,
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

// ensureConnected 确保底层连接与 channel 可用；连接断开时自动重连（指数退避）。
func (c *RabbitMQClient) ensureConnected(ctx context.Context) error {
	if c.conn != nil && !c.conn.IsClosed() && c.channel != nil {
		return nil
	}
	backoff := time.Second
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		conn, err := amqp.Dial(c.url)
		if err == nil {
			ch, chErr := conn.Channel()
			if chErr == nil {
				if setupErr := c.setupChannel(ch); setupErr == nil {
					c.closeOld()
					c.conn = conn
					c.channel = ch
					slog.Info("rabbitmq reconnected")
					return nil
				}
				_ = ch.Close()
				err = fmt.Errorf("setup channel: %w", chErr)
			}
			_ = conn.Close()
			if err == nil {
				err = fmt.Errorf("open channel: %w", chErr)
			}
		}
		slog.Warn("rabbitmq reconnect failed", "error", err, "retryIn", backoff)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

// closeOld 关闭旧的连接与 channel（重连成功时调用）
func (c *RabbitMQClient) closeOld() {
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

func (c *RabbitMQClient) setup() error {
	return c.setupChannel(c.channel)
}

func (c *RabbitMQClient) setupChannel(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(
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

	queue, err := ch.QueueDeclare(
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

	if err := ch.QueueBind(
		queue.Name,
		c.routingKey,
		c.exchange,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("bind queue failed: %w", err)
	}

	return ch.Qos(8, 0, false)
}

// PublishExpiredFilePurge 生产者：发布过期文件清理消息
func (c *RabbitMQClient) PublishExpiredFilePurge(ctx context.Context, fileID string) error {
	if fileID == "" {
		return fmt.Errorf("file id is required")
	}
	if err := c.ensureConnected(ctx); err != nil {
		return err
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

// ConsumeExpiredFilePurge 消费者：消费过期文件清理消息。
// 连接中断时自动重连（指数退避），避免回收站清理永久停摆。
func (c *RabbitMQClient) ConsumeExpiredFilePurge(ctx context.Context, handler func(context.Context, string) error) error {
	if c == nil {
		return nil
	}

	for {
		if err := c.ensureConnected(ctx); err != nil {
			return err
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
			slog.Warn("rabbitmq consume failed, will retry", "error", err)
			sleepCtx(ctx, 2*time.Second)
			continue
		}

	consumeLoop:
		for {
			select {
			case <-ctx.Done():
				return nil
			case msg, ok := <-deliveries:
				if !ok {
					// delivery channel 关闭（连接断开）：退出内层循环触发重连
					break consumeLoop
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
		slog.Warn("rabbitmq delivery channel closed, reconnecting")
		sleepCtx(ctx, 2*time.Second)
	}
}

func sleepCtx(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
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
