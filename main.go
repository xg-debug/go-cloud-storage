package main

import (
	"fmt"
	"go-cloud-storage/internal/pkg/aliyunoss"
	"go-cloud-storage/internal/pkg/cache"
	"go-cloud-storage/internal/pkg/config"
	"go-cloud-storage/internal/pkg/mysql"
	"go-cloud-storage/internal/router"
	"log"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("加载配置文件时出现错误：%v", err)
	}
	// 初始化数据库
	if err := mysql.InitDB(&cfg.Database); err != nil {
		log.Fatalf("数据库初始化失败：%v", err)
	}
	// defer确保程序退出时自动关闭数据库连接
	defer mysql.Close()

	// 初始化Redis
	if err := cache.InitRedis(&cfg.Redis); err != nil {
		log.Fatalf("Redis初始化失败: %v", err)
	}
	defer cache.Close() // 确保程序退出时关闭Redis连接

	// 初始化Oss
	ossService, err := aliyunoss.NewOSSService(&cfg.AliyunOss)
	if err != nil {
		log.Fatalf("OSS初始化失败: " + err.Error())
	}

	// 初始化其他组件（Redis、HTTP服务器等）

	r := router.SetUpRouter(mysql.GormDB, ossService)

	port := fmt.Sprintf(":%d", cfg.Server.Port)
	if err := r.Run(port); err != nil {
		log.Fatal("服务器启动失败...")
	}
}
