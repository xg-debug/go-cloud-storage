package transport

import (
	"errors"
	"github.com/gin-gonic/gin"

	contracts "go-cloud-storage/backend/pkg/contracts"
	rpc "go-cloud-storage/backend/pkg/rpc"
	"go-cloud-storage/backend/pkg/utils"
	userclient "go-cloud-storage/backend/services/auth-service/internal/adapters/userclient"
)

func RegisterAuthInternal(r *gin.Engine, token string, users *userclient.AuthUsers, stores ...sessionStore) {
	var sessions sessionStore = redisSessions{}
	if len(stores) > 0 {
		sessions = stores[0]
	}
	r.POST(rpc.Prefix+"auth/introspect", rpc.Protect(token), func(c *gin.Context) {
		var in struct {
			Token string `json:"token"`
		}
		if !rpc.Bind(c, &in) {
			return
		}
		claims, err := utils.ParseTokenWithType(in.Token, "access")
		if err != nil {
			c.Status(401)
			return
		}
		var identity contracts.Identity
		if err = users.Client.Call(c.Request.Context(), "user/session", contracts.UserRequest{UserID: claims.UserId, Version: claims.CredentialVersion}, &identity); err != nil {
			c.Status(rpc.AuthFailureStatus(err))
			return
		}
		if err = sessions.validate(c.Request.Context(), claims.UserId, claims.SessionID, ""); err != nil {
			if errors.Is(err, contracts.ErrSessionInvalid) {
				c.Status(401)
			} else {
				c.Status(503)
			}
			return
		}
		c.JSON(200, identity)
	})
}
