package router

import (
	"context"
	"go-cloud-storage/backend/infrastructure/cache"
	"go-cloud-storage/backend/infrastructure/email"
	"go-cloud-storage/backend/infrastructure/messaging"
	"go-cloud-storage/backend/infrastructure/minio"
	"go-cloud-storage/backend/infrastructure/mq"
	persistenceinfra "go-cloud-storage/backend/infrastructure/persistence"
	storageinfra "go-cloud-storage/backend/infrastructure/storage"
	"go-cloud-storage/backend/internal/controller"
	"go-cloud-storage/backend/internal/middleware"
	"go-cloud-storage/backend/internal/ports"
	"go-cloud-storage/backend/internal/repositories"
	"go-cloud-storage/backend/pkg/config"
	"log/slog"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"go-cloud-storage/backend/internal/services"

	"gorm.io/gorm"
)

func SetUpRouter(appCtx context.Context, db *gorm.DB, minioService *minio.MinioService, rabbitClient *mq.RabbitMQClient, cfg *config.Config) *gin.Engine {
	mqCfg := &cfg.RabbitMQ
	ginServer := gin.New()
	ginServer.Use(gin.Recovery())

	_ = ginServer.SetTrustedProxies(nil)
	ginServer.Use(middleware.RequestIDMiddleware())
	ginServer.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		slog.Info("request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency", time.Since(start).String(),
			"ip", c.ClientIP(),
		)
	})

	middleware.InitRateLimiter(cfg.Security.RateLimitRPS, cfg.Security.RateLimitRPS*2)
	ginServer.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.Server.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-CSRF-Token"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	userRepo := repositories.NewUserRepository(db)
	fileRepo := repositories.NewFileRepository(db)
	recycleRepo := repositories.NewRecycleRepository(db)
	favoriteRepo := repositories.NewFavoriteRepository(db)
	shareRepo := repositories.NewShareRepository(db)
	storageQuotaRepo := repositories.NewStorageQuotaRepository(db)
	notificationRepo := repositories.NewNotificationRepository(db)

	emailService := email.NewEmailService(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.From)

	sseBroker := services.NewSSEBroker()
	notificationService := services.NewNotificationService(notificationRepo, sseBroker)
	userService := services.NewUserService(db, userRepo, fileRepo, storageQuotaRepo, minioService, emailService, cfg.Server.PublicBaseURL)

	// Legacy FileService remains for file use cases that have not been migrated yet.
	// Upload and core download streaming now use application ports/adapters.
	redisClient := cache.GetClient()
	fileService := services.NewFileService(db, redisClient, fileRepo, storageQuotaRepo, shareRepo, minioService)

	storagePort := storageinfra.NewMinIOStorageV2(minioService)
	uploadFiles := persistenceinfra.NewGormUploadFileRepository(db)
	uploadQuotas := persistenceinfra.NewGormUploadQuotaRepository(db)
	txManager := persistenceinfra.NewGormTransactionManager(db)
	uploadSessions := cache.NewRedisUploadSessionStore(redisClient, 24*time.Hour, 48*time.Hour)

	eventBus := messaging.NewLocalEventBus()
	eventBus.Subscribe("file.uploaded.v1", func(ctx context.Context, event ports.Event) error {
		data, ok := event.Data.(services.FileUploadedEvent)
		if !ok {
			return nil
		}
		return notificationService.CreateUploadCompleteNotification(uint(event.UserID), data.FileName)
	})

	uploadApplication := services.NewUploadApplication(
		uploadFiles,
		uploadQuotas,
		txManager,
		uploadSessions,
		storagePort,
		eventBus,
	)
	uploadApplication.StartChunkUploadCleanup(appCtx)

	// The GORM upload adapter also satisfies FileReadRepository, so download and
	// preview streaming share the same metadata boundary without depending on the
	// legacy FileService or MinIO concrete type.
	downloadApplication := services.NewDownloadApplication(uploadFiles, storagePort)

	recyclePurgeService := services.NewRecyclePurgeService(db, minioService, recycleRepo, fileRepo, shareRepo, favoriteRepo, storageQuotaRepo)
	var recyclePublisher services.RecycleJobPublisher
	if rabbitClient != nil {
		recyclePublisher = rabbitClient
	}
	recycleService := services.NewRecycleService(db, recycleRepo, fileRepo, recyclePurgeService, recyclePublisher)
	favoriteService := services.NewFavoriteService(favoriteRepo, fileRepo, fileService)
	categoryService := services.NewCategoryService(db, fileRepo, fileService)
	shareService := services.NewShareService(shareRepo, fileRepo, minioService, notificationService)
	statsService := services.NewStatsService(fileRepo, storageQuotaRepo, shareRepo)
	storageQuotaService := services.NewStorageQuotaService(storageQuotaRepo)

	loginCtrl := controller.NewLoginController(userService)
	fileCtrl := controller.NewFileController(fileService, cfg, notificationService)
	uploadCtrl := controller.NewUploadController(uploadApplication, cfg)
	downloadCtrl := controller.NewDownloadController(downloadApplication)
	userCtrl := controller.NewUserController(userService)
	recycleCtrl := controller.NewRecycleController(recycleService)
	favoriteCtrl := controller.NewFavoriteController(favoriteService)
	categoryCtrl := controller.NewCategoryController(categoryService, fileService)
	shareBruteProtector := middleware.NewShareBruteProtector(5, 15*time.Minute)
	shareCtrl := controller.NewShareController(shareService, shareBruteProtector)
	statsCtrl := controller.NewStatsController(statsService, storageQuotaService)
	notificationCtrl := controller.NewNotificationController(notificationService, sseBroker)

	startRecycleCleanupWorkers(appCtx, recycleService, rabbitClient, mqCfg)

	ginServer.POST("/login", middleware.NewIPRateLimiter(10, time.Minute), loginCtrl.Login)
	ginServer.POST("/register", middleware.NewIPRateLimiter(5, time.Minute), loginCtrl.Register)
	ginServer.POST("/refresh-token", middleware.NewIPRateLimiter(30, time.Minute), loginCtrl.RefreshToken)
	ginServer.POST("/forgot-password", middleware.NewIPRateLimiter(3, 10*time.Minute), userCtrl.ForgotPassword)
	ginServer.POST("/reset-password", middleware.NewIPRateLimiter(5, 10*time.Minute), userCtrl.ResetPassword)

	authGroup := ginServer.Group("")
	authGroup.Use(middleware.JWTAuthMiddleware(userService.ValidateSession))
	authGroup.Use(middleware.CSRFMiddleware())
	authGroup.Use(middleware.RateLimitMiddleware())
	authGroup.GET("/me", userCtrl.GetProfile)
	authGroup.POST("/logout", loginCtrl.Logout)

	user := ginServer.Group("user")
	user.Use(middleware.JWTAuthMiddleware(userService.ValidateSession))
	user.Use(middleware.CSRFMiddleware())
	user.Use(middleware.RateLimitMiddleware())
	{
		user.PUT("/update", userCtrl.UpdateProfile)
		user.PUT("/password", userCtrl.UpdatePassword)
		user.POST("/avatar", userCtrl.UpdateAvatar)
		user.GET("/stats", statsCtrl.GetUserDashboardStats)
		user.GET("/quota", statsCtrl.GetUserStorage)
	}

	file := ginServer.Group("file")
	file.Use(middleware.JWTAuthMiddleware(userService.ValidateSession))
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
	favorite.Use(middleware.JWTAuthMiddleware(userService.ValidateSession))
	favorite.Use(middleware.CSRFMiddleware())
	favorite.Use(middleware.RateLimitMiddleware())
	{
		favorite.GET("", favoriteCtrl.GetFavoriteList)
		favorite.POST("/:fileId", favoriteCtrl.Favorite)
		favorite.DELETE("/:fileId", favoriteCtrl.UnFavorite)
	}

	recycle := ginServer.Group("recycle")
	recycle.Use(middleware.JWTAuthMiddleware(userService.ValidateSession))
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
	category.Use(middleware.JWTAuthMiddleware(userService.ValidateSession))
	category.Use(middleware.CSRFMiddleware())
	category.Use(middleware.RateLimitMiddleware())
	{
		category.POST("/files", categoryCtrl.GetFilesByCategory)
	}

	share := ginServer.Group("share")
	share.Use(middleware.JWTAuthMiddleware(userService.ValidateSession))
	share.Use(middleware.CSRFMiddleware())
	share.Use(middleware.RateLimitMiddleware())
	{
		share.POST("", shareCtrl.CreateShare)
		share.GET("", shareCtrl.GetUserShares)
		share.GET("/:shareId", shareCtrl.GetShareDetail)
		share.PUT("/:shareId", shareCtrl.UpdateShare)
		share.PUT("/:shareId/cancel", shareCtrl.CancelShare)
	}

	notification := ginServer.Group("notification")
	notification.Use(middleware.JWTAuthMiddleware(userService.ValidateSession))
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

	ginServer.GET("/s/:token", middleware.NewIPRateLimiter(60, time.Minute), shareCtrl.AccessShare)
	ginServer.GET("/s/:token/download", middleware.NewIPRateLimiter(60, time.Minute), shareCtrl.DownloadSharedFile)

	return ginServer
}

func startRecycleCleanupWorkers(ctx context.Context, recycleService services.RecycleService, rabbitClient *mq.RabbitMQClient, mqCfg *config.RabbitMQConfig) {
	interval := 60 * time.Second
	if mqCfg != nil && mqCfg.ScanIntervalSeconds > 0 {
		interval = time.Duration(mqCfg.ScanIntervalSeconds) * time.Second
	}

	go func() {
		if rabbitClient == nil {
			return
		}
		if err := rabbitClient.ConsumeExpiredFilePurge(ctx, func(ctx context.Context, fileID string) error {
			return recycleService.PurgeExpired(ctx, []string{fileID})
		}); err != nil && ctx.Err() == nil {
			slog.Error("recycle cleanup consumer exited", "error", err)
		}
	}()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		_, err := recycleService.DispatchExpiredPurgeJobs(ctx, 200)
		if err != nil && ctx.Err() == nil {
			slog.Error("dispatch recycle cleanup job failed", "error", err)
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				n, dispatchErr := recycleService.DispatchExpiredPurgeJobs(ctx, 200)
				if dispatchErr != nil {
					if ctx.Err() == nil {
						slog.Error("dispatch recycle cleanup job failed", "error", dispatchErr)
					}
					continue
				}
				if n > 0 {
					slog.Info("dispatched recycle cleanup jobs", "count", n)
				}
			}
		}
	}()
}
