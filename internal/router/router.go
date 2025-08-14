package router

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go-cloud-storage/internal/controller"
	"go-cloud-storage/internal/middleware"
	"go-cloud-storage/internal/repositories"
	"time"

	"go-cloud-storage/internal/pkg/oss"
	"go-cloud-storage/internal/services"
	"gorm.io/gorm"
)

func SetUpRouter(db *gorm.DB, ossService *oss.OSSService) *gin.Engine {
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
	shareRepo := repositories.NewShareRepository(db)
	fileRepo := repositories.NewFileRepository(db)
	recycleRepo := repositories.NewRecycleRepository(db)

	// 初始化服务
	userService := services.NewUserService(userRepo, fileRepo, ossService)
	shareService := services.NewShareService(shareRepo)
	fileService := services.NewFileService(db, fileRepo)
	recycleService := services.NewRecycleService(db, recycleRepo, fileRepo)

	loginCtrl := controller.NewLoginController(userService)
	shareCtrl := controller.NewShareController(shareService)
	fileCtrl := controller.NewFileController(fileService)
	userCtrl := controller.NewUserController(userService)
	uploadCtrl := controller.NewUploadController(ossService, fileService)
	recycleCtrl := controller.NewRecycleController(recycleService)

	ginServer.POST("/login", loginCtrl.Login)
	ginServer.POST("/register", loginCtrl.Register)
	ginServer.POST("/refresh-token", loginCtrl.RefreshToken)

	authGroup := ginServer.Group("/")
	authGroup.Use(middleware.JWTAuthMiddleware())
	authGroup.GET("/me", userCtrl.GetProfile)
	authGroup.POST("/logout", loginCtrl.Logout)

	user := ginServer.Group("user")
	user.Use(middleware.JWTAuthMiddleware())
	{
		user.PUT("/update", userCtrl.UpdateProfile)
		user.PUT("/password", userCtrl.UpdatePassword)
		user.POST("/avatar", userCtrl.UpdateAvatar)
	}

	file := ginServer.Group("file")
	file.Use(middleware.JWTAuthMiddleware()) // 为路由组注册中间件
	{
		file.POST("/list", fileCtrl.GetFiles)
		file.POST("/create-folder", fileCtrl.CreateFolder)
		file.POST("/upload", uploadCtrl.Upload)
		file.DELETE("/:fileId", fileCtrl.Delete)
		file.POST("/rename", fileCtrl.Rename)
		file.POST("/move")
		file.GET("/info")
		file.GET("/preview")
	}

	shareGroup := ginServer.Group("/share")
	shareGroup.Use(middleware.JWTAuthMiddleware())
	{
		shareGroup.POST("/create", shareCtrl.CreateShare)
		shareGroup.GET("/list", shareCtrl.ListUserShares)
		shareGroup.DELETE("/:id", shareCtrl.ListUserShares)
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

	return ginServer
}
