package transport

import (
	"errors"

	"github.com/gin-gonic/gin"

	contracts "go-cloud-storage/backend/pkg/contracts"
	rpc "go-cloud-storage/backend/pkg/rpc"
	"go-cloud-storage/backend/pkg/utils"
	application "go-cloud-storage/backend/services/user-service/internal/application"
	domain "go-cloud-storage/backend/services/user-service/internal/domain"
)

func RegisterUserInternal(r *gin.Engine, token string, users application.IdentityUserService) {
	group := r.Group(rpc.Prefix, rpc.Protect(token))
	group.POST("user/:action", func(c *gin.Context) {
		var in contracts.UserRequest
		if !rpc.Bind(c, &in) {
			return
		}
		var err error
		var result any = gin.H{}
		switch c.Param("action") {
		case "authenticate":
			var user *domain.User
			user, err = users.AuthenticateUser(in.Account, in.Password)
			if err == nil {
				version := utils.CredentialVersion(user.Password)
				user.Password = ""
				result = contracts.Credentials{User: contracts.User{Id: user.Id, Username: user.Username, Email: user.Email, Phone: user.Phone, Avatar: user.Avatar, OpenId: user.OpenId, RegisterTime: user.RegisterTime, RootFolderId: user.RootFolderId}, Version: version}
			}
		case "session":
			err = users.ValidateSession(in.UserID, in.Version)
			if err == nil {
				var root string
				root, err = users.RootFolderID(in.UserID)
				result = contracts.Identity{UserID: in.UserID, RootID: root}
			}
		case "register":
			err = users.RegisterUser(in.Account, in.Password, in.Confirm)
		case "forgot":
			err = users.ForgotPassword(in.Account)
		case "reset":
			err = users.ResetPassword(in.Token, in.Password)
		default:
			c.Status(404)
			return
		}
		if err != nil {
			if c.Param("action") == "session" && !errors.Is(err, application.ErrSessionInvalid) {
				c.Status(503)
			} else {
				c.Status(401)
			}
			return
		}
		c.JSON(200, result)
	})
}
