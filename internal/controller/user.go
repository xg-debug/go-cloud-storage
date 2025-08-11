package controller

import (
	"go-cloud-storage/internal/models/dto"
	"go-cloud-storage/internal/services"
	"go-cloud-storage/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

type PrivateInfo struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    string `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenId       string `json:"open_id"`
}

type UserInfoUpdate struct {
	Username string `json:"username"`
	Phone    string `json:"phone"`
}

type UserController struct {
	userService services.UserService
}

func NewUserController(service services.UserService) *UserController {
	return &UserController{userService: service}
}

// GetProfile 获取当前用户信息
func (c *UserController) GetProfile(ctx *gin.Context) {
	userId := ctx.GetInt("userId")
	profile, err := c.userService.GetProfile(userId)
	if err != nil {
		utils.Fail(ctx, http.StatusNotFound, "用户不存在")
	}
	utils.Success(ctx, profile)
}

// UpdateProfile 更新用户信息
func (c *UserController) UpdateProfile(ctx *gin.Context) {
	var req UserInfoUpdate
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "参数错误")
		return
	}
	userId := ctx.GetInt("userId")
	err := c.userService.UpdateUserInfo(userId, req.Username, req.Phone)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "更新用户信息失败")
		return
	}
	utils.Success(ctx, gin.H{"message": "更新用户信息成功"})
}

// UpdatePassword 修改密码
func (c *UserController) UpdatePassword(ctx *gin.Context) {
	var req dto.ChangePasswordDTO
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "参数错误")
		return
	}
	userId := ctx.GetInt("userId")
	err := c.userService.ChangePassword(userId, req.OldPassword, req.NewPassword)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	utils.Success(ctx, gin.H{"message": "修改密码成功"})
}
