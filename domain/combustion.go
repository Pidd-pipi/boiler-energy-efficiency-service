package domain

import (
	"fmt"
	"time"

	"example.com/boiler-energy-efficiency-service/config"
)

// DiagnosisResult 燃烧工况诊断结论枚举（前端 web/enums.js 与后端保持一致）。
type DiagnosisResult string

const (
	// ResultUnderAir 缺氧燃烧（过量空气系数偏低）。
	ResultUnderAir DiagnosisResult = "under_air"
	// ResultNormal 燃烧正常。
	ResultNormal DiagnosisResult = "normal"
	// ResultExcessAir 过剩空气（过量空气系数偏高）。
	ResultExcessAir DiagnosisResult = "excess_air"
)

// Valid 校验诊断结论是否合法。
func (r DiagnosisResult) Valid() bool {
	switch r {
	case ResultUnderAir, ResultNormal, ResultExcessAir:
		return true
	}
	return false
}

// Label 返回诊断结论的中文展示名。
func (r DiagnosisResult) Label() string {
	switch r {
	case ResultUnderAir:
		return "缺氧燃烧"
	case ResultNormal:
		return "燃烧正常"
	case ResultExcessAir:
		return "过剩空气"
	}
	return string(r)
}

// Suggestion 返回诊断结论对应的调整建议。
func (r DiagnosisResult) Suggestion() string {
	switch r {
	case ResultUnderAir:
		return "风量不足导致缺氧燃烧，建议增大送风量、适当开启风门挡板，并检查炉膛负压与燃料雾化情况，防止不完全燃烧损失。"
	case ResultExcessAir:
		return "过量空气偏大导致排烟热损失增加，建议减小送风量或关小风门，优化配风比，控制过量空气系数在 1.2~1.8 区间。"
	case ResultNormal:
		return "燃烧工况正常，过量空气系数处于合理区间，建议维持当前配风参数并持续监测。"
	}
	return "请结合现场情况调整燃烧参数。"
}

// CombustionStatus 燃烧工况诊断实体。
type CombustionStatus struct {
	ID                   string          `json:"id"`
	BoilerID             string          `json:"boiler_id"`
	RunDataID            string          `json:"run_data_id"`
	OxygenContent        float64         `json:"oxygen_content"`
	ExcessAirCoefficient float64         `json:"excess_air_coefficient"`
	Result               DiagnosisResult `json:"result"`
	Suggestion           string          `json:"suggestion"`
	Timestamp            time.Time       `json:"timestamp"`
	CreatedAt            time.Time       `json:"created_at"`
}

// ExcessAirCoefficient 过量空气系数 = 21 / (21 - 氧含量)。
// 氧含量 >= 21 时返回错误（物理上不可能）。
func ExcessAirCoefficient(oxygen float64) (float64, error) {
	if oxygen < 0 {
		return 0, fmt.Errorf("氧含量不能为负: %v", oxygen)
	}
	if oxygen >= 21 {
		return 0, fmt.Errorf("氧含量必须小于 21%%，当前: %v%%", oxygen)
	}
	return 21.0 / (21.0 - oxygen), nil
}

// DiagnoseCombustion 依据过量空气系数诊断燃烧工况并给出调整建议。
// 低于 ExcessAirLow 判为缺氧，高于 ExcessAirHigh 判为过剩，其余正常。
func DiagnoseCombustion(cfg *config.Config, rd *RunData) (*CombustionStatus, error) {
	coef, err := ExcessAirCoefficient(rd.OxygenContent)
	if err != nil {
		return nil, NewError(KindInvalidInput, "燃烧工况诊断失败：%v", err)
	}
	result := ResultNormal
	switch {
	case coef < cfg.ExcessAirLow:
		result = ResultUnderAir
	case coef > cfg.ExcessAirHigh:
		result = ResultExcessAir
	}
	return &CombustionStatus{
		BoilerID:             rd.BoilerID,
		RunDataID:            rd.ID,
		OxygenContent:        round2(rd.OxygenContent),
		ExcessAirCoefficient: round2(coef),
		Result:               result,
		Suggestion:           result.Suggestion(),
		Timestamp:            rd.Timestamp,
		CreatedAt:            rd.Timestamp,
	}, nil
}
