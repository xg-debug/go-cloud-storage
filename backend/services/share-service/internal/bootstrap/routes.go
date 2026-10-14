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
	eventing "go-cloud-storage/backend/pkg/platform/eventing"
	ports "go-cloud-storage/backend/pkg/ports"
	rpc "go-cloud-storage/backend/pkg/rpc"
	fileclient "go-cloud-storage/backend/services/share-service/internal/adapters/fileclient"
	messaging "go-cloud-storage/backend/services/share-service/internal/adapters/messaging"
	repository "go-cloud-storage/backend/services/share-service/internal/adapters/repository"
	application "go-cloud-storage/backend/services/share-service/internal/application"
	transport "go-cloud-storage/backend/services/share-service/internal/transport"
)

type ServiceClients struct {
	Auth, File *rpc.Client
	Token      string
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
	repo := repository.NewEventShares(db)
	shareService := application.NewShareService(repo, &fileclient.Files{Client: clients.File}, storage, nil)
	shareCtrl := transport.NewShareController(shareService, middleware.NewShareBruteProtector(5, 15*time.Minute))
	transport.RegisterShareInternal(ginServer, clients.Token, repo)
	go eventing.Relay(ctx, db, cfg.RabbitMQ.URL)
	go eventing.Consume(ctx, cfg.RabbitMQ.URL, "cloudbox.share", messaging.ShareEventHandler(db))
	share := ginServer.Group("share")
	share.Use(auth)
	share.Use(middleware.CSRFMiddleware())
	share.Use(middleware.RateLimitMiddleware())
	{
		share.POST("", shareCtrl.CreateShare)
		share.GET("", shareCtrl.GetUserShares)
		share.GET("/:shareId", shareCtrl.GetShareDetail)
		share.PUT("/:shareId", shareCtrl.UpdateShare)
		share.PUT("/:shareId/cancel", shareCtrl.CancelShare)
	}

	ginServer.GET("/s/:token", middleware.NewIPRateLimiter(60, time.Minute), shareCtrl.AccessShare)
	ginServer.GET("/s/:token/download", middleware.NewIPRateLimiter(60, time.Minute), shareCtrl.DownloadSharedFile)
	return ginServer, nil
}
