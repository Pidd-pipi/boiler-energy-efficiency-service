package domain

import (
	"testing"
	"time"
)

func TestBoilerStateMachine_AllowedTransitions(t *testing.T) {
	now := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	b := NewBoiler("测试锅炉", BoilerTypeSteam, 10, 2, now)

	// 停炉 -> 启动中
	if _, ok := b.CanTransitionTo(BoilerStatusStarting); !ok {
		t.Fatal("stopped -> starting 应为合法迁移")
	}
	// 停炉 -> 运行：不允许
	if _, ok := b.CanTransitionTo(BoilerStatusRunning); ok {
		t.Fatal("stopped -> running 应被禁止")
	}
	// 停炉 -> 停炉确认：不允许
	if _, ok := b.CanTransitionTo(BoilerStatusStopped); ok {
		t.Fatal("stopped -> stopped 应被禁止")
	}
}

func TestBoilerStateMachine_RunningCannotStopDirectly(t *testing.T) {
	now := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	b := NewBoiler("1#锅炉", BoilerTypeSteam, 10, 2, now)

	for _, s := range []BoilerStatus{BoilerStatusStarting, BoilerStatusRunning} {
		if err := b.TransitionTo(s, now); err != nil {
			t.Fatalf("迁移到 %s 失败: %v", s, err)
		}
	}
	// 运行中禁止直接进入停炉。
	if err := b.TransitionTo(BoilerStatusStopped, now); err == nil {
		t.Fatal("running -> stopped 必须被状态机拒绝")
	}
	// 必须先压火再停炉。
	if err := b.TransitionTo(BoilerStatusFiringDown, now); err != nil {
		t.Fatalf("running -> firing_down 失败: %v", err)
	}
	if err := b.TransitionTo(BoilerStatusStopped, now); err != nil {
		t.Fatalf("firing_down -> stopped 失败: %v", err)
	}
	if b.Status != BoilerStatusStopped {
		t.Fatalf("期望停炉状态，实际 %s", b.Status)
	}
}

func TestBoilerStateMachine_FullLifecycle(t *testing.T) {
	now := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	b := NewBoiler("2#锅炉", BoilerTypeHotWater, 7, 3, now)

	sequence := []struct {
		to   BoilerStatus
		want bool
	}{
		{BoilerStatusStarting, true},
		{BoilerStatusRunning, true},
		{BoilerStatusFiringDown, true},
		{BoilerStatusRunning, true}, // 压火后可恢复运行
		{BoilerStatusFiringDown, true},
		{BoilerStatusStopped, true},
	}
	for i, step := range sequence {
		if err := b.TransitionTo(step.to, now); (err == nil) != step.want {
			t.Fatalf("步骤 %d 迁移到 %s 期望合法=%v，实际 err=%v", i, step.to, step.want, err)
		}
	}
}

func TestBoiler_AddRunSecondsAndResetBlowdown(t *testing.T) {
	now := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	b := NewBoiler("3#锅炉", BoilerTypeSteam, 10, 2, now)
	b.AddRunSeconds(300)
	b.AddRunSeconds(300)
	if b.RunSecondsTotal != 600 || b.RunSecondsSinceBlowdown != 600 {
		t.Fatalf("运行时长累计错误: total=%v since=%v", b.RunSecondsTotal, b.RunSecondsSinceBlowdown)
	}
	b.ResetBlowdownClock(now)
	if b.RunSecondsSinceBlowdown != 0 || b.RunSecondsTotal != 600 {
		t.Fatalf("排污计时重置错误: total=%v since=%v", b.RunSecondsTotal, b.RunSecondsSinceBlowdown)
	}
}

func TestBoiler_Validate(t *testing.T) {
	now := time.Now()
	if err := NewBoiler("", BoilerTypeSteam, 10, 2, now).Validate(); err == nil {
		t.Fatal("空名称应校验失败")
	}
	if err := NewBoiler("x", BoilerType("gas"), 10, 2, now).Validate(); err == nil {
		t.Fatal("非法类型应校验失败")
	}
	if err := NewBoiler("x", BoilerTypeSteam, 0, 2, now).Validate(); err == nil {
		t.Fatal("非正额定蒸发量应校验失败")
	}
	if err := NewBoiler("x", BoilerTypeSteam, 10, 2, now).Validate(); err != nil {
		t.Fatalf("合法锅炉应通过校验: %v", err)
	}
}
