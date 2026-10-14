package userclient

import (
	"context"
	"errors"
	"net/http"

	contracts "go-cloud-storage/backend/pkg/contracts"
	rpc "go-cloud-storage/backend/pkg/rpc"
)

// AuthUsers exposes only the operations used by auth-service, not profile/avatar mutation.
type AuthUsers struct{ Client *rpc.Client }

func (u *AuthUsers) AuthenticateUser(account, password string) (*contracts.User, error) {
	var result contracts.Credentials
	err := u.Client.Call(context.Background(), "user/authenticate", contracts.UserRequest{Account: account, Password: password}, &result)
	if err == nil && (result.User.Id <= 0 || result.Version == "") {
		err = errors.New("invalid credential response")
	}
	result.User.SessionVersion = result.Version
	return &result.User, err
}
func (u *AuthUsers) ValidateSession(id int, version string) error {
	err := u.Client.Call(context.Background(), "user/session", contracts.UserRequest{UserID: id, Version: version}, nil)
	var rpc *rpc.RPCError
	if errors.As(err, &rpc) && rpc.Status == http.StatusUnauthorized {
		return contracts.ErrSessionInvalid
	}
	return err
}
func (u *AuthUsers) RegisterUser(email, password, confirm string) error {
	return u.Client.Call(context.Background(), "user/register", contracts.UserRequest{Account: email, Password: password, Confirm: confirm}, nil)
}
func (u *AuthUsers) ForgotPassword(email string) error {
	return u.Client.Call(context.Background(), "user/forgot", contracts.UserRequest{Account: email}, nil)
}
func (u *AuthUsers) ResetPassword(token, password string) error {
	return u.Client.Call(context.Background(), "user/reset", contracts.UserRequest{Token: token, Password: password}, nil)
}
