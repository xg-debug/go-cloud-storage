package router

import (
	"context"
	"go-cloud-storage/internal/controller"
	"go-cloud-storage/internal/middleware"
	"go-cloud-storage/internal/pkg/cache"
	"go-cloud-storage/internal/pkg/config"
	"go-cloud-storage/internal/pkg/minio"
	"go-cloud-storage/internal/pkg/mq"
	"go-cloud-storage/internal/repositories"
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"go-cloud-storage/internal/services"

	"gorm.io/gorm"
)

func SetUpRouter(db *gorm.DB, minioService *minio.MinioService, rabbitClient *mq.RabbitMQClient, mqCfg *config.RabbitMQConfig) *gin.Engine {
	// 创建一个服务
	ginServer := gin.Default()

	// 配置 CORS 中间件
	ginServer.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:8080"}, // 你的前端地址
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 初始化仓库
	userRepo := repositories.NewUserRepository(db)
	fileRepo := repositories.NewFileRepository(db)
	recycleRepo := repositories.NewRecycleRepository(db)
	favoriteRepo := repositories.NewFavoriteRepository(db)
	shareRepo := repositories.NewShareRepository(db)
	storageQuotaRepo := repositories.NewStorageQuotaRepository(db)

	// 初始化服务
	userService := services.NewUserService(userRepo, fileRepo, storageQuotaRepo, minioService)
	fileService := services.NewFileService(db, cache.GetClient(), fileRepo, storageQuotaRepo, minioService)
	recyclePurgeService := services.NewRecyclePurgeService(db, minioService, recycleRepo, fileRepo, shareRepo, favoriteRepo)
	recycleService := services.NewRecycleService(db, recycleRepo, fileRepo, recyclePurgeService, rabbitClient)
	favoriteService := services.NewFavoriteService(favoriteRepo, fileService)
	categoryService := services.NewCategoryService(db, fileRepo)
	shareService := services.NewShareService(shareRepo, fileRepo)
	statsService := services.NewStatsService(fileRepo, storageQuotaRepo, shareRepo)
	storageQuotaService := services.NewStorageQuotaService(storageQuotaRepo)

	loginCtrl := controller.NewLoginController(userService)
	fileCtrl := controller.NewFileController(fileService)
	userCtrl := controller.NewUserController(userService)
	recycleCtrl := controller.NewRecycleController(recycleService)
	favoriteCtrl := controller.NewFavoriteController(favoriteService)
	categoryCtrl := controller.NewCategoryController(categoryService, fileService)
	shareCtrl := controller.NewShareController(shareService)
	statsCtrl := controller.NewStatsController(statsService, storageQuotaService)

	if rabbitClient != nil {
		startRecycleCleanupWorkers(recycleService, rabbitClient, mqCfg)
	}

	ginServer.POST("/login", loginCtrl.Login)
	ginServer.POST("/register", loginCtrl.Register)
	ginServer.POST("/refresh-token", loginCtrl.RefreshToken)

	authGroup := ginServer.Group("")
	authGroup.Use(middleware.JWTAuthMiddleware())
	authGroup.GET("/me", userCtrl.GetProfile)
	authGroup.POST("/logout", loginCtrl.Logout)

	user := ginServer.Group("user")
	user.Use(middleware.JWTAuthMiddleware())
	{
		user.PUT("/update", userCtrl.UpdateProfile)
		user.PUT("/password", userCtrl.UpdatePassword)
		user.POST("/avatar", userCtrl.UpdateAvatar)
		user.GET("/stats", statsCtrl.GetUserDashboardStats)
		user.GET("/quota", statsCtrl.GetUserStorage)
	}

	file := ginServer.Group("file")
	file.Use(middleware.JWTAuthMiddleware()) // 为路由组注册中间件
	{
		file.POST("/list", fileCtrl.GetFiles)
		file.POST("/create-folder", fileCtrl.CreateFolder)
		// 普通文件上传
		file.POST("/upload", fileCtrl.UploadFile)

		// 大文件上传
		file.POST("/chunk/init", fileCtrl.ChunkUploadInit)
		file.POST("/chunk/upload", fileCtrl.ChunkUploadPart)
		file.POST("/chunk/merge", fileCtrl.ChunkUploadMerge)
		file.POST("/chunk/cancel", fileCtrl.ChunkUploadCancel)

		file.DELETE("/:fileId", fileCtrl.Delete)
		file.POST("/rename", fileCtrl.Rename)
		file.GET("/folders/tree", fileCtrl.GetFolderTree)
		file.POST("/move", fileCtrl.MoveFile)
		file.GET("/preview/:fileId", fileCtrl.PreviewFile)
		file.GET("/recent", fileCtrl.GetRecentFiles)
		file.POST("/search", fileCtrl.SearchFiles)
		file.GET("/download/:fileId", fileCtrl.Download)
	}

	favorite := ginServer.Group("favorite")
	favorite.Use(middleware.JWTAuthMiddleware())
	{
		favorite.GET("", favoriteCtrl.GetFavoriteList)
		favorite.POST("/:fileId", favoriteCtrl.Favorite)
		favorite.DELETE("/:fileId", favoriteCtrl.UnFavorite)
	}

	recycle := ginServer.Group("recycle")
	recycle.Use(middleware.JWTAuthMiddleware()) // 为路由组注册中间件
	{
		recycle.GET("", recycleCtrl.ListRecycleFiles)

		recycle.DELETE("/:fileId", recycleCtrl.DeletePermanent)
		recycle.DELETE("/batch", recycleCtrl.DeleteSelected)
		recycle.DELETE("", recycleCtrl.ClearRecycleBin)

		recycle.PUT("/:fileId/restore", recycleCtrl.RestoreFile)
		recycle.PUT("/batch", recycleCtrl.RestoreSelected)
	}

	// 分类路由
	category := ginServer.Group("category")
	category.Use(middleware.JWTAuthMiddleware())
	{
		category.POST("/files", categoryCtrl.GetFilesByCategory)
	}

	// 分享路由
	share := ginServer.Group("share")
	share.Use(middleware.JWTAuthMiddleware())
	{
		share.POST("", shareCtrl.CreateShare)                // 创建分享
		share.GET("", shareCtrl.GetUserShares)               // 获取用户分享列表
		share.GET("/:shareId", shareCtrl.GetShareDetail)     // 获取分享详情
		share.PUT("/:shareId", shareCtrl.UpdateShare)        // 更新分享设置
		share.PUT("/:shareId/cancel", shareCtrl.CancelShare) // 取消分享
	}

	// 公开分享访问路由（无需认证）
	ginServer.GET("/s/:token", shareCtrl.AccessShare)                 // 访问分享
	ginServer.GET("/s/:token/download", shareCtrl.DownloadSharedFile) // 下载分享文件

	return ginServer
}

func startRecycleCleanupWorkers(recycleService services.RecycleService, rabbitClient *mq.RabbitMQClient, mqCfg *config.RabbitMQConfig) {
	interval := 60 * time.Second
	if mqCfg != nil && mqCfg.ScanIntervalSeconds > 0 {
		interval = time.Duration(mqCfg.ScanIntervalSeconds) * time.Second
	}
	ctx := context.Background()

	go func() {
		if err := rabbitClient.ConsumeExpiredFilePurge(ctx, func(ctx context.Context, fileID string) error {
			return recycleService.DeleteSelected(ctx, []string{fileID})
		}); err != nil {
			log.Printf("recycle cleanup consumer exited: %v", err)
		}
	}()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		_, err := recycleService.DispatchExpiredPurgeJobs(ctx, 200)
		if err != nil {
			log.Printf("dispatch recycle cleanup job failed: %v", err)
		}

		for range ticker.C {
			n, dispatchErr := recycleService.DispatchExpiredPurgeJobs(ctx, 200)
			if dispatchErr != nil {
				log.Printf("dispatch recycle cleanup job failed: %v", dispatchErr)
				continue
			}
			if n > 0 {
				log.Printf("dispatched %d recycle cleanup jobs", n)
			}
		}
	}()
}
