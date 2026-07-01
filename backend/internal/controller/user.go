package controller

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"go-cloud-storage/backend/internal/models/dto"
	"go-cloud-storage/backend/internal/services"
	"go-cloud-storage/backend/pkg/utils"
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
		return
	}
	utils.Success(ctx, profile)
}

// UpdateProfile 更新用户信息：用户名、手机号
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
		slog.Error("修改密码失败", "error", err)
		utils.Fail(ctx, http.StatusInternalServerError, "修改密码失败")
		return
	}
	utils.Success(ctx, gin.H{"message": "修改密码成功"})
}

// ForgotPassword 忘记密码 - 发送重置邮件
func (c *UserController) ForgotPassword(ctx *gin.Context) {
	var req dto.ForgotPasswordDTO
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "请输入邮箱地址")
		return
	}
	err := c.userService.ForgotPassword(req.Email)
	if err != nil {
		slog.Error("忘记密码失败", "error", err)
		utils.Fail(ctx, http.StatusBadRequest, "发送重置邮件失败")
		return
	}
	utils.Success(ctx, gin.H{"message": "密码重置邮件已发送，请检查邮箱"})
}

// ResetPassword 重置密码
func (c *UserController) ResetPassword(ctx *gin.Context) {
	var req dto.ResetPasswordDTO
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "参数错误")
		return
	}
	err := c.userService.ResetPassword(req.Token, req.Password)
	if err != nil {
		slog.Error("重置密码失败", "error", err)
		utils.Fail(ctx, http.StatusBadRequest, "重置密码失败")
		return
	}
	utils.Success(ctx, gin.H{"message": "密码重置成功，请登录"})
}

// UpdateAvatar 更新头像
func (c *UserController) UpdateAvatar(ctx *gin.Context) {
	userId := ctx.GetInt("userId")

	file, header, err := ctx.Request.FormFile("avatar")
	if err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "获取头像失败")
		return
	}
	defer file.Close()

	avatarURL, err := c.userService.UploadAvatar(ctx, userId, file, header)
	if err != nil {
		slog.Error("上传头像失败", "error", err)
		utils.Fail(ctx, http.StatusInternalServerError, "上传头像失败")
		return
	}

	utils.Success(ctx, gin.H{"message": "上传成功", "avatar": avatarURL})
}
