package controller

import (
	"fmt"
	"go-cloud-storage/internal/pkg/aliyunoss"
	"go-cloud-storage/internal/pkg/utils"
	"go-cloud-storage/internal/repositories"
	"go-cloud-storage/internal/services"
	"net/http"
	"strconv"

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

func (c *UploadController) InitUpload(ctx *gin.Context) {
	userId := ctx.GetInt("userId")
	if userId == 0 {
		utils.Fail(ctx, http.StatusUnauthorized, "用户未登录")
		return
	}

	var req struct {
		FileName  string `json:"fileName" binding:"required"`
		FileSize  int64  `json:"fileSize" binding:"required"`
		FileHash  string `json:"fileHash" binding:"required"`
		ChunkSize int64  `json:"chunkSize" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	// 检查文件是否已存在（秒传）
	exists, existingFile, err := c.fileService.CheckFileExistsByMD5(userId, req.FileHash)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "检查文件存在性失败: "+err.Error())
		return
	}

	if exists {
		utils.Success(ctx, gin.H{
			"message": "文件已存在，秒传成功",
			"status":  "instant_upload_success",
			"file":    existingFile,
		})
		return
	}

	task, err := c.uploadService.InitUpload(ctx, userId, req.FileName, req.FileSize, req.ChunkSize, req.FileHash)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, task)
}

func (c *UploadController) GetChunkPresignedURL(ctx *gin.Context) {
	taskId := ctx.Param("taskId")
	partNumberStr := ctx.Param("partNumber")
	partNumber, err := strconv.Atoi(partNumberStr)
	if err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "无效的分片编号")
		return
	}

	url, err := c.uploadService.GetChunkPresignedURL(ctx, taskId, partNumber)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, gin.H{"url": url})
}

func (c *UploadController) MarkChunkUploaded(ctx *gin.Context) {
	taskId := ctx.Param("taskId")
	partNumberStr := ctx.Param("partNumber")
	partNumber, err := strconv.Atoi(partNumberStr)
	if err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "无效的分片编号")
		return
	}

	var req struct {
		ETag string `json:"etag" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	err = c.uploadService.MarkChunkUploaded(ctx, taskId, partNumber, req.ETag)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, "ok")
}

func (c *UploadController) GetTask(ctx *gin.Context) {
	taskId := ctx.Param("taskId")
	task, err := c.uploadService.GetTask(taskId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	ctx.JSON(http.StatusOK, task)
}

func (c *UploadController) CompleteUpload(ctx *gin.Context) {
	taskId := ctx.Param("taskId")
	err := c.uploadService.CompleteUpload(ctx, taskId)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	utils.Success(ctx, "ok")
}

// 获取未完成的上传任务
func (c *UploadController) GetIncompleteTasks(ctx *gin.Context) {
	userId := ctx.GetInt("userId")
	tasks, err := c.uploadService.GetIncompleteTasks(userId)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	utils.Success(ctx, tasks)
}

// 删除上传任务
func (c *UploadController) DeleteTask(ctx *gin.Context) {
	taskId := ctx.Param("taskId")
	userId := ctx.GetInt("userId")

	err := c.uploadService.DeleteTask(taskId, userId)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	utils.Success(ctx, "ok")
}

// 检查文件是否存在（秒传）
func (c *UploadController) CheckFileExists(ctx *gin.Context) {
	var req struct {
		FileHash string `json:"fileHash" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "参数错误")
		return
	}

	userId := ctx.GetInt("userId")
	exists, fileInfo, err := c.fileService.CheckFileExistsByMD5(userId, req.FileHash)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, gin.H{
		"exists": exists,
		"file":   fileInfo,
	})
}
