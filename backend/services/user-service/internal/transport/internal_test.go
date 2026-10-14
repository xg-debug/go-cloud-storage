package transport

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"go-cloud-storage/backend/pkg/contracts"
	"go-cloud-storage/backend/pkg/utils"
	application "go-cloud-storage/backend/services/user-service/internal/application"
	domain "go-cloud-storage/backend/services/user-service/internal/domain"
)

type identityStub struct {
	application.IdentityUserService
	err error
}

func (u *identityStub) AuthenticateUser(string, string) (*domain.User, error) {
	return &domain.User{Id: 7, Password: "private-hash", RootFolderId: "root"}, nil
}
func (u *identityStub) ValidateSession(int, string) error { return u.err }
func (u *identityStub) RootFolderID(int) (string, error)  { return "root", nil }
func TestUserRPCContract(t *testing.T) {
	utils.InitJWTSecret(strings.Repeat("s", 32))
	u := &identityStub{}
	r := gin.New()
	token := strings.Repeat("t", 32)
	RegisterUserInternal(r, token, u)
	for _, tc := range []struct {
		op   string
		err  error
		code int
	}{{"authenticate", nil, 200}, {"session", nil, 200}, {"session", contracts.ErrSessionInvalid, 401}, {"session", errors.New("database offline"), 503}} {
		u.err = tc.err
		req := httptest.NewRequest("POST", "/internal/v1/user/"+tc.op, strings.NewReader(`{"userId":7,"version":"v"}`))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != tc.code || strings.Contains(w.Body.String(), "private-hash") {
			t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
		}
	}
}
