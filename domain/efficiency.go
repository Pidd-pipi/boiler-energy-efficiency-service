package domain

import (
	"time"

	"example.com/boiler-energy-efficiency-service/config"
)

// EfficiencyRecord 能效记录实体（正平衡法计算结果）。
type EfficiencyRecord struct {
	ID                   string    `json:"id"`
	BoilerID             string    `json:"boiler_id"`
	RunDataID            string    `json:"run_data_id"`
	Efficiency           float64   `json:"efficiency"`             // 热效率 %
	UnitCoalConsumption  float64   `json:"unit_coal_consumption"`  // 单位煤耗 kg 燃料 / t 产出
	ExcessAirCoefficient float64   `json:"excess_air_coefficient"` // 过量空气系数
	InputHeatKJ          float64   `json:"input_heat_kj"`          // 输入热量 kJ/h
	UsefulHeatKJ         float64   `json:"useful_heat_kj"`         // 有效利用热量 kJ/h
	SteamOutput          float64   `json:"steam_output"`           // 产出量 t/h（用于日报聚合）
	FuelAmount           float64   `json:"fuel_amount"`            // 燃料量 kg/h（用于日报聚合）
	IntervalMinutes      float64   `json:"interval_minutes"`       // 采样周期（分钟，用于日报聚合）
	Timestamp            time.Time `json:"timestamp"`
	CreatedAt            time.Time `json:"created_at"`
}

// RejectionReason 描述能效计算被拒绝的原因（计算输入缺失时）。
type RejectionReason string

const (
	// RejectMissingFuel 缺少燃料量。
	RejectMissingFuel RejectionReason = "missing_fuel"
	// RejectMissingOutput 缺少产出量（蒸汽量或循环水量）。
	RejectMissingOutput RejectionReason = "missing_output"
	// RejectInvalidValue 输入数值非法（负数等）。
	RejectInvalidValue RejectionReason = "invalid_value"
)

// String 返回拒绝原因的中文说明。
func (r RejectionReason) String() string {
	switch r {
	case RejectMissingFuel:
		return "燃料量缺失"
	case RejectMissingOutput:
		return "产出量缺失（蒸汽量/循环水量）"
	case RejectInvalidValue:
		return "输入数值非法"
	}
	return string(r)
}

// CalculateEfficiency 按正平衡法计算锅炉热效率与单位煤耗。
//
//	蒸汽锅炉: 有效利用热量 = 蒸汽量 × (蒸汽焓 - 给水焓)
//	热水锅炉: 有效利用热量 = 循环水量 × 比热 × (出水温度 - 回水温度)
//	输入热量  = 燃料量 × 燃料低位发热量
//	热效率    = 有效利用热量 / 输入热量 × 100%
//	单位煤耗  = 燃料量 / 产出量
//
// 计算输入缺失（缺燃料量或产出量）时返回 ErrCalculationRejected 并携带拒绝原因。
func CalculateEfficiency(cfg *config.Config, boiler *Boiler, rd *RunData) (*EfficiencyRecord, RejectionReason, error) {
	if rd.FuelAmount <= 0 {
		return nil, RejectMissingFuel, NewError(KindCalculationRejected, "能效计算被拒绝：缺少燃料量")
	}
	if rd.FuelAmount < 0 || rd.SteamOutput < 0 || rd.FeedWaterFlow < 0 {
		return nil, RejectInvalidValue, NewError(KindCalculationRejected, "能效计算被拒绝：存在负数输入")
	}

	var outputPerHour float64 // t/h
	var usefulHeat float64    // kJ/h
	switch boiler.Type {
	case BoilerTypeSteam:
		if rd.SteamOutput <= 0 {
			return nil, RejectMissingOutput, NewError(KindCalculationRejected, "能效计算被拒绝：蒸汽锅炉缺少蒸汽量")
		}
		outputPerHour = rd.SteamOutput
		usefulHeat = rd.SteamOutput * 1000 * (cfg.SteamEnthalpyKJPerKg - cfg.FeedWaterEnthalpyKJPerKg)
	case BoilerTypeHotWater:
		if rd.FeedWaterFlow <= 0 {
			return nil, RejectMissingOutput, NewError(KindCalculationRejected, "能效计算被拒绝：热水锅炉缺少循环水量")
		}
		supply := rd.SupplyWaterTemp
		if supply <= 0 {
			supply = cfg.HotWaterSupplyTempC
		}
		ret := rd.ReturnWaterTemp
		if ret <= 0 {
			ret = cfg.HotWaterReturnTempC
		}
		if supply <= ret {
			return nil, RejectInvalidValue, NewError(KindCalculationRejected, "能效计算被拒绝：出水温度必须高于回水温度")
		}
		outputPerHour = rd.FeedWaterFlow
		usefulHeat = rd.FeedWaterFlow * 1000 * cfg.WaterSpecificHeatKJPerKgC * (supply - ret)
	default:
		return nil, RejectInvalidValue, NewError(KindCalculationRejected, "能效计算被拒绝：未知锅炉类型 %q", boiler.Type)
	}

	inputHeat := rd.FuelAmount * cfg.CoalLowerHeatingValueKJPerKg
	if inputHeat <= 0 {
		return nil, RejectInvalidValue, NewError(KindCalculationRejected, "能效计算被拒绝：输入热量非正")
	}

	efficiency := usefulHeat / inputHeat * 100.0
	unitConsumption := 0.0
	if outputPerHour > 0 {
		unitConsumption = rd.FuelAmount / outputPerHour // kg 燃料 / t 产出
	}
	excessAir := 0.0
	if rd.OxygenContent > 0 && rd.OxygenContent < 21 {
		if coef, err := ExcessAirCoefficient(rd.OxygenContent); err == nil {
			excessAir = coef
		}
	}

	return &EfficiencyRecord{
		BoilerID:             boiler.ID,
		RunDataID:            rd.ID,
		Efficiency:           round2(efficiency),
		UnitCoalConsumption:  round2(unitConsumption),
		ExcessAirCoefficient: round2(excessAir),
		InputHeatKJ:          round2(inputHeat),
		UsefulHeatKJ:         round2(usefulHeat),
		SteamOutput:          round2(outputPerHour),
		FuelAmount:           round2(rd.FuelAmount),
		IntervalMinutes:      rd.IntervalMinutes,
		Timestamp:            rd.Timestamp,
		CreatedAt:            rd.Timestamp,
	}, "", nil
}

// round2 保留两位小数，保证输出稳定可断言。
func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100.0
}
