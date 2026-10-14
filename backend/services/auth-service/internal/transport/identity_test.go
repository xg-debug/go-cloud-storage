package transport

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	authrpc "go-cloud-storage/backend/pkg/auth"
	"go-cloud-storage/backend/pkg/contracts"
	rpc "go-cloud-storage/backend/pkg/rpc"
	"go-cloud-storage/backend/pkg/utils"
	userclient "go-cloud-storage/backend/services/auth-service/internal/adapters/userclient"
)

const testToken = "internal-test-token-of-at-least-32-bytes"

type userStub struct {
	LoginUsers
	version    string
	sessionErr error
}

func (u *userStub) AuthenticateUser(account, password string) (*contracts.User, error) {
	if account != "user" || password != "password" {
		return nil, errors.New("invalid")
	}
	return &contracts.User{Id: 7, RootFolderId: "root"}, nil
}
func (u *userStub) ValidateSession(id int, version string) error {
	if u.sessionErr != nil {
		return u.sessionErr
	}
	if id != 7 || version != u.version {
		return contracts.ErrSessionInvalid
	}
	return nil
}
func (u *userStub) RootFolderID(int) (string, error) { return "root", nil }

func TestIdentityHTTPChainAndRevocation(t *testing.T) {
	utils.InitJWTSecret(strings.Repeat("s", 32))
	users := &userStub{version: utils.CredentialVersion("hash-that-must-never-cross-http")}
	userRouter := gin.New()
	registerUserFixture(userRouter, users)
	userServer := httptest.NewServer(userRouter)
	defer userServer.Close()
	userClient, _ := rpc.NewClient(userServer.URL, testToken)
	remote := &userclient.AuthUsers{Client: userClient}
	authenticated, err := remote.AuthenticateUser("user", "password")
	if err != nil {
		t.Fatal(err)
	}
	if authenticated.SessionVersion != users.version {
		t.Fatal("credential version lost")
	}
	authRouter := gin.New()
	RegisterAuthInternal(authRouter, testToken, remote)
	authServer := httptest.NewServer(authRouter)
	defer authServer.Close()
	authClient, _ := rpc.NewClient(authServer.URL, testToken)
	token, err := utils.GenerateAccessToken(7, time.Minute, users.version)
	if err != nil {
		t.Fatal(err)
	}
	protected := gin.New()
	protected.Use(authrpc.Authenticate(authClient, nil))
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
	if errors.Is(remote.ValidateSession(7, users.version), contracts.ErrSessionInvalid) {
		t.Fatal("dependency failure marked invalid")
	}
	users.sessionErr = nil
	check(204)
	users.version = "revoked"
	check(401)
	if !errors.Is(remote.ValidateSession(7, "old"), contracts.ErrSessionInvalid) {
		t.Fatal("revoked session not classified")
	}
	authServer.Close()
	check(503)
	wrong, _ := rpc.NewClient(userServer.URL, strings.Repeat("x", 32))
	if err = wrong.Call(context.Background(), "user/session", contracts.UserRequest{UserID: 7}, nil); err == nil {
		t.Fatal("invalid service token accepted")
	}
}
func TestInvalidAuthorizationNeverFallsBackToCookie(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Basic x")
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "valid-cookie"})
	if _, err := authrpc.AccessToken(req); err == nil {
		t.Fatal("invalid authorization bypassed")
	}
}

// Test double exercises the published user RPC contract without importing its implementation.
func registerUserFixture(r *gin.Engine, u *userStub) {
	r.POST(rpc.Prefix+"user/:action", rpc.Protect(testToken), func(c *gin.Context) {
		var in contracts.UserRequest
		if !rpc.Bind(c, &in) {
			return
		}
		switch c.Param("action") {
		case "authenticate":
			user, err := u.AuthenticateUser(in.Account, in.Password)
			if err != nil {
				c.Status(401)
				return
			}
			c.JSON(200, contracts.Credentials{User: *user, Version: u.version})
		case "session":
			err := u.ValidateSession(in.UserID, in.Version)
			if err != nil {
				if errors.Is(err, contracts.ErrSessionInvalid) {
					c.Status(401)
				} else {
					c.Status(503)
				}
				return
			}
			c.JSON(200, contracts.Identity{UserID: 7, RootID: "root"})
		default:
			c.Status(404)
		}
	})
}
