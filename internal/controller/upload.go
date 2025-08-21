package controller

import (
	"crypto/md5"
	"database/sql"
	"fmt"
	"go-cloud-storage/internal/models"
	"go-cloud-storage/internal/pkg/aliyunoss"
	"go-cloud-storage/internal/pkg/utils"
	"go-cloud-storage/internal/repositories"
	"go-cloud-storage/internal/services"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type UploadController struct {
	ossService       *aliyunoss.OSSService
	fileService      services.FileService
	fileChunkService services.FileChunkService
	storageQuotaRepo repositories.StorageQuotaRepository
}

// services.FileService 是接口, 本身就是引用类型，不需要加 *

func NewUploadController(oss *aliyunoss.OSSService, fs services.FileService, fcs services.FileChunkService, storageQuotaRepo repositories.StorageQuotaRepository) *UploadController {
	return &UploadController{
		ossService:       oss,
		fileService:      fs,
		fileChunkService: fcs,
		storageQuotaRepo: storageQuotaRepo,
	}
}

// 上传文件
func (c *UploadController) Upload(ctx *gin.Context) {
	// 获取参数
	parentId := ctx.PostForm("parentId")
	userId := ctx.GetInt("userId")

	// 获取上传的文件
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "获取文件失败: "+err.Error())
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "打开文件失败: "+err.Error())
		return
	}
	defer file.Close()

	// 调用 OSS 上传
	fileInfo, err := c.ossService.UploadFromStream(ctx, file, fileHeader.Filename, userId, parentId, 100*1024*1024)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	// 保存文件信息到数据库
	err = c.fileService.CreateFileInfo(fileInfo)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "数据库保存上传文件元数据失败: "+err.Error())
		return
	}

	// 更新用户存储配额
	fmt.Println("上传文件大小：", fileInfo.Size)
	if fileInfo.Size > 0 {
		err = c.storageQuotaRepo.UpdateUsedSpace(userId, fileInfo.Size)
		if err != nil {
			// 这里只记录错误，不影响上传成功
			ctx.Error(err)
		}
	}

	utils.Success(ctx, fileInfo)
}

// InitUpload 初始化分片上传
func (c *UploadController) InitUpload(ctx *gin.Context) {
	var req struct {
		FileName string `json:"fileName" binding:"required"`
		FileId   string `json:"fileId" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "参数格式错误"})
		return
	}

	// 从上下文获取用户ID
	userId := ctx.GetInt("userId")

	// 生成OSS对象键
	objectKey := fmt.Sprintf("files/%d/%s/%s", userId, req.FileId, req.FileName)

	// 初始化分片上传
	uploadId, err := c.ossService.InitiateMultipartUpload(ctx, objectKey)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("初始化分片上传失败: %v", err)})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "初始化分片上传成功",
		"data": gin.H{
			"uploadId":  uploadId,
			"objectKey": objectKey,
		},
	})
}

// UploadChunk 上传分片（使用数据库管理）
func (c *UploadController) UploadChunk(ctx *gin.Context) {
	// 获取参数
	fileId := ctx.PostForm("fileId")
	chunkIndexStr := ctx.PostForm("chunkIndex")
	chunkHash := ctx.PostForm("chunkHash")
	uploadId := ctx.PostForm("uploadId")
	objectKey := ctx.PostForm("objectKey")

	if fileId == "" || chunkIndexStr == "" || chunkHash == "" || uploadId == "" || objectKey == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "缺少必要参数"})
		return
	}

	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "分片索引格式错误"})
		return
	}

	// 获取上传的文件
	file, err := ctx.FormFile("chunk")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "获取分片文件失败"})
		return
	}

	// 打开文件
	src, err := file.Open()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "打开分片文件失败"})
		return
	}
	defer src.Close()

	// 读取文件内容
	chunkData, err := io.ReadAll(src)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "读取分片文件失败"})
		return
	}

	// 验证分片哈希
	actualHash := fmt.Sprintf("%x", md5.Sum(chunkData))
	if actualHash != chunkHash {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "分片哈希验证失败"})
		return
	}

	// 上传分片到数据库和OSS
	err = c.fileChunkService.UploadChunk(fileId, chunkIndex, chunkData, chunkHash, uploadId, objectKey)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("上传分片失败: %v", err)})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "分片上传成功",
		"data": gin.H{
			"chunkIndex": chunkIndex,
		},
	})
}

// MergeChunks 合并分片（使用数据库管理）
func (c *UploadController) MergeChunks(ctx *gin.Context) {
	userIdInterface, exists := ctx.Get("userId")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}
	userId := userIdInterface.(int)

	var req struct {
		FileId      string `json:"fileId" binding:"required"`
		FileName    string `json:"fileName" binding:"required"`
		TotalChunks int    `json:"totalChunks" binding:"required"`
		FileSize    int64  `json:"fileSize" binding:"required"`
		FileHash    string `json:"fileHash" binding:"required"`
		ParentId    string `json:"parentId"`
		ObjectKey   string `json:"objectKey" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "参数格式错误"})
		return
	}

	// 检查是否已存在相同哈希的文件（秒传功能）
	exists, existingFile, err := c.fileService.CheckFileExistsByMD5(userId, req.FileHash)
	if err == nil && exists && existingFile != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"message": "文件秒传成功",
			"file":    existingFile,
		})
		return
	}

	// 合并分片
	err = c.fileChunkService.MergeChunks(req.FileId, req.ObjectKey, req.TotalChunks)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("合并分片失败: %v", err)})
		return
	}

	// 获取文件扩展名
	fileExt := ""
	if len(req.FileName) > 0 {
		for i := len(req.FileName) - 1; i >= 0; i-- {
			if req.FileName[i] == '.' {
				fileExt = req.FileName[i+1:]
				break
			}
		}
	}

	// 生成文件URL
	fileURL := c.ossService.GenerateObjectURL(req.ObjectKey)

	// 创建文件记录
	fileInfo := &models.File{
		Id:            req.FileId,
		UserId:        userId,
		Name:          req.FileName,
		Size:          req.FileSize,
		SizeStr:       utils.FormatFileSize(req.FileSize),
		IsDir:         false,
		FileExtension: fileExt,
		OssObjectKey:  req.ObjectKey,
		FileHash:      req.FileHash,
		ParentId:      sql.NullString{String: req.ParentId, Valid: req.ParentId != ""},
		IsDeleted:     false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		FileURL:       fileURL,
		ThumbnailURL:  fileURL,
	}

	err = c.fileService.CreateFileInfo(fileInfo)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "创建文件记录失败"})
		return
	}

	// 更新用户存储配额
	if req.FileSize > 0 {
		err = c.storageQuotaRepo.UpdateUsedSpace(userId, req.FileSize)
		if err != nil {
			// 这里只记录错误，不影响上传成功
			ctx.Error(err)
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "文件上传成功",
		"data": gin.H{
			"file": fileInfo,
		},
	})
}

// GetUploadedChunks 获取已上传的分片列表（使用数据库管理）
func (c *UploadController) GetUploadedChunks(ctx *gin.Context) {
	fileId := ctx.Query("fileId")
	if fileId == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "缺少fileId参数"})
		return
	}

	uploadedChunks, err := c.fileChunkService.GetUploadedChunks(fileId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "获取已上传分片失败"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"uploadedChunks": uploadedChunks})
}

// CheckFileExists 检查文件是否已存在（秒传检查）
func (c *UploadController) CheckFileExists(ctx *gin.Context) {
	fileHash := ctx.Query("fileHash")
	fileName := ctx.Query("fileName")
	fileSize := ctx.Query("fileSize")

	fmt.Printf("CheckFileExists - fileHash: %s, fileName: %s, fileSize: %s\n", fileHash, fileName, fileSize)

	if fileHash == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "缺少fileHash参数"})
		return
	}

	userIdInterface, exists := ctx.Get("userId")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}
	userId := userIdInterface.(int)

	fmt.Printf("CheckFileExists - userId: %d\n", userId)

	fileExists, file, err := c.fileService.CheckFileExistsByMD5(userId, fileHash)
	if err != nil {
		fmt.Printf("CheckFileExists - 检查文件失败: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "检查文件失败: " + err.Error()})
		return
	}

	fmt.Printf("CheckFileExists - fileExists: %t, file: %+v\n", fileExists, file)

	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data": gin.H{
			"exists": fileExists,
			"file":   file,
		},
	})
}
