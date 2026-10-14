package bootstrap

import (
	"context"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"go-cloud-storage/backend/pkg/config"
	contracts "go-cloud-storage/backend/pkg/contracts"
	middleware "go-cloud-storage/backend/pkg/httpmiddleware"
	rpc "go-cloud-storage/backend/pkg/rpc"
	"go-cloud-storage/backend/pkg/utils"
	userclient "go-cloud-storage/backend/services/auth-service/internal/adapters/userclient"
	transport "go-cloud-storage/backend/services/auth-service/internal/transport"
)

type ServiceClients struct {
	User  *rpc.Client
	Token string
}

func setUp(ctx context.Context, cfg *config.Config, clients ServiceClients) (*gin.Engine, error) {
	ginServer := gin.New()
	ginServer.Use(gin.Recovery(), middleware.RequestIDMiddleware())
	if err := ginServer.SetTrustedProxies(cfg.Server.TrustedProxies); err != nil {
		return nil, err
	}
	ginServer.GET("/healthz", func(c *gin.Context) { c.Status(200) })
	ginServer.HEAD("/healthz", func(c *gin.Context) { c.Status(200) })
	ginServer.Use(cors.New(cors.Config{AllowOrigins: cfg.Server.AllowedOrigins, AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}, AllowHeaders: []string{"Origin", "Content-Type", "Authorization", "X-CSRF-Token"}, AllowCredentials: true, MaxAge: 12 * time.Hour}))
	middleware.InitRateLimiter(cfg.Security.RateLimitRPS, cfg.Security.RateLimitRPS*2)
	users := &userclient.AuthUsers{Client: clients.User}
	login := transport.NewLoginController(users)
	transport.RegisterAuthInternal(ginServer, clients.Token, users)
	ginServer.POST("/login", middleware.NewIPRateLimiter(10, time.Minute), login.Login)
	ginServer.POST("/register", middleware.NewIPRateLimiter(5, time.Minute), login.Register)
	ginServer.POST("/refresh-token", middleware.NewIPRateLimiter(30, time.Minute), login.RefreshToken)
	ginServer.POST("/forgot-password", middleware.NewIPRateLimiter(3, 10*time.Minute), func(c *gin.Context) {
		var in contracts.ForgotPasswordDTO
		if c.ShouldBindJSON(&in) != nil {
			utils.Fail(c, 400, "参数错误")
			return
		}
		if err := users.ForgotPassword(in.Email); err != nil {
			utils.Fail(c, 503, "服务暂不可用")
			return
		}
		utils.Success(c, nil)
	})
	ginServer.POST("/reset-password", middleware.NewIPRateLimiter(5, 10*time.Minute), func(c *gin.Context) {
		var in contracts.ResetPasswordDTO
		if c.ShouldBindJSON(&in) != nil {
			utils.Fail(c, 400, "参数错误")
			return
		}
		if err := users.ResetPassword(in.Token, in.Password); err != nil {
			utils.Fail(c, 400, "重置失败")
			return
		}
		utils.Success(c, nil)
	})
	// Local JWT validation plus user-service session version validation avoids a self-RPC.
	ginServer.POST("/logout", middleware.JWTAuthMiddleware(users.ValidateSession), middleware.CSRFMiddleware(), login.Logout)
	return ginServer, nil
}
