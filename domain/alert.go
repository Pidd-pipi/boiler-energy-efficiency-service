package domain

import (
	"fmt"
	"time"
)

// AlertType 运行告警类型枚举（前端 web/enums.js 与后端保持一致）。
type AlertType string

const (
	// AlertFlueTempSpike 排烟温度突升。
	AlertFlueTempSpike AlertType = "flue_temp_spike"
	// AlertOxygenAbnormal 氧含量异常。
	AlertOxygenAbnormal AlertType = "oxygen_abnormal"
	// AlertPressureHigh 压力过高。
	AlertPressureHigh AlertType = "pressure_high"
	// AlertWaterLow 水位过低。
	AlertWaterLow AlertType = "water_low"
)

// AllAlertTypes 返回全部告警类型。
func AllAlertTypes() []AlertType {
	return []AlertType{
		AlertFlueTempSpike,
		AlertOxygenAbnormal,
		AlertPressureHigh,
		AlertWaterLow,
	}
}

// Valid 校验告警类型是否合法。
func (t AlertType) Valid() bool {
	switch t {
	case AlertFlueTempSpike, AlertOxygenAbnormal, AlertPressureHigh, AlertWaterLow:
		return true
	}
	return false
}

// Label 返回告警类型的中文展示名。
func (t AlertType) Label() string {
	switch t {
	case AlertFlueTempSpike:
		return "排烟温度突升"
	case AlertOxygenAbnormal:
		return "氧含量异常"
	case AlertPressureHigh:
		return "压力过高"
	case AlertWaterLow:
		return "水位过低"
	}
	return string(t)
}

// AlertLevel 告警级别。
type AlertLevel string

const (
	// LevelWarning 一般告警。
	LevelWarning AlertLevel = "warning"
	// LevelCritical 严重告警。
	LevelCritical AlertLevel = "critical"
)

// Valid 校验告警级别。
func (l AlertLevel) Valid() bool {
	switch l {
	case LevelWarning, LevelCritical:
		return true
	}
	return false
}

// Label 返回告警级别中文名。
func (l AlertLevel) Label() string {
	switch l {
	case LevelWarning:
		return "一般"
	case LevelCritical:
		return "严重"
	}
	return string(l)
}

// AlertStatus 告警状态。
type AlertStatus string

const (
	// AlertOpen 未确认。
	AlertOpen AlertStatus = "open"
	// AlertAcknowledged 已确认。
	AlertAcknowledged AlertStatus = "acknowledged"
	// AlertEscalated 已升级。
	AlertEscalated AlertStatus = "escalated"
	// AlertResolved 已处置关闭。
	AlertResolved AlertStatus = "resolved"
)

// Valid 校验告警状态。
func (s AlertStatus) Valid() bool {
	switch s {
	case AlertOpen, AlertAcknowledged, AlertEscalated, AlertResolved:
		return true
	}
	return false
}

// Label 返回告警状态中文名。
func (s AlertStatus) Label() string {
	switch s {
	case AlertOpen:
		return "待确认"
	case AlertAcknowledged:
		return "已确认"
	case AlertEscalated:
		return "已升级"
	case AlertResolved:
		return "已处置"
	}
	return string(s)
}

// RunAlert 运行告警实体。
type RunAlert struct {
	ID          string      `json:"id"`
	BoilerID    string      `json:"boiler_id"`
	Type        AlertType   `json:"type"`
	Level       AlertLevel  `json:"level"`
	Status      AlertStatus `json:"status"`
	Message     string      `json:"message"`
	Value       float64     `json:"value"`
	Baseline    float64     `json:"baseline"`
	ConfirmBy   string      `json:"confirm_by"`
	ConfirmNote string      `json:"confirm_note"`
	CreatedAt   time.Time   `json:"created_at"`
	ConfirmedAt time.Time   `json:"confirmed_at"`
	EscalatedAt time.Time   `json:"escalated_at"`
}

// NewRunAlert 构造新告警，初始状态为待确认。
func NewRunAlert(boilerID string, typ AlertType, level AlertLevel, message string, value, baseline float64, now time.Time) *RunAlert {
	return &RunAlert{
		BoilerID:  boilerID,
		Type:      typ,
		Level:     level,
		Status:    AlertOpen,
		Message:   message,
		Value:     value,
		Baseline:  baseline,
		CreatedAt: now,
	}
}

// IsOpen 是否处于待确认/未处置状态。
func (a *RunAlert) IsOpen() bool {
	return a.Status == AlertOpen
}

// Acknowledge 确认告警（仅待确认状态可确认；已升级告警仍可确认并转为已确认）。
func (a *RunAlert) Acknowledge(operator, note string, now time.Time) error {
	if a.Status == AlertResolved {
		return NewError(KindConflict, "告警 %s 已处置，不能重复确认", a.ID)
	}
	a.Status = AlertAcknowledged
	a.ConfirmBy = operator
	a.ConfirmNote = note
	a.ConfirmedAt = now
	return nil
}

// Escalate 升级告警：级别提升为严重，状态置为已升级。
func (a *RunAlert) Escalate(now time.Time) error {
	if a.Status == AlertResolved {
		return NewError(KindConflict, "告警 %s 已处置，不能升级", a.ID)
	}
	if a.Status == AlertEscalated {
		return NewError(KindConflict, "告警 %s 已升级", a.ID)
	}
	a.Status = AlertEscalated
	a.Level = LevelCritical
	a.EscalatedAt = now
	return nil
}

// Resolve 处置关闭告警。
func (a *RunAlert) Resolve(operator, note string, now time.Time) error {
	if a.Status == AlertResolved {
		return NewError(KindConflict, "告警 %s 已处置", a.ID)
	}
	a.Status = AlertResolved
	a.ConfirmBy = operator
	a.ConfirmNote = note
	a.ConfirmedAt = now
	return nil
}

// IsDueEscalation 判断待确认告警是否超过升级时限。
func (a *RunAlert) IsDueEscalation(now time.Time, after time.Duration) bool {
	if a.Status != AlertOpen {
		return false
	}
	return now.Sub(a.CreatedAt) >= after
}

// Summary 返回告警摘要文本。
func (a *RunAlert) Summary() string {
	return fmt.Sprintf("[%s/%s] %s", a.Type.Label(), a.Level.Label(), a.Message)
}
