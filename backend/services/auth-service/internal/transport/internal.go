package transport

import (
	"github.com/gin-gonic/gin"

	contracts "go-cloud-storage/backend/pkg/contracts"
	rpc "go-cloud-storage/backend/pkg/rpc"
	"go-cloud-storage/backend/pkg/utils"
	userclient "go-cloud-storage/backend/services/auth-service/internal/adapters/userclient"
)

func RegisterAuthInternal(r *gin.Engine, token string, users *userclient.AuthUsers) {
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
		c.JSON(200, identity)
	})
}
