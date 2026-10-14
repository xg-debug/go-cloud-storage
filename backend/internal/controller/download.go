package controller

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"go-cloud-storage/backend/internal/services"
	"go-cloud-storage/backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

// DownloadController is the HTTP adapter for download and preview-stream use cases.
// Object storage remains behind DownloadApplication -> ports.Storage.
type DownloadController struct {
	download services.DownloadApplication
}

func NewDownloadController(download services.DownloadApplication) *DownloadController {
	return &DownloadController{download: download}
}

func (c *DownloadController) GetDownloadInfo(ctx *gin.Context) {
	info, err := c.download.GetDownloadInfo(ctx.Request.Context(), ctx.GetInt("userId"), ctx.Param("fileId"))
	if err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "获取下载信息失败")
		return
	}
	utils.Success(ctx, info)
}

func (c *DownloadController) Download(ctx *gin.Context) {
	fileID := ctx.Param("fileId")
	userID := ctx.GetInt("userId")
	rangeHeader := ctx.GetHeader("Range")

	if rangeHeader == "" {
		u, file, err := c.download.GetPresignedDownloadURL(ctx.Request.Context(), userID, fileID)
		if err != nil {
			utils.Fail(ctx, http.StatusInternalServerError, "生成下载链接失败")
			return
		}
		ctx.Header("Content-Disposition", contentDispositionAttachment(file.Name))
		ctx.Redirect(http.StatusFound, u)
		return
	}

	objectSize, err := c.download.GetObjectSize(ctx.Request.Context(), userID, fileID)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "获取文件信息失败")
		return
	}
	start, end, rangeStatus, err := parseSingleRange(rangeHeader, objectSize)
	if err != nil {
		if rangeStatus == http.StatusRequestedRangeNotSatisfiable {
			ctx.Header("Content-Range", fmt.Sprintf("bytes */%d", objectSize))
			ctx.Status(http.StatusRequestedRangeNotSatisfiable)
			return
		}
		utils.Fail(ctx, http.StatusBadRequest, "无效的 Range 头")
		return
	}

	reader, file, actualSize, err := c.download.DownloadRange(ctx.Request.Context(), userID, fileID, start, end)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "下载失败")
		return
	}
	defer reader.Close()

	ctx.Header("Content-Disposition", contentDispositionAttachment(file.Name))
	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header("Content-Length", fmt.Sprintf("%d", end-start+1))
	ctx.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, actualSize))
	ctx.Header("Accept-Ranges", "bytes")
	ctx.Status(http.StatusPartialContent)
	_, _ = io.Copy(ctx.Writer, reader)
}

func (c *DownloadController) PreviewStream(ctx *gin.Context) {
	fileID := ctx.Param("fileId")
	userID := ctx.GetInt("userId")
	objectSize, err := c.download.GetObjectSize(ctx.Request.Context(), userID, fileID)
	if err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "获取文件信息失败")
		return
	}

	start, end := int64(0), objectSize-1
	statusCode := http.StatusOK
	if rangeHeader := ctx.GetHeader("Range"); rangeHeader != "" {
		var rangeStatus int
		start, end, rangeStatus, err = parseSingleRange(rangeHeader, objectSize)
		if err != nil {
			if rangeStatus == http.StatusRequestedRangeNotSatisfiable {
				ctx.Header("Content-Range", fmt.Sprintf("bytes */%d", objectSize))
				ctx.Status(http.StatusRequestedRangeNotSatisfiable)
				return
			}
			utils.Fail(ctx, http.StatusBadRequest, "无效的 Range 头")
			return
		}
		statusCode = http.StatusPartialContent
	}

	reader, file, actualSize, err := c.download.DownloadRange(ctx.Request.Context(), userID, fileID, start, end)
	if err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "获取文件流失败")
		return
	}
	defer reader.Close()

	ctx.Header("Content-Type", mimeTypeByExtension(file.FileExtension))
	ctx.Header("Content-Disposition", "inline; filename=\""+url.QueryEscape(file.Name)+"\"")
	ctx.Header("Accept-Ranges", "bytes")
	ctx.Header("Content-Length", fmt.Sprintf("%d", end-start+1))
	if statusCode == http.StatusPartialContent {
		ctx.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, actualSize))
	}
	ctx.Status(statusCode)
	_, _ = io.Copy(ctx.Writer, reader)
}
