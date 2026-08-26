package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"example.com/boiler-energy-efficiency-service/domain"
)

func fixedTime() time.Time {
	return time.Date(2026, 8, 25, 10, 0, 0, 0, time.Local)
}

func TestStore_BoilerCRUD(t *testing.T) {
	s, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	b := domain.NewBoiler("1#锅炉", domain.BoilerTypeSteam, 10, 2, fixedTime())
	if err := s.CreateBoiler(b); err != nil {
		t.Fatal(err)
	}
	if b.ID == "" {
		t.Fatal("创建后应分配 ID")
	}
	got, err := s.GetBoiler(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "1#锅炉" {
		t.Fatalf("名称不一致: %s", got.Name)
	}
	if _, err := s.GetBoiler("not_exist"); err == nil {
		t.Fatal("不存在的锅炉应返回错误")
	}
	if n := s.CountBoilers(); n != 1 {
		t.Fatalf("锅炉数量错误: %d", n)
	}
}

func TestStore_RunDataAndIndexes(t *testing.T) {
	s, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	// 先建锅炉。
	b := domain.NewBoiler("1#锅炉", domain.BoilerTypeSteam, 10, 2, fixedTime())
	if err := s.CreateBoiler(b); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		rd := domain.NewRunData(b.ID, fixedTime().Add(time.Duration(i)*time.Minute))
		rd.FuelAmount = 900
		rd.SteamOutput = 8.5
		if err := s.CreateRunData(rd); err != nil {
			t.Fatal(err)
		}
	}
	list, err := s.ListRunDataByBoiler(b.ID, 0)
	if err != nil || len(list) != 5 {
		t.Fatalf("运行数据列表错误: %v %d", err, len(list))
	}
	recent, err := s.RecentRunDataByBoiler(b.ID, 3)
	if err != nil || len(recent) != 3 {
		t.Fatalf("最近数据错误: %v %d", err, len(recent))
	}
	all, _ := s.ListRunDataByBoiler(b.ID, 0)
	if !recent[len(recent)-1].Timestamp.Equal(all[len(all)-1].Timestamp) {
		t.Fatal("最近数据应包含最新一条记录")
	}
	if recent[0].Timestamp.After(recent[2].Timestamp) {
		t.Fatal("最近数据应按时间正序返回")
	}
}

func TestStore_AlertFilter(t *testing.T) {
	s, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	now := fixedTime()
	a1 := domain.NewRunAlert("blr_000001", domain.AlertFlueTempSpike, domain.LevelWarning, "m1", 180, 150, now)
	a2 := domain.NewRunAlert("blr_000001", domain.AlertWaterLow, domain.LevelCritical, "m2", 30, 0, now)
	a3 := domain.NewRunAlert("blr_000002", domain.AlertOxygenAbnormal, domain.LevelWarning, "m3", 2, 0, now)
	for _, a := range []*domain.RunAlert{a1, a2, a3} {
		if err := s.CreateAlert(a); err != nil {
			t.Fatal(err)
		}
	}
	// 确认 a2。
	if err := a2.Acknowledge("o", "", now); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateAlert(a2); err != nil {
		t.Fatal(err)
	}
	open, err := s.ListAlerts(AlertFilter{OpenOnly: true})
	if err != nil || len(open) != 2 {
		t.Fatalf("未处置告警数量错误: %d err=%v", len(open), err)
	}
	byBoiler, err := s.ListAlerts(AlertFilter{BoilerID: "blr_000001"})
	if err != nil || len(byBoiler) != 2 {
		t.Fatalf("按锅炉过滤错误: %d", len(byBoiler))
	}
	// 同类型未处置去重查询。
	dup, err := s.OpenAlertOfType("blr_000001", domain.AlertFlueTempSpike)
	if err != nil || dup == nil {
		t.Fatalf("应找到同类型未处置告警: %v", err)
	}
}

func TestStore_JSONPersistenceRoundtrip(t *testing.T) {
	dir := t.TempDir()
	s1, err := New(Options{DataDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	b := domain.NewBoiler("1#锅炉", domain.BoilerTypeSteam, 10, 2, fixedTime())
	if err := s1.CreateBoiler(b); err != nil {
		t.Fatal(err)
	}
	rd := domain.NewRunData(b.ID, fixedTime())
	rd.FuelAmount = 900
	rd.SteamOutput = 8.5
	if err := s1.CreateRunData(rd); err != nil {
		t.Fatal(err)
	}
	if err := s1.Save(); err != nil {
		t.Fatal(err)
	}
	// 快照文件存在。
	if _, err := os.Stat(filepath.Join(dir, "store.json")); err != nil {
		t.Fatalf("快照文件缺失: %v", err)
	}

	// 重新打开恢复。
	s2, err := New(Options{DataDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s2.GetBoiler(b.ID)
	if err != nil {
		t.Fatalf("恢复锅炉失败: %v", err)
	}
	if got.Name != "1#锅炉" {
		t.Fatalf("恢复的名称错误: %s", got.Name)
	}
	runList, err := s2.ListRunDataByBoiler(b.ID, 0)
	if err != nil || len(runList) != 1 {
		t.Fatalf("恢复运行数据失败: %v %d", err, len(runList))
	}
	// 恢复后继续分配 ID 不冲突。
	b2 := domain.NewBoiler("2#锅炉", domain.BoilerTypeSteam, 7, 2, fixedTime())
	if err := s2.CreateBoiler(b2); err != nil {
		t.Fatal(err)
	}
	if b2.ID == b.ID {
		t.Fatal("恢复后 ID 序列应继续递增")
	}
}

func TestStore_AuditAndReport(t *testing.T) {
	s, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	now := fixedTime()
	entry := domain.NewAuditEntry("req_x", domain.ActionTransition, "boiler", "blr_000001", "o", "状态迁移", now)
	if err := s.AppendAudit(entry); err != nil {
		t.Fatal(err)
	}
	audits, err := s.ListAudit(AuditFilter{Action: domain.ActionTransition})
	if err != nil || len(audits) != 1 {
		t.Fatalf("审计查询错误: %d %v", len(audits), err)
	}
	r := &domain.DailyReport{BoilerID: "blr_000001", BoilerName: "1#锅炉", Date: "2026-08-25", AvgEfficiency: 82.5}
	if err := s.UpsertDailyReport(r); err != nil {
		t.Fatal(err)
	}
	r2 := &domain.DailyReport{BoilerID: "blr_000001", BoilerName: "1#锅炉", Date: "2026-08-25", AvgEfficiency: 83.1}
	if err := s.UpsertDailyReport(r2); err != nil {
		t.Fatal(err)
	}
	if n := s.CountReports(); n != 1 {
		t.Fatalf("日报应去重: %d", n)
	}
	got, err := s.GetDailyReport("blr_000001", "2026-08-25")
	if err != nil {
		t.Fatal(err)
	}
	if got.AvgEfficiency != 83.1 {
		t.Fatalf("日报未更新: %v", got.AvgEfficiency)
	}
}
