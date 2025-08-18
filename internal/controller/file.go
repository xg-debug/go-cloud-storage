package controller

import (
	"go-cloud-storage/internal/pkg/utils"
	"go-cloud-storage/internal/services"
	"net/http"
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
	ParentId string `json:"parentId"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
}

type RenameFileRequest struct {
	FileId  string `json:"fileId"`
	NewName string `json:"newName"`
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
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	files, total, err := c.fileService.GetFiles(ctx, userId, req.ParentId, req.Page, req.PageSize)
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

func (c *FileController) UploadFile(ctx *gin.Context) {
	var req struct {
		Name      string `json:"name"`
		Extension string `json:"extension"`
		Size      int64  `json:"size"`
		ParentId  string `json:"parent_id"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "接收前端参数错误")
		return
	}
	file, err := c.fileService.UploadFile(req.Name, req.Extension, req.Size, req.ParentId)
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
	previewData, err := c.fileService.PreviewFile(ctx, userId, fileId)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(ctx, previewData)
}
