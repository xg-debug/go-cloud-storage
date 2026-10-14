package router

import (
	"context"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go-cloud-storage/backend/infrastructure/cache"
	"go-cloud-storage/backend/infrastructure/email"
	"go-cloud-storage/backend/infrastructure/eventing"
	persistenceinfra "go-cloud-storage/backend/infrastructure/persistence"
	"go-cloud-storage/backend/internal/controller"
	"go-cloud-storage/backend/internal/distributed"
	"go-cloud-storage/backend/internal/middleware"
	"go-cloud-storage/backend/internal/models/dto"
	"go-cloud-storage/backend/internal/ports"
	"go-cloud-storage/backend/internal/repositories"
	"go-cloud-storage/backend/internal/services"
	"go-cloud-storage/backend/pkg/config"
	"go-cloud-storage/backend/pkg/utils"
	"gorm.io/gorm"
	"time"
)

type ServiceClients struct {
	Auth, User, File, Share *distributed.Client
	Token                   string
}

// SetUpService mounts only routes and dependencies owned by this process.
func SetUpService(ctx context.Context, role string, db *gorm.DB, storage ports.Storage, cfg *config.Config, clients ServiceClients) (*gin.Engine, error) {
	ginServer := gin.New()
	ginServer.Use(gin.Recovery(), middleware.RequestIDMiddleware())
	if err := ginServer.SetTrustedProxies(cfg.Server.TrustedProxies); err != nil {
		return nil, err
	}
	ginServer.GET("/healthz", func(c *gin.Context) { c.Status(200) })
	ginServer.HEAD("/healthz", func(c *gin.Context) { c.Status(200) })
	ginServer.Use(cors.New(cors.Config{AllowOrigins: cfg.Server.AllowedOrigins, AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}, AllowHeaders: []string{"Origin", "Content-Type", "Authorization", "X-CSRF-Token"}, AllowCredentials: true, MaxAge: 12 * time.Hour}))
	middleware.InitRateLimiter(cfg.Security.RateLimitRPS, cfg.Security.RateLimitRPS*2)
	var ensure func(context.Context, distributed.Identity) error
	if role == "file" {
		ensure = distributed.EnsureOwner(db, cfg.Security.DefaultQuotaGB)
	}
	auth := distributed.Authenticate(clients.Auth, ensure)
	protected := func(prefix string) *gin.RouterGroup {
		group := ginServer.Group(prefix)
		group.Use(auth, middleware.CSRFMiddleware(), middleware.RateLimitMiddleware())
		return group
	}
	switch role {
	case "auth":
		users := &distributed.AuthUsers{Client: clients.User}
		login := controller.NewLoginController(users)
		distributed.RegisterAuthInternal(ginServer, clients.Token, users)
		ginServer.POST("/login", middleware.NewIPRateLimiter(10, time.Minute), login.Login)
		ginServer.POST("/register", middleware.NewIPRateLimiter(5, time.Minute), login.Register)
		ginServer.POST("/refresh-token", middleware.NewIPRateLimiter(30, time.Minute), login.RefreshToken)
		ginServer.POST("/forgot-password", middleware.NewIPRateLimiter(3, 10*time.Minute), func(c *gin.Context) {
			var in dto.ForgotPasswordDTO
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
			var in dto.ResetPasswordDTO
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
	case "user":
		mail := email.NewEmailService(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.From)
		users := services.NewIndependentUserService(db, storage, mail, cfg.Server.PublicBaseURL)
		distributed.RegisterUserInternal(ginServer, clients.Token, users)
		ctrl := controller.NewUserController(users)
		protected("").GET("/me", ctrl.GetProfile)
		group := protected("user")
		group.PUT("/update", ctrl.UpdateProfile)
		group.PUT("/password", ctrl.UpdatePassword)
		group.POST("/avatar", ctrl.UpdateAvatar)
	case "file":
		fileRepo := repositories.NewFileRepository(db)
		quotaRepo := repositories.NewStorageQuotaRepository(db)
		recycleRepo := repositories.NewRecycleRepository(db)
		favoriteRepo := repositories.NewFavoriteRepository(db)
		shareRepo := &distributed.Shares{Client: clients.Share}
		fileService := services.NewFileService(db, cache.GetClient(), fileRepo, quotaRepo, shareRepo, storage)
		files := persistenceinfra.NewGormUploadFileRepository(db)
		upload := services.NewUploadApplication(files, persistenceinfra.NewGormUploadQuotaRepository(db), persistenceinfra.NewGormTransactionManager(db), cache.NewRedisUploadSessionStore(cache.GetClient(), 24*time.Hour, 48*time.Hour), storage, &eventing.Publisher{DB: db})
		upload.StartChunkUploadCleanup(ctx)
		purge := services.NewRecyclePurgeService(db, storage, recycleRepo, fileRepo, shareRepo, favoriteRepo, quotaRepo)
		recycleService := services.NewRecycleService(db, recycleRepo, fileRepo, purge, nil)
		startRecycleCleanupWorkers(ctx, recycleService, nil, &cfg.RabbitMQ)
		go services.RunObjectCleanup(ctx, db, storage)
		fileCtrl := controller.NewFileController(fileService, cfg, nil)
		uploadCtrl := controller.NewUploadController(upload, cfg)
		downloadCtrl := controller.NewDownloadController(services.NewDownloadApplication(files, storage))
		favoriteCtrl := controller.NewFavoriteController(services.NewFavoriteService(favoriteRepo, fileRepo, fileService))
		recycleCtrl := controller.NewRecycleController(recycleService)
		categoryCtrl := controller.NewCategoryController(services.NewCategoryService(db, fileRepo, fileService), fileService)
		stats := controller.NewStatsController(services.NewStatsService(fileRepo, quotaRepo, shareRepo), services.NewStorageQuotaService(quotaRepo))
		group := protected("user")
		group.GET("/stats", stats.GetUserDashboardStats)
		group.GET("/quota", stats.GetUserStorage)
		distributed.RegisterFileInternal(ginServer, clients.Token, db)
		go eventing.Relay(ctx, db, cfg.RabbitMQ.URL)
		file := ginServer.Group("file")
		file.Use(auth)
		file.Use(middleware.CSRFMiddleware())
		file.Use(middleware.RateLimitMiddleware())
		{
			file.POST("/list", fileCtrl.GetFiles)
			file.POST("/create-folder", fileCtrl.CreateFolder)

			file.POST("/upload", uploadCtrl.UploadFile)
			file.POST("/chunk/init", uploadCtrl.ChunkUploadInit)
			file.POST("/chunk/upload", uploadCtrl.ChunkUploadPart)
			file.POST("/chunk/merge", uploadCtrl.ChunkUploadMerge)
			file.POST("/chunk/cancel", uploadCtrl.ChunkUploadCancel)
			file.GET("/chunk/progress", uploadCtrl.GetChunkUploadProgress)

			file.DELETE("/:fileId", fileCtrl.Delete)
			file.POST("/rename", fileCtrl.Rename)
			file.GET("/folders/tree", fileCtrl.GetFolderTree)
			file.POST("/move", fileCtrl.MoveFile)
			file.POST("/copy", fileCtrl.CopyFile)
			file.GET("/preview/:fileId", fileCtrl.PreviewFile)
			file.GET("/preview-stream/:fileId", downloadCtrl.PreviewStream)
			file.GET("/recent", fileCtrl.GetRecentFiles)
			file.POST("/search", fileCtrl.SearchFiles)
			file.GET("/search/history", fileCtrl.GetSearchHistory)
			file.GET("/duplicates", fileCtrl.GetDuplicateFiles)
			file.DELETE("/search/history", fileCtrl.DeleteSearchHistory)
			file.GET("/download/:fileId", downloadCtrl.Download)
			file.GET("/download-info/:fileId", downloadCtrl.GetDownloadInfo)
			file.POST("/download-batch", fileCtrl.DownloadBatch)
		}

		favorite := ginServer.Group("favorite")
		favorite.Use(auth)
		favorite.Use(middleware.CSRFMiddleware())
		favorite.Use(middleware.RateLimitMiddleware())
		{
			favorite.GET("", favoriteCtrl.GetFavoriteList)
			favorite.POST("/:fileId", favoriteCtrl.Favorite)
			favorite.DELETE("/:fileId", favoriteCtrl.UnFavorite)
		}

		recycle := ginServer.Group("recycle")
		recycle.Use(auth)
		recycle.Use(middleware.CSRFMiddleware())
		recycle.Use(middleware.RateLimitMiddleware())
		{
			recycle.GET("", recycleCtrl.ListRecycleFiles)
			recycle.DELETE("/:fileId", recycleCtrl.DeletePermanent)
			recycle.DELETE("/batch", recycleCtrl.DeleteSelected)
			recycle.DELETE("", recycleCtrl.ClearRecycleBin)
			recycle.PUT("/:fileId/restore", recycleCtrl.RestoreFile)
			recycle.PUT("/batch", recycleCtrl.RestoreSelected)
		}

		category := ginServer.Group("category")
		category.Use(auth)
		category.Use(middleware.CSRFMiddleware())
		category.Use(middleware.RateLimitMiddleware())
		{
			category.POST("/files", categoryCtrl.GetFilesByCategory)
		}

	case "share":
		repo := distributed.NewEventShares(db)
		shareService := services.NewShareService(repo, &distributed.Files{Client: clients.File}, storage, nil)
		shareCtrl := controller.NewShareController(shareService, middleware.NewShareBruteProtector(5, 15*time.Minute))
		distributed.RegisterShareInternal(ginServer, clients.Token, repo)
		go eventing.Relay(ctx, db, cfg.RabbitMQ.URL)
		go eventing.Consume(ctx, cfg.RabbitMQ.URL, "cloudbox.share", distributed.ShareEventHandler(db))
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
	case "notification":
		broker := services.NewSSEBroker()
		notificationCtrl := controller.NewNotificationController(services.NewNotificationService(repositories.NewNotificationRepository(db), broker), broker)
		go distributed.NotificationFanout(ctx, cache.GetClient(), broker)
		go eventing.Consume(ctx, cfg.RabbitMQ.URL, "cloudbox.notification", distributed.NotificationEventHandler(db, cache.GetClient()))
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

	default:
		return nil, fmt.Errorf("unknown service role %q", role)
	}
	return ginServer, nil
}
