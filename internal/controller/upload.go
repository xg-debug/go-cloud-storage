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

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/gin-gonic/gin"
)

type UploadController struct {
	ossService       *aliyunoss.OSSService
	fileService      services.FileService
	uploadService    services.UploadService
	storageQuotaRepo repositories.StorageQuotaRepository
}

func NewUploadController(oss *aliyunoss.OSSService, fs services.FileService, us services.UploadService, storageQuotaRepo repositories.StorageQuotaRepository) *UploadController {
	return &UploadController{
		ossService:       oss,
		fileService:      fs,
		uploadService:    us,
		storageQuotaRepo: storageQuotaRepo,
	}
}

// 上传文件（普通文件）
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
		FileName string `json:"fileName"`
		FileId   string `json:"fileId"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "参数错误")
		return
	}

	userId := ctx.GetInt("userId")

	// 生成objectKey，添加文件扩展名
	var fileExt string
	for i := len(req.FileName) - 1; i >= 0; i-- {
		if req.FileName[i] == '.' {
			fileExt = req.FileName[i:]
			break
		}
	}
	objectKey := fmt.Sprintf("files/%d/%s%s", userId, req.FileId, fileExt)

	// 初始化OSS分片上传
	uploadId, err := c.ossService.InitiateMultipartUpload(ctx, objectKey)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "初始化分片上传失败: "+err.Error())
		return
	}

	utils.Success(ctx, gin.H{
		"uploadId":  uploadId,
		"objectKey": objectKey,
	})
}

// UploadChunk 上传分片
func (c *UploadController) UploadChunk(ctx *gin.Context) {
	// 获取参数
	fileId := ctx.PostForm("fileId")
	chunkIndexStr := ctx.PostForm("chunkIndex")
	chunkHash := ctx.PostForm("chunkHash")
	uploadId := ctx.PostForm("uploadId")
	objectKey := ctx.PostForm("objectKey")

	if fileId == "" || chunkIndexStr == "" || chunkHash == "" || uploadId == "" || objectKey == "" {
		utils.Fail(ctx, http.StatusBadRequest, "缺少必要参数")
		return
	}

	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "分片索引格式错误")
		return
	}

	// 获取上传的文件
	file, err := ctx.FormFile("chunk")
	if err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "获取分片文件失败")
		return
	}

	// 打开文件
	src, err := file.Open()
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "打开分片文件失败")
		return
	}
	defer src.Close()

	// 读取文件内容
	chunkData, err := io.ReadAll(src)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "读取分片文件失败")
		return
	}

	// 验证分片哈希
	actualHash := fmt.Sprintf("%x", md5.Sum(chunkData))
	if actualHash != chunkHash {
		utils.Fail(ctx, http.StatusBadRequest, "分片哈希验证失败")
		return
	}

	// 上传分片到OSS
	etag, err := c.ossService.UploadPart(ctx, objectKey, uploadId, chunkIndex, chunkData)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "上传分片失败: "+err.Error())
		return
	}

	utils.Success(ctx, gin.H{
		"chunkIndex": chunkIndex,
		"etag":       etag,
	})
}

// MergeChunks 合并分片
func (c *UploadController) MergeChunks(ctx *gin.Context) {
	userId := ctx.GetInt("userId")

	var req struct {
		FileId      string            `json:"fileId" binding:"required"`
		FileName    string            `json:"fileName" binding:"required"`
		TotalChunks int               `json:"totalChunks" binding:"required"`
		FileSize    int64             `json:"fileSize" binding:"required"`
		FileHash    string            `json:"fileHash" binding:"required"`
		ParentId    string            `json:"parentId"`
		ObjectKey   string            `json:"objectKey" binding:"required"`
		UploadId    string            `json:"uploadId" binding:"required"`
		ChunkETags  map[string]string `json:"chunkETags" binding:"required"` // chunkIndex -> etag
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "参数格式错误: "+err.Error())
		return
	}

	// 检查是否已存在相同哈希的文件（秒传功能）
	exists, existingFile, err := c.fileService.CheckFileExistsByMD5(userId, req.FileHash)
	if err == nil && exists && existingFile != nil {
		utils.Success(ctx, gin.H{
			"message": "文件秒传成功",
			"file":    existingFile,
		})
		return
	}

	// 准备分片信息用于合并
	var parts []oss.UploadPart

	for i := 1; i <= req.TotalChunks; i++ {
		etag, ok := req.ChunkETags[strconv.Itoa(i)]
		if !ok {
			utils.Fail(ctx, http.StatusBadRequest, fmt.Sprintf("缺少分片 %d 的ETag", i))
			return
		}
		parts = append(parts, oss.UploadPart{
			PartNumber: int32(i),
			ETag:       &etag,
		})
	}

	// 完成分片上传
	err = c.ossService.CompleteMultipartUpload(ctx, req.ObjectKey, req.UploadId, parts)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "合并分片失败: "+err.Error())
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
		utils.Fail(ctx, http.StatusInternalServerError, "创建文件记录失败: "+err.Error())
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

	utils.Success(ctx, gin.H{
		"file": fileInfo,
	})
}

// GetUploadedChunks 获取已上传的分片列表
func (c *UploadController) GetUploadedChunks(ctx *gin.Context) {
	fileId := ctx.Query("fileId")
	if fileId == "" {
		utils.Fail(ctx, http.StatusBadRequest, "缺少fileId参数")
		return
	}

	// 由于我们使用OSS直接管理分片，这里返回空数组
	// 实际项目中可以根据需要实现分片状态的持久化
	utils.Success(ctx, gin.H{
		"uploadedChunks": []int{},
	})
}

// CheckFileExists 检查文件是否已存在（秒传检查）
func (c *UploadController) CheckFileExists(ctx *gin.Context) {
	fileHash := ctx.Query("fileHash")
	if fileHash == "" {
		utils.Fail(ctx, http.StatusBadRequest, "缺少fileHash参数")
		return
	}

	userId := ctx.GetInt("userId")

	fileExists, file, err := c.fileService.CheckFileExistsByMD5(userId, fileHash)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "检查文件失败: "+err.Error())
		return
	}

	utils.Success(ctx, gin.H{
		"exists": fileExists,
		"file":   file,
	})
}
