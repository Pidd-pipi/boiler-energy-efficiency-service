package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"
)

// env helpers 负责把环境变量解析为配置项，解析失败时返回错误，
// 避免静默使用错误参数。

// getString 读取字符串环境变量，为空时返回默认值。
func getString(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// getBool 读取布尔环境变量（true/1/yes）。
func getBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

// getFloat 读取浮点环境变量，解析失败时返回错误。
func getFloat(key string, def float64) (float64, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("环境变量 %s 无法解析为数字: %q", key, v)
	}
	return f, nil
}

// getInt 读取整数环境变量，解析失败时返回错误。
func getInt(key string, def int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("环境变量 %s 无法解析为整数: %q", key, v)
	}
	return n, nil
}

// getDuration 读取时长环境变量（如 1h、30m），解析失败时返回错误。
func getDuration(key string, def time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("环境变量 %s 无法解析为时长: %q", key, v)
	}
	return d, nil
}

// parseLogLevel 解析 LOG_LEVEL 环境变量，默认 info。
func parseLogLevel(v string) (slog.Level, error) {
	switch v {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("环境变量 LOG_LEVEL 非法: %q（支持 debug/info/warn/error）", v)
	}
}

// Load 从环境变量读取全部可覆盖配置项并返回。
// 优先级：环境变量 > 默认值。
func Load() (*Config, error) {
	c := Default()

	var err error
	if c.SteamEnthalpyKJPerKg, err = getFloat("STEAM_ENTHALPY", c.SteamEnthalpyKJPerKg); err != nil {
		return nil, err
	}
	if c.FeedWaterEnthalpyKJPerKg, err = getFloat("FEED_WATER_ENTHALPY", c.FeedWaterEnthalpyKJPerKg); err != nil {
		return nil, err
	}
	if c.CoalLowerHeatingValueKJPerKg, err = getFloat("COAL_LHV", c.CoalLowerHeatingValueKJPerKg); err != nil {
		return nil, err
	}
	if c.WaterSpecificHeatKJPerKgC, err = getFloat("WATER_SPECIFIC_HEAT", c.WaterSpecificHeatKJPerKgC); err != nil {
		return nil, err
	}
	if c.HotWaterSupplyTempC, err = getFloat("HOT_WATER_SUPPLY_TEMP", c.HotWaterSupplyTempC); err != nil {
		return nil, err
	}
	if c.HotWaterReturnTempC, err = getFloat("HOT_WATER_RETURN_TEMP", c.HotWaterReturnTempC); err != nil {
		return nil, err
	}
	if c.ExcessAirLow, err = getFloat("EXCESS_AIR_LOW", c.ExcessAirLow); err != nil {
		return nil, err
	}
	if c.ExcessAirHigh, err = getFloat("EXCESS_AIR_HIGH", c.ExcessAirHigh); err != nil {
		return nil, err
	}
	if c.FlueTempSpikeDelta, err = getFloat("FLUE_TEMP_SPIKE_DELTA", c.FlueTempSpikeDelta); err != nil {
		return nil, err
	}
	if c.FlueTempCriticalDelta, err = getFloat("FLUE_TEMP_CRITICAL_DELTA", c.FlueTempCriticalDelta); err != nil {
		return nil, err
	}
	if c.FlueTempBaselineWindow, err = getInt("FLUE_TEMP_BASELINE_WINDOW", c.FlueTempBaselineWindow); err != nil {
		return nil, err
	}
	if c.OxygenLow, err = getFloat("OXYGEN_LOW", c.OxygenLow); err != nil {
		return nil, err
	}
	if c.OxygenHigh, err = getFloat("OXYGEN_HIGH", c.OxygenHigh); err != nil {
		return nil, err
	}
	if c.OxygenJumpDelta, err = getFloat("OXYGEN_JUMP_DELTA", c.OxygenJumpDelta); err != nil {
		return nil, err
	}
	if c.PressureHighThreshold, err = getFloat("PRESSURE_HIGH", c.PressureHighThreshold); err != nil {
		return nil, err
	}
	if c.WaterLowThreshold, err = getFloat("WATER_LOW", c.WaterLowThreshold); err != nil {
		return nil, err
	}
	if c.AlertEscalationAfter, err = getDuration("ALERT_ESCALATION_AFTER", c.AlertEscalationAfter); err != nil {
		return nil, err
	}
	if c.SweepInterval, err = getDuration("SWEEP_INTERVAL", c.SweepInterval); err != nil {
		return nil, err
	}
	if c.BlowdownBaseIntervalHours, err = getFloat("BLOWDOWN_BASE_INTERVAL_HOURS", c.BlowdownBaseIntervalHours); err != nil {
		return nil, err
	}
	if c.BlowdownReferenceHardness, err = getFloat("BLOWDOWN_REFERENCE_HARDNESS", c.BlowdownReferenceHardness); err != nil {
		return nil, err
	}
	if c.BlowdownMaxIntervalHours, err = getFloat("BLOWDOWN_MAX_INTERVAL_HOURS", c.BlowdownMaxIntervalHours); err != nil {
		return nil, err
	}
	if c.BlowdownMinIntervalHours, err = getFloat("BLOWDOWN_MIN_INTERVAL_HOURS", c.BlowdownMinIntervalHours); err != nil {
		return nil, err
	}
	if c.BlowdownMissFactor, err = getFloat("BLOWDOWN_MISS_FACTOR", c.BlowdownMissFactor); err != nil {
		return nil, err
	}
	if c.DefaultSampleIntervalMinutes, err = getFloat("SAMPLE_INTERVAL_MINUTES", c.DefaultSampleIntervalMinutes); err != nil {
		return nil, err
	}

	if port := os.Getenv("PORT"); port != "" {
		if _, err := strconv.Atoi(port); err != nil {
			return nil, fmt.Errorf("环境变量 PORT 无法解析为端口: %q", port)
		}
		c.ListenAddr = "0.0.0.0:" + port
	} else if addr := os.Getenv("LISTEN_ADDR"); addr != "" {
		c.ListenAddr = addr
	}
	c.DataDir = getString("DATA_DIR", c.DataDir)
	c.SeedDemo = getBool("SEED_DEMO", c.SeedDemo)
	if c.LogLevel, err = parseLogLevel(os.Getenv("LOG_LEVEL")); err != nil {
		return nil, err
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}
