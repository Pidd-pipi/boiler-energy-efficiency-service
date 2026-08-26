package service

import (
	"fmt"
	"time"

	"example.com/boiler-energy-efficiency-service/domain"
)

// RunIngestInput 运行数据上报入参。
type RunIngestInput struct {
	FuelAmount      float64   `json:"fuel_amount"`       // 燃料量 kg/h
	SteamOutput     float64   `json:"steam_output"`      // 蒸汽量 t/h
	FeedWaterFlow   float64   `json:"feed_water_flow"`   // 给水/循环水流量 t/h
	FlueGasTemp     float64   `json:"flue_gas_temp"`     // 排烟温度 ℃
	OxygenContent   float64   `json:"oxygen_content"`    // 氧含量 %
	SteamPressure   float64   `json:"steam_pressure"`    // 蒸汽压力 MPa
	WaterLevel      float64   `json:"water_level"`       // 水位 %
	SupplyWaterTemp float64   `json:"supply_water_temp"` // 热水出水温度 ℃
	ReturnWaterTemp float64   `json:"return_water_temp"` // 热水回水温度 ℃
	IntervalMinutes float64   `json:"interval_minutes"`  // 采样周期分钟，0 使用配置默认
	Timestamp       time.Time `json:"-"`
	Operator        string    `json:"operator"`
}

// RunIngestResult 运行数据上报结果（能效 + 诊断 + 告警联动产物）。
type RunIngestResult struct {
	RunData            *domain.RunData          `json:"run_data"`
	Efficiency         *domain.EfficiencyRecord `json:"efficiency"`
	EfficiencyRejected bool                     `json:"efficiency_rejected"`
	RejectReason       string                   `json:"reject_reason,omitempty"`
	Combustion         *domain.CombustionStatus `json:"combustion"`
	Alerts             []*domain.RunAlert       `json:"alerts"`
}

// IngestRunData 运行数据上报主链路：
//
//	采集 -> 能效计算（正平衡） -> 燃烧工况诊断 -> 异常告警判定 -> 运行时长累计。
//
// 能效计算输入缺失时仅拒绝生成能效记录（不影响采集与诊断）。
func (s *Services) IngestRunData(traceID, boilerID string, in RunIngestInput) (*RunIngestResult, error) {
	b, err := s.Store.GetBoiler(boilerID)
	if err != nil {
		return nil, err
	}

	ts := in.Timestamp
	if ts.IsZero() {
		ts = s.now()
	}
	interval := in.IntervalMinutes
	if interval <= 0 {
		interval = s.Cfg.DefaultSampleIntervalMinutes
	}

	rd := domain.NewRunData(boilerID, ts)
	rd.FuelAmount = in.FuelAmount
	rd.SteamOutput = in.SteamOutput
	rd.FeedWaterFlow = in.FeedWaterFlow
	rd.FlueGasTemp = in.FlueGasTemp
	rd.OxygenContent = in.OxygenContent
	rd.SteamPressure = in.SteamPressure
	rd.WaterLevel = in.WaterLevel
	rd.SupplyWaterTemp = in.SupplyWaterTemp
	rd.ReturnWaterTemp = in.ReturnWaterTemp
	rd.IntervalMinutes = interval
	if err := s.Store.CreateRunData(rd); err != nil {
		return nil, err
	}

	result := &RunIngestResult{RunData: rd, Alerts: []*domain.RunAlert{}}

	// 1) 能效计算（正平衡法）。
	eff, reject, err := domain.CalculateEfficiency(s.Cfg, b, rd)
	if err != nil {
		if domain.IsKind(err, domain.KindCalculationRejected) {
			result.EfficiencyRejected = true
			result.RejectReason = reject.String()
		} else {
			return nil, err
		}
	} else {
		eff.BoilerID = boilerID
		if err := s.Store.CreateEfficiency(eff); err != nil {
			return nil, err
		}
		result.Efficiency = eff
	}

	// 2) 燃烧工况诊断。
	if rd.OxygenContent >= 0 && rd.OxygenContent < 21 {
		cmb, err := domain.DiagnoseCombustion(s.Cfg, rd)
		if err != nil {
			return nil, err
		}
		if err := s.Store.CreateCombustion(cmb); err != nil {
			return nil, err
		}
		result.Combustion = cmb
	}

	// 3) 异常告警判定。
	alerts, err := s.detectAlerts(traceID, b, rd)
	if err != nil {
		return nil, err
	}
	result.Alerts = alerts

	// 4) 运行时长累计（仅运行状态；排污周期依据）。
	if b.Status == domain.BoilerStatusRunning {
		b.AddRunSeconds(interval * 60.0)
		b.TouchLastRun(ts)
		if err := s.Store.UpdateBoiler(b); err != nil {
			return nil, err
		}
	}

	_ = s.Audit(traceID, domain.ActionRunIngest, "rundata", rd.ID, in.Operator,
		fmt.Sprintf("锅炉 %s 上报运行数据：燃料 %v kg/h、产汽 %v t/h、排烟 %v℃、氧 %v%%",
			b.Name, rd.FuelAmount, rd.SteamOutput, rd.FlueGasTemp, rd.OxygenContent))

	return result, nil
}

// detectAlerts 依据运行数据与历史基线生成告警。
// 规则：排烟温度相对基线突升超阈值、氧含量越限/突跳、压力过高、水位过低。
// 同一锅炉同一类型未处置告警已存在时不再重复生成（防刷屏）。
func (s *Services) detectAlerts(traceID string, b *domain.Boiler, rd *domain.RunData) ([]*domain.RunAlert, error) {
	var out []*domain.RunAlert
	now := s.now()

	create := func(typ domain.AlertType, level domain.AlertLevel, message string, value, baseline float64) error {
		// 同类型未处置告警去重。
		if existing, err := s.Store.OpenAlertOfType(b.ID, typ); err == nil {
			_ = existing
			return nil
		}
		a := domain.NewRunAlert(b.ID, typ, level, message, value, baseline, now)
		if err := s.Store.CreateAlert(a); err != nil {
			return err
		}
		out = append(out, a)
		_ = s.Audit(traceID, domain.ActionRunIngest, "alert", a.ID, "",
			fmt.Sprintf("生成告警 %s：%s", a.Type.Label(), a.Message))
		return nil
	}

	// 排烟温度突升（相对最近基线均值）。
	if rd.FlueGasTemp > 0 {
		recent, _ := s.Store.RecentRunDataByBoiler(b.ID, s.Cfg.FlueTempBaselineWindow)
		var sum float64
		var n int
		for _, prev := range recent {
			if prev.ID == rd.ID || prev.FlueGasTemp <= 0 {
				continue
			}
			sum += prev.FlueGasTemp
			n++
		}
		if n > 0 {
			baseline := sum / float64(n)
			delta := rd.FlueGasTemp - baseline
			if delta > s.Cfg.FlueTempSpikeDelta {
				level := domain.LevelWarning
				if delta > s.Cfg.FlueTempCriticalDelta {
					level = domain.LevelCritical
				}
				msg := fmt.Sprintf("排烟温度突升 %.1f℃（当前 %.1f℃，基线 %.1f℃），请检查受热面结焦或燃烧异常",
					delta, rd.FlueGasTemp, baseline)
				if err := create(domain.AlertFlueTempSpike, level, msg, rd.FlueGasTemp, baseline); err != nil {
					return nil, err
				}
			}
		}
	}

	// 氧含量越限或相对上一采样点突跳。
	if rd.OxygenContent >= 0 {
		msg := ""
		switch {
		case rd.OxygenContent < s.Cfg.OxygenLow:
			msg = fmt.Sprintf("氧含量偏低 %.1f%%（低于 %.1f%%），可能缺氧燃烧，请检查配风", rd.OxygenContent, s.Cfg.OxygenLow)
		case rd.OxygenContent > s.Cfg.OxygenHigh:
			msg = fmt.Sprintf("氧含量偏高 %.1f%%（高于 %.1f%%），风量过大，请检查漏风与配风", rd.OxygenContent, s.Cfg.OxygenHigh)
		}
		if msg == "" {
			if prev, err := s.Store.RecentRunDataByBoiler(b.ID, 2); err == nil && len(prev) >= 2 {
				p := prev[len(prev)-2]
				if p.ID != rd.ID && p.OxygenContent >= 0 {
					if d := rd.OxygenContent - p.OxygenContent; d > s.Cfg.OxygenJumpDelta || d < -s.Cfg.OxygenJumpDelta {
						msg = fmt.Sprintf("氧含量突跳 %.1f 个百分点（%.1f%% -> %.1f%%），请检查仪表与燃烧工况",
							d, p.OxygenContent, rd.OxygenContent)
					}
				}
			}
		}
		if msg != "" {
			if err := create(domain.AlertOxygenAbnormal, domain.LevelWarning, msg, rd.OxygenContent, 0); err != nil {
				return nil, err
			}
		}
	}

	// 蒸汽压力过高（蒸汽锅炉）。
	if b.Type == domain.BoilerTypeSteam && rd.SteamPressure > s.Cfg.PressureHighThreshold {
		msg := fmt.Sprintf("蒸汽压力过高 %.2f MPa（阈值 %.2f MPa），请关注安全阀与负荷", rd.SteamPressure, s.Cfg.PressureHighThreshold)
		if err := create(domain.AlertPressureHigh, domain.LevelCritical, msg, rd.SteamPressure, s.Cfg.PressureHighThreshold); err != nil {
			return nil, err
		}
	}

	// 水位过低。
	if rd.WaterLevel > 0 && rd.WaterLevel < s.Cfg.WaterLowThreshold {
		msg := fmt.Sprintf("水位过低 %.1f%%（阈值 %.1f%%），请立即补水并检查给水泵", rd.WaterLevel, s.Cfg.WaterLowThreshold)
		if err := create(domain.AlertWaterLow, domain.LevelCritical, msg, rd.WaterLevel, s.Cfg.WaterLowThreshold); err != nil {
			return nil, err
		}
	}

	return out, nil
}
