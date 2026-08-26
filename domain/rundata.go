package domain

import "time"

// RunData 锅炉运行数据采集实体。
// 每条运行数据对应一个采样周期（IntervalMinutes 分钟）。
type RunData struct {
	ID              string    `json:"id"`
	BoilerID        string    `json:"boiler_id"`
	FuelAmount      float64   `json:"fuel_amount"`       // 燃料量 kg/h
	SteamOutput     float64   `json:"steam_output"`      // 蒸汽量/产热量 t/h
	FeedWaterFlow   float64   `json:"feed_water_flow"`   // 给水/循环水流量 t/h
	FlueGasTemp     float64   `json:"flue_gas_temp"`     // 排烟温度 ℃
	OxygenContent   float64   `json:"oxygen_content"`    // 氧含量 %
	SteamPressure   float64   `json:"steam_pressure"`    // 蒸汽压力 MPa
	WaterLevel      float64   `json:"water_level"`       // 水位 %
	SupplyWaterTemp float64   `json:"supply_water_temp"` // 热水锅炉出水温度 ℃
	ReturnWaterTemp float64   `json:"return_water_temp"` // 热水锅炉回水温度 ℃
	IntervalMinutes float64   `json:"interval_minutes"`  // 采样周期（分钟）
	Timestamp       time.Time `json:"timestamp"`
}

// NewRunData 构造运行数据实体。
func NewRunData(boilerID string, now time.Time) *RunData {
	return &RunData{
		BoilerID:        boilerID,
		IntervalMinutes: 0,
		Timestamp:       now,
	}
}

// HasFuel 是否具备燃料量输入。
func (r *RunData) HasFuel() bool { return r.FuelAmount > 0 }

// HasOutput 是否具备产出量输入（蒸汽锅炉看蒸汽量，热水锅炉看循环水量）。
func (r *RunData) HasOutput() bool { return r.SteamOutput > 0 || r.FeedWaterFlow > 0 }
