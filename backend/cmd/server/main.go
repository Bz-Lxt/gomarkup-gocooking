// GoCooking API 入口：加载配置 → slog → Postgres 迁移（advisory lock）→ 种子 → Gin。
// 优雅退出 15s。时区由容器 TZ=Asia/Shanghai 与 pkg/timeutil 共同保证。
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gocooking/internal/config"
	"gocooking/internal/db"
	"gocooking/internal/handler"
	"gocooking/internal/service"
	"gocooking/pkg/logger"
)

func main() {
	// 启动顺序固定：config → logger → db → migrate → seed → listen。
	// seed 失败必须退出，避免空库被当成「启动成功」。

	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	logger.Init(cfg.LogLevel, cfg.Env)
	logger.Info("starting", "env", cfg.Env, "addr", cfg.HTTPAddr)

	gdb, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		logger.Error("db open failed", "err", err)
		os.Exit(1)
	}
	if err := db.Migrate(gdb); err != nil {
		logger.Error("migrate failed", "err", err)
		os.Exit(1)
	}
	if err := db.Seed(gdb); err != nil {
		logger.Error("seed failed", "err", err)
		os.Exit(1)
	}

	r := handler.NewRouter(handler.Deps{
		Catalog: service.NewCatalog(gdb),
		Planner: service.NewPlanner(gdb),
		Secret:  cfg.JWTSecret,
	})
	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: r, ReadHeaderTimeout: 8 * time.Second}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("listen failed", "err", err)
			os.Exit(1)
		}
	}()
	logger.Info("listening", "addr", cfg.HTTPAddr)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown", "err", err)
	}
}
