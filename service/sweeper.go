package service

import (
	"context"
	"log/slog"
	"time"
)

// Sweeper 告警升级扫描定时任务。
// 每 SweepInterval 扫描一次未确认超时告警并自动升级；
// 触达 service -> store -> 告警页。
type Sweeper struct {
	Services *Services
	Interval time.Duration
}

// NewSweeper 构造扫描器。
func NewSweeper(svc *Services, interval time.Duration) *Sweeper {
	if interval <= 0 {
		interval = 10 * time.Minute
	}
	return &Sweeper{Services: svc, Interval: interval}
}

// Run 启动后台扫描循环，直到 ctx 取消。
func (sw *Sweeper) Run(ctx context.Context) {
	slog.Info("告警升级扫描已启动", "interval", sw.Interval.String())
	ticker := time.NewTicker(sw.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("告警升级扫描已停止")
			return
		case now := <-ticker.C:
			n, err := sw.Services.EscalateDue(now)
			if err != nil {
				slog.Error("告警升级扫描失败", "err", err)
				continue
			}
			if n > 0 {
				slog.Info("自动升级告警", "count", n)
			}
		}
	}
}
