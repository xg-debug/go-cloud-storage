package middleware

import (
	"github.com/gin-gonic/gin"
	"go-cloud-storage/internal/pkg/utils"
	"net/http"
	"strings"
)

func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 适用于前后端分离架构
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}
		// Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
		// 分割Bearer和Token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be 'Bearer <token>'"})
			return
		}

		tokenString := parts[1] // 获取真正的Token部分

		// 1. 从Cookie获取Token：适用于前后端不分离
		//tokenString, _ := c.Cookie("token")
		//fmt.Println("token: ", tokenString)
		//if tokenString == "" {
		//	c.Redirect(http.StatusFound, "/") // 跳转到首页(登录页)
		//	c.Abort()
		//	return
		//}

		// 2.解析并验证JWT
		claims, err := utils.ParseToken(tokenString)
		if err != nil {
			c.SetCookie("token", "", -1, "/", "", false, true) // 清除无效Cookie
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "无效Token"})
			return
		}

		// 存储用户信息到上下文
		c.Set("userId", int(claims.UserId))
		c.Next()
	}
}
