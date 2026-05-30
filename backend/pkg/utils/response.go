package utils

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// 错误码字典
const (
	// 鉴权类 10000-10099
	CodeUnauthorized     = 10001
	CodeTokenExpired     = 10002
	CodeForbidden        = 10003
	CodeLoginFailed      = 10004
	CodeTokenRefreshFail = 10005
	CodeExtractCodeWrong = 10006

	// 参数类 20000-20099
	CodeInvalidParam  = 20001
	CodeFileNotFound  = 20002
	CodeUserNotFound  = 20003
	CodeFileNotYours  = 20004
	CodePageTooLarge  = 20005

	// 业务类 30000-30099
	CodeDuplicateName   = 30001
	CodeQuotaExceeded   = 30002
	CodeMoveToSelf      = 30003
	CodeShareExpired    = 30004
	CodeShareCancelled  = 30005
	CodeExtractLocked   = 30006
	CodeFileTypeForbid  = 30007
	CodeFileTooLarge    = 30008
	CodeRateLimited     = 30009

	// 存储类 40000-40099
	CodeUploadFailed  = 40001
	CodeMinIOError    = 40002
	CodeDownloadFailed = 40003

	// 系统类 50000-50099
	CodeInternalError = 50001
	CodeDBError       = 50002
)

// 错误码对应的默认消息
var codeMessages = map[int]string{
	CodeUnauthorized:     "未登录或登录已过期",
	CodeTokenExpired:     "令牌已过期",
	CodeForbidden:        "无权限访问",
	CodeLoginFailed:      "用户名或密码错误",
	CodeTokenRefreshFail: "刷新令牌失败",
	CodeExtractCodeWrong: "提取码错误",
	CodeInvalidParam:     "参数错误",
	CodeFileNotFound:     "文件不存在",
	CodeUserNotFound:     "用户不存在",
	CodeFileNotYours:     "无权操作此文件",
	CodePageTooLarge:     "分页大小超过限制",
	CodeDuplicateName:    "同名文件已存在",
	CodeQuotaExceeded:    "存储空间不足",
	CodeMoveToSelf:       "不能移动到自身或子目录",
	CodeShareExpired:     "分享链接已过期",
	CodeShareCancelled:   "分享已被取消",
	CodeExtractLocked:    "提取码尝试次数过多，请稍后再试",
	CodeFileTypeForbid:   "不支持的文件类型",
	CodeFileTooLarge:     "文件大小超过限制",
	CodeRateLimited:      "请求过于频繁，请稍后再试",
	CodeUploadFailed:     "上传失败",
	CodeMinIOError:       "存储服务异常",
	CodeDownloadFailed:   "下载失败",
	CodeInternalError:    "服务器内部错误",
	CodeDBError:          "数据库操作失败",
}

type Response struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	RequestId string      `json:"requestId"`
}

func Success(ctx *gin.Context, data interface{}) {
	ctx.JSON(http.StatusOK, Response{
		Code:      200,
		Message:   "success",
		Data:      data,
		RequestId: GetRequestID(ctx),
	})
}

func Fail(ctx *gin.Context, code int, message string) {
	// 如果 message 为空，使用错误码默认消息
	if message == "" {
		if defaultMsg, ok := codeMessages[code]; ok {
			message = defaultMsg
		}
	}
	ctx.JSON(http.StatusOK, Response{
		Code:      code,
		Message:   message,
		RequestId: GetRequestID(ctx),
	})
}

// GetRequestID 从 context 中获取 requestId
func GetRequestID(ctx *gin.Context) string {
	if rid, exists := ctx.Get("requestId"); exists {
		return rid.(string)
	}
	return ""
}
