package distributed

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"go-cloud-storage/backend/internal/models"
	"go-cloud-storage/backend/internal/services"
	"go-cloud-storage/backend/pkg/utils"
	"net/http"
	"strings"
)

type Identity struct {
	UserID int    `json:"userId"`
	RootID string `json:"rootId"`
}
type UserRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
	Confirm  string `json:"confirm"`
	UserID   int    `json:"userId"`
	Version  string `json:"version"`
	Token    string `json:"token"`
}
type Credentials struct {
	User    models.User `json:"user"`
	Version string      `json:"version"`
}

// AuthUsers exposes only the operations used by auth-service, not profile/avatar mutation.
type AuthUsers struct{ Client *Client }

func (u *AuthUsers) AuthenticateUser(account, password string) (*models.User, error) {
	var result Credentials
	err := u.Client.Call(context.Background(), "user/authenticate", UserRequest{Account: account, Password: password}, &result)
	if err == nil && (result.User.Id <= 0 || result.Version == "") {
		err = errors.New("invalid credential response")
	}
	result.User.SessionVersion = result.Version
	return &result.User, err
}
func (u *AuthUsers) ValidateSession(id int, version string) error {
	err := u.Client.Call(context.Background(), "user/session", UserRequest{UserID: id, Version: version}, nil)
	var rpc *RPCError
	if errors.As(err, &rpc) && rpc.Status == http.StatusUnauthorized {
		return services.ErrSessionInvalid
	}
	return err
}
func (u *AuthUsers) RegisterUser(email, password, confirm string) error {
	return u.Client.Call(context.Background(), "user/register", UserRequest{Account: email, Password: password, Confirm: confirm}, nil)
}
func (u *AuthUsers) ForgotPassword(email string) error {
	return u.Client.Call(context.Background(), "user/forgot", UserRequest{Account: email}, nil)
}
func (u *AuthUsers) ResetPassword(token, password string) error {
	return u.Client.Call(context.Background(), "user/reset", UserRequest{Token: token, Password: password}, nil)
}

func RegisterUserInternal(r *gin.Engine, token string, users services.IdentityUserService) {
	group := r.Group(Prefix, Protect(token))
	group.POST("user/:action", func(c *gin.Context) {
		var in UserRequest
		if !Bind(c, &in) {
			return
		}
		var err error
		var result any = gin.H{}
		switch c.Param("action") {
		case "authenticate":
			var user *models.User
			user, err = users.AuthenticateUser(in.Account, in.Password)
			if err == nil {
				version := utils.CredentialVersion(user.Password)
				user.Password = ""
				result = Credentials{User: *user, Version: version}
			}
		case "session":
			err = users.ValidateSession(in.UserID, in.Version)
			if err == nil {
				var root string
				root, err = users.RootFolderID(in.UserID)
				result = Identity{UserID: in.UserID, RootID: root}
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
			if c.Param("action") == "session" && !errors.Is(err, services.ErrSessionInvalid) {
				c.Status(503)
			} else {
				c.Status(401)
			}
			return
		}
		c.JSON(200, result)
	})
}
func RegisterAuthInternal(r *gin.Engine, token string, users *AuthUsers) {
	r.POST(Prefix+"auth/introspect", Protect(token), func(c *gin.Context) {
		var in struct {
			Token string `json:"token"`
		}
		if !Bind(c, &in) {
			return
		}
		claims, err := utils.ParseTokenWithType(in.Token, "access")
		if err != nil {
			c.Status(401)
			return
		}
		var identity Identity
		if err = users.Client.Call(c.Request.Context(), "user/session", UserRequest{UserID: claims.UserId, Version: claims.CredentialVersion}, &identity); err != nil {
			c.Status(authFailureStatus(err))
			return
		}
		c.JSON(200, identity)
	})
}
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
func Authenticate(auth *Client, ensure func(context.Context, Identity) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := AccessToken(c.Request)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"code": utils.CodeUnauthorized, "message": "未登录或登录已过期"})
			return
		}
		validate := func() (Identity, error) {
			var out Identity
			err := auth.Call(c.Request.Context(), "auth/introspect", gin.H{"token": token}, &out)
			if err == nil && out.UserID <= 0 {
				err = errors.New("invalid identity")
			}
			return out, err
		}
		identity, err := validate()
		if err != nil {
			if status := authFailureStatus(err); status == 401 {
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
