package domain

import "time"

// DailyReport 锅炉运行日报实体。
// 每日按锅炉聚合产汽量、煤耗、平均热效率、告警次数，用于月度能效考核。
type DailyReport struct {
	ID                   string    `json:"id"`
	BoilerID             string    `json:"boiler_id"`
	BoilerName           string    `json:"boiler_name"`
	Date                 string    `json:"date"` // 2006-01-02
	RunDataCount         int       `json:"run_data_count"`
	SteamOutputTotal     float64   `json:"steam_output_total"`     // 当日累计产汽量 t
	CoalConsumptionTotal float64   `json:"coal_consumption_total"` // 当日累计煤耗 kg
	AvgEfficiency        float64   `json:"avg_efficiency"`         // 平均热效率 %
	AlertCount           int       `json:"alert_count"`            // 当日告警次数
	CreatedAt            time.Time `json:"created_at"`
}
