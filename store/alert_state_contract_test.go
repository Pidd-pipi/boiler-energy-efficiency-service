package store

import (
	"testing"

	"example.com/boiler-energy-efficiency-service/domain"
)

func newStateAlert(t *testing.T) (*Store, *domain.RunAlert) {
	t.Helper()
	s, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	now := fixedTime()
	a := domain.NewRunAlert("b1", domain.AlertFlueTempSpike, domain.LevelWarning, "test", 150, 120, now)
	if err := s.CreateAlert(a); err != nil {
		t.Fatal(err)
	}
	// 升级为 escalated。
	got, err := s.GetAlert(a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := got.Escalate(now); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateAlert(got); err != nil {
		t.Fatal(err)
	}
	return s, got
}

// TestStoreOpenAlertOfTypeSeesEscalated 升级告警应仍参与同类型去重。
func TestStoreOpenAlertOfTypeSeesEscalated(t *testing.T) {
	s, a := newStateAlert(t)
	existing, err := s.OpenAlertOfType(a.BoilerID, a.Type)
	if err != nil {
		t.Fatalf("升级告警应仍被视为未处置用于去重: %v", err)
	}
	if existing == nil || existing.ID != a.ID {
		t.Fatalf("去重应命中已升级告警 %s，实际 %+v", a.ID, existing)
	}
}

// TestStoreCountOpenAlertsIncludesEscalated 未处置数量应包含已升级告警。
func TestStoreCountOpenAlertsIncludesEscalated(t *testing.T) {
	s, a := newStateAlert(t)
	if n := s.CountOpenAlerts(a.BoilerID); n != 1 {
		t.Fatalf("未处置告警数应包含已升级告警，实际 %d", n)
	}
}

// TestStoreListOpenOnlyIncludesEscalated 未处置过滤应包含已升级告警。
func TestStoreListOpenOnlyIncludesEscalated(t *testing.T) {
	s, a := newStateAlert(t)
	list, err := s.ListAlerts(AlertFilter{BoilerID: a.BoilerID, OpenOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != a.ID {
		t.Fatalf("未处置列表应包含已升级告警，实际 %+v", list)
	}
}
