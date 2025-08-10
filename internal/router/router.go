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

	// 初始化服务
	userService := services.NewUserService(userRepo, fileRepo)
	shareService := services.NewShareService(shareRepo)
	fileService := services.NewFileService(fileRepo)

	// 创建需要OSS服务的控制器
	//shareCtrl := controller.NewShareController(ossService)
	//uploadCtrl := controller.NewUploadController(ossService)

	loginCtrl := controller.NewLoginController(userService)
	shareCtrl := controller.NewShareController(shareService)
	fileCtrl := controller.NewFileController(fileService)
	userCtrl := controller.NewUserController(userService)
	uploadCtrl := controller.NewUploadController(ossService, fileService)

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
	folder := ginServer.Group("folder")
	folder.Use(middleware.JWTAuthMiddleware())
	{
		folder.POST("/create", fileCtrl.CreateFolder)
	}

	shareGroup := ginServer.Group("/share")
	shareGroup.Use(middleware.JWTAuthMiddleware())
	{
		shareGroup.POST("/create", shareCtrl.CreateShare)
		shareGroup.GET("/list", shareCtrl.ListUserShares)
		shareGroup.DELETE("/:id", shareCtrl.ListUserShares)
	}

	return ginServer
}
