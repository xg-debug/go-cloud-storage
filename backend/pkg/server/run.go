package server

import (
	"context"
	"errors"
	"net/http"
	"time"
)

// Run keeps streaming responses unbounded while limiting idle/header reads.
func Run(ctx context.Context, addr string, handler http.Handler) error {
	srv := &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 90 * time.Second}
	result := make(chan error, 1)
	go func() { result <- srv.ListenAndServe() }()
	select {
	case err := <-result:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdown); err != nil {
			_ = srv.Close()
			return err
		}
		return nil
	}
}
