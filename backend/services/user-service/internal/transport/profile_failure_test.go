package transport

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	application "go-cloud-storage/backend/services/user-service/internal/application"
	vo "go-cloud-storage/backend/services/user-service/internal/domain/vo"
	"gorm.io/gorm"
	"net/http/httptest"
	"strings"
	"testing"
)

type profileFailure struct {
	application.UserService
	err error
}

func (s profileFailure) GetProfile(int) (*vo.UserProfileResponse, error) { return nil, s.err }
func TestProfileDistinguishesMissingUserAndDependencyFailure(t *testing.T) {
	for _, tc := range []struct {
		name    string
		err     error
		status  int
		message string
	}{
		{"missing", fmt.Errorf("wrapped: %w", gorm.ErrRecordNotFound), 200, "用户不存在"},
		{"database unavailable", errors.New("connection pool unavailable"), 503, "用户服务暂不可用"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/me", NewUserController(profileFailure{err: tc.err}).GetProfile)
			out := httptest.NewRecorder()
			router.ServeHTTP(out, httptest.NewRequest("GET", "/me", nil))
			if out.Code != tc.status || !strings.Contains(out.Body.String(), tc.message) {
				t.Fatalf("status=%d body=%s", out.Code, out.Body)
			}
		})
	}
}
