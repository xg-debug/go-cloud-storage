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
	transaction "go-cloud-storage/backend/pkg/platform/persistence"
	ports "go-cloud-storage/backend/pkg/ports"
	rpc "go-cloud-storage/backend/pkg/rpc"
	sessioncache "go-cloud-storage/backend/services/file-service/internal/adapters/cache"
	persistenceinfra "go-cloud-storage/backend/services/file-service/internal/adapters/persistence"
	repository "go-cloud-storage/backend/services/file-service/internal/adapters/repository"
	shareclient "go-cloud-storage/backend/services/file-service/internal/adapters/shareclient"
	application "go-cloud-storage/backend/services/file-service/internal/application"
	transport "go-cloud-storage/backend/services/file-service/internal/transport"
)

type ServiceClients struct {
	Auth, Share *rpc.Client
	Token       string
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

	ensure = repository.EnsureOwner(db, cfg.Security.DefaultQuotaGB)

	auth := authrpc.Authenticate(clients.Auth, ensure)
	protected := func(prefix string) *gin.RouterGroup {
		group := ginServer.Group(prefix)
		group.Use(auth, middleware.CSRFMiddleware(), middleware.RateLimitMiddleware())
		return group
	}
	fileRepo := repository.NewFileRepository(db)
	quotaRepo := repository.NewStorageQuotaRepository(db)
	recycleRepo := repository.NewRecycleRepository(db)
	favoriteRepo := repository.NewFavoriteRepository(db)
	shareRepo := &shareclient.Shares{Client: clients.Share}
	fileService := application.NewFileService(db, cache.GetClient(), fileRepo, quotaRepo, shareRepo, storage)
	files := persistenceinfra.NewGormUploadFileRepository(db)
	upload := application.NewUploadApplication(files, persistenceinfra.NewGormUploadQuotaRepository(db), transaction.NewGormTransactionManager(db), sessioncache.NewRedisUploadSessionStore(cache.GetClient(), 24*time.Hour, 48*time.Hour), storage, &eventing.Publisher{DB: db})
	upload.StartChunkUploadCleanup(ctx)
	purge := application.NewRecyclePurgeService(db, storage, recycleRepo, fileRepo, shareRepo, favoriteRepo, quotaRepo)
	recycleService := application.NewRecycleService(db, recycleRepo, fileRepo, purge, nil)
	startRecycleCleanupWorkers(ctx, recycleService, nil, &cfg.RabbitMQ)
	go application.RunObjectCleanup(ctx, db, storage)
	fileCtrl := transport.NewFileController(fileService, cfg, nil)
	uploadCtrl := transport.NewUploadController(upload, cfg)
	downloadCtrl := transport.NewDownloadController(application.NewDownloadApplication(files, storage))
	favoriteCtrl := transport.NewFavoriteController(application.NewFavoriteService(favoriteRepo, fileRepo, fileService))
	recycleCtrl := transport.NewRecycleController(recycleService)
	categoryCtrl := transport.NewCategoryController(application.NewCategoryService(db, fileRepo, fileService), fileService)
	stats := transport.NewStatsController(application.NewStatsService(fileRepo, quotaRepo, shareRepo), application.NewStorageQuotaService(quotaRepo))
	group := protected("user")
	group.GET("/stats", stats.GetUserDashboardStats)
	group.GET("/quota", stats.GetUserStorage)
	transport.RegisterFileInternal(ginServer, clients.Token, db)
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

	return ginServer, nil
}
