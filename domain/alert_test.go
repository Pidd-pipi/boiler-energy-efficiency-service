package domain

import (
	"testing"
	"time"
)

func TestRunAlert_Acknowledge(t *testing.T) {
	now := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	a := NewRunAlert("blr_000001", AlertFlueTempSpike, LevelWarning, "排烟温度突升", 180, 150, now)
	if err := a.Acknowledge("司炉工", "已现场确认", now.Add(time.Minute)); err != nil {
		t.Fatalf("确认失败: %v", err)
	}
	if a.Status != AlertAcknowledged || a.ConfirmBy != "司炉工" {
		t.Fatalf("确认结果错误: %+v", a)
	}
	// 重复确认应报错。
	if err := a.Acknowledge("x", "", now); err == nil {
		t.Fatal("重复确认应报错")
	}
}

func TestRunAlert_Escalate(t *testing.T) {
	now := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	a := NewRunAlert("blr_000001", AlertOxygenAbnormal, LevelWarning, "氧含量异常", 2, 0, now)
	if err := a.Escalate(now.Add(time.Hour)); err != nil {
		t.Fatalf("升级失败: %v", err)
	}
	if a.Status != AlertEscalated || a.Level != LevelCritical {
		t.Fatalf("升级结果错误: %+v", a)
	}
	if a.IsOpen() != true {
		t.Fatal("升级后仍应视为未处置")
	}
}

func TestRunAlert_IsDueEscalation(t *testing.T) {
	now := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	a := NewRunAlert("blr_000001", AlertWaterLow, LevelCritical, "水位过低", 30, 0, now)
	if a.IsDueEscalation(now.Add(59*time.Minute), time.Hour) {
		t.Fatal("未满 1 小时不应升级")
	}
	if !a.IsDueEscalation(now.Add(time.Hour+time.Second), time.Hour) {
		t.Fatal("超过 1 小时应升级")
	}
	// 已确认的不再升级。
	if err := a.Acknowledge("o", "", now); err != nil {
		t.Fatal(err)
	}
	if a.IsDueEscalation(now.Add(2*time.Hour), time.Hour) {
		t.Fatal("已确认告警不应再升级")
	}
}

func TestAlertType_Valid(t *testing.T) {
	for _, typ := range AllAlertTypes() {
		if !typ.Valid() {
			t.Fatalf("枚举 %s 应合法", typ)
		}
	}
	if AlertType("unknown").Valid() {
		t.Fatal("非法告警类型应校验失败")
	}
}
