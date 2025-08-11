package controller

import (
	"github.com/gin-gonic/gin"
	"go-cloud-storage/internal/pkg/oss"
	"go-cloud-storage/internal/services"
	"go-cloud-storage/utils"
	"net/http"
)

type UploadController struct {
	ossService  *oss.OSSService
	fileService services.FileService
}

// services.FileService 是接口, 本身就是引用类型，不需要加 *

func NewUploadController(oss *oss.OSSService, service services.FileService) *UploadController {
	return &UploadController{ossService: oss, fileService: service}
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

	// 保存到数据库
	err = c.fileService.CreateFromFileInfo(fileInfo)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "数据库保存上传文件元数据失败: "+err.Error())
		return
	}
	utils.Success(ctx, fileInfo)
}

// 分块上传文件
func (c *UploadController) UploadChunk(ctx *gin.Context) {

}
