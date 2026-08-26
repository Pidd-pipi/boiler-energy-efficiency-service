package service

import (
	"context"
	"testing"
	"time"

	"example.com/boiler-energy-efficiency-service/domain"
)

// TestSweeperRunExitsOnCancel 取消 ctx 后告警升级扫描 goroutine 应立即退出，不泄漏。
func TestSweeperRunExitsOnCancel(t *testing.T) {
	svc, _ := newTestServices(t)
	sw := NewSweeper(svc, 5*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		sw.Run(ctx)
		close(done)
	}()

	time.Sleep(30 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("告警升级扫描未在取消后退出，goroutine 泄漏")
	}
}

// TestSweeperStopsEscalatingAfterCancel 取消后扫描不应继续自动升级告警。
func TestSweeperStopsEscalatingAfterCancel(t *testing.T) {
	svc, _ := newTestServices(t)
	b := mustCreateSteamBoiler(t, svc)

	base := time.Date(2026, 8, 25, 10, 0, 0, 0, time.Local)
	svc.Now = func() time.Time { return base }
	svc.Cfg.AlertEscalationAfter = time.Hour

	if _, err := svc.IngestRunData("t1", b.ID, RunIngestInput{
		FuelAmount: 900, SteamOutput: 8.5, OxygenContent: 8, IntervalMinutes: 5,
	}); err != nil {
		t.Fatal(err)
	}
	res, err := svc.IngestRunData("t2", b.ID, RunIngestInput{
		FuelAmount: 900, SteamOutput: 8.5, OxygenContent: 2.0, IntervalMinutes: 5,
	})
	if err != nil || len(res.Alerts) == 0 {
		t.Fatalf("应生成告警: %v", err)
	}
	alertID := res.Alerts[0].ID
	svc.Now = func() time.Time { return base.Add(2 * time.Hour) }

	// 先取消 ctx，再启动扫描：取消后不得再升级。
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sw := NewSweeper(svc, 5*time.Millisecond)
	go sw.Run(ctx)
	time.Sleep(40 * time.Millisecond)

	a, err := svc.Store.GetAlert(alertID)
	if err != nil {
		t.Fatal(err)
	}
	if a.Status == domain.AlertEscalated {
		t.Fatalf("取消后扫描不应继续自动升级告警")
	}
}
