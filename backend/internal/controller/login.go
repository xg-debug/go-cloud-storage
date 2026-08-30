package controller

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go-cloud-storage/backend/infrastructure/cache"
	"go-cloud-storage/backend/internal/services"
	"go-cloud-storage/backend/pkg/utils"
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
	UserInfo interface{} `json:"user_info"`
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
		utils.Fail(ctx, http.StatusBadRequest, "参数错误")
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
	accessToken, err := utils.GenerateAccessToken(user.Id, 2*time.Hour)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "生成访问令牌失败")
		return
	}
	refreshToken, err := utils.GenerateRefreshToken(user.Id, refreshTokenExpire)

	// 存入redis
	rdb := cache.GetClient()
	refreshKey := fmt.Sprintf("user:%d:refresh_token", user.Id)

	// 仅存储刷新令牌，访问令牌无需存储（无状态JWT）
	if rdb == nil {
		// Redis 不可用时降级：签发令牌但跳过持久化（刷新接口将因无法校验而拒绝）
		slog.Warn("redis unavailable, refresh token persistence skipped", "userId", user.Id)
	} else if err = rdb.Set(ctx.Request.Context(), refreshKey, refreshToken, refreshTokenExpire).Err(); err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "刷新令牌存储失败")
		return
	}

	loginResp := LoginResponse{
		UserInfo: user,
	}
	setAuthCookies(ctx, accessToken, refreshToken, 2*time.Hour, refreshTokenExpire)
	utils.Success(ctx, loginResp)
}

func (c *LoginController) RefreshToken(ctx *gin.Context) {
	refreshToken, err := ctx.Cookie("refresh_token")
	if err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "未找到RefreshToken")
		return
	}
	// 解析并验证刷新令牌
	claims, err := utils.ParseTokenWithType(refreshToken, "refresh")
	if err != nil {
		utils.Fail(ctx, http.StatusUnauthorized, "无效RefreshToken")
		return
	}
	// 检查refresh_token是否存在于Redis
	rdb := cache.GetClient()
	if rdb == nil {
		utils.Fail(ctx, http.StatusUnauthorized, "RefreshToken已失效")
		return
	}
	refreshKey := fmt.Sprintf("user:%d:refresh_token", claims.UserId)
	storedToken, err := rdb.Get(ctx.Request.Context(), refreshKey).Result()
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
	setAccessCookie(ctx, newToken, 2*time.Hour)
	setCSRFCookie(ctx)
	utils.Success(ctx, gin.H{"authenticated": true})
}

func (c *LoginController) Register(ctx *gin.Context) {
	var req RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "参数错误")
		return
	}
	// 调用注册服务
	err := c.userService.RegisterUser(req.Email, req.Password, req.PasswordConfirm)
	if err != nil {
		utils.Fail(ctx, http.StatusBadRequest, "注册失败")
		return
	}
	// 注册成功
	utils.Success(ctx, gin.H{"message": "注册成功"})
}

func (c *LoginController) Logout(ctx *gin.Context) {
	userId := ctx.GetInt("userId")
	// 删除刷新令牌
	refreshKey := fmt.Sprintf("user:%d:refresh_token", userId)
	if rdb := cache.GetClient(); rdb != nil {
		if err := rdb.Del(ctx.Request.Context(), refreshKey).Err(); err != nil {
			slog.Error("删除refresh token缓存失败", "error", err)
		}
	}
	clearAuthCookies(ctx)
	utils.Success(ctx, gin.H{"message": "退出成功"})
}

func setAuthCookies(ctx *gin.Context, accessToken, refreshToken string, accessTTL, refreshTTL time.Duration) {
	setAccessCookie(ctx, accessToken, accessTTL)
	setRefreshCookie(ctx, refreshToken, refreshTTL)
	setCSRFCookie(ctx)
}

func setAccessCookie(ctx *gin.Context, token string, ttl time.Duration) {
	setCookie(ctx, "access_token", token, int(ttl.Seconds()), true)
}

func setRefreshCookie(ctx *gin.Context, token string, ttl time.Duration) {
	setCookie(ctx, "refresh_token", token, int(ttl.Seconds()), true)
}

func setCSRFCookie(ctx *gin.Context) {
	setCookie(ctx, "csrf_token", utils.NewUUID(), int((24 * time.Hour).Seconds()), false)
}

func clearAuthCookies(ctx *gin.Context) {
	setCookie(ctx, "access_token", "", -1, true)
	setCookie(ctx, "refresh_token", "", -1, true)
	setCookie(ctx, "csrf_token", "", -1, false)
}

func setCookie(ctx *gin.Context, name, value string, maxAge int, httpOnly bool) {
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie(name, value, maxAge, "/", "", isSecureRequest(ctx), httpOnly)
}

func isSecureRequest(ctx *gin.Context) bool {
	return ctx.Request.TLS != nil || ctx.GetHeader("X-Forwarded-Proto") == "https"
}
