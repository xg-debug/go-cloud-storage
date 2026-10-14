package transport

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"go-cloud-storage/backend/pkg/utils"
	application "go-cloud-storage/backend/services/file-service/internal/application"
)

type FavoriteController struct {
	favoriteService application.FavoriteService
}

func NewFavoriteController(service application.FavoriteService) *FavoriteController {
	return &FavoriteController{favoriteService: service}
}

func (c *FavoriteController) GetFavoriteList(ctx *gin.Context) {
	pageStr := ctx.DefaultQuery("page", "1")
	pageSizeStr := ctx.DefaultQuery("pageSize", "10")
	userId := ctx.GetInt("userId")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	favoriteList, total, err := c.favoriteService.GetFavorites(userId, page, pageSize)
	if err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "获取收藏列表失败")
		return
	}
	utils.Success(ctx, gin.H{"favoriteList": favoriteList, "page": page, "pageSize": pageSize, "total": total})
}

func (c *FavoriteController) Favorite(ctx *gin.Context) {
	fileId := ctx.Param("fileId")
	userId := ctx.GetInt("userId")
	if err := c.favoriteService.AddToFavorite(fileId, userId); err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "收藏失败")
		return
	}
	utils.Success(ctx, nil)
}

func (c *FavoriteController) UnFavorite(ctx *gin.Context) {
	fileId := ctx.Param("fileId")
	userId := ctx.GetInt("userId")
	if err := c.favoriteService.CancelFavorite(userId, fileId); err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "取消收藏失败")
		return
	}
	utils.Success(ctx, nil)
}
