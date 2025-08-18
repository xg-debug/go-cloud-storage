package controller

import (
	"go-cloud-storage/internal/pkg/oss"
	"go-cloud-storage/internal/pkg/utils"
	"go-cloud-storage/internal/repositories"
	"go-cloud-storage/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UploadController struct {
	ossService       *oss.OSSService
	fileService      services.FileService
	storageQuotaRepo repositories.StorageQuotaRepository
}

// services.FileService 是接口, 本身就是引用类型，不需要加 *

func NewUploadController(oss *oss.OSSService, service services.FileService, storageQuotaRepo repositories.StorageQuotaRepository) *UploadController {
	return &UploadController{
		ossService:       oss,
		fileService:      service,
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
	if fileInfo.Size > 0 {
		err = c.storageQuotaRepo.UpdateUsedSpace(userId, fileInfo.Size)
		if err != nil {
			// 这里只记录错误，不影响上传成功
			ctx.Error(err)
		}
	}

	utils.Success(ctx, fileInfo)
}

// 分块上传文件
func (c *UploadController) UploadChunk(ctx *gin.Context) {

}
