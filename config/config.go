// Package config 集中管理能效公式系数、燃烧工况阈值、告警阈值与
// 排污周期等全部可调业务参数，保证领域规则有唯一可验证的数据来源。
package config

import (
	"fmt"
	"log/slog"
	"net"
	"time"
)

// Config 保存本项目全部可调系数与阈值。
// 所有字段都可以通过环境变量覆盖（见 Load）。
type Config struct {
	// ---- 能效正平衡计算系数 ----
	// SteamEnthalpyKJPerKg 饱和蒸汽焓（kJ/kg），用于蒸汽锅炉有效利用热量计算。
	SteamEnthalpyKJPerKg float64
	// FeedWaterEnthalpyKJPerKg 给水焓（kJ/kg）。
	FeedWaterEnthalpyKJPerKg float64
	// CoalLowerHeatingValueKJPerKg 燃料低位发热量（kJ/kg）。
	CoalLowerHeatingValueKJPerKg float64
	// WaterSpecificHeatKJPerKgC 水的比热容（kJ/(kg·℃)），用于热水锅炉计算。
	WaterSpecificHeatKJPerKgC float64
	// HotWaterSupplyTempC 热水锅炉默认出水温度（℃）。
	HotWaterSupplyTempC float64
	// HotWaterReturnTempC 热水锅炉默认回水温度（℃）。
	HotWaterReturnTempC float64

	// ---- 燃烧工况诊断阈值 ----
	// ExcessAirLow 过量空气系数下限，低于该值判为缺氧燃烧。
	ExcessAirLow float64
	// ExcessAirHigh 过量空气系数上限，高于该值判为过剩空气。
	ExcessAirHigh float64

	// ---- 告警判定阈值 ----
	// FlueTempSpikeDelta 排烟温度相对基线突升阈值（℃）。
	FlueTempSpikeDelta float64
	// FlueTempCriticalDelta 排烟温度突升达到该值判为严重告警（℃）。
	FlueTempCriticalDelta float64
	// FlueTempBaselineWindow 排烟温度基线滑动窗口条数。
	FlueTempBaselineWindow int
	// OxygenLow 氧含量下限（%）。
	OxygenLow float64
	// OxygenHigh 氧含量上限（%）。
	OxygenHigh float64
	// OxygenJumpDelta 氧含量相对上一采样点突跳阈值（百分点）。
	OxygenJumpDelta float64
	// PressureHighThreshold 蒸汽压力过高阈值（MPa）。
	PressureHighThreshold float64
	// WaterLowThreshold 水位过低阈值（%）。
	WaterLowThreshold float64
	// AlertEscalationAfter 告警未确认超时升级时长。
	AlertEscalationAfter time.Duration
	// SweepInterval 告警升级扫描周期。
	SweepInterval time.Duration

	// ---- 排污管理 ----
	// BlowdownBaseIntervalHours 基准排污周期（小时），对应基准硬度。
	BlowdownBaseIntervalHours float64
	// BlowdownReferenceHardness 基准水质硬度（mmol/L）。
	BlowdownReferenceHardness float64
	// BlowdownMaxIntervalHours 排污周期上限（小时）。
	BlowdownMaxIntervalHours float64
	// BlowdownMinIntervalHours 排污周期下限（小时）。
	BlowdownMinIntervalHours float64
	// BlowdownMissFactor 超期倍数，累计运行时长超过 2 倍周期进入「需关注」。
	BlowdownMissFactor float64

	// ---- 运行数据采样 ----
	// DefaultSampleIntervalMinutes 每条运行数据默认代表的采样时长（分钟），
	// 用于累计锅炉运行时长（排污周期依据）。
	DefaultSampleIntervalMinutes float64

	// ---- 服务参数 ----
	// ListenAddr 监听地址（默认 0.0.0.0:8080，可由 PORT 覆盖）。
	ListenAddr string
	// LogLevel 日志级别（debug/info/warn/error），默认 info。
	LogLevel slog.Level
	// DataDir 数据目录，用于 JSON 文件持久化；为空则不落盘。
	DataDir string
	// SeedDemo 首次启动且数据为空时是否写入演示数据。
	SeedDemo bool
}

// Default 返回一份带默认值的配置，保证离线可运行。
func Default() *Config {
	return &Config{
		SteamEnthalpyKJPerKg:         2760.0,  // 饱和蒸汽焓（约 1.0 MPa）
		FeedWaterEnthalpyKJPerKg:     293.0,   // 70℃ 给水焓
		CoalLowerHeatingValueKJPerKg: 20934.0, // 5000 kcal/kg
		WaterSpecificHeatKJPerKgC:    4.1868,
		HotWaterSupplyTempC:          85.0,
		HotWaterReturnTempC:          60.0,

		ExcessAirLow:  1.2,
		ExcessAirHigh: 1.8,

		FlueTempSpikeDelta:     30.0,
		FlueTempCriticalDelta:  60.0,
		FlueTempBaselineWindow: 10,
		OxygenLow:              3.0,
		OxygenHigh:             15.0,
		OxygenJumpDelta:        3.0,
		PressureHighThreshold:  1.6,
		WaterLowThreshold:      40.0,
		AlertEscalationAfter:   time.Hour,
		SweepInterval:          10 * time.Minute,

		BlowdownBaseIntervalHours: 48.0,
		BlowdownReferenceHardness: 2.0,
		BlowdownMaxIntervalHours:  720.0,
		BlowdownMinIntervalHours:  8.0,
		BlowdownMissFactor:        2.0,

		DefaultSampleIntervalMinutes: 5.0,

		ListenAddr: "0.0.0.0:8080",
		LogLevel:   slog.LevelInfo,
		DataDir:    "data",
		SeedDemo:   true,
	}
}

// String 返回便于日志输出的配置摘要（不含敏感信息）。
func (c *Config) String() string {
	return fmt.Sprintf(
		"listen=%s data_dir=%s seed_demo=%t excess_air=[%v,%v] flue_spike=%.1fC escalation=%s sweep=%s blowdown_base=%.1fh",
		c.ListenAddr, c.DataDir, c.SeedDemo,
		c.ExcessAirLow, c.ExcessAirHigh,
		c.FlueTempSpikeDelta, c.AlertEscalationAfter, c.SweepInterval,
		c.BlowdownBaseIntervalHours,
	)
}

// Validate 校验配置合法性，启动前拒绝明显错误的参数。
func (c *Config) Validate() error {
	if c.ListenAddr == "" {
		return fmt.Errorf("监听地址不能为空")
	}
	if _, _, err := net.SplitHostPort(c.ListenAddr); err != nil {
		return fmt.Errorf("监听地址格式错误 %q: %w", c.ListenAddr, err)
	}
	if c.SteamEnthalpyKJPerKg <= c.FeedWaterEnthalpyKJPerKg {
		return fmt.Errorf("蒸汽焓必须大于给水焓")
	}
	if c.CoalLowerHeatingValueKJPerKg <= 0 {
		return fmt.Errorf("燃料低位发热量必须大于 0")
	}
	if c.WaterSpecificHeatKJPerKgC <= 0 {
		return fmt.Errorf("水的比热容必须大于 0")
	}
	if c.ExcessAirLow <= 0 || c.ExcessAirHigh <= c.ExcessAirLow {
		return fmt.Errorf("过量空气系数阈值必须满足 0 < low < high")
	}
	if c.FlueTempSpikeDelta < 0 || c.FlueTempCriticalDelta < c.FlueTempSpikeDelta {
		return fmt.Errorf("排烟温度突升阈值必须满足 0 <= spike <= critical")
	}
	if c.FlueTempBaselineWindow <= 0 {
		return fmt.Errorf("排烟温度基线窗口必须大于 0")
	}
	if c.OxygenLow < 0 || c.OxygenHigh <= c.OxygenLow {
		return fmt.Errorf("氧含量阈值必须满足 0 <= low < high")
	}
	if c.OxygenJumpDelta < 0 {
		return fmt.Errorf("氧含量突跳阈值不能为负数")
	}
	if c.PressureHighThreshold <= 0 || c.WaterLowThreshold <= 0 {
		return fmt.Errorf("压力/水位告警阈值必须大于 0")
	}
	if c.AlertEscalationAfter <= 0 || c.SweepInterval <= 0 {
		return fmt.Errorf("告警升级时长与扫描周期必须大于 0")
	}
	if c.BlowdownBaseIntervalHours <= 0 || c.BlowdownReferenceHardness <= 0 {
		return fmt.Errorf("排污基准周期与基准硬度必须大于 0")
	}
	if c.BlowdownMinIntervalHours <= 0 || c.BlowdownMaxIntervalHours < c.BlowdownMinIntervalHours {
		return fmt.Errorf("排污周期夹取区间必须满足 0 < min <= max")
	}
	if c.BlowdownMissFactor < 1 {
		return fmt.Errorf("排污超期倍数不能小于 1")
	}
	if c.DefaultSampleIntervalMinutes <= 0 {
		return fmt.Errorf("默认采样周期必须大于 0")
	}
	return nil
}
