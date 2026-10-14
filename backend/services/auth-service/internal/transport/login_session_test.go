package transport

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	contracts "go-cloud-storage/backend/pkg/contracts"
	"go-cloud-storage/backend/pkg/utils"
)

type sessionUsers struct {
	LoginUsers
	err error
}

func (s sessionUsers) ValidateSession(int, string) error { return s.err }
func TestRefreshPreservesCookiesDuringDependencyFailure(t *testing.T) {
	utils.InitJWTSecret(strings.Repeat("s", 32))
	for _, tc := range []struct {
		name   string
		err    error
		status int
		clear  bool
	}{
		{"user outage", errors.New("database unavailable"), 503, false},
		{"revoked", contracts.ErrSessionInvalid, 200, true},
		{"redis unavailable", nil, 503, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			token, err := utils.GenerateRefreshToken(42, time.Hour, "version")
			if err != nil {
				t.Fatal(err)
			}
			r := gin.New()
			r.POST("/refresh", NewLoginController(sessionUsers{err: tc.err}).RefreshToken)
			req := httptest.NewRequest("POST", "/refresh", nil)
			req.AddCookie(&http.Cookie{Name: "refresh_token", Value: token})
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.status {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			cleared := false
			for _, c := range w.Result().Cookies() {
				if c.Name == "refresh_token" && c.MaxAge < 0 {
					cleared = true
				}
			}
			if cleared != tc.clear {
				t.Fatalf("cleared=%v wanted=%v", cleared, tc.clear)
			}
		})
	}
}
