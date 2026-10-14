package ports

import (
	"context"
	"time"
)

// Event is the transport-neutral envelope used by application modules.
// Concrete serialization/routing belongs to messaging adapters.
type Event struct {
	ID          string    `json:"eventId"`
	Type        string    `json:"eventType"`
	AggregateID string    `json:"aggregateId,omitempty"`
	UserID      int       `json:"userId,omitempty"`
	OccurredAt  time.Time `json:"occurredAt"`
	Data        any       `json:"data,omitempty"`
}

// EventBus publishes application events; the production adapter persists them in an Outbox.
type EventBus interface {
	Publish(ctx context.Context, event Event) error
}
