package controller

import (
	"go-cloud-storage/internal/pkg/utils"
	"go-cloud-storage/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CategoryController struct {
	categoryService services.CategoryService
	fileService     services.FileService
}

// GetFilesByCategoryRequest 获取分类文件请求
type GetFilesByCategoryRequest struct {
	FileType  string `json:"fileType" binding:"required"` // image, video, audio, document
	SortBy    string `json:"sortBy"`                      // name, size, updated_at
	SortOrder string `json:"sortOrder"`                   // asc, desc
	Page      int    `json:"page"`
	PageSize  int    `json:"pageSize"`
}

func NewCategoryController(categoryService services.CategoryService, fileService services.FileService) *CategoryController {
	return &CategoryController{
		categoryService: categoryService,
		fileService:     fileService,
	}
}

// GetFilesByCategory 获取特定类型的文件
// @Description 根据文件类型(图片、视频、音频、文档)获取文件列表，支持排序
func (c *CategoryController) GetFilesByCategory(ctx *gin.Context) {
	var req GetFilesByCategoryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	userId := ctx.GetInt("userId")
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	// 验证文件类型
	if !isValidFileType(req.FileType) {
		utils.Fail(ctx, http.StatusBadRequest, "无效的文件类型")
		return
	}

	// 验证排序方式
	if req.SortBy != "" && !isValidSortBy(req.SortBy) {
		utils.Fail(ctx, http.StatusBadRequest, "无效的排序字段")
		return
	}

	// 验证排序顺序
	if req.SortOrder != "" && !isValidSortOrder(req.SortOrder) {
		utils.Fail(ctx, http.StatusBadRequest, "无效的排序顺序")
		return
	}

	// 设置默认排序
	if req.SortBy == "" {
		req.SortBy = "updated_at"
	}
	if req.SortOrder == "" {
		req.SortOrder = "desc"
	}

	files, total, err := c.categoryService.GetFilesByCategory(userId, req.FileType, req.SortBy, req.SortOrder, req.Page, req.PageSize)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "获取文件列表失败: "+err.Error())
		return
	}

	utils.Success(ctx, gin.H{"list": files, "total": total})
}

// 验证文件类型是否有效
func isValidFileType(fileType string) bool {
	validTypes := []string{"image", "video", "audio", "document"}
	for _, t := range validTypes {
		if t == fileType {
			return true
		}
	}
	return false
}

// 验证排序字段是否有效
func isValidSortBy(sortBy string) bool {
	validFields := []string{"name", "size", "created_at"}
	for _, f := range validFields {
		if f == sortBy {
			return true
		}
	}
	return false
}

// 验证排序顺序是否有效
func isValidSortOrder(sortOrder string) bool {
	return sortOrder == "asc" || sortOrder == "desc"
}
