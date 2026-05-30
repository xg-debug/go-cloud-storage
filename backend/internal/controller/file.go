package controller

import (
	"context"
	"fmt"
	"go-cloud-storage/backend/infrastructure/cache"
	"go-cloud-storage/backend/internal/services"
	"go-cloud-storage/backend/pkg/config"
	"go-cloud-storage/backend/pkg/utils"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type FileController struct {
	fileService   services.FileService
	securityCfg   config.SecurityConfig
}

func NewFileController(service services.FileService, cfg *config.Config) *FileController {
	return &FileController{fileService: service, securityCfg: cfg.Security}
}

// GetFilesRequest Gin 对 JSON 解析时，json:"xxx" 的名字要和 前端传的字段一致，且大小写敏感。
type GetFilesRequest struct {
	ParentId  string `json:"parentId" form:"parentId"`
	Page      int    `json:"page" form:"page"`
	PageSize  int    `json:"pageSize" form:"pageSize"`
	SortBy    string `json:"sortBy" form:"sortBy"`
	SortOrder string `json:"sortOrder" form:"sortOrder"`
}

type RenameFileRequest struct {
	FileId  string `json:"fileId"`
	NewName string `json:"newName"`
}

type SearchFilesRequest struct {
	Keyword  string `json:"keyword" binding:"required"`
	ParentId string `json:"parentId"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
}

func (c *FileController) GetFiles(ctx *gin.Context) {
	var req GetFilesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "参数错误")
		return
	}
	userId := ctx.GetInt("userId")

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}
	files, total, err := c.fileService.GetFiles(ctx, userId, req.ParentId, req.Page, req.PageSize, req.SortBy, req.SortOrder)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "查询文件列表失败")
		return
	}

	utils.Success(ctx, gin.H{"list": files, "total": total})
}

func (c *FileController) CreateFolder(ctx *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		ParentId string `json:"parentId"`
	}
	userId := ctx.GetInt("userId")
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		utils.Fail(ctx, http.StatusBadRequest, "文件夹名称不能为空")
		return
	}
	folder, err := c.fileService.CreateFolder(userId, req.Name, req.ParentId)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "创建文件夹失败")
		return
	}
	utils.Success(ctx, folder)
}

// UploadFile 上传小文件 10MB 以内
func (c *FileController) UploadFile(ctx *gin.Context) {
	userId := ctx.GetInt("userId")

	fileHash := ctx.PostForm("fileHash")

	// 获取上传的文件（对应前端 FormData.append('file', file)）
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "读取文件失败")
		return
	}

	// 验证文件扩展名
	if !c.isAllowedExtension(fileHeader.Filename) {
		utils.Fail(ctx, http.StatusBadRequest, "不支持的文件类型")
		return
	}

	// 验证文件大小
	maxSize := int64(c.securityCfg.MaxFileSizeMB) * 1024 * 1024
	if fileHeader.Size > maxSize {
		utils.Fail(ctx, http.StatusBadRequest, fmt.Sprintf("文件大小超过限制（最大 %dMB）", c.securityCfg.MaxFileSizeMB))
		return
	}

	// 获取其他表单参数 (对应前端 FormData.append('parentId', ...))
	parentId := ctx.PostForm("parentId")

	// 打开文件流（获取 io.Reader）
	srcFile, err := fileHeader.Open()
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "打开文件流失败")
		return
	}
	defer srcFile.Close() // 关闭流

	file, err := c.fileService.UploadFile(ctx, srcFile, userId, fileHeader.Filename, fileHeader.Size, fileHash, parentId)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "上传文件失败")
		return
	}
	utils.Success(ctx, file)
}

func (c *FileController) Rename(ctx *gin.Context) {
	var req RenameFileRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "参数错误")
		return
	}
	userId := ctx.GetInt("userId")
	err := c.fileService.Rename(userId, req.FileId, req.NewName)
	if err != nil {
		utils.Fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(ctx, gin.H{"message": "重命名成功"})
}

func (c *FileController) Delete(ctx *gin.Context) {
	fileId := ctx.Param("fileId")
	userId := ctx.GetInt("userId")
	err := c.fileService.Delete(fileId, userId)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	utils.Success(ctx, gin.H{"message": "删除成功"})
}

func (c *FileController) GetRecentFiles(ctx *gin.Context) {
	timeRange := ctx.Query("timeRange")
	userId := ctx.GetInt("userId")
	resultMap, err := c.fileService.GetRecentFiles(userId, timeRange)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	utils.Success(ctx, resultMap)
}

func (c *FileController) PreviewFile(ctx *gin.Context) {
	fileId := ctx.Param("fileId")
	if fileId == "" {
		utils.Fail(ctx, http.StatusBadRequest, "文件ID不能为空")
		return
	}

	userId := ctx.GetInt("userId")

	previewData, err := c.fileService.PreviewFile(userId, fileId)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	// PDF 使用后端代理流式传输，确保 Content-Disposition: inline 生效
	if previewData.PreviewType == "pdf" {
		token := ""
		if auth := ctx.GetHeader("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			token = auth[7:]
		}
		scheme := "http"
		if ctx.Request.TLS != nil || ctx.GetHeader("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		proxyURL := fmt.Sprintf("%s://%s/file/preview-stream/%s?token=%s",
			scheme, ctx.Request.Host, fileId, url.QueryEscape(token))
		previewData.FileURL = proxyURL
	}

	utils.Success(ctx, previewData)
}

func (c *FileController) PreviewStream(ctx *gin.Context) {
	fileId := ctx.Param("fileId")
	userId := ctx.GetInt("userId")

	reader, fileInfo, err := c.fileService.PreviewStream(ctx, userId, fileId)
	if err != nil {
		utils.Fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	defer reader.Close()

	contentType := mimeTypeByExtension(fileInfo.FileExtension)
	ctx.Header("Content-Type", contentType)
	ctx.Header("Content-Disposition", "inline; filename=\""+url.QueryEscape(fileInfo.Name)+"\"")
	ctx.Header("Accept-Ranges", "bytes")
	io.Copy(ctx.Writer, reader)
}

func (c *FileController) SearchFiles(ctx *gin.Context) {
	var req SearchFilesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	userId := ctx.GetInt("userId")

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}

	// 调用服务层搜索文件
	files, total, err := c.fileService.SearchFiles(userId, req.Keyword, req.ParentId, req.Page, req.PageSize)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "搜索文件失败: "+err.Error())
		return
	}

	// 保存搜索历史到 Redis（异步，不影响响应）
	go saveSearchHistory(userId, req.Keyword)

	utils.Success(ctx, gin.H{"list": files, "total": total})
}

// ChunkUploadInit 初始化分片上传
func (c *FileController) ChunkUploadInit(ctx *gin.Context) {
	var req struct {
		FileName string `json:"fileName" binding:"required"`
		FileHash string `json:"fileHash" binding:"required"`
		FileSize int64  `json:"fileSize" binding:"required"`
		ParentId string `json:"parentId"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "参数错误："+err.Error())
		return
	}

	userId := ctx.GetInt("userId")

	// 验证文件扩展名
	if !c.isAllowedExtension(req.FileName) {
		utils.Fail(ctx, http.StatusBadRequest, "不支持的文件类型")
		return
	}

	// 验证文件大小
	maxSize := int64(c.securityCfg.MaxFileSizeMB) * 1024 * 1024
	if req.FileSize > maxSize {
		utils.Fail(ctx, http.StatusBadRequest, fmt.Sprintf("文件大小超过限制（最大 %dMB）", c.securityCfg.MaxFileSizeMB))
		return
	}

	// 调用 Service 层逻辑
	resp, err := c.fileService.InitChunkUpload(ctx, userId, req.FileName, req.FileHash, req.ParentId, req.FileSize)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, resp)
}

// ChunkUploadPart 上传单个分片
// 前端以 multipart/form-data 方式提交: chunk(文件流), chunkIndex(整型), fileHash(字符串)
func (c *FileController) ChunkUploadPart(ctx *gin.Context) {
	// 1. 获取参数
	fileHash := ctx.PostForm("fileHash")
	chunkIndexStr := ctx.PostForm("chunkIndex")
	chunkHash := ctx.PostForm("chunkHash") // 可选：分片 SHA-256，用于完整性校验

	if fileHash == "" || chunkIndexStr == "" {
		utils.Fail(ctx, http.StatusBadRequest, "缺少必要参数 fileHash 或 chunkIndex")
		return
	}

	chunkIndex, _ := strconv.Atoi(chunkIndexStr)
	userId := ctx.GetInt("userId")

	fileHeader, err := ctx.FormFile("chunk")
	if err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "未找到分片文件流")
		return
	}

	srcfile, err := fileHeader.Open()
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "无法读取分片文件")
		return
	}
	defer srcfile.Close()

	// 流式上传: 数据从 HTTP body 直通 MinIO，零内存缓冲
	// chunkHash 非空时，服务端会边上传边校验 hash
	err = c.fileService.UploadChunk(ctx, userId, fileHash, chunkIndex, srcfile, fileHeader.Size, chunkHash)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "分片上传失败: "+err.Error())
		return
	}

	utils.Success(ctx, gin.H{"chunkIndex": chunkIndex, "status": "uploaded"})
}

// ChunkUploadMerge 合并分片
func (c *FileController) ChunkUploadMerge(ctx *gin.Context) {
	var req struct {
		FileHash string `json:"fileHash" binding:"required"`
		FileName string `json:"fileName" binding:"required"`
		FileSize int64  `json:"fileSize" binding:"required"`
		ParentId string `json:"parentId"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	userId := ctx.GetInt("userId")

	// 调用 Service 层逻辑
	// Service 层逻辑：从 Redis 取出所有 Parts (ETags) -> 调用 MinIO CompleteMultipartUpload -> 写入数据库 -> 清理 Redis
	file, err := c.fileService.MergeChunks(ctx, userId, req.FileHash, req.FileName, req.ParentId, req.FileSize)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "合并文件失败: "+err.Error())
		return
	}
	// 返回完整的文件对象/仅返回 URL
	utils.Success(ctx, file)
}

// GetChunkUploadProgress 查询服务端上传进度（断点续传用）
func (c *FileController) GetChunkUploadProgress(ctx *gin.Context) {
	fileHash := ctx.Query("fileHash")
	if fileHash == "" {
		utils.Fail(ctx, http.StatusBadRequest, "缺少 fileHash 参数")
		return
	}
	userId := ctx.GetInt("userId")

	progress, err := c.fileService.GetChunkUploadProgress(ctx, userId, fileHash)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "查询进度失败: "+err.Error())
		return
	}
	utils.Success(ctx, progress)
}

// ChunkUploadCancel 取消上传
func (c *FileController) ChunkUploadCancel(ctx *gin.Context) {
	var req struct {
		FileHash string `json:"fileHash" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "参数错误")
		return
	}
	userId := ctx.GetInt("userId")

	// Service 层逻辑：获取 UploadID -> 调用 MinIO AbortMultipartUpload -> 清理 Redis
	err := c.fileService.CancelChunkUpload(ctx, userId, req.FileHash)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "取消失败: "+err.Error())
		return
	}
	utils.Success(ctx, gin.H{"message": "上传已取消"})
}

// GetFolderTree 获取文件夹树
func (c *FileController) GetFolderTree(ctx *gin.Context) {
	userId := ctx.GetInt("userId")

	tree, err := c.fileService.GetFolderTree(ctx, userId)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "获取文件夹树失败: ")
		return
	}

	utils.Success(ctx, gin.H{"list": tree})
}

// MoveFile 移动文件
func (c *FileController) MoveFile(ctx *gin.Context) {
	var req struct {
		FileId         string `json:"fileId"`
		TargetFolderId string `json:"targetFolderId"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "参数错误")
		return
	}

	userId := ctx.GetInt("userId")

	if err := c.fileService.MoveFile(ctx, userId, req.FileId, req.TargetFolderId); err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "移动失败: ")
		return
	}

	utils.Success(ctx, gin.H{"message": "移动成功"})
}

// GetDownloadInfo 返回下载策略：推荐分块大小、块数、是否支持 Range 等
func (c *FileController) GetDownloadInfo(ctx *gin.Context) {
	fileId := ctx.Param("fileId")
	userId := ctx.GetInt("userId")

	info, err := c.fileService.GetDownloadInfo(ctx, userId, fileId)
	if err != nil {
		utils.Fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(ctx, info)
}

func (c *FileController) Download(ctx *gin.Context) {
	fileId := ctx.Param("fileId")
	userId := ctx.GetInt("userId")

	file, err := c.fileService.GetFileById(fileId)
	if err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "文件不存在")
		return
	}

	// 大文件 (>100MB): 返回 MinIO 预签名 URL 让客户端直连
	// 客户端可用 Range 头并行下载，且不经过应用服务器
	if file.Size > 100*1024*1024 {
		u, _, err := c.fileService.GetPresignedDownloadURL(ctx, userId, fileId)
		if err != nil {
			utils.Fail(ctx, http.StatusInternalServerError, "生成下载链接失败")
			return
		}
		ctx.Header("Content-Disposition", "attachment; filename=\""+file.Name+"\"")
		ctx.Redirect(http.StatusFound, u)
		return
	}

	objSize, err := c.fileService.GetObjectSize(ctx, userId, fileId)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "获取文件信息失败")
		return
	}

	rangeHeader := ctx.GetHeader("Range")

	// 支持 Range 请求: 客户端可多线程分段下载
	if rangeHeader != "" {
		var start, end int64
		_, err := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end)
		if err != nil {
			// 尝试 "bytes=0-" 格式
			_, err = fmt.Sscanf(rangeHeader, "bytes=%d-", &start)
			if err != nil {
				utils.Fail(ctx, http.StatusBadRequest, "无效的 Range 头")
				return
			}
			end = objSize - 1
		}

		if start > end || start >= objSize {
			ctx.Header("Content-Range", fmt.Sprintf("bytes */%d", objSize))
			ctx.Status(http.StatusRequestedRangeNotSatisfiable)
			return
		}
		if end >= objSize {
			end = objSize - 1
		}

		reader, fileInfo, infoSize, err := c.fileService.DownloadRange(ctx, userId, fileId, start, end)
		if err != nil {
			utils.Fail(ctx, http.StatusInternalServerError, "下载失败")
			return
		}
		defer reader.Close()

		contentLen := end - start + 1
		ctx.Header("Content-Disposition", "attachment; filename=\""+fileInfo.Name+"\"")
		ctx.Header("Content-Type", "application/octet-stream")
		ctx.Header("Content-Length", fmt.Sprintf("%d", contentLen))
		ctx.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, infoSize))
		ctx.Header("Accept-Ranges", "bytes")
		ctx.Status(http.StatusPartialContent)
		io.Copy(ctx.Writer, reader)
		return
	}

	// 无 Range: 全量下载
	reader, fileInfo, err := c.fileService.Download(ctx, userId, fileId)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "下载失败")
		return
	}
	defer reader.Close()

	ctx.Header("Content-Disposition", "attachment; filename=\""+fileInfo.Name+"\"")
	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header("Content-Length", fmt.Sprintf("%d", objSize))
	ctx.Header("Accept-Ranges", "bytes")
	io.Copy(ctx.Writer, reader)
}

// CopyFile 复制文件/文件夹
func (c *FileController) CopyFile(ctx *gin.Context) {
	var req struct {
		FileId         string `json:"fileId" binding:"required"`
		TargetFolderId string `json:"targetFolderId"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "参数错误")
		return
	}
	userId := ctx.GetInt("userId")
	if err := c.fileService.CopyFile(ctx, userId, req.FileId, req.TargetFolderId); err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "复制失败: "+err.Error())
		return
	}
	utils.Success(ctx, gin.H{"message": "复制成功"})
}

func (c *FileController) isAllowedExtension(fileName string) bool {
	if len(c.securityCfg.AllowedExtensions) == 0 {
		return true // 未配置白名单则全部允许
	}
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext == "" {
		return false
	}
	for _, allowed := range c.securityCfg.AllowedExtensions {
		if strings.EqualFold(ext, allowed) {
			return true
		}
	}
	return false
}

func mimeTypeByExtension(ext string) string {
	if t := mime.TypeByExtension("." + strings.TrimPrefix(strings.ToLower(ext), ".")); t != "" {
		return t
	}
	return "application/octet-stream"
}

func saveSearchHistory(userId int, keyword string) {
	client := cache.GetClient()
	if client == nil {
		return
	}
	key := fmt.Sprintf("search_history:%d", userId)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	client.ZAdd(ctx, key, &redis.Z{Score: float64(time.Now().Unix()), Member: keyword})
	// 只保留最近50条
	client.ZRemRangeByRank(ctx, key, 0, -51)
	// 设置过期时间30天
	client.Expire(ctx, key, 30*24*time.Hour)
}

// GetSearchHistory 获取搜索历史
func (c *FileController) GetSearchHistory(ctx *gin.Context) {
	userId := ctx.GetInt("userId")
	client := cache.GetClient()
	if client == nil {
		utils.Success(ctx, gin.H{"list": []string{}})
		return
	}

	key := fmt.Sprintf("search_history:%d", userId)
	ctxBg, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	results, err := client.ZRevRange(ctxBg, key, 0, 9).Result()
	if err != nil {
		utils.Success(ctx, gin.H{"list": []string{}})
		return
	}
	utils.Success(ctx, gin.H{"list": results})
}

// DeleteSearchHistory 删除搜索历史
func (c *FileController) DeleteSearchHistory(ctx *gin.Context) {
	userId := ctx.GetInt("userId")
	client := cache.GetClient()
	if client == nil {
		utils.Success(ctx, nil)
		return
	}

	keyword := ctx.Query("keyword")
	key := fmt.Sprintf("search_history:%d", userId)
	ctxBg, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if keyword != "" {
		client.ZRem(ctxBg, key, keyword)
	} else {
		client.Del(ctxBg, key)
	}
	utils.Success(ctx, nil)
}
