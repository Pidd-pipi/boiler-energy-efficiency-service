package domain

import (
	"math"
	"time"

	"example.com/boiler-energy-efficiency-service/config"
)

// BlowdownRecord 排污执行记录实体。
type BlowdownRecord struct {
	ID          string    `json:"id"`
	BoilerID    string    `json:"boiler_id"`
	ExecutedAt  time.Time `json:"executed_at"`
	DurationMin float64   `json:"duration_min"`
	Operator    string    `json:"operator"`
	Note        string    `json:"note"`
	BlowdownNo  int64     `json:"blowdown_no"` // 该锅炉第几次排污
}

// NewBlowdownRecord 构造排污记录。
func NewBlowdownRecord(boilerID, operator, note string, durationMin float64, no int64, now time.Time) *BlowdownRecord {
	return &BlowdownRecord{
		BoilerID:    boilerID,
		ExecutedAt:  now,
		DurationMin: durationMin,
		Operator:    operator,
		Note:        note,
		BlowdownNo:  no,
	}
}

// BlowdownIntervalHours 依据水质硬度计算排污周期（小时）。
//
//	周期 = 基准周期 × (基准硬度 / 实际硬度)，并夹在 [下限, 上限] 区间。
//	硬度越高，水垢沉积越快，排污周期越短。
func BlowdownIntervalHours(cfg *config.Config, hardness float64) float64 {
	if hardness <= 0 {
		return cfg.BlowdownBaseIntervalHours
	}
	hours := cfg.BlowdownBaseIntervalHours * (cfg.BlowdownReferenceHardness / hardness)
	hours = math.Max(hours, cfg.BlowdownMinIntervalHours)
	hours = math.Min(hours, cfg.BlowdownMaxIntervalHours)
	return round2(hours)
}

// BlowdownPlan 排污计划（按累计运行时长与水质硬度提示排污时机）。
type BlowdownPlan struct {
	BoilerID            string    `json:"boiler_id"`
	IntervalHours       float64   `json:"interval_hours"`
	AccumulatedRunHours float64   `json:"accumulated_run_hours"`
	Hardness            float64   `json:"hardness"`
	Due                 bool      `json:"due"`             // 累计运行时长 >= 排污周期
	NeedsAttention      bool      `json:"needs_attention"` // 超 2 倍周期未排污
	RemainingHours      float64   `json:"remaining_hours"`
	NextDueAt           time.Time `json:"next_due_at"`
	LastBlowdownAt      time.Time `json:"last_blowdown_at"`
}

// BuildBlowdownPlan 构建排污计划。
// 规则：累计运行时长超过 BlowdownMissFactor 倍周期未排污 -> 需关注。
func BuildBlowdownPlan(cfg *config.Config, b *Boiler, lastBlowdownAt time.Time, now time.Time) *BlowdownPlan {
	interval := BlowdownIntervalHours(cfg, b.WaterHardness)
	accRun := b.RunSecondsSinceBlowdown / 3600.0
	due := accRun >= interval
	needsAttention := accRun >= interval*cfg.BlowdownMissFactor
	remaining := interval - accRun
	if remaining < 0 {
		remaining = 0
	}
	nextDue := now.Add(time.Duration(remaining * float64(time.Hour)))
	return &BlowdownPlan{
		BoilerID:            b.ID,
		IntervalHours:       interval,
		AccumulatedRunHours: round2(accRun),
		Hardness:            round2(b.WaterHardness),
		Due:                 due,
		NeedsAttention:      needsAttention,
		RemainingHours:      round2(remaining),
		NextDueAt:           nextDue,
		LastBlowdownAt:      lastBlowdownAt,
	}
}
