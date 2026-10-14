package gateway

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	middleware "go-cloud-storage/backend/pkg/httpmiddleware"
)

// New retains the single-upstream compatibility mode.
func New(upstream string) (http.Handler, error) {
	return NewServices(map[string]string{"auth": upstream, "user": upstream, "file": upstream, "share": upstream, "notification": upstream})
}

func makeProxy(upstream string) (*httputil.ReverseProxy, error) {
	target, err := url.Parse(upstream)
	if err != nil || target.Host == "" || (target.Scheme != "http" && target.Scheme != "https") || target.User != nil || target.RawQuery != "" || target.Fragment != "" || (target.Path != "" && target.Path != "/") {
		return nil, errors.New("gateway upstream must be an http(s) origin")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 5 * time.Minute
	proxy := &httputil.ReverseProxy{
		Transport: transport, FlushInterval: -1,
		Rewrite: func(req *httputil.ProxyRequest) {
			req.SetURL(target)
			if strings.HasPrefix(req.Out.URL.Path, "/api/") {
				req.Out.URL.Path = strings.TrimPrefix(req.Out.URL.Path, "/api")
				req.Out.URL.RawPath = ""
			}
			req.Out.Header.Del("X-User-ID")
			req.Out.Header.Del("X-User-Roles")
			// ReverseProxy removes incoming Forwarded and X-Forwarded-* before Rewrite.
			req.SetXForwarded()
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			slog.Warn("gateway upstream unavailable", "path", r.URL.Path, "error", err)
			http.Error(w, "upstream unavailable", http.StatusBadGateway)
		},
	}
	return proxy, nil
}

// NewServices routes public APIs to their owning service. Internal APIs and the
// storage service are never exposed by this gateway.
func NewServices(upstreams map[string]string) (http.Handler, error) {
	proxies := map[string]*httputil.ReverseProxy{}
	for _, role := range []string{"auth", "user", "file", "share", "notification"} {
		p, err := makeProxy(upstreams[role])
		if err != nil {
			return nil, err
		}
		proxies[role] = p
	}
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		defer func() {
			if value := recover(); value != nil {
				// A broken upstream stream must abort the connection, not look successful.
				if value == http.ErrAbortHandler {
					panic(value)
				}
				slog.Error("gateway handler panicked", "path", c.Request.URL.Path)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}, middleware.RequestIDMiddleware())
	_ = engine.SetTrustedProxies(nil)
	engine.GET("/healthz", func(c *gin.Context) { c.Status(http.StatusOK) })
	engine.HEAD("/healthz", func(c *gin.Context) { c.Status(http.StatusOK) })
	engine.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/") {
			path = strings.TrimPrefix(path, "/api")
		}
		role := RouteService(path)
		if role == "" {
			c.Status(http.StatusNotFound)
			return
		}
		c.Request.Header.Set("X-Request-Id", c.GetString("requestId"))
		proxies[role].ServeHTTP(c.Writer, c.Request)
	})
	return engine, nil
}

func RouteService(path string) string {
	if path == "/user/stats" || path == "/user/quota" {
		return "file"
	}
	first := strings.Split(strings.TrimPrefix(path, "/"), "/")[0]
	switch first {
	case "login", "register", "refresh-token", "forgot-password", "reset-password", "logout":
		return "auth"
	case "me", "user":
		return "user"
	case "file", "favorite", "recycle", "category":
		return "file"
	case "share", "s":
		return "share"
	case "notification":
		return "notification"
	}
	return ""
}
