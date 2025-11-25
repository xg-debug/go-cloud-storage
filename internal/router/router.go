package router

import (
	"go-cloud-storage/internal/controller"
	"go-cloud-storage/internal/middleware"
	"go-cloud-storage/internal/pkg/cache"
	"go-cloud-storage/internal/pkg/minio"
	"go-cloud-storage/internal/repositories"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"go-cloud-storage/internal/services"

	"gorm.io/gorm"
)

func SetUpRouter(db *gorm.DB, minioService *minio.MinioService) *gin.Engine {
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
	recycleService := services.NewRecycleService(db, minioService, recycleRepo, fileRepo)
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
		file.POST("/move")
		file.GET("/preview/:fileId", fileCtrl.PreviewFile)
		file.GET("/recent", fileCtrl.GetRecentFiles)
		file.POST("/search", fileCtrl.SearchFiles)
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
		share.PUT("/:shareId/cancel", shareCtrl.CancelShare) // 取消分享
		share.DELETE("/:shareId", shareCtrl.DeleteShare)     // 删除分享记录
	}

	// 公开分享访问路由（无需认证）
	ginServer.GET("/s/:token", shareCtrl.AccessShare)                 // 访问分享
	ginServer.GET("/s/:token/download", shareCtrl.DownloadSharedFile) // 下载分享文件

	return ginServer
}
