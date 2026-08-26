package domain

import (
	"testing"
	"time"
)

// TestAlertIsOpenIncludesEscalated 已升级告警仍应视为未处置（待处理）。
func TestAlertIsOpenIncludesEscalated(t *testing.T) {
	a := NewRunAlert("b1", AlertFlueTempSpike, LevelWarning, "test", 150, 120, time.Now())
	if err := a.Escalate(time.Now()); err != nil {
		t.Fatal(err)
	}
	if !a.IsOpen() {
		t.Fatalf("已升级告警应仍视为未处置，IsOpen 应为 true")
	}
}

// TestAlertAcknowledgeRejectsDoubleAck 已确认告警不能重复确认。
func TestAlertAcknowledgeRejectsDoubleAck(t *testing.T) {
	a := NewRunAlert("b1", AlertFlueTempSpike, LevelWarning, "test", 150, 120, time.Now())
	now := time.Now()
	if err := a.Acknowledge("op1", "note", now); err != nil {
		t.Fatal(err)
	}
	if err := a.Acknowledge("op2", "again", now); err == nil {
		t.Fatalf("已确认告警重复确认应报错")
	}
}

// TestAlertEscalateRejectsAcknowledged 已确认告警不能再升级。
func TestAlertEscalateRejectsAcknowledged(t *testing.T) {
	a := NewRunAlert("b1", AlertFlueTempSpike, LevelWarning, "test", 150, 120, time.Now())
	now := time.Now()
	if err := a.Acknowledge("op1", "note", now); err != nil {
		t.Fatal(err)
	}
	if err := a.Escalate(now); err == nil {
		t.Fatalf("已确认告警升级应报错")
	}
}
