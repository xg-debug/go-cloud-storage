package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	contracts "go-cloud-storage/backend/pkg/contracts"
	rpc "go-cloud-storage/backend/pkg/rpc"
	"go-cloud-storage/backend/pkg/utils"
)

func AccessToken(r *http.Request) (string, error) {
	if value := r.Header.Get("Authorization"); value != "" {
		parts := strings.SplitN(value, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			return "", errors.New("invalid authorization")
		}
		return parts[1], nil
	}
	if value := r.URL.Query().Get("token"); value != "" {
		return value, nil
	}
	cookie, err := r.Cookie("access_token")
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}
func Authenticate(auth *rpc.Client, ensure func(context.Context, contracts.Identity) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := AccessToken(c.Request)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"code": utils.CodeUnauthorized, "message": "未登录或登录已过期"})
			return
		}
		validate := func() (contracts.Identity, error) {
			var out contracts.Identity
			err := auth.Call(c.Request.Context(), "auth/introspect", gin.H{"token": token}, &out)
			if err == nil && out.UserID <= 0 {
				err = errors.New("invalid identity")
			}
			return out, err
		}
		identity, err := validate()
		if err != nil {
			if status := rpc.AuthFailureStatus(err); status == 401 {
				c.AbortWithStatusJSON(401, gin.H{"code": utils.CodeTokenExpired, "message": "登录已失效"})
			} else {
				c.AbortWithStatusJSON(503, gin.H{"code": 503, "message": "认证服务暂不可用，请稍后重试"})
			}
			return
		}
		if ensure != nil {
			if err = ensure(c.Request.Context(), identity); err != nil {
				c.AbortWithStatus(503)
				return
			}
		}
		c.Set("userId", identity.UserID)
		c.Set("validateSession", func() error { _, err := validate(); return err })
		c.Next()
	}
}
