package controller

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go-cloud-storage/internal/models"
	"go-cloud-storage/internal/pkg/aliyunoss"
	"go-cloud-storage/internal/pkg/cache"
	"go-cloud-storage/internal/pkg/utils"
	"go-cloud-storage/internal/repositories"
	"go-cloud-storage/internal/services"

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

// 检查分片文件状态 - 适配前端 /api/file/check/:fileHash
func (c *UploadController) CheckChunkFileStatus(ctx *gin.Context) {
	fileHash := ctx.Param("fileHash")
	if fileHash == "" {
		utils.Fail(ctx, http.StatusBadRequest, "fileHash参数不能为空")
		return
	}

	userId := ctx.GetInt("userId")

	// 首先检查文件是否已完全上传（秒传）
	exists, fileInfo, err := c.fileService.CheckFileExistsByMD5(userId, fileHash)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	if exists {
		utils.Success(ctx, gin.H{
			"status": "completed",
			"url":    fileInfo.FileURL,
		})
		return
	}

	// 检查Redis中的分片上传信息
	chunkInfo, err := cache.GetChunkUploadInfo(ctx, fileHash)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "获取分片信息失败: "+err.Error())
		return
	}

	if chunkInfo == nil {
		// 没有找到分片信息，返回新上传状态
		utils.Success(ctx, gin.H{
			"status": "new",
		})
		return
	}

	if chunkInfo.Status == "completed" {
		utils.Success(ctx, gin.H{
			"status": "completed",
		})
		return
	}

	// 返回部分上传状态
	utils.Success(ctx, gin.H{
		"status":         "uploading",
		"uploadedChunks": chunkInfo.UploadedChunks,
	})
}

// 分片文件上传 - 适配前端 /api/chunk/upload
func (c *UploadController) ChunkUpload(ctx *gin.Context) {
	// 获取表单参数
	fileHash := ctx.PostForm("fileHash")
	fileName := ctx.PostForm("fileName")
	chunkIndexStr := ctx.PostForm("chunkIndex")
	totalChunksStr := ctx.PostForm("totalChunks")
	biz := ctx.PostForm("biz")

	if fileHash == "" || fileName == "" || chunkIndexStr == "" || totalChunksStr == "" {
		utils.Fail(ctx, http.StatusBadRequest, "缺少必要参数")
		return
	}

	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "chunkIndex参数无效")
		return
	}

	totalChunks, err := strconv.Atoi(totalChunksStr)
	if err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "totalChunks参数无效")
		return
	}

	// 获取上传的分片文件
	fileHeader, err := ctx.FormFile("chunk")
	if err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "获取分片文件失败: "+err.Error())
		return
	}

	userId := ctx.GetInt("userId")

	// 检查或创建Redis中的分片信息
	chunkInfo, err := cache.GetChunkUploadInfo(ctx, fileHash)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "获取分片信息失败: "+err.Error())
		return
	}

	if chunkInfo == nil {
		// 创建新的分片信息
		chunkInfo = &cache.ChunkUploadInfo{
			FileHash:       fileHash,
			FileName:       fileName,
			TotalChunks:    totalChunks,
			UploadedChunks: []int{},
			Status:         "uploading",
			CreatedAt:      time.Now().Unix(),
			UpdatedAt:      time.Now().Unix(),
		}
		err = cache.SaveChunkUploadInfo(ctx, fileHash, chunkInfo)
		if err != nil {
			utils.Fail(ctx, http.StatusInternalServerError, "保存分片信息失败: "+err.Error())
			return
		}
	}

	// 检查分片是否已上传
	for _, uploadedChunk := range chunkInfo.UploadedChunks {
		if uploadedChunk == chunkIndex {
			utils.Success(ctx, gin.H{"message": "分片已存在"})
			return
		}
	}

	// 上传分片到OSS
	file, err := fileHeader.Open()
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "打开分片文件失败: "+err.Error())
		return
	}
	defer file.Close()

	// 读取分片数据
	chunkData, err := io.ReadAll(file)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "读取分片数据失败: "+err.Error())
		return
	}

	// 检查是否已有uploadId，如果没有则初始化分片上传
	if chunkInfo.UploadId == "" || chunkInfo.ObjectKey == "" {
		// 生成OSS对象键
		objectKey := c.ossService.GenerateObjectKey(userId, "", fileName)

		// 初始化分片上传
		uploadId, err := c.ossService.InitiateMultipartUpload(ctx, objectKey)
		if err != nil {
			utils.Fail(ctx, http.StatusInternalServerError, "初始化分片上传失败: "+err.Error())
			return
		}

		// 更新分片信息
		chunkInfo.UploadId = uploadId
		chunkInfo.ObjectKey = objectKey
		chunkInfo.UpdatedAt = time.Now().Unix()

		err = cache.SaveChunkUploadInfo(ctx, fileHash, chunkInfo)
		if err != nil {
			utils.Fail(ctx, http.StatusInternalServerError, "保存分片信息失败: "+err.Error())
			return
		}

		fmt.Printf("初始化分片上传 - uploadId: %s, objectKey: %s\n", uploadId, objectKey)
	}

	fmt.Printf("上传分片 %d - uploadId: %s, objectKey: %s, 数据大小: %d bytes\n",
		chunkIndex, chunkInfo.UploadId, chunkInfo.ObjectKey, len(chunkData))

	// 上传分片到OSS
	uploadPart, err := c.ossService.UploadPart(ctx, chunkInfo.ObjectKey, chunkInfo.UploadId, chunkIndex+1, chunkData)
	if err != nil {
		fmt.Printf("OSS分片上传失败: fileHash=%s, chunkIndex=%d, error=%v\n", fileHash, chunkIndex, err)
		utils.Fail(ctx, http.StatusInternalServerError, "上传分片到OSS失败: "+err.Error())
		return
	}

	fmt.Printf("用户%d上传分片成功: fileHash=%s, chunkIndex=%d, partNumber=%d, ETag=%s, biz=%s\n",
		userId, fileHash, chunkIndex, chunkIndex+1, *uploadPart.ETag, biz)

	// 只有OSS上传成功后才更新Redis中的分片信息
	err = cache.UpdateUploadedChunk(ctx, fileHash, chunkIndex)
	if err != nil {
		fmt.Printf("更新Redis分片信息失败: fileHash=%s, chunkIndex=%d, error=%v\n", fileHash, chunkIndex, err)
		utils.Fail(ctx, http.StatusInternalServerError, "更新分片信息失败: "+err.Error())
		return
	}

	utils.Success(ctx, gin.H{
		"message":    "分片上传成功",
		"chunkIndex": chunkIndex,
		"partNumber": chunkIndex + 1,
	})
}

// 合并分片文件 - 适配前端 /api/chunk/merge
func (c *UploadController) MergeChunkFile(ctx *gin.Context) {
	var req struct {
		FileHash    string `json:"fileHash" binding:"required"`
		FileName    string `json:"fileName" binding:"required"`
		TotalChunks int    `json:"totalChunks" binding:"required"`
		Biz         string `json:"biz"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	// 检查Redis中的分片信息
	chunkInfo, err := cache.GetChunkUploadInfo(ctx, req.FileHash)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "获取分片信息失败: "+err.Error())
		return
	}

	if chunkInfo == nil {
		utils.Fail(ctx, http.StatusBadRequest, "分片信息不存在")
		return
	}

	if len(chunkInfo.UploadedChunks) != req.TotalChunks {
		utils.Fail(ctx, http.StatusBadRequest, "分片未完全上传")
		return
	}

	if chunkInfo.UploadId == "" || chunkInfo.ObjectKey == "" {
		utils.Fail(ctx, http.StatusBadRequest, "分片上传信息不完整")
		return
	}

	// 获取已上传的分片列表
	parts, err := c.ossService.ListParts(ctx, chunkInfo.ObjectKey, chunkInfo.UploadId)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "获取分片列表失败: "+err.Error())
		return
	}

	// 添加详细的调试信息
	fmt.Printf("调试信息 - Redis分片数量: %d, OSS分片数量: %d, 期望分片数量: %d\n",
		len(chunkInfo.UploadedChunks), len(parts), req.TotalChunks)
	fmt.Printf("Redis已上传分片: %v\n", chunkInfo.UploadedChunks)

	// 检查OSS分片数量
	if len(parts) == 0 {
		utils.Fail(ctx, http.StatusBadRequest, "OSS中没有找到任何分片")
		return
	}

	// 严格检查：OSS分片数量必须等于期望数量
	if len(parts) != req.TotalChunks {
		// 数据不一致，需要重新同步
		utils.Fail(ctx, http.StatusBadRequest, fmt.Sprintf("数据不一致 - OSS分片数量: %d, 期望: %d, Redis记录: %d。请重新上传缺失的分片",
			len(parts), req.TotalChunks, len(chunkInfo.UploadedChunks)))
		return
	}

	// 构建分片列表用于合并，按分片编号排序
	var uploadParts []oss.UploadPart
	for _, part := range parts {
		// 只取前 req.TotalChunks 个分片
		if len(uploadParts) < req.TotalChunks {
			uploadParts = append(uploadParts, oss.UploadPart{
				PartNumber: part.PartNumber,
				ETag:       part.ETag,
			})
		}
	}

	fmt.Printf("准备合并的分片数量: %d\n", len(uploadParts))

	// 完成分片上传合并
	err = c.ossService.CompleteMultipartUpload(ctx, chunkInfo.ObjectKey, chunkInfo.UploadId, uploadParts)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "合并分片失败: "+err.Error())
		return
	}

	// 生成文件URL
	fileURL := c.ossService.GenerateObjectURL(chunkInfo.ObjectKey)

	// 获取用户ID
	userId := ctx.GetInt("userId")

	// 创建文件记录到数据库
	fileSize := int64(0)
	for _, part := range parts {
		fileSize += part.Size
	}

	// 创建文件记录
	fileRecord := &models.File{
		Id:            utils.NewUUID(),
		UserId:        userId,
		Name:          req.FileName,
		Size:          fileSize,
		SizeStr:       utils.FormatFileSize(fileSize),
		IsDir:         false,
		FileExtension: strings.ToLower(strings.TrimPrefix(filepath.Ext(req.FileName), ".")),
		OssObjectKey:  chunkInfo.ObjectKey,
		FileHash:      req.FileHash,
		ParentId:      sql.NullString{Valid: false}, // 根目录
		IsDeleted:     false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		FileURL:       fileURL,
		ThumbnailURL:  fileURL,
	}

	// 保存文件记录到数据库
	err = c.fileService.CreateFileInfo(fileRecord)
	if err != nil {
		fmt.Printf("保存文件记录失败: %v\n", err)
		// 不返回错误，因为文件已经上传成功
	} else {
		fmt.Printf("用户%d合并文件成功并保存到数据库: fileId=%s, fileHash=%s, fileName=%s, fileSize=%d, url=%s\n",
			userId, fileRecord.Id, req.FileHash, req.FileName, fileSize, fileURL)
	}

	// 清理Redis中的分片信息
	err = cache.DeleteChunkUploadInfo(ctx, req.FileHash)
	if err != nil {
		// 记录错误但不影响返回结果
		fmt.Printf("清理分片信息失败: %v\n", err)
	}

	utils.Success(ctx, gin.H{
		"url":      fileURL,
		"fileHash": req.FileHash,
		"fileName": req.FileName,
		"message":  "文件合并成功",
	})
}

// 获取登录信息 - 适配前端需要
func (c *UploadController) GetLoginInfo(ctx *gin.Context) {
	userId := ctx.GetInt("userId")
	if userId == 0 {
		utils.Fail(ctx, http.StatusUnauthorized, "用户未登录")
		return
	}

	utils.Success(ctx, gin.H{
		"userId": userId,
		"status": "logged_in",
	})
}

// 取消分片上传 - 新增API
func (c *UploadController) CancelChunkUpload(ctx *gin.Context) {
	var req struct {
		FileHash string `json:"fileHash" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	// 获取分片信息
	chunkInfo, err := cache.GetChunkUploadInfo(ctx, req.FileHash)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "获取分片信息失败: "+err.Error())
		return
	}

	if chunkInfo != nil && chunkInfo.UploadId != "" && chunkInfo.ObjectKey != "" {
		// 取消OSS分片上传
		err = c.ossService.AbortMultipartUpload(ctx, chunkInfo.ObjectKey, chunkInfo.UploadId)
		if err != nil {
			fmt.Printf("取消OSS分片上传失败: %v\n", err)
		}
	}

	// 清理Redis中的分片信息
	err = cache.DeleteChunkUploadInfo(ctx, req.FileHash)
	if err != nil {
		fmt.Printf("清理分片信息失败: %v\n", err)
	}

	utils.Success(ctx, gin.H{"message": "分片上传已取消"})
}

// 暂停分片上传 - 新增API
func (c *UploadController) PauseChunkUpload(ctx *gin.Context) {
	var req struct {
		FileHash string `json:"fileHash" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	// 获取分片信息
	chunkInfo, err := cache.GetChunkUploadInfo(ctx, req.FileHash)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "获取分片信息失败: "+err.Error())
		return
	}

	if chunkInfo == nil {
		utils.Fail(ctx, http.StatusBadRequest, "分片信息不存在")
		return
	}

	// 更新状态为暂停
	chunkInfo.Status = "paused"
	chunkInfo.UpdatedAt = time.Now().Unix()

	err = cache.SaveChunkUploadInfo(ctx, req.FileHash, chunkInfo)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "保存分片信息失败: "+err.Error())
		return
	}

	utils.Success(ctx, gin.H{"message": "分片上传已暂停"})
}

// 继续分片上传 - 新增API
func (c *UploadController) ResumeChunkUpload(ctx *gin.Context) {
	var req struct {
		FileHash string `json:"fileHash" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	// 获取分片信息
	chunkInfo, err := cache.GetChunkUploadInfo(ctx, req.FileHash)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "获取分片信息失败: "+err.Error())
		return
	}

	if chunkInfo == nil {
		utils.Fail(ctx, http.StatusBadRequest, "分片信息不存在")
		return
	}

	// 更新状态为上传中
	chunkInfo.Status = "uploading"
	chunkInfo.UpdatedAt = time.Now().Unix()

	err = cache.SaveChunkUploadInfo(ctx, req.FileHash, chunkInfo)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "保存分片信息失败: "+err.Error())
		return
	}

	utils.Success(ctx, gin.H{"message": "分片上传已继续"})
}
