package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	// 清除可能影响的环境变量。
	for _, k := range []string{"PORT", "LISTEN_ADDR", "DATA_DIR", "SEED_DEMO", "EXCESS_AIR_LOW", "SWEEP_INTERVAL", "ALERT_ESCALATION_AFTER"} {
		_ = os.Unsetenv(k)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ListenAddr != "0.0.0.0:8080" {
		t.Fatalf("默认端口错误: %s", cfg.ListenAddr)
	}
	if cfg.ExcessAirLow != 1.2 || cfg.ExcessAirHigh != 1.8 {
		t.Fatalf("默认工况阈值错误: %v %v", cfg.ExcessAirLow, cfg.ExcessAirHigh)
	}
	if cfg.SweepInterval != 10*time.Minute {
		t.Fatalf("默认扫描周期错误: %v", cfg.SweepInterval)
	}
	if cfg.AlertEscalationAfter != time.Hour {
		t.Fatalf("默认升级时限错误: %v", cfg.AlertEscalationAfter)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("PORT", "18020")
	t.Setenv("DATA_DIR", "/tmp/bess-data")
	t.Setenv("SEED_DEMO", "false")
	t.Setenv("EXCESS_AIR_LOW", "1.15")
	t.Setenv("EXCESS_AIR_HIGH", "2.0")
	t.Setenv("SWEEP_INTERVAL", "5m")
	t.Setenv("ALERT_ESCALATION_AFTER", "30m")
	t.Setenv("FLUE_TEMP_SPIKE_DELTA", "25")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ListenAddr != "0.0.0.0:18020" {
		t.Fatalf("PORT 覆盖失败: %s", cfg.ListenAddr)
	}
	if cfg.DataDir != "/tmp/bess-data" {
		t.Fatalf("DATA_DIR 覆盖失败: %s", cfg.DataDir)
	}
	if cfg.SeedDemo {
		t.Fatal("SEED_DEMO=false 未生效")
	}
	if cfg.ExcessAirLow != 1.15 || cfg.ExcessAirHigh != 2.0 {
		t.Fatalf("工况阈值覆盖失败: %v %v", cfg.ExcessAirLow, cfg.ExcessAirHigh)
	}
	if cfg.SweepInterval != 5*time.Minute {
		t.Fatalf("SWEEP_INTERVAL 覆盖失败: %v", cfg.SweepInterval)
	}
	if cfg.AlertEscalationAfter != 30*time.Minute {
		t.Fatalf("升级时限覆盖失败: %v", cfg.AlertEscalationAfter)
	}
	if cfg.FlueTempSpikeDelta != 25 {
		t.Fatalf("排烟突升阈值覆盖失败: %v", cfg.FlueTempSpikeDelta)
	}
}

func TestLoad_InvalidPort(t *testing.T) {
	t.Setenv("PORT", "abc")
	if _, err := Load(); err == nil {
		t.Fatal("非法 PORT 应报错")
	}
}

func TestLoad_InvalidDuration(t *testing.T) {
	t.Setenv("SWEEP_INTERVAL", "not-a-duration")
	if _, err := Load(); err == nil {
		t.Fatal("非法时长应报错")
	}
}
