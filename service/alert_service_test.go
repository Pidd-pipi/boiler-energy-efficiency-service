package service

import (
	"testing"
	"time"

	"example.com/boiler-energy-efficiency-service/domain"
	"example.com/boiler-energy-efficiency-service/store"
)

func TestAlertService_AckEscalateResolve(t *testing.T) {
	svc, _ := newTestServices(t)
	b := mustCreateSteamBoiler(t, svc)

	if _, err := svc.IngestRunData("t1", b.ID, RunIngestInput{
		FuelAmount: 900, SteamOutput: 8.5, FlueGasTemp: 150, OxygenContent: 8, IntervalMinutes: 5,
	}); err != nil {
		t.Fatal(err)
	}
	res, err := svc.IngestRunData("t2", b.ID, RunIngestInput{
		FuelAmount: 900, SteamOutput: 8.5, FlueGasTemp: 200, OxygenContent: 2.5, IntervalMinutes: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Alerts) == 0 {
		t.Fatal("应生成告警")
	}
	a := res.Alerts[0]

	// 确认。
	ackd, err := svc.AckAlert("trace-ack", a.ID, "司炉工", "现场已处置")
	if err != nil {
		t.Fatal(err)
	}
	if ackd.Status != domain.AlertAcknowledged {
		t.Fatalf("确认状态错误: %s", ackd.Status)
	}
	// 重复确认报错。
	if _, err := svc.AckAlert("trace-ack2", a.ID, "x", ""); err == nil {
		t.Fatal("重复确认应报错")
	}
	// 处置。
	resolved, err := svc.ResolveAlert("trace-res", a.ID, "司炉工", "修复完成")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Status != domain.AlertResolved {
		t.Fatalf("处置状态错误: %s", resolved.Status)
	}
}

func TestAlertService_EscalateDue(t *testing.T) {
	svc, _ := newTestServices(t)
	b := mustCreateSteamBoiler(t, svc)

	// 让服务时钟可推进。
	base := time.Date(2026, 8, 25, 10, 0, 0, 0, time.Local)
	svc.Now = func() time.Time { return base }
	svc.Cfg.AlertEscalationAfter = time.Hour

	if _, err := svc.IngestRunData("t1", b.ID, RunIngestInput{
		FuelAmount: 900, SteamOutput: 8.5, FlueGasTemp: 150, OxygenContent: 8, IntervalMinutes: 5,
	}); err != nil {
		t.Fatal(err)
	}
	res, err := svc.IngestRunData("t2", b.ID, RunIngestInput{
		FuelAmount: 900, SteamOutput: 8.5, FlueGasTemp: 200, OxygenContent: 2.5, IntervalMinutes: 5,
	})
	if err != nil || len(res.Alerts) == 0 {
		t.Fatalf("应生成告警: %v", err)
	}
	alertID := res.Alerts[0].ID

	// 未超时：不升级。
	n, err := svc.EscalateDue(base.Add(30 * time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("未超时不应升级: %d", n)
	}
	// 超时：全部未确认告警升级（排烟突升 + 氧异常共 2 条）。
	n, err = svc.EscalateDue(base.Add(2 * time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if n < 1 {
		t.Fatalf("应至少升级 1 条: %d", n)
	}
	a, err := svc.Store.GetAlert(alertID)
	if err != nil {
		t.Fatal(err)
	}
	if a.Status != domain.AlertEscalated || a.Level != domain.LevelCritical {
		t.Fatalf("升级结果错误: %+v", a)
	}
	// 已升级不再重复升级。
	n, err = svc.EscalateDue(base.Add(3 * time.Hour))
	if err != nil || n != 0 {
		t.Fatalf("已升级不应重复: %d %v", n, err)
	}
}

func TestListAlertsFilter(t *testing.T) {
	svc, _ := newTestServices(t)
	b := mustCreateSteamBoiler(t, svc)
	if _, err := svc.IngestRunData("t", b.ID, RunIngestInput{
		FuelAmount: 900, SteamOutput: 8.5, FlueGasTemp: 150, OxygenContent: 8, IntervalMinutes: 5,
	}); err != nil {
		t.Fatal(err)
	}
	open, err := svc.ListAlerts(store.AlertFilter{OpenOnly: true})
	if err != nil || len(open) != 0 {
		t.Fatalf("正常数据不应有未处置告警: %d", len(open))
	}
}
