package service

import (
	"testing"

	"example.com/boiler-energy-efficiency-service/domain"
)

func TestBlowdown_ExecuteResetsClock(t *testing.T) {
	svc, _ := newTestServices(t)
	b := mustCreateSteamBoiler(t, svc)

	// 累计运行 10 条 × 60 分钟 = 10h。
	for i := 0; i < 10; i++ {
		if _, err := svc.IngestRunData("t", b.ID, RunIngestInput{
			FuelAmount: 900, SteamOutput: 8.5, OxygenContent: 8, IntervalMinutes: 60,
		}); err != nil {
			t.Fatal(err)
		}
	}
	b2, _ := svc.GetBoiler(b.ID)
	if b2.RunSecondsSinceBlowdown != 10*3600 {
		t.Fatalf("累计排污计时错误: %v", b2.RunSecondsSinceBlowdown)
	}

	rec, plan, err := svc.ExecuteBlowdown("trace-bd", b.ID, ExecuteBlowdownInput{
		Operator: "司炉工", DurationMin: 8, Note: "定期排污",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.BlowdownNo != 1 {
		t.Fatalf("排污序号错误: %d", rec.BlowdownNo)
	}
	if plan.AccumulatedRunHours != 0 {
		t.Fatalf("排污后累计运行应归零: %v", plan.AccumulatedRunHours)
	}
	b3, _ := svc.GetBoiler(b.ID)
	if b3.RunSecondsSinceBlowdown != 0 {
		t.Fatalf("锅炉排污计时未重置: %v", b3.RunSecondsSinceBlowdown)
	}
	if b3.RunSecondsTotal != 10*3600 {
		t.Fatalf("累计总时长不应被重置: %v", b3.RunSecondsTotal)
	}

	// 再执行一次，序号递增。
	rec2, _, err := svc.ExecuteBlowdown("trace-bd2", b.ID, ExecuteBlowdownInput{Operator: "o"})
	if err != nil {
		t.Fatal(err)
	}
	if rec2.BlowdownNo != 2 {
		t.Fatalf("第二次排污序号错误: %d", rec2.BlowdownNo)
	}
}

func TestBlowdown_PlanNeedsAttention(t *testing.T) {
	svc, _ := newTestServices(t)
	b := mustCreateSteamBoiler(t, svc)
	svc.Cfg.BlowdownBaseIntervalHours = 8
	svc.Cfg.BlowdownReferenceHardness = 2
	svc.Cfg.BlowdownMinIntervalHours = 4
	svc.Cfg.BlowdownMissFactor = 2
	b.WaterHardness = 2 // 周期 8h，超 16h 需关注

	for i := 0; i < 17; i++ {
		if _, err := svc.IngestRunData("t", b.ID, RunIngestInput{
			FuelAmount: 900, SteamOutput: 8.5, OxygenContent: 8, IntervalMinutes: 60,
		}); err != nil {
			t.Fatal(err)
		}
	}
	plan, _, err := svc.GetBlowdownDetail(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Due {
		t.Fatal("17h 应到期")
	}
	if !plan.NeedsAttention {
		t.Fatal("17h 超 2 倍周期(16h)应需关注")
	}
	_ = domain.BoilerStatusRunning
}
