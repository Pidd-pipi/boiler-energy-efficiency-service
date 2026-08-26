package domain

import (
	"testing"
	"time"

	"example.com/boiler-energy-efficiency-service/config"
)

func newSteamBoiler() *Boiler {
	return NewBoiler("蒸汽锅炉", BoilerTypeSteam, 10, 2, time.Now())
}

func newHotWaterBoiler() *Boiler {
	return NewBoiler("热水锅炉", BoilerTypeHotWater, 7, 3, time.Now())
}

func TestCalculateEfficiency_Steam(t *testing.T) {
	cfg := config.Default()
	rd := NewRunData("blr_000001", time.Now())
	rd.FuelAmount = 900  // kg/h
	rd.SteamOutput = 8.5 // t/h
	rd.OxygenContent = 8 // %
	rd.IntervalMinutes = 5

	eff, reject, err := CalculateEfficiency(cfg, newSteamBoiler(), rd)
	if err != nil {
		t.Fatalf("计算失败: %v", err)
	}
	if reject != "" {
		t.Fatalf("不应拒绝: %s", reject)
	}
	// 理论验证：有效热量 = 8.5*1000*(2760-293) kJ/h；输入 = 900*20934 kJ/h
	want := 8.5 * 1000 * (cfg.SteamEnthalpyKJPerKg - cfg.FeedWaterEnthalpyKJPerKg) / (900 * cfg.CoalLowerHeatingValueKJPerKg) * 100
	if diff := eff.Efficiency - want; diff > 0.01 || diff < -0.01 {
		t.Fatalf("热效率计算错误: got=%v want=%v", eff.Efficiency, want)
	}
	// 单位煤耗 = 900/8.5 kg/t
	if wantUC := 900.0 / 8.5; eff.UnitCoalConsumption != round2(wantUC) {
		t.Fatalf("单位煤耗错误: got=%v want=%v", eff.UnitCoalConsumption, round2(wantUC))
	}
	// 过量空气系数 = 21/(21-8)
	if wantCoef := round2(21.0 / 13.0); eff.ExcessAirCoefficient != wantCoef {
		t.Fatalf("过量空气系数错误: got=%v want=%v", eff.ExcessAirCoefficient, wantCoef)
	}
}

func TestCalculateEfficiency_MissingFuelRejected(t *testing.T) {
	cfg := config.Default()
	rd := NewRunData("blr_000001", time.Now())
	rd.FuelAmount = 0
	rd.SteamOutput = 8.5

	_, reject, err := CalculateEfficiency(cfg, newSteamBoiler(), rd)
	if err == nil {
		t.Fatal("缺少燃料量应拒绝")
	}
	if !IsKind(err, KindCalculationRejected) {
		t.Fatalf("应为计算拒绝错误: %v", err)
	}
	if reject != RejectMissingFuel {
		t.Fatalf("拒绝原因错误: %s", reject)
	}
}

func TestCalculateEfficiency_MissingOutputRejected(t *testing.T) {
	cfg := config.Default()
	rd := NewRunData("blr_000001", time.Now())
	rd.FuelAmount = 900
	rd.SteamOutput = 0

	_, reject, err := CalculateEfficiency(cfg, newSteamBoiler(), rd)
	if err == nil {
		t.Fatal("蒸汽锅炉缺少蒸汽量应拒绝")
	}
	if reject != RejectMissingOutput {
		t.Fatalf("拒绝原因错误: %s", reject)
	}
}

func TestCalculateEfficiency_HotWater(t *testing.T) {
	cfg := config.Default()
	rd := NewRunData("blr_000002", time.Now())
	rd.FuelAmount = 620
	rd.FeedWaterFlow = 130
	rd.SupplyWaterTemp = 82
	rd.ReturnWaterTemp = 58
	rd.OxygenContent = 8

	eff, _, err := CalculateEfficiency(cfg, newHotWaterBoiler(), rd)
	if err != nil {
		t.Fatalf("热水锅炉计算失败: %v", err)
	}
	want := 130 * 1000 * cfg.WaterSpecificHeatKJPerKgC * (82 - 58) / (620 * cfg.CoalLowerHeatingValueKJPerKg) * 100
	if diff := eff.Efficiency - want; diff > 0.01 || diff < -0.01 {
		t.Fatalf("热水锅炉热效率错误: got=%v want=%v", eff.Efficiency, want)
	}
	// 热水锅炉缺循环水量应拒绝。
	rd2 := NewRunData("blr_000002", time.Now())
	rd2.FuelAmount = 620
	rd2.FeedWaterFlow = 0
	if _, _, err := CalculateEfficiency(cfg, newHotWaterBoiler(), rd2); err == nil {
		t.Fatal("热水锅炉缺少循环水量应拒绝")
	}
}
