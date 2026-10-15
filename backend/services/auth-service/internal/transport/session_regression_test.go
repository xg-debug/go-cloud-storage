package transport

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"go-cloud-storage/backend/pkg/contracts"
	"go-cloud-storage/backend/pkg/utils"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type memorySessions struct {
	sid, refresh string
	fail         error
}

func (m *memorySessions) start(_ context.Context, _ int, sid, refresh string, _ time.Duration) error {
	m.sid = sid
	m.refresh = refresh
	return m.fail
}
func (m *memorySessions) validate(_ context.Context, _ int, sid, refresh string) error {
	if m.fail != nil {
		return m.fail
	}
	if sid == "" || sid != m.sid || (refresh != "" && refresh != m.refresh) {
		return contracts.ErrSessionInvalid
	}
	return nil
}
func (m *memorySessions) revoke(_ context.Context, _ int, sid string) error {
	if m.fail != nil {
		return m.fail
	}
	if m.sid == sid {
		m.sid = ""
		m.refresh = ""
	}
	return nil
}
func TestLogoutRevokesAccessAndRefreshIncludingConcurrentRefresh(t *testing.T) {
	utils.InitJWTSecret(strings.Repeat("s", 32))
	sessions := &memorySessions{sid: "current", refresh: "refresh"}
	token, _ := utils.GenerateAccessToken(7, time.Hour, "version", "current")
	// A refresh that has already validated may mint this token after logout.
	refreshed, _ := utils.GenerateAccessToken(7, time.Hour, "version", "current")
	controller := NewLoginController(sessionUsers{})
	controller.sessions = sessions
	router := gin.New()
	router.POST("/logout", func(c *gin.Context) { c.Set("userId", 7) }, controller.Logout)
	req := httptest.NewRequest("POST", "/logout", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	out := httptest.NewRecorder()
	router.ServeHTTP(out, req)
	if out.Code != 200 || sessions.sid != "" {
		t.Fatalf("logout failed: %s", out.Body)
	}
	for _, jwt := range []string{token, refreshed} {
		claims, _ := utils.ParseTokenWithType(jwt, "access")
		if !errors.Is(sessions.validate(context.Background(), 7, claims.SessionID, ""), contracts.ErrSessionInvalid) {
			t.Fatal("access token survived logout")
		}
	}
	if !errors.Is(sessions.validate(context.Background(), 7, "current", "refresh"), contracts.ErrSessionInvalid) {
		t.Fatal("refresh survived logout")
	}
}
func TestLogoutDependencyFailureDoesNotClaimSuccess(t *testing.T) {
	utils.InitJWTSecret(strings.Repeat("s", 32))
	token, _ := utils.GenerateAccessToken(7, time.Hour, "version", "sid")
	controller := NewLoginController(sessionUsers{})
	controller.sessions = &memorySessions{fail: errors.New("redis offline")}
	r := gin.New()
	r.POST("/logout", func(c *gin.Context) { c.Set("userId", 7) }, controller.Logout)
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	out := httptest.NewRecorder()
	r.ServeHTTP(out, req)
	if out.Code != 503 || len(out.Result().Cookies()) != 0 {
		t.Fatalf("status=%d cookies=%v", out.Code, out.Result().Cookies())
	}
}
