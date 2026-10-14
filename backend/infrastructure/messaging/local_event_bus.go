package messaging

import (
	"context"
	"fmt"
	"sync"

	"go-cloud-storage/backend/internal/ports"
)

// EventHandler is an in-process subscriber used while the system is still a
// modular monolith. The same application EventBus port can later be backed by
// RabbitMQ/Outbox without changing publishers.
type EventHandler func(ctx context.Context, event ports.Event) error

type LocalEventBus struct {
	mu       sync.RWMutex
	handlers map[string][]EventHandler
}

func NewLocalEventBus() *LocalEventBus {
	return &LocalEventBus{handlers: make(map[string][]EventHandler)}
}

func (b *LocalEventBus) Subscribe(eventType string, handler EventHandler) {
	if eventType == "" || handler == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

func (b *LocalEventBus) Publish(ctx context.Context, event ports.Event) error {
	b.mu.RLock()
	handlers := append([]EventHandler(nil), b.handlers[event.Type]...)
	b.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			return fmt.Errorf("handle event %s: %w", event.Type, err)
		}
	}
	return nil
}

var _ ports.EventBus = (*LocalEventBus)(nil)
