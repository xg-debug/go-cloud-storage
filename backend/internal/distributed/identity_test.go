package distributed

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go-cloud-storage/backend/internal/models"
	"go-cloud-storage/backend/internal/services"
	"go-cloud-storage/backend/pkg/utils"
)

const testToken = "internal-test-token-of-at-least-32-bytes"

type userStub struct {
	services.UserService
	version    string
	sessionErr error
}

func (u *userStub) AuthenticateUser(account, password string) (*models.User, error) {
	if account != "user" || password != "password" {
		return nil, errors.New("invalid")
	}
	return &models.User{Id: 7, Password: "hash-that-must-never-cross-http", RootFolderId: "root"}, nil
}
func (u *userStub) ValidateSession(id int, version string) error {
	if u.sessionErr != nil {
		return u.sessionErr
	}
	if id != 7 || version != u.version {
		return services.ErrSessionInvalid
	}
	return nil
}
func (u *userStub) RootFolderID(int) (string, error) { return "root", nil }

func TestIdentityHTTPChainAndRevocation(t *testing.T) {
	utils.InitJWTSecret(strings.Repeat("s", 32))
	users := &userStub{version: utils.CredentialVersion("hash-that-must-never-cross-http")}
	userRouter := gin.New()
	RegisterUserInternal(userRouter, testToken, users)
	userServer := httptest.NewServer(userRouter)
	defer userServer.Close()
	userClient, _ := NewClient(userServer.URL, testToken)
	remote := &AuthUsers{Client: userClient}
	authenticated, err := remote.AuthenticateUser("user", "password")
	if err != nil {
		t.Fatal(err)
	}
	if authenticated.Password != "" || authenticated.SessionVersion != users.version {
		t.Fatal("password hash leaked or credential version lost")
	}
	authRouter := gin.New()
	RegisterAuthInternal(authRouter, testToken, remote)
	authServer := httptest.NewServer(authRouter)
	defer authServer.Close()
	authClient, _ := NewClient(authServer.URL, testToken)
	token, err := utils.GenerateAccessToken(7, time.Minute, users.version)
	if err != nil {
		t.Fatal(err)
	}
	protected := gin.New()
	protected.Use(Authenticate(authClient, nil))
	protected.GET("/file", func(c *gin.Context) {
		if c.GetInt("userId") != 7 {
			t.Error("forged identity accepted")
		}
		c.Status(204)
	})
	check := func(expected int) {
		req := httptest.NewRequest("GET", "/file", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-User-ID", "999")
		out := httptest.NewRecorder()
		protected.ServeHTTP(out, req)
		if out.Code != expected {
			t.Fatalf("got %d want %d", out.Code, expected)
		}
	}
	check(204)
	users.sessionErr = errors.New("database offline")
	check(503)
	if errors.Is(remote.ValidateSession(7, users.version), services.ErrSessionInvalid) {
		t.Fatal("dependency failure marked invalid")
	}
	users.sessionErr = nil
	check(204)
	users.version = "revoked"
	check(401)
	if !errors.Is(remote.ValidateSession(7, "old"), services.ErrSessionInvalid) {
		t.Fatal("revoked session not classified")
	}
	authServer.Close()
	check(503)
	wrong, _ := NewClient(userServer.URL, strings.Repeat("x", 32))
	if err = wrong.Call(context.Background(), "user/session", UserRequest{UserID: 7}, nil); err == nil {
		t.Fatal("invalid service token accepted")
	}
}
func TestInvalidAuthorizationNeverFallsBackToCookie(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Basic x")
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "valid-cookie"})
	if _, err := AccessToken(req); err == nil {
		t.Fatal("invalid authorization bypassed")
	}
}
