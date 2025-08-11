package controller

import (
	"go-cloud-storage/internal/models"
	"go-cloud-storage/internal/services"
	"go-cloud-storage/utils"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type RecycleController struct {
	recycleService services.RecycleService
}

func NewRecycleController(service services.RecycleService) *RecycleController {
	return &RecycleController{recycleService: service}
}

// TrashItem 回收站项目响应结构
type TrashItem struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	OriginalPath string `json:"originalPath"`
	DeletedDate  string `json:"deletedDate"`
	Size         string `json:"size"`
}

// ListRecycleFiles 获取回收站文件列表
func (rc *RecycleController) ListRecycleFiles(c *gin.Context) {
	userId := c.GetInt("userId")
	// 获取回收站记录
	recycleRecords, err := rc.recycleService.GetRecycleFiles(userId)
	if err != nil {
		utils.Fail(c, http.StatusInternalServerError, "获取回收站删除文件列表失败")
		return
	}
	// 构建响应数据
	//var trashItems []TrashItem
	//for _, record := range recycleRecords {
	//	// 获取文件信息
	//	file, err := rc.fileRepo.GetFileById(record.FileId)
	//	if err != nil {
	//		continue // 跳过无法找到的文件
	//	}
	//
	//	item := TrashItem{
	//		ID:           record.FileId,
	//		Name:         file.Name,
	//		Type:         getFileType(file),
	//		OriginalPath: record.OriginalPath,
	//		DeletedDate:  record.DeletedAt.Format("2006-01-02"),
	//		Size:         formatFileSize(file.Size, file.IsDir),
	//	}
	//	trashItems = append(trashItems, item)
	//}

	utils.Success(c, gin.H{"data": recycleRecords})
}

func (rc *RecycleController) DeletePermanent(c *gin.Context) {
	fileId := c.Param("fileId")
	if fileId == "" {
		utils.Fail(c, http.StatusBadRequest, "文件ID不能为空")
		return
	}

	err := rc.recycleService.DeleteOne(fileId)
	if err != nil {
		utils.Fail(c, http.StatusInternalServerError, "删除文件失败")
		return
	}

	utils.Success(c, nil)
}

func (rc *RecycleController) DeleteSelected(c *gin.Context) {
	var fileIDs []string
	if err := c.ShouldBindJSON(&fileIDs); err != nil {
		utils.Fail(c, http.StatusBadRequest, "请求参数错误")
		return
	}
	if len(fileIDs) == 0 {
		utils.Fail(c, http.StatusBadRequest, "文件ID列表不能为空")
		return
	}
	if err := rc.recycleService.DeleteSelected(fileIDs); err != nil {
		utils.Fail(c, http.StatusInternalServerError, "批量删除失败")
		return
	}
	utils.Success(c, nil)
}

func (rc *RecycleController) ClearRecycleBin(c *gin.Context) {
	userId := c.GetInt("userId")

	if err := rc.recycleService.ClearRecycles(userId); err != nil {
		utils.Fail(c, http.StatusInternalServerError, "清空回收站失败")
		return
	}
	utils.Success(c, nil)
}

// RestoreFile 恢复单个文件
func (rc *RecycleController) RestoreFile(c *gin.Context) {
	fileId := c.Param("fileId")

	if fileId == "" {
		utils.Fail(c, http.StatusBadRequest, "文件ID不能为空")
		return
	}

	if err := rc.recycleService.RestoreOne(fileId); err != nil {
		utils.Fail(c, http.StatusInternalServerError, "恢复文件失败")
		return
	}

	utils.Success(c, nil)
}

// RestoreSelected 批量恢复文件
func (rc *RecycleController) RestoreSelected(c *gin.Context) {
	var fileIDs []string
	if err := c.ShouldBindJSON(&fileIDs); err != nil {
		utils.Fail(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	if len(fileIDs) == 0 {
		utils.Fail(c, http.StatusBadRequest, "文件ID列表不能为空")
		return
	}
	if err := rc.recycleService.RestoreSelected(fileIDs); err != nil {
		utils.Fail(c, http.StatusInternalServerError, "批量恢复文件失败")
		return
	}

	utils.Success(c, nil)
}

func (rc *RecycleController) RestoreAll(c *gin.Context) {
	userId := c.GetInt("userId")
	if err := rc.recycleService.RestoreAll(userId); err != nil {
		utils.Fail(c, http.StatusInternalServerError, "恢复全部文件失败")
		return
	}
	utils.Success(c, nil)
}

// 辅助函数：获取文件类型
func getFileType(file *models.File) string {
	if file.IsDir {
		return "folder"
	}
	return "file"
}

// 辅助函数：格式化文件大小
func formatFileSize(size int64, isDir bool) string {
	if isDir {
		return "-"
	}

	const unit = 1024
	if size < unit {
		return strconv.FormatInt(size, 10) + " B"
	}

	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB"}
	return strings.TrimSuffix(strconv.FormatFloat(float64(size)/float64(div), 'f', 1, 64), ".0") + " " + units[exp]
}
