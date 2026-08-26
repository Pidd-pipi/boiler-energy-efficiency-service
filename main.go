// 工业锅炉能效与运行监测服务入口。
//
// 启动方式：
//
//	go run .                      # 默认监听 8080
//	PORT=18020 go run .           # 端口覆盖
//	DATA_DIR=/tmp/bess go run .   # 指定持久化目录
package main

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example.com/boiler-energy-efficiency-service/config"
	"example.com/boiler-energy-efficiency-service/httpapi"
	"example.com/boiler-energy-efficiency-service/service"
	"example.com/boiler-energy-efficiency-service/store"
)

//go:embed all:web
var webFS embed.FS

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("配置加载失败", "err", err)
		os.Exit(1)
	}

	// 全局结构化日志，级别由 LOG_LEVEL 控制。
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	})))
	slog.Info("配置加载完成", "config", cfg.String(), "log_level", cfg.LogLevel.String())

	st, err := store.New(store.Options{DataDir: cfg.DataDir})
	if err != nil {
		slog.Error("初始化存储失败", "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := st.Close(); err != nil {
			slog.Error("存储落盘失败", "err", err)
		}
	}()

	svc := service.NewServices(cfg, st)

	// 可选演示数据：仓库为空时写入，保证前端开箱可用。
	if cfg.SeedDemo {
		if err := svc.SeedDemoData(); err != nil {
			slog.Error("写入演示数据失败", "err", err)
			os.Exit(1)
		}
	}

	webRoot, err := fs.Sub(webFS, "web")
	if err != nil {
		slog.Error("加载内嵌前端资源失败", "err", err)
		os.Exit(1)
	}

	handler := httpapi.NewRouter(svc, webRoot)

	// 告警升级扫描定时任务。
	sweepCtx, sweepCancel := context.WithCancel(context.Background())
	defer sweepCancel()
	go service.NewSweeper(svc, cfg.SweepInterval).Run(sweepCtx)

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           handler,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("服务启动", "listen", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	// 优雅关闭：优先响应 SIGINT/SIGTERM，也处理启动失败。
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-quit:
		slog.Info("收到退出信号，开始优雅关闭", "signal", sig.String())
	case err := <-errCh:
		if err != nil {
			slog.Error("服务启动失败", "err", err)
			os.Exit(1)
		}
		// ListenAndServe 正常返回 ErrServerClosed 之外的情况由上方处理；
		// 走到这里说明意外结束，按退出流程走。
		slog.Info("服务监听意外结束，开始关闭")
	}
	sweepCancel()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP 服务关闭异常", "err", err)
	}
	if err := st.Close(); err != nil {
		slog.Error("存储落盘失败", "err", err)
	}
	slog.Info("服务已退出")
}
