package transport

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	authrpc "go-cloud-storage/backend/pkg/auth"
	contracts "go-cloud-storage/backend/pkg/contracts"
	middleware "go-cloud-storage/backend/pkg/httpmiddleware"
	"go-cloud-storage/backend/pkg/utils"
)

type LoginUsers interface {
	AuthenticateUser(string, string) (*contracts.User, error)
	ValidateSession(int, string) error
	RegisterUser(string, string, string) error
}

type LoginController struct {
	userService LoginUsers
	sessions    sessionStore
}

func NewLoginController(service LoginUsers) *LoginController {
	return &LoginController{userService: service, sessions: redisSessions{}}
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
	version := user.SessionVersion
	if version == "" {
		utils.Fail(ctx, http.StatusServiceUnavailable, "认证服务返回无效会话")
		return
	}

	// 根据记住我设置不同过期时间
	var refreshTokenExpire time.Duration
	if req.Remember { // 用户勾选了”记住我“
		refreshTokenExpire = 7 * 24 * time.Hour // 刷新令牌有效期延长至7天
	} else {
		refreshTokenExpire = 24 * time.Hour // 默认刷新有效期24小时
	}

	sid := utils.NewUUID()
	// 生成JWT Token
	accessToken, err := utils.GenerateAccessToken(user.Id, 2*time.Hour, version, sid)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "生成访问令牌失败")
		return
	}
	refreshToken, err := utils.GenerateRefreshToken(user.Id, refreshTokenExpire, version, sid)
	if err != nil {
		utils.Fail(ctx, http.StatusInternalServerError, "生成刷新令牌失败")
		return
	}

	if err = c.sessions.start(ctx.Request.Context(), user.Id, sid, refreshToken, refreshTokenExpire); err != nil {
		ctx.JSON(503, gin.H{"code": 503, "message": "会话服务暂不可用"})
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
	if err := c.userService.ValidateSession(claims.UserId, claims.CredentialVersion); err != nil {
		if !errors.Is(err, contracts.ErrSessionInvalid) {
			ctx.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "认证服务暂不可用，请稍后重试"})
			return
		}
		middleware.ClearAuthCookies(ctx)
		utils.Fail(ctx, http.StatusUnauthorized, "登录已失效，请重新登录")
		return
	}
	if err = c.sessions.validate(ctx.Request.Context(), claims.UserId, claims.SessionID, refreshToken); err != nil {
		if errors.Is(err, contracts.ErrSessionInvalid) {
			middleware.ClearAuthCookies(ctx)
			utils.Fail(ctx, 401, "RefreshToken已失效")
		} else {
			ctx.JSON(503, gin.H{"code": 503, "message": "会话服务暂不可用"})
		}
		return
	}
	// 生成新的 访问令牌（始终2小时）
	newToken, err := utils.GenerateAccessToken(claims.UserId, 2*time.Hour, claims.CredentialVersion, claims.SessionID)
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
	token, err := authrpc.AccessToken(ctx.Request)
	if err != nil {
		ctx.Status(401)
		return
	}
	claims, err := utils.ParseTokenWithType(token, "access")
	if err != nil || claims.UserId != ctx.GetInt("userId") {
		ctx.Status(401)
		return
	}
	if err = c.sessions.revoke(ctx.Request.Context(), claims.UserId, claims.SessionID); err != nil {
		ctx.JSON(503, gin.H{"code": 503, "message": "会话服务暂不可用，请重试退出"})
		return
	}
	middleware.ClearAuthCookies(ctx)
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

func setCookie(ctx *gin.Context, name, value string, maxAge int, httpOnly bool) {
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie(name, value, maxAge, "/", "", isSecureRequest(ctx), httpOnly)
}

func isSecureRequest(ctx *gin.Context) bool {
	return ctx.Request.TLS != nil || ctx.GetHeader("X-Forwarded-Proto") == "https"
}
