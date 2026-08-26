package service

import (
	"testing"
	"time"

	"example.com/boiler-energy-efficiency-service/config"
	"example.com/boiler-energy-efficiency-service/domain"
	"example.com/boiler-energy-efficiency-service/store"
)

// newTestServices 构造无持久化的测试服务，使用固定时钟。
func newTestServices(t *testing.T) (*Services, *store.Store) {
	t.Helper()
	cfg := config.Default()
	st, err := store.New(store.Options{})
	if err != nil {
		t.Fatal(err)
	}
	svc := NewServices(cfg, st)
	svc.Now = func() time.Time {
		return time.Date(2026, 8, 25, 10, 0, 0, 0, time.Local)
	}
	return svc, st
}

func mustCreateSteamBoiler(t *testing.T, svc *Services) *domain.Boiler {
	t.Helper()
	b, err := svc.CreateBoiler("test", CreateBoilerInput{
		Name: "1#蒸汽锅炉", Type: domain.BoilerTypeSteam, RatedCapacity: 10, WaterHardness: 2, Operator: "tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	// 迁移到运行。
	for _, s := range []domain.BoilerStatus{domain.BoilerStatusStarting, domain.BoilerStatusRunning} {
		if _, err := svc.Transition("test", b.ID, s, "tester"); err != nil {
			t.Fatal(err)
		}
	}
	return b
}

func TestIngestRunData_FullPipeline(t *testing.T) {
	svc, st := newTestServices(t)
	b := mustCreateSteamBoiler(t, svc)

	res, err := svc.IngestRunData("trace-1", b.ID, RunIngestInput{
		FuelAmount: 900, SteamOutput: 8.5, FeedWaterFlow: 9,
		FlueGasTemp: 152, OxygenContent: 8, SteamPressure: 1.2, WaterLevel: 60,
		IntervalMinutes: 5, Operator: "tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.RunData == nil || res.RunData.ID == "" {
		t.Fatal("应生成运行数据")
	}
	if res.Efficiency == nil {
		t.Fatal("应生成能效记录")
	}
	if res.Efficiency.Efficiency <= 0 {
		t.Fatalf("热效率应为正: %v", res.Efficiency.Efficiency)
	}
	if res.Combustion == nil || res.Combustion.Result != domain.ResultNormal {
		t.Fatalf("诊断结果错误: %+v", res.Combustion)
	}
	if len(res.Alerts) != 0 {
		t.Fatalf("正常数据不应产生告警: %d", len(res.Alerts))
	}
	// 运行时长累计（5 分钟）。
	b2, _ := svc.GetBoiler(b.ID)
	if b2.RunSecondsTotal != 300 {
		t.Fatalf("运行时长累计错误: %v", b2.RunSecondsTotal)
	}
	// 审计留痕。
	audits, _ := svc.ListAudit(store.AuditFilter{Action: domain.ActionRunIngest})
	if len(audits) != 1 {
		t.Fatalf("应留运行数据审计: %d", len(audits))
	}
	_ = st
}

func TestIngestRunData_MissingFuelRejectsEfficiency(t *testing.T) {
	svc, _ := newTestServices(t)
	b := mustCreateSteamBoiler(t, svc)

	res, err := svc.IngestRunData("trace-2", b.ID, RunIngestInput{
		FuelAmount: 0, SteamOutput: 8.5, OxygenContent: 8, IntervalMinutes: 5,
	})
	if err != nil {
		t.Fatalf("缺少燃料量不应阻断采集: %v", err)
	}
	if !res.EfficiencyRejected {
		t.Fatal("缺少燃料量应拒绝能效记录")
	}
	if res.RunData == nil {
		t.Fatal("运行数据仍应入库")
	}
}

func TestIngestRunData_AlertGeneration(t *testing.T) {
	svc, _ := newTestServices(t)
	b := mustCreateSteamBoiler(t, svc)

	// 先上报一条基线（排烟 150℃、氧 8%、压力 1.2、水位 60）。
	if _, err := svc.IngestRunData("t1", b.ID, RunIngestInput{
		FuelAmount: 900, SteamOutput: 8.5, FlueGasTemp: 150, OxygenContent: 8,
		SteamPressure: 1.2, WaterLevel: 60, IntervalMinutes: 5,
	}); err != nil {
		t.Fatal(err)
	}

	// 异常数据：排烟突升到 200（Δ=50>30）、氧 2.5%（<3）、压力 2.0（>1.6）、水位 30（<40）。
	res, err := svc.IngestRunData("t2", b.ID, RunIngestInput{
		FuelAmount: 900, SteamOutput: 8.5, FlueGasTemp: 200, OxygenContent: 2.5,
		SteamPressure: 2.0, WaterLevel: 30, IntervalMinutes: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	types := map[domain.AlertType]bool{}
	for _, a := range res.Alerts {
		types[a.Type] = true
	}
	for _, want := range []domain.AlertType{
		domain.AlertFlueTempSpike, domain.AlertOxygenAbnormal,
		domain.AlertPressureHigh, domain.AlertWaterLow,
	} {
		if !types[want] {
			t.Fatalf("应生成 %s 告警，实际: %v", want, types)
		}
	}
	// 排烟突升 50℃ 应为 warning（<60）。
	for _, a := range res.Alerts {
		if a.Type == domain.AlertFlueTempSpike && a.Level != domain.LevelWarning {
			t.Fatalf("排烟突升 50℃ 应为 warning: %+v", a)
		}
	}
}

func TestIngestRunData_AlertDedup(t *testing.T) {
	svc, _ := newTestServices(t)
	b := mustCreateSteamBoiler(t, svc)

	if _, err := svc.IngestRunData("t1", b.ID, RunIngestInput{
		FuelAmount: 900, SteamOutput: 8.5, FlueGasTemp: 150, OxygenContent: 8, IntervalMinutes: 5,
	}); err != nil {
		t.Fatal(err)
	}
	// 连续两次异常，同类型告警不应重复生成。
	if _, err := svc.IngestRunData("t2", b.ID, RunIngestInput{
		FuelAmount: 900, SteamOutput: 8.5, FlueGasTemp: 200, OxygenContent: 2.5, IntervalMinutes: 5,
	}); err != nil {
		t.Fatal(err)
	}
	res2, err := svc.IngestRunData("t3", b.ID, RunIngestInput{
		FuelAmount: 900, SteamOutput: 8.5, FlueGasTemp: 205, OxygenContent: 2.0, IntervalMinutes: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res2.Alerts) != 0 {
		t.Fatalf("同类型未处置告警不应重复生成: %d", len(res2.Alerts))
	}
}

func TestIngestRunData_StoppedBoilerNotAccumulateRuntime(t *testing.T) {
	svc, _ := newTestServices(t)
	b, err := svc.CreateBoiler("test", CreateBoilerInput{
		Name: "停炉锅炉", Type: domain.BoilerTypeSteam, RatedCapacity: 10, WaterHardness: 2, Operator: "tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	// 停炉状态上报数据不累计运行时长。
	if _, err := svc.IngestRunData("t", b.ID, RunIngestInput{
		FuelAmount: 900, SteamOutput: 8.5, OxygenContent: 8, IntervalMinutes: 5,
	}); err != nil {
		t.Fatal(err)
	}
	b2, _ := svc.GetBoiler(b.ID)
	if b2.RunSecondsTotal != 0 {
		t.Fatalf("停炉状态不应累计运行时长: %v", b2.RunSecondsTotal)
	}
}
