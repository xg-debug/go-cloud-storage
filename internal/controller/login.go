package controller

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go-cloud-storage/internal/pkg/cache"
	"go-cloud-storage/internal/pkg/utils"
	"go-cloud-storage/internal/services"
	"net/http"
	"time"
)

type LoginController struct {
	userService services.UserService
}

func NewLoginController(service services.UserService) *LoginController {
	return &LoginController{userService: service}
}

type LoginRequest struct {
	Account  string `json:"account" binding:"required"` // 支持邮箱、手机号
	Password string `json:"password" binding:"required"`
	Remember bool   `json:"remember"` // 记住我标志
}

type LoginResponse struct {
	Token        string      `json:"token"`
	RefreshToken string      `json:"refresh_token,omitempty"`
	UserInfo     interface{} `json:"user_info"`
}

type RegisterRequest struct {
	Email           string `json:"email" binding:"required"` // 邮箱
	Password        string `json:"password" binding:"required"`
	PasswordConfirm string `json:"password_confirm" binding:"required"`
}

// Login 处理 邮箱/手机号 密码 登录
func (c *LoginController) Login(ctx *gin.Context) {
	var req LoginRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Fail(ctx, http.StatusBadRequest, err.Error())
		return
	}

	// 调用认证服务
	user, err := c.userService.AuthenticateUser(req.Account, req.Password)
	if err != nil {
		utils.Fail(ctx, http.StatusUnauthorized, "认证失败")
		return
	}
	user.Password = ""

	// 根据记住我设置不同过期时间
	var refreshTokenExpire time.Duration
	if req.Remember { // 用户勾选了”记住我“
		refreshTokenExpire = 7 * 24 * time.Hour // 刷新令牌有效期延长至7天
	} else {
		refreshTokenExpire = 24 * time.Hour // 默认刷新有效期24小时
	}

	// 生成JWT Token
	accessToken, err := utils.GenerateAccessToken(user.Id, 2*time.Hour)          // 访问令牌
	refreshToken, err := utils.GenerateRefreshToken(user.Id, refreshTokenExpire) // 刷新令牌

	// 存入redis
	rdb := cache.GetClient()
	refreshKey := fmt.Sprintf("user:%d:refresh_token", user.Id)

	// 仅存储刷新令牌，访问令牌无需存储（无状态JWT）
	err = rdb.Set(ctx.Request.Context(), refreshKey, refreshToken, refreshTokenExpire).Err()
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "刷新令牌存储失败")
		return
	}

	loginResp := LoginResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
		UserInfo:     user,
	}
	ctx.SetCookie(
		"refresh_token",                   // cookie 名称
		refreshToken,                      // cookie 值，即 refresh_token
		int(refreshTokenExpire.Seconds()), // 过期时间，单位秒
		"/",                               // 路径，通常设置根路径
		"",                                // 域名，根据你的环境配置，localhost时可为空或""
		true,                              // Secure，只允许HTTPS请求携带
		true,                              // HttpOnly，前端JS无法读取，防止XSS窃取
	)
	utils.Success(ctx, loginResp)
}

func (c *LoginController) RefreshToken(ctx *gin.Context) {
	refreshToken, err := ctx.Cookie("refresh_token")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, "未找到RefreshToken")
		return
	}
	// 解析并验证刷新令牌
	claims, err := utils.ParseTokenWithType(refreshToken, "refresh")
	if err != nil {
		utils.Fail(ctx, http.StatusUnauthorized, "无效RefreshToken")
		return
	}
	// 检查refresh_token是否存在于Redis
	refreshKey := fmt.Sprintf("user:%d:refresh_token", claims.UserId)
	storedToken, err := cache.GetClient().Get(ctx.Request.Context(), refreshKey).Result()
	if err != nil || storedToken != refreshToken {
		utils.Fail(ctx, http.StatusUnauthorized, "RefreshToken已失效")
		return
	}
	// 生成新的 访问令牌（始终2小时）
	newToken, err := utils.GenerateAccessToken(claims.UserId, 2*time.Hour)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "生成新令牌失败")
		return
	}
	utils.Success(ctx, gin.H{"token": newToken})
}

func (c *LoginController) Register(ctx *gin.Context) {
	var req RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// 调用注册服务
	err := c.userService.RegisterUser(req.Email, req.Password, req.PasswordConfirm)
	if err != nil {
		utils.Fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	// 注册成功
	utils.Success(ctx, gin.H{"message": "注册成功"})
}

func (c *LoginController) Logout(ctx *gin.Context) {
	userId := ctx.GetInt("userId")
	// 删除刷新令牌
	refreshKey := fmt.Sprintf("user:%d:refresh_token", userId)
	cache.GetClient().Del(ctx.Request.Context(), refreshKey)
	ctx.SetCookie("refresh_token", "", -1, "/", "", true, true) // 删除浏览器 Cookie
	utils.Success(ctx, gin.H{"message": "退出成功"})
}
