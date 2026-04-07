# Go Cloud Storage

一个前后端分离的云存储系统，支持登录鉴权、文件上传下载、分片上传、收藏、分享、回收站、容量统计等能力。

- 后端：Go + Gin + GORM + MySQL + Redis + MinIO + RabbitMQ(可选)
- 前端：Vue 3 + Vue Router + Vuex + Element Plus

## 项目结构

```text
go-cloud-storage/
├── backend/                      # Go 后端
│   ├── cmd/                      # 启动入口(main)
│   ├── conf/                     # 后端配置文件
│   ├── infrastructure/           # 基础设施层(mysql/redis/minio/mq/aliyun)
│   ├── internal/                 # 业务核心(controller/service/repository/router)
│   ├── migrations/               # GORM 迁移入口
│   ├── pkg/                      # 通用能力(config/utils)
│   ├── go.mod
│   └── go.sum
├── front/                        # Vue 前端
│   ├── public/
│   ├── src/
│   │   ├── api/                  # 接口封装
│   │   ├── components/           # 通用组件
│   │   ├── router/               # 路由
│   │   ├── store/                # Vuex
│   │   ├── utils/                # 工具与请求封装
│   │   └── views/                # 页面
│   └── package.json
├── docs/                         # 文档
├── image/                        # README 展示图片
├── db.sql                        # 数据库初始化脚本
└── README.md
```

## 功能概览

- 用户认证：注册、登录、Token 刷新、退出
- 文件管理：文件/文件夹管理、预览、下载、搜索、最近文件
- 大文件上传：分片初始化、分片上传、合并、取消
- 收藏与分类：收藏夹、按文件类型分类查看
- 分享：创建分享链接、公开访问、分享文件下载
- 回收站：删除、恢复、批量操作、过期清理
- 统计：用户容量与概览统计

## 运行环境

- Go `go 1.25.6`
- Node.js `24.13.1`
- MySQL `8+`
- Redis `7+`
- MinIO（对象存储）
- RabbitMQ

## 快速开始

### 1. 导入数据库

在 MySQL 中创建数据库后导入根目录脚本：

```bash
mysql -u <user> -p <database_name> < db.sql
```

### 2. 配置后端

编辑配置文件：

- `backend/conf/go-cloud-storage.dev.yaml`

至少确认以下配置：

- `Server.port`
- `Database.*`
- `Redis.*`
- `minio.*`
- `rabbitmq.enabled`（不使用可保持 `false`）

### 3. 启动后端

```bash
cd backend
go mod tidy
go run ./cmd
```

默认端口来自配置文件（当前示例为 `8081`）。

### 4. 启动前端

```bash
cd front
npm install
npm run serve
```

默认访问：`http://localhost:8080`

前端请求地址在 `front/src/utils/request.js`，当前默认指向：`http://localhost:8081`。

## 后端分层说明

- `internal/controller`：HTTP 接口层
- `internal/services`：业务编排层
- `internal/repositories`：数据访问层
- `internal/router`：依赖装配与路由注册
- `internal/middleware`：鉴权等中间件
- `internal/models`：实体与 DTO/VO
- `infrastructure/*`：数据库、缓存、对象存储、消息队列等实现

## 主要 API 路由

- 认证：`/login`、`/register`、`/refresh-token`、`/logout`
- 用户：`/me`、`/user/update`、`/user/password`、`/user/avatar`、`/user/stats`、`/user/quota`
- 文件：`/file/list`、`/file/upload`、`/file/chunk/*`、`/file/preview/:fileId`、`/file/download/:fileId`
- 收藏：`/favorite`
- 回收站：`/recycle`
- 分类：`/category/files`
- 分享：`/share`、`/s/:token`

## 数据迁移说明

项目保留了 `backend/migrations/migrate.go` 的 `AutoMigrate` 实现，但默认启动流程未自动调用迁移逻辑。当前建议以 `db.sql` 为初始化基线。

## 页面预览

![首页](image/img.png)
![文件管理](image/img_1.png)
![回收站](image/img_2.png)
![分享](image/img_3.png)
![个人中心](image/img_4.png)
![登录](image/img_5.png)
![更多界面](image/img_6.png)
