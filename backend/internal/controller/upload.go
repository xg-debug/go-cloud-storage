package controller

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"go-cloud-storage/backend/internal/services"
	"go-cloud-storage/backend/pkg/config"
	"go-cloud-storage/backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

// UploadController is the HTTP adapter for upload use cases. It is deliberately
// thin: request binding/validation stays here, while the application service only
// sees context.Context, typed DTOs and io.Reader.
type UploadController struct {
	upload      services.UploadApplication
	securityCfg config.SecurityConfig
}

func NewUploadController(upload services.UploadApplication, cfg *config.Config) *UploadController {
	return &UploadController{upload: upload, securityCfg: cfg.Security}
}

type chunkUploadInitRequest struct {
	FileName    string `json:"fileName" binding:"required"`
	FileHash    string `json:"fileHash" binding:"required"`
	FileSize    int64  `json:"fileSize" binding:"required"`
	ParentID    string `json:"parentId"`
	ChunkSize   int64  `json:"chunkSize"`
	TotalChunks int    `json:"totalChunks"`
}

type chunkUploadMergeRequest struct {
	FileHash    string `json:"fileHash" binding:"required"`
	FileName    string `json:"fileName" binding:"required"`
	FileSize    int64  `json:"fileSize" binding:"required"`
	ParentID    string `json:"parentId"`
	ChunkSize   int64  `json:"chunkSize"`
	TotalChunks int    `json:"totalChunks"`
}

type chunkUploadCancelRequest struct {
	FileHash string `json:"fileHash" binding:"required"`
}

func (c *UploadController) UploadFile(ctx *gin.Context) {
	userID := ctx.GetInt("userId")
	fileHash := ctx.PostForm("fileHash")
	if !isSHA256Hex(fileHash) {
		utils.Fail(ctx, http.StatusBadRequest, "无效的文件hash")
		return
	}

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "读取文件失败")
		return
	}
	if !c.isAllowedExtension(fileHeader.Filename) {
		utils.Fail(ctx, http.StatusBadRequest, "不支持的文件类型")
		return
	}
	maxSize := int64(c.securityCfg.MaxFileSizeMB) * 1024 * 1024
	if fileHeader.Size > maxSize {
		utils.Fail(ctx, http.StatusBadRequest, fmt.Sprintf("文件大小超过限制（最大 %dMB）", c.securityCfg.MaxFileSizeMB))
		return
	}
	if fileHeader.Size > normalUploadMaxSize {
		utils.Fail(ctx, http.StatusBadRequest, "普通上传文件不能超过10MB，请使用分片上传")
		return
	}

	src, err := fileHeader.Open()
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "打开文件流失败")
		return
	}
	defer src.Close()

	file, err := c.upload.UploadFile(
		ctx.Request.Context(),
		src,
		userID,
		fileHeader.Filename,
		fileHeader.Size,
		fileHash,
		ctx.PostForm("parentId"),
	)
	if err != nil {
		slog.Error("上传文件失败", "error", err)
		utils.Fail(ctx, http.StatusInternalServerError, "上传文件失败")
		return
	}
	utils.Success(ctx, file)
}

func (c *UploadController) ChunkUploadInit(ctx *gin.Context) {
	var req chunkUploadInitRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "参数错误")
		return
	}
	if !isSHA256Hex(req.FileHash) {
		utils.Fail(ctx, http.StatusBadRequest, "无效的文件hash")
		return
	}
	if !c.isAllowedExtension(req.FileName) {
		utils.Fail(ctx, http.StatusBadRequest, "不支持的文件类型")
		return
	}
	maxSize := int64(c.securityCfg.MaxFileSizeMB) * 1024 * 1024
	if req.FileSize > maxSize {
		utils.Fail(ctx, http.StatusBadRequest, fmt.Sprintf("文件大小超过限制（最大 %dMB）", c.securityCfg.MaxFileSizeMB))
		return
	}

	result, err := c.upload.InitChunkUpload(ctx.Request.Context(), services.InitChunkUploadInput{
		UserID:      ctx.GetInt("userId"),
		FileName:    req.FileName,
		FileHash:    req.FileHash,
		ParentID:    req.ParentID,
		FileSize:    req.FileSize,
		ChunkSize:   req.ChunkSize,
		TotalChunks: req.TotalChunks,
	})
	if err != nil {
		slog.Error("初始化上传失败", "error", err)
		utils.Fail(ctx, http.StatusInternalServerError, "初始化上传失败")
		return
	}
	utils.Success(ctx, result)
}

func (c *UploadController) ChunkUploadPart(ctx *gin.Context) {
	fileHash := ctx.PostForm("fileHash")
	chunkIndexText := ctx.PostForm("chunkIndex")
	chunkHash := ctx.PostForm("chunkHash")
	if fileHash == "" || chunkIndexText == "" {
		utils.Fail(ctx, http.StatusBadRequest, "缺少必要参数 fileHash 或 chunkIndex")
		return
	}
	if !isSHA256Hex(fileHash) {
		utils.Fail(ctx, http.StatusBadRequest, "无效的文件hash")
		return
	}
	chunkIndex, err := strconv.Atoi(chunkIndexText)
	if err != nil || chunkIndex < 0 {
		utils.Fail(ctx, http.StatusBadRequest, "无效的分片索引")
		return
	}

	fileHeader, err := ctx.FormFile("chunk")
	if err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "未找到分片文件流")
		return
	}
	src, err := fileHeader.Open()
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "无法读取分片文件")
		return
	}
	defer src.Close()

	if err := c.upload.UploadChunk(
		ctx.Request.Context(),
		ctx.GetInt("userId"),
		fileHash,
		chunkIndex,
		src,
		fileHeader.Size,
		chunkHash,
	); err != nil {
		slog.Error("分片上传失败", "error", err)
		utils.Fail(ctx, http.StatusInternalServerError, "分片上传失败")
		return
	}
	utils.Success(ctx, gin.H{"chunkIndex": chunkIndex, "status": "uploaded"})
}

func (c *UploadController) ChunkUploadMerge(ctx *gin.Context) {
	var req chunkUploadMergeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "参数错误")
		return
	}
	if !isSHA256Hex(req.FileHash) {
		utils.Fail(ctx, http.StatusBadRequest, "无效的文件hash")
		return
	}

	file, err := c.upload.MergeChunks(
		ctx.Request.Context(),
		ctx.GetInt("userId"),
		req.FileHash,
		req.FileName,
		req.ParentID,
		req.FileSize,
		req.ChunkSize,
		req.TotalChunks,
	)
	if err != nil {
		slog.Error("合并文件失败", "error", err)
		utils.Fail(ctx, http.StatusInternalServerError, "合并文件失败")
		return
	}
	utils.Success(ctx, file)
}

func (c *UploadController) GetChunkUploadProgress(ctx *gin.Context) {
	fileHash := ctx.Query("fileHash")
	if fileHash == "" {
		utils.Fail(ctx, http.StatusBadRequest, "缺少 fileHash 参数")
		return
	}
	result, err := c.upload.GetChunkUploadProgress(ctx.Request.Context(), ctx.GetInt("userId"), fileHash)
	if err != nil {
		slog.Error("查询进度失败", "error", err)
		utils.Fail(ctx, http.StatusInternalServerError, "查询进度失败")
		return
	}
	utils.Success(ctx, result)
}

func (c *UploadController) ChunkUploadCancel(ctx *gin.Context) {
	var req chunkUploadCancelRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "参数错误")
		return
	}
	if err := c.upload.CancelChunkUpload(ctx.Request.Context(), ctx.GetInt("userId"), req.FileHash); err != nil {
		slog.Error("取消上传失败", "error", err)
		utils.Fail(ctx, http.StatusInternalServerError, "取消上传失败")
		return
	}
	utils.Success(ctx, gin.H{"message": "上传已取消"})
}

func (c *UploadController) isAllowedExtension(fileName string) bool {
	if len(c.securityCfg.AllowedExtensions) == 0 {
		return true
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

var _ io.Reader = nil // keep io in this transport file explicit for upload boundaries
