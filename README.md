# Go Cloud Storage

基于 **Go + Gin + MinIO + Vue 3** 的私有云存储系统，支持文件上传/下载、分片续传、文件共享、收藏夹、回收站、容量统计等功能，可用作 golang 练习项目。

## 功能特性

- **账号体系**：用户注册、登录、刷新/注销 Token、头像与密码维护、JWT 鉴权中间件。
- **文件管理**：
  - 普通/大文件上传、断点续传（`/file/chunk/*`）、秒传校验、批量移动/重命名。
  - 目录树、最近文件、模糊搜索、在线预览/下载。
  - MinIO 对象存储封装，结合 MySQL 元数据、Redis 缓存提升查询性能。
- **收藏夹与分类**：一键收藏/取消收藏，按类型（图片/文档/视频/音频等）聚合文件列表。
- **回收站**：软删除、批量恢复或永久删除，定期清理。
- **分享链路**：生成分享链接、查看分享详情、取消分享、免登录公开下载。
- **容量与统计**：用户容量配额、上传/分享数据面板，便于管理员掌握资源使用情况。

## 技术栈

| 层级 | 技术                                        |
| --- |-------------------------------------------|
| 后端 | Go 1.22、Gin、Gorm、MySQL、Redis、MinIO、Docker |
| 前端 | Vue 3、Vue Router、Vuex、Element Plus、Axios  |

## 目录结构

```
go-cloud-storage
├── conf/                         # 环境配置（YAML）
├── internal/
│   ├── controller/               # HTTP 控制器，处理路由请求
│   ├── middleware/               # JWT 等中间件
│   ├── pkg/                      # cache、config、minio、mysql 等通用组件
│   ├── repositories/             # 数据访问层
│   └── services/                 # 业务服务层
├── front/                        # Vue 3 前端项目
│   ├── src/api                   # 后端接口封装
│   ├── src/components            # 公共组件（侧边栏、上传器等）
│   ├── src/views                 # 页面：Dashboard、Files、Recycle、Share 等
│   └── package.json
├── main.go                       # 程序入口：加载配置、初始化依赖、启动 Gin
└── go.mod / go.sum
```

## 快速开始

### 1. 准备环境

- Go 1.21+（推荐 1.22）
- Node.js 18+/npm（或 pnpm/yarn）
- MySQL 8、Redis、MinIO

### 2. 配置

1. 复制 `conf/go-cloud-storage.dev.yaml`，根据环境（数据库、Redis、MinIO 账号、存储路径）修改。
2. 若使用阿里云 OSS，保持 `storageType` 为 `aliyun` 并填入 AccessKey；若自建 MinIO，则保持 `storageType: "minio"` 并填写 endpoint。

### 3. 启动后端

```bash
go mod download
go run main.go
# 默认监听 :8081，可在配置文件 Server.port 中修改
```

启动期间程序会依次完成：

1. 读取配置 (`config.LoadConfig`)
2. 初始化 MySQL / Redis 连接池
3. 建立 MinIO Service（bucket 自动创建）
4. 注册 Gin 路由、CORS、JWT 中间件

### 4. 启动前端

```bash
cd front
npm install
npm run serve
# 默认 http://localhost:8080
```

若与后端端口或域名不一致，请在 `front/src/api/request.js`（或等价文件）中调整 `baseURL`。


## License

自定义/未声明。如需开源请补充许可证说明。
