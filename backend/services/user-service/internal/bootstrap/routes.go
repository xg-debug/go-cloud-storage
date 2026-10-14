package bootstrap

import (
	"context"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	authrpc "go-cloud-storage/backend/pkg/auth"
	"go-cloud-storage/backend/pkg/config"
	contracts "go-cloud-storage/backend/pkg/contracts"
	middleware "go-cloud-storage/backend/pkg/httpmiddleware"
	ports "go-cloud-storage/backend/pkg/ports"
	rpc "go-cloud-storage/backend/pkg/rpc"
	email "go-cloud-storage/backend/services/user-service/internal/adapters/email"
	application "go-cloud-storage/backend/services/user-service/internal/application"
	transport "go-cloud-storage/backend/services/user-service/internal/transport"
)

type ServiceClients struct {
	Auth  *rpc.Client
	Token string
}

func setUp(ctx context.Context, db *gorm.DB, storage ports.Storage, cfg *config.Config, clients ServiceClients) (*gin.Engine, error) {
	ginServer := gin.New()
	ginServer.Use(gin.Recovery(), middleware.RequestIDMiddleware())
	if err := ginServer.SetTrustedProxies(cfg.Server.TrustedProxies); err != nil {
		return nil, err
	}
	ginServer.GET("/healthz", func(c *gin.Context) { c.Status(200) })
	ginServer.HEAD("/healthz", func(c *gin.Context) { c.Status(200) })
	ginServer.Use(cors.New(cors.Config{AllowOrigins: cfg.Server.AllowedOrigins, AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}, AllowHeaders: []string{"Origin", "Content-Type", "Authorization", "X-CSRF-Token"}, AllowCredentials: true, MaxAge: 12 * time.Hour}))
	middleware.InitRateLimiter(cfg.Security.RateLimitRPS, cfg.Security.RateLimitRPS*2)
	var ensure func(context.Context, contracts.Identity) error
	auth := authrpc.Authenticate(clients.Auth, ensure)
	protected := func(prefix string) *gin.RouterGroup {
		group := ginServer.Group(prefix)
		group.Use(auth, middleware.CSRFMiddleware(), middleware.RateLimitMiddleware())
		return group
	}
	mail := email.NewEmailService(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.From)
	users := application.NewIndependentUserService(db, storage, mail, cfg.Server.PublicBaseURL)
	transport.RegisterUserInternal(ginServer, clients.Token, users)
	ctrl := transport.NewUserController(users)
	protected("").GET("/me", ctrl.GetProfile)
	group := protected("user")
	group.PUT("/update", ctrl.UpdateProfile)
	group.PUT("/password", ctrl.UpdatePassword)
	group.POST("/avatar", ctrl.UpdateAvatar)
	return ginServer, nil
}
