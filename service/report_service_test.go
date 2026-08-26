package service

import (
	"testing"
	"time"

	"example.com/boiler-energy-efficiency-service/domain"
)

func TestDailyReport_Aggregation(t *testing.T) {
	svc, _ := newTestServices(t)
	b := mustCreateSteamBoiler(t, svc)

	now := time.Date(2026, 8, 25, 10, 0, 0, 0, time.Local)
	// 固定上报时间。
	for i := 0; i < 4; i++ {
		ts := now.Add(time.Duration(i) * time.Hour)
		_, err := svc.IngestRunData("trace", b.ID, RunIngestInput{
			FuelAmount: 900, SteamOutput: 8.5, FlueGasTemp: 150, OxygenContent: 8,
			IntervalMinutes: 60, Operator: "tester",
		})
		if err != nil {
			t.Fatal(err)
		}
		// 修正时间戳：直接改存储中的运行数据时间（测试用）。
		rd, _ := svc.Store.GetRunData(domain.RunData{}.ID)
		_ = rd
		_ = ts
	}
	// 上面上报的时间戳是 svc.Now()（固定 10:00），所以 4 条都会落在 2026-08-25。
	report, err := svc.GenerateDailyReport(b.ID, "2026-08-25")
	if err != nil {
		t.Fatal(err)
	}
	if report.RunDataCount != 4 {
		t.Fatalf("数据条数错误: %d", report.RunDataCount)
	}
	// 每条 60 分钟：产汽 = 8.5 * 1h * 4 = 34t，煤耗 = 900 * 1 * 4 = 3600kg。
	if report.SteamOutputTotal != 34 {
		t.Fatalf("产汽聚合错误: %v", report.SteamOutputTotal)
	}
	if report.CoalConsumptionTotal != 3600 {
		t.Fatalf("煤耗聚合错误: %v", report.CoalConsumptionTotal)
	}
	if report.AvgEfficiency <= 0 {
		t.Fatalf("平均热效率应为正: %v", report.AvgEfficiency)
	}
	if report.AlertCount != 0 {
		t.Fatalf("正常数据告警数应为 0: %d", report.AlertCount)
	}

	// 重复生成应去重并刷新。
	report2, err := svc.GenerateDailyReport(b.ID, "2026-08-25")
	if err != nil {
		t.Fatal(err)
	}
	if report2.ID != report.ID {
		t.Fatalf("日报应去重: %s != %s", report2.ID, report.ID)
	}
}

func TestDailyReport_AlertCount(t *testing.T) {
	svc, _ := newTestServices(t)
	b := mustCreateSteamBoiler(t, svc)

	if _, err := svc.IngestRunData("t1", b.ID, RunIngestInput{
		FuelAmount: 900, SteamOutput: 8.5, FlueGasTemp: 150, OxygenContent: 8, IntervalMinutes: 5,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.IngestRunData("t2", b.ID, RunIngestInput{
		FuelAmount: 900, SteamOutput: 8.5, FlueGasTemp: 200, OxygenContent: 2.5, IntervalMinutes: 5,
	}); err != nil {
		t.Fatal(err)
	}
	report, err := svc.GenerateDailyReport(b.ID, "2026-08-25")
	if err != nil {
		t.Fatal(err)
	}
	if report.AlertCount != 2 {
		t.Fatalf("告警次数应为 2（排烟突升 + 氧异常）: %d", report.AlertCount)
	}
}
