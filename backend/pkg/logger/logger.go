// Package logger 是全项目唯一日志入口。禁止业务代码直接 fmt.Println。
// production 即使 LOG_LEVEL=debug 也会被抬到 info。
package logger

import (
	"log/slog"
	"os"
	"strings"
	"sync"
)

var (
	mu     sync.RWMutex
	global *slog.Logger
)

// Init 按环境初始化全局 slog。production 屏蔽 debug。
func Init(level string, env string) *slog.Logger {
	lv := parseLevel(level, env)
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lv})
	l := slog.New(h)
	mu.Lock()
	global = l
	mu.Unlock()
	slog.SetDefault(l)
	return l
}

func parseLevel(level, env string) slog.Level {
	if strings.EqualFold(env, "production") && strings.TrimSpace(level) == "" {
		return slog.LevelInfo
	}
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		if strings.EqualFold(env, "production") {
			return slog.LevelInfo
		}
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func L() *slog.Logger {
	mu.RLock()
	l := global
	mu.RUnlock()
	if l == nil {
		return Init("info", "development")
	}
	return l
}

func Info(msg string, args ...any)  { L().Info(msg, args...) }
func Warn(msg string, args ...any)  { L().Warn(msg, args...) }
func Error(msg string, args ...any) { L().Error(msg, args...) }
func Debug(msg string, args ...any) { L().Debug(msg, args...) }
