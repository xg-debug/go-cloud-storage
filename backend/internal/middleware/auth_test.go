package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go-cloud-storage/backend/pkg/utils"
)

func TestJWTSessionValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	utils.InitJWTSecret("test-secret-at-least-thirty-two-bytes-long")
	current := "version-one"
	router := gin.New()
	router.GET("/private", JWTAuthMiddleware(func(id int, version string) error {
		if id != 1 || version != current {
			return errors.New("revoked")
		}
		return nil
	}), func(c *gin.Context) { c.Status(200) })
	token, err := utils.GenerateAccessToken(1, time.Hour, current)
	if err != nil {
		t.Fatal(err)
	}
	request := func(token string) int {
		req := httptest.NewRequest(http.MethodGet, "/private", nil)
		req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		return resp.Code
	}
	if code := request(token); code != 200 {
		t.Fatalf("valid token: %d", code)
	}
	current = "version-two"
	if code := request(token); code != 401 {
		t.Fatalf("revoked token: %d", code)
	}
	refresh, _ := utils.GenerateRefreshToken(1, time.Hour, current)
	if code := request(refresh); code != 401 {
		t.Fatalf("refresh token used as access: %d", code)
	}
}
