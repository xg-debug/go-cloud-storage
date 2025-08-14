package controller

import (
	"go-cloud-storage/internal/pkg/utils"
	"go-cloud-storage/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type RecycleController struct {
	recycleService services.RecycleService
}

func NewRecycleController(service services.RecycleService) *RecycleController {
	return &RecycleController{recycleService: service}
}

// ListRecycleFiles 获取回收站文件列表
func (rc *RecycleController) ListRecycleFiles(c *gin.Context) {
	userId := c.GetInt("userId")
	// 获取回收站记录
	records, err := rc.recycleService.GetRecycleFiles(userId)
	if err != nil {
		utils.Fail(c, http.StatusInternalServerError, "获取回收站删除文件失败")
		return
	}

	utils.Success(c, gin.H{"data": records})
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
