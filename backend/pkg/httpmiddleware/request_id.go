package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 优先使用客户端传入的 X-Request-Id
		requestID := c.GetHeader("X-Request-Id")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("requestId", requestID)
		c.Header("X-Request-Id", requestID)
		c.Next()
	}
}
