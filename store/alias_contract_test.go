package store

import (
	"testing"
	"time"

	"example.com/boiler-energy-efficiency-service/domain"
)

func newAliasTestStore(t *testing.T) (*Store, *domain.Boiler) {
	t.Helper()
	s, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	b := domain.NewBoiler("别名锅炉", domain.BoilerTypeSteam, 10, 2, fixedTime())
	if err := s.CreateBoiler(b); err != nil {
		t.Fatal(err)
	}
	return s, b
}

func createEff(s *Store, b *domain.Boiler, eff float64) {
	s.CreateEfficiency(&domain.EfficiencyRecord{BoilerID: b.ID, Efficiency: eff, Timestamp: fixedTime()})
}

func TestStoreEfficiencyListNoAliasPollution(t *testing.T) {
	s, b := newAliasTestStore(t)
	createEff(s, b, 10)
	createEff(s, b, 20)
	createEff(s, b, 30)

	first, err := s.ListEfficiencyByBoiler(b.ID, 0)
	if err != nil || len(first) != 3 {
		t.Fatalf("首次读取应返回 3 条: %v", err)
	}
	// 第二次带 limit 读取同一集合，不得改写第一次保留的结果。
	if _, err := s.ListEfficiencyByBoiler(b.ID, 2); err != nil {
		t.Fatal(err)
	}
	if len(first) != 3 {
		t.Fatalf("首次读取的切片被改写，长度 %d", len(first))
	}
	if first[0].Efficiency != 10 || first[1].Efficiency != 20 || first[2].Efficiency != 30 {
		t.Fatalf("首次读取的内容被污染: %v %v %v", first[0].Efficiency, first[1].Efficiency, first[2].Efficiency)
	}
}

func TestStoreCombustionListNoAliasPollution(t *testing.T) {
	s, b := newAliasTestStore(t)
	for _, ox := range []float64{5, 8, 11} {
		rd := domain.NewRunData(b.ID, fixedTime())
		rd.OxygenContent = ox
		s.CreateRunData(rd)
		if err := s.CreateCombustion(&domain.CombustionStatus{BoilerID: b.ID, RunDataID: rd.ID, OxygenContent: ox, Timestamp: fixedTime()}); err != nil {
			t.Fatal(err)
		}
	}
	first, err := s.ListCombustionByBoiler(b.ID, 0)
	if err != nil || len(first) != 3 {
		t.Fatalf("首次读取应返回 3 条: %v", err)
	}
	if _, err := s.ListCombustionByBoiler(b.ID, 1); err != nil {
		t.Fatal(err)
	}
	if len(first) != 3 {
		t.Fatalf("首次读取的切片被改写，长度 %d", len(first))
	}
	if first[0].OxygenContent != 5 || first[1].OxygenContent != 8 || first[2].OxygenContent != 11 {
		t.Fatalf("首次读取的内容被污染: %v %v %v", first[0].OxygenContent, first[1].OxygenContent, first[2].OxygenContent)
	}
}

func TestStoreBlowdownListNoAliasPollution(t *testing.T) {
	s, b := newAliasTestStore(t)
	for i := 0; i < 3; i++ {
		rec := domain.NewBlowdownRecord(b.ID, "op", "", 5, int64(i+1), fixedTime().Add(time.Duration(i)*time.Minute))
		if err := s.CreateBlowdown(rec); err != nil {
			t.Fatal(err)
		}
	}
	first, err := s.ListBlowdownByBoiler(b.ID, 0)
	if err != nil || len(first) != 3 {
		t.Fatalf("首次读取应返回 3 条: %v", err)
	}
	if _, err := s.ListBlowdownByBoiler(b.ID, 1); err != nil {
		t.Fatal(err)
	}
	if len(first) != 3 {
		t.Fatalf("首次读取的切片被改写，长度 %d", len(first))
	}
	if first[0].BlowdownNo != 1 || first[1].BlowdownNo != 2 || first[2].BlowdownNo != 3 {
		t.Fatalf("首次读取的内容被污染: %v %v %v", first[0].BlowdownNo, first[1].BlowdownNo, first[2].BlowdownNo)
	}
}

func TestStoreReportListNoAliasPollution(t *testing.T) {
	s, b := newAliasTestStore(t)
	for i, date := range []string{"2026-08-24", "2026-08-25"} {
		r := &domain.DailyReport{BoilerID: b.ID, BoilerName: b.Name, Date: date, RunDataCount: i + 1}
		if err := s.UpsertDailyReport(r); err != nil {
			t.Fatal(err)
		}
	}
	first, err := s.ListDailyReports("")
	if err != nil || len(first) != 2 {
		t.Fatalf("首次读取应返回 2 条: %v", err)
	}
	if _, err := s.ListDailyReports("2026-08-25"); err != nil {
		t.Fatal(err)
	}
	if len(first) != 2 {
		t.Fatalf("首次读取的切片被改写，长度 %d", len(first))
	}
	if first[0].Date != "2026-08-24" || first[1].Date != "2026-08-25" {
		t.Fatalf("首次读取的内容被污染: %v %v", first[0].Date, first[1].Date)
	}
}

// TestStoreCombustionAllListNoAliasPollution 校验全量诊断列表复用缓冲区同样污染旧结果。
func TestStoreCombustionAllListNoAliasPollution(t *testing.T) {
	s, b := newAliasTestStore(t)
	for _, ox := range []float64{5, 8, 11} {
		rd := domain.NewRunData(b.ID, fixedTime())
		rd.OxygenContent = ox
		s.CreateRunData(rd)
		if err := s.CreateCombustion(&domain.CombustionStatus{BoilerID: b.ID, RunDataID: rd.ID, OxygenContent: ox, Timestamp: fixedTime()}); err != nil {
			t.Fatal(err)
		}
	}
	first, err := s.ListCombustion(0)
	if err != nil || len(first) != 3 {
		t.Fatalf("首次读取应返回 3 条: %v", err)
	}
	if _, err := s.ListCombustion(2); err != nil {
		t.Fatal(err)
	}
	if len(first) != 3 {
		t.Fatalf("首次读取的切片被改写，长度 %d", len(first))
	}
	if first[0].OxygenContent != 5 || first[1].OxygenContent != 8 || first[2].OxygenContent != 11 {
		t.Fatalf("首次读取的内容被污染: %v %v %v", first[0].OxygenContent, first[1].OxygenContent, first[2].OxygenContent)
	}
}
