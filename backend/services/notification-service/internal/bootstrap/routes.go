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
	cache "go-cloud-storage/backend/pkg/platform/cache"
	eventing "go-cloud-storage/backend/pkg/platform/eventing"
	rpc "go-cloud-storage/backend/pkg/rpc"
	messaging "go-cloud-storage/backend/services/notification-service/internal/adapters/messaging"
	repository "go-cloud-storage/backend/services/notification-service/internal/adapters/repository"
	application "go-cloud-storage/backend/services/notification-service/internal/application"
	transport "go-cloud-storage/backend/services/notification-service/internal/transport"
)

type ServiceClients struct {
	Auth  *rpc.Client
	Token string
}

func setUp(ctx context.Context, db *gorm.DB, cfg *config.Config, clients ServiceClients) (*gin.Engine, error) {
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
	broker := application.NewSSEBroker()
	notificationCtrl := transport.NewNotificationController(application.NewNotificationService(repository.NewNotificationRepository(db), broker), broker)
	go messaging.NotificationFanout(ctx, cache.GetClient(), broker)
	go eventing.Consume(ctx, cfg.RabbitMQ.URL, "cloudbox.notification", messaging.NotificationEventHandler(db, cache.GetClient()))
	notification := ginServer.Group("notification")
	notification.Use(auth)
	notification.Use(middleware.CSRFMiddleware())
	notification.Use(middleware.RateLimitMiddleware())
	{
		notification.GET("/stream", notificationCtrl.NotificationSSE)
		notification.GET("", notificationCtrl.GetNotifications)
		notification.GET("/unread-count", notificationCtrl.GetUnreadCount)
		notification.PUT("/:id/read", notificationCtrl.MarkAsRead)
		notification.PUT("/read-all", notificationCtrl.MarkAllAsRead)
		notification.DELETE("/:id", notificationCtrl.DeleteNotification)
		notification.DELETE("/all", notificationCtrl.DeleteAllNotifications)
	}

	return ginServer, nil
}
