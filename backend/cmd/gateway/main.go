package main

import (
	"context"
	"go-cloud-storage/backend/internal/gateway"
	"go-cloud-storage/backend/pkg/server"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	upstream := os.Getenv("GCS_GATEWAY_UPSTREAM")
	if upstream == "" {
		upstream = "http://127.0.0.1:8081"
	}
	addr := os.Getenv("GCS_GATEWAY_LISTEN")
	if addr == "" {
		addr = ":8080"
	}
	var handler http.Handler
	var err error
	if os.Getenv("GCS_AUTH_URL") != "" {
		handler, err = gateway.NewServices(map[string]string{"auth": os.Getenv("GCS_AUTH_URL"), "user": os.Getenv("GCS_USER_URL"), "file": os.Getenv("GCS_FILE_URL"), "share": os.Getenv("GCS_SHARE_URL"), "notification": os.Getenv("GCS_NOTIFICATION_URL")})
	} else {
		handler, err = gateway.New(upstream)
	}
	if err != nil {
		slog.Error("invalid gateway configuration", "error", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err = server.Run(ctx, addr, handler); err != nil {
		slog.Error("gateway stopped", "error", err)
		os.Exit(1)
	}
}
