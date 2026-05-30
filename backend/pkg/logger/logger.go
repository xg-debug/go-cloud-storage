package logger

import (
	"context"
	"log/slog"
	"os"
)

var defaultLogger *slog.Logger

func Init(env string) {
	level := slog.LevelInfo
	opts := &slog.HandlerOptions{
		Level: level,
	}
	var handler slog.Handler
	if env == "prod" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
}

func Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

func Error(msg string, args ...any) {
	slog.Error(msg, args...)
}

func Warn(msg string, args ...any) {
	slog.Warn(msg, args...)
}

func Debug(msg string, args ...any) {
	slog.Debug(msg, args...)
}

// WithRequest 为日志添加 request 相关字段
func WithRequest(ctx context.Context) *slog.Logger {
	// 尝试从 context 中提取 requestId
	if rid, ok := ctx.Value("requestId").(string); ok {
		return slog.With("requestId", rid)
	}
	return slog.Default()
}
