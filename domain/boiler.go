package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// BoilerType 锅炉类型。
type BoilerType string

const (
	// BoilerTypeSteam 蒸汽锅炉。
	BoilerTypeSteam BoilerType = "steam"
	// BoilerTypeHotWater 热水锅炉。
	BoilerTypeHotWater BoilerType = "hot_water"
)

// Valid 校验锅炉类型是否合法。
func (t BoilerType) Valid() bool {
	switch t {
	case BoilerTypeSteam, BoilerTypeHotWater:
		return true
	}
	return false
}

// Label 返回锅炉类型的中文展示名。
func (t BoilerType) Label() string {
	switch t {
	case BoilerTypeSteam:
		return "蒸汽锅炉"
	case BoilerTypeHotWater:
		return "热水锅炉"
	}
	return string(t)
}

// BoilerStatus 锅炉运行状态枚举（前端 web/enums.js 与后端保持一致）。
type BoilerStatus string

const (
	// BoilerStatusStopped 停炉。
	BoilerStatusStopped BoilerStatus = "stopped"
	// BoilerStatusStarting 启动中。
	BoilerStatusStarting BoilerStatus = "starting"
	// BoilerStatusRunning 运行中。
	BoilerStatusRunning BoilerStatus = "running"
	// BoilerStatusFiringDown 压火。
	BoilerStatusFiringDown BoilerStatus = "firing_down"
)

// AllBoilerStatuses 返回全部状态枚举，用于测试与前端枚举对齐。
func AllBoilerStatuses() []BoilerStatus {
	return []BoilerStatus{
		BoilerStatusStopped,
		BoilerStatusStarting,
		BoilerStatusRunning,
		BoilerStatusFiringDown,
	}
}

// Valid 校验状态是否合法。
func (s BoilerStatus) Valid() bool {
	switch s {
	case BoilerStatusStopped, BoilerStatusStarting, BoilerStatusRunning, BoilerStatusFiringDown:
		return true
	}
	return false
}

// Label 返回状态的中文展示名。
func (s BoilerStatus) Label() string {
	switch s {
	case BoilerStatusStopped:
		return "停炉"
	case BoilerStatusStarting:
		return "启动中"
	case BoilerStatusRunning:
		return "运行中"
	case BoilerStatusFiringDown:
		return "压火"
	}
	return string(s)
}

// boilerTransitionTable 状态机迁移表。
// 核心规则：运行中禁止直接进入停炉，必须先压火（firing_down）。
var boilerTransitionTable = map[BoilerStatus]map[BoilerStatus]string{
	BoilerStatusStopped: {
		BoilerStatusStarting: "点火启动",
	},
	BoilerStatusStarting: {
		BoilerStatusRunning: "启动完成进入运行",
		BoilerStatusStopped: "启动失败终止",
	},
	BoilerStatusRunning: {
		BoilerStatusFiringDown: "压火待停",
	},
	BoilerStatusFiringDown: {
		BoilerStatusStopped: "确认停炉",
		BoilerStatusRunning: "恢复运行",
	},
}

// AllowedTransitions 返回全部允许的迁移（源状态 -> 目标状态列表）。
func AllowedTransitions() map[BoilerStatus][]BoilerStatus {
	out := make(map[BoilerStatus][]BoilerStatus, len(boilerTransitionTable))
	for from, targets := range boilerTransitionTable {
		list := make([]BoilerStatus, 0, len(targets))
		for to := range targets {
			list = append(list, to)
		}
		sort.Slice(list, func(i, j int) bool { return list[i] < list[j] })
		out[from] = list
	}
	return out
}

// TransitionReason 返回 from -> to 是否允许及迁移说明。
func TransitionReason(from, to BoilerStatus) (string, bool) {
	targets, ok := boilerTransitionTable[from]
	if !ok {
		return "", false
	}
	reason, ok := targets[to]
	return reason, ok
}

// Boiler 锅炉台账实体。
type Boiler struct {
	ID                      string       `json:"id"`
	Name                    string       `json:"name"`
	Type                    BoilerType   `json:"type"`
	RatedCapacity           float64      `json:"rated_capacity"` // 额定蒸发量 t/h
	WaterHardness           float64      `json:"water_hardness"` // 水质硬度 mmol/L
	Status                  BoilerStatus `json:"status"`
	RunSecondsTotal         float64      `json:"run_seconds_total"`          // 累计运行秒数
	RunSecondsSinceBlowdown float64      `json:"run_seconds_since_blowdown"` // 距上次排污累计运行秒数
	LastRunAt               time.Time    `json:"last_run_at"`
	CreatedAt               time.Time    `json:"created_at"`
	UpdatedAt               time.Time    `json:"updated_at"`
}

// NewBoiler 构造新锅炉，默认状态为停炉。
func NewBoiler(name string, typ BoilerType, ratedCapacity, hardness float64, now time.Time) *Boiler {
	return &Boiler{
		Name:                    name,
		Type:                    typ,
		RatedCapacity:           ratedCapacity,
		WaterHardness:           hardness,
		Status:                  BoilerStatusStopped,
		RunSecondsTotal:         0,
		RunSecondsSinceBlowdown: 0,
		CreatedAt:               now,
		UpdatedAt:               now,
	}
}

// CanTransitionTo 判断能否迁移到目标状态，返回迁移原因与是否允许。
func (b *Boiler) CanTransitionTo(target BoilerStatus) (string, bool) {
	return TransitionReason(b.Status, target)
}

// TransitionTo 执行状态迁移，不合法时返回 KindStateTransition 错误。
func (b *Boiler) TransitionTo(target BoilerStatus, now time.Time) error {
	if !target.Valid() {
		return NewError(KindInvalidInput, "非法目标状态: %q", target)
	}
	reason, ok := TransitionReason(b.Status, target)
	if !ok {
		return NewError(
			KindStateTransition,
			"禁止状态迁移 %s -> %s（运行中禁止直接停炉，必须先压火）",
			b.Status, target,
		)
	}
	b.Status = target
	b.UpdatedAt = now
	_ = reason
	return nil
}

// AddRunSeconds 累计运行时长（排污周期与运行统计依赖该字段）。
func (b *Boiler) AddRunSeconds(sec float64) {
	if sec <= 0 {
		return
	}
	b.RunSecondsTotal += sec
	b.RunSecondsSinceBlowdown += sec
	b.UpdatedAt = time.Now()
}

// ResetBlowdownClock 排污执行后重置距上次排污的运行计时。
func (b *Boiler) ResetBlowdownClock(now time.Time) {
	b.RunSecondsSinceBlowdown = 0
	b.UpdatedAt = now
}

// TouchLastRun 更新最近运行时间。
func (b *Boiler) TouchLastRun(now time.Time) {
	b.LastRunAt = now
	b.UpdatedAt = now
}

// Validate 校验锅炉基础字段。
func (b *Boiler) Validate() error {
	if strings.TrimSpace(b.Name) == "" {
		return NewError(KindInvalidInput, "锅炉名称不能为空")
	}
	if !b.Type.Valid() {
		return NewError(KindInvalidInput, "锅炉类型必须为 steam 或 hot_water，当前: %q", b.Type)
	}
	if b.RatedCapacity <= 0 {
		return NewError(KindInvalidInput, "额定蒸发量必须大于 0")
	}
	if b.WaterHardness < 0 {
		return NewError(KindInvalidInput, "水质硬度不能为负数")
	}
	return nil
}

// Summary 返回锅炉的简要描述。
func (b *Boiler) Summary() string {
	return fmt.Sprintf("%s(%s %v t/h)", b.Name, b.Type.Label(), b.RatedCapacity)
}
