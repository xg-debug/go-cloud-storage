package controller

import (
	"fmt"
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
	userId, exists := ctx.Get("userId")
	if !exists {
		utils.Fail(ctx, http.StatusBadRequest, "用户未登录")
		return
	}
	profile, err := c.userService.GetProfile(userId.(int))
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
	fmt.Println(req.Username, req.Phone)
	userId, _ := ctx.Get("userId")
	err := c.userService.UpdateUserInfo(userId.(int), req.Username, req.Phone)
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
	userId, exists := ctx.Get("userId")
	if !exists {
		utils.Fail(ctx, http.StatusBadRequest, "用户未登录")
		return
	}
	err := c.userService.ChangePassword(userId.(int), req.OldPassword, req.NewPassword)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	utils.Success(ctx, gin.H{"message": "修改密码成功"})
}
