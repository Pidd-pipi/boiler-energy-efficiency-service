package domain

import (
	"testing"
	"time"

	"example.com/boiler-energy-efficiency-service/config"
)

func TestBlowdownIntervalHours(t *testing.T) {
	cfg := config.Default()
	cfg.BlowdownBaseIntervalHours = 48
	cfg.BlowdownReferenceHardness = 2
	cfg.BlowdownMinIntervalHours = 8
	cfg.BlowdownMaxIntervalHours = 720

	// 硬度 2 -> 48h；硬度 4 -> 24h；硬度 1 -> 96h
	if got := BlowdownIntervalHours(cfg, 2); got != 48 {
		t.Fatalf("硬度2: got=%v want=48", got)
	}
	if got := BlowdownIntervalHours(cfg, 4); got != 24 {
		t.Fatalf("硬度4: got=%v want=24", got)
	}
	if got := BlowdownIntervalHours(cfg, 1); got != 96 {
		t.Fatalf("硬度1: got=%v want=96", got)
	}
	// 上限与下限裁剪。
	cfg.BlowdownMaxIntervalHours = 100
	if got := BlowdownIntervalHours(cfg, 0.5); got != 100 {
		t.Fatalf("超上限: got=%v want=100", got)
	}
	if got := BlowdownIntervalHours(cfg, 100); got != cfg.BlowdownMinIntervalHours {
		t.Fatalf("超下限: got=%v want=%v", got, cfg.BlowdownMinIntervalHours)
	}
}

func TestBuildBlowdownPlan(t *testing.T) {
	cfg := config.Default()
	cfg.BlowdownBaseIntervalHours = 48
	cfg.BlowdownReferenceHardness = 2
	cfg.BlowdownMissFactor = 2

	now := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	b := NewBoiler("1#锅炉", BoilerTypeSteam, 10, 2, now)
	last := time.Time{}

	// 累计运行 20h < 48h：未到期。
	b.AddRunSeconds(20 * 3600)
	plan := BuildBlowdownPlan(cfg, b, last, now)
	if plan.Due || plan.NeedsAttention {
		t.Fatalf("20h 不应到期: %+v", plan)
	}

	// 累计运行 50h >= 48h：到期但未需关注。
	b.AddRunSeconds(30 * 3600)
	plan = BuildBlowdownPlan(cfg, b, last, now)
	if !plan.Due {
		t.Fatal("50h 应到期")
	}
	if plan.NeedsAttention {
		t.Fatal("50h 未超 2 倍周期，不应需关注")
	}

	// 累计运行 100h >= 96h：需关注。
	b.AddRunSeconds(50 * 3600)
	plan = BuildBlowdownPlan(cfg, b, last, now)
	if !plan.NeedsAttention {
		t.Fatal("100h 应需关注")
	}
}
