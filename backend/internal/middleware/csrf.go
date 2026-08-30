package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go-cloud-storage/backend/pkg/utils"
)

func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isSafeMethod(c.Request.Method) {
			c.Next()
			return
		}

		// Bearer token clients are not vulnerable to browser-driven CSRF in the same way,
		// because cross-site forms cannot set Authorization headers.
		if strings.HasPrefix(strings.ToLower(c.GetHeader("Authorization")), "bearer ") {
			c.Next()
			return
		}

		headerToken := c.GetHeader("X-CSRF-Token")
		cookieToken, err := c.Cookie("csrf_token")
		if err != nil || headerToken == "" || cookieToken == "" || headerToken != cookieToken {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":      utils.CodeForbidden,
				"message":   "CSRF token missing or invalid",
				"requestId": utils.GetRequestID(c),
			})
			return
		}

		c.Next()
	}
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}
