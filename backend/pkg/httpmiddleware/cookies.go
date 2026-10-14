package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ClearAuthCookies clears browser credentials after logout or a credential change.
func ClearAuthCookies(ctx *gin.Context) {
	ctx.SetSameSite(http.SameSiteLaxMode)
	secure := ctx.Request.TLS != nil || ctx.GetHeader("X-Forwarded-Proto") == "https"
	for _, name := range []string{"access_token", "refresh_token", "csrf_token"} {
		ctx.SetCookie(name, "", -1, "/", "", secure, name != "csrf_token")
	}
}
