package gateway

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestProxyPreservesAPIContractAndStripsIdentity(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/file/upload" || r.URL.RawQuery != "parentId=abc" {
			t.Errorf("unexpected URL %s", r.URL)
		}
		if r.Header.Get("X-Request-Id") == "" {
			t.Error("request ID not propagated")
		}
		if r.Header.Get("X-User-ID") != "" || r.Header.Get("X-User-Roles") != "" {
			t.Error("untrusted identity forwarded")
		}
		if r.Header.Get("X-Forwarded-For") == "attacker" {
			t.Error("untrusted forwarded header retained")
		}
		if r.Header.Get("Cookie") != "access_token=test" || r.Header.Get("X-CSRF-Token") != "csrf" {
			t.Error("session headers lost")
		}
		data, _ := io.ReadAll(r.Body)
		if string(data) != "content" {
			t.Error("upload body changed")
		}
		w.Header().Set("Set-Cookie", "access_token=next; HttpOnly; Path=/")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(data)
	}))
	defer upstream.Close()
	handler, err := New(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/file/upload", "/file/upload"} {
		req := httptest.NewRequest(http.MethodPost, path+"?parentId=abc", strings.NewReader("content"))
		req.Header.Set("X-User-ID", "99")
		req.Header.Set("X-User-Roles", "admin")
		req.Header.Set("X-Forwarded-For", "attacker")
		req.Header.Set("Cookie", "access_token=test")
		req.Header.Set("X-CSRF-Token", "csrf")
		ctx, cancel := context.WithCancel(req.Context())
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req.WithContext(ctx))
		cancel()
		if res.Code != http.StatusCreated || res.Body.String() != "content" || res.Header().Get("Set-Cookie") == "" {
			t.Fatalf("bad proxy response: %v", res)
		}
	}
	for _, path := range []string{"/internal/storage/v1/delete", "/api/internal/storage/v1/delete", "/admin"} {
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, httptest.NewRequest(http.MethodPost, path, nil))
		if res.Code != 404 {
			t.Errorf("exposed %s", path)
		}
	}
}

func TestProxyStreamsSSEBeforeUpstreamCloses(t *testing.T) {
	release := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: ready\n\n")
		w.(http.Flusher).Flush()
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer upstream.Close()
	handler, _ := New(upstream.URL)
	srv := httptest.NewServer(handler)
	defer srv.Close()
	defer close(release)
	client := srv.Client()
	client.Timeout = 2 * time.Second
	res, err := client.Get(srv.URL + "/api/notification/stream")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	data := make([]byte, len("data: ready\n\n"))
	if _, err = io.ReadFull(res.Body, data); err != nil {
		t.Fatal(err)
	}
	if string(data) != "data: ready\n\n" {
		t.Fatal(string(data))
	}
}
