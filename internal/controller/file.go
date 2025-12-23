package controller

import (
	"go-cloud-storage/internal/pkg/utils"
	"go-cloud-storage/internal/services"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type FileController struct {
	fileService services.FileService
}

func NewFileController(service services.FileService) *FileController {
	return &FileController{fileService: service}
}

// GetFilesRequest Gin 对 JSON 解析时，json:"xxx" 的名字要和 前端传的字段一致，且大小写敏感。
type GetFilesRequest struct {
	ParentId string `json:"parentId" form:"parentId"`
	//Page     int    `json:"page" form:"page"`
	//PageSize int    `json:"pageSize" form:"pageSize"`
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

	//if req.Page <= 0 {
	//	req.Page = 1
	//}
	//if req.PageSize <= 0 {
	//	req.PageSize = 20
	//}
	//files, total, err := c.fileService.GetFiles(ctx, userId, req.ParentId, req.Page, req.PageSize)
	files, total, err := c.fileService.GetFiles(ctx, userId, req.ParentId)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "查询文件列表失败")
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

	// 获取文件信息和预览数据
	previewData, err := c.fileService.PreviewFile(userId, fileId)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, previewData)
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
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	// 调用服务层搜索文件
	files, total, err := c.fileService.SearchFiles(userId, req.Keyword, req.ParentId, req.Page, req.PageSize)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "搜索文件失败: "+err.Error())
		return
	}

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

	// 调用 Service 层逻辑
	// Service 层应该处理：查询是否秒传 -> 查询 Redis 是否有 UploadID -> 调用 MinIO InitiateMultipartUpload
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

	if fileHash == "" || chunkIndexStr == "" {
		utils.Fail(ctx, http.StatusBadRequest, "缺少必要参数 fileHash 或 chunkIndex")
		return
	}

	chunkIndex, _ := strconv.Atoi(chunkIndexStr)
	userId := ctx.GetInt("userId")

	// 获取上传的文件分片流
	fileHeader, err := ctx.FormFile("chunk")
	if err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "未找到分片文件流")
		return
	}

	// 3.打开文件流
	srcfile, err := fileHeader.Open()
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "无法读取分片文件")
		return
	}
	defer srcfile.Close()

	// 4.读取二进制数据 (MinIO PutObjectPart 需要 Reader 或 []byte，这里读入内存传给 Service)
	// 注意：分片通常为 5MB~20MB，读入内存是安全的
	data, err := io.ReadAll(srcfile)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "读取分片数据失败")
		return
	}

	// 5.调用 Service 上传: 这里要保证 并发安全 + 幂等。
	// Service 层逻辑：根据 fileHash 从 Redis 获取 UploadID -> 调用 MinIO UploadPart -> 保存 ETag 到 Redis
	err = c.fileService.UploadChunk(ctx, userId, fileHash, chunkIndex, data)
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

func (c *FileController) Download(ctx *gin.Context) {
	fileId := ctx.Param("fileId")

	reader, fileInfo, err := c.fileService.Download(ctx, fileId)
	if err != nil {
		//utils.Fail(ctx, http.StatusInternalServerError, "下载失败")
		ctx.Status(http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	// 设置下载头
	ctx.Header("Content-Disposition", "attachment; filename=\""+fileInfo.Name+"\"")
	ctx.Header("Content-Type", "application/octet-stream")

	// 返回数据流给前端
	_, err = io.Copy(ctx.Writer, reader)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "文件下载失败")
		return
	}
}
