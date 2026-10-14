package messaging

import (
	"context"
	"errors"
	"testing"

	eventing "go-cloud-storage/backend/pkg/platform/eventing"
	ports "go-cloud-storage/backend/pkg/ports"
)

func TestMalformedNotification(t *testing.T) {
	ctx := context.Background()
	if err := NotificationEventHandler(nil, nil)(ctx, ports.Event{Type: "file.uploaded.v1", UserID: 1, Data: map[string]string{}}); !errors.Is(err, eventing.ErrInvalidEvent) {
		t.Fatalf("unexpected result: %v", err)
	}
}
