package middleware

import (
	"github.com/gin-gonic/gin"
	"go-cloud-storage/backend/pkg/utils"
	"net/http"
	"strings"
)

func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		tokenString := ""
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"code":      utils.CodeUnauthorized,
					"message":   "Authorization header format must be 'Bearer <token>'",
					"requestId": utils.GetRequestID(c),
				})
				return
			}
			tokenString = parts[1]
		}

		// EventSource 不支持自定义 header，允许通过 query param 传递 token
		if tokenString == "" {
			tokenString = c.Query("token")
		}
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":      utils.CodeUnauthorized,
				"message":   "未登录或登录已过期",
				"requestId": utils.GetRequestID(c),
			})
			return
		}

		claims, err := utils.ParseToken(tokenString)
		if err != nil {
			c.SetCookie("token", "", -1, "/", "", false, true)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":      utils.CodeTokenExpired,
				"message":   "令牌无效或已过期",
				"requestId": utils.GetRequestID(c),
			})
			return
		}

		c.Set("userId", int(claims.UserId))
		c.Next()
	}
}
