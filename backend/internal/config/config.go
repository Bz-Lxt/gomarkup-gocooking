// Package config 只从环境变量读取。Docker 内 DATABASE_URL 指向 postgres 服务名，
// 本地默认 27343。JWT_SECRET 为空直接拒绝启动。
package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Env         string
	LogLevel    string
	HTTPAddr    string
	DatabaseURL string
	JWTSecret   string
}

func Load() (Config, error) {
	// 本地默认连 27343，与 docker-compose 开发端口一致。

	cfg := Config{
		Env:         getenv("APP_ENV", "development"),
		LogLevel:    getenv("LOG_LEVEL", "info"),
		HTTPAddr:    getenv("HTTP_ADDR", ":8080"),
		DatabaseURL: getenv("DATABASE_URL", "postgres://gocooking:gocooking@127.0.0.1:27343/gocooking?sslmode=disable"),
		JWTSecret:   getenv("JWT_SECRET", "gocooking-dev-jwt-secret-change-me"),
	}
	var missing []string
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if strings.TrimSpace(cfg.JWTSecret) == "" {
		missing = append(missing, "JWT_SECRET")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required env: %s", strings.Join(missing, ","))
	}
	return cfg, nil
}

func getenv(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}
