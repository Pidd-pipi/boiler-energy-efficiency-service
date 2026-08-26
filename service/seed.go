package service

import (
	"log/slog"
	"time"

	"example.com/boiler-energy-efficiency-service/domain"
)

// SeedDemoData 在数据为空时写入演示锅炉与最近几天的运行数据，
// 保证前端页面开箱即有真实可交互的数据。
// 幂等：仅当仓库没有任何锅炉时执行。
func (s *Services) SeedDemoData() error {
	if s.Store.CountBoilers() > 0 {
		return nil
	}
	now := s.now()
	slog.Info("仓库为空，写入演示数据")

	steam, err := s.CreateBoiler("seed", CreateBoilerInput{
		Name:          "1# 蒸汽锅炉",
		Type:          domain.BoilerTypeSteam,
		RatedCapacity: 10,
		WaterHardness: 2.5,
		Operator:      "system",
	})
	if err != nil {
		return err
	}
	hot, err := s.CreateBoiler("seed", CreateBoilerInput{
		Name:          "2# 热水锅炉",
		Type:          domain.BoilerTypeHotWater,
		RatedCapacity: 7,
		WaterHardness: 4.0,
		Operator:      "system",
	})
	if err != nil {
		return err
	}

	// 状态迁移到运行。
	if _, err := s.Transition("seed", steam.ID, domain.BoilerStatusStarting, "system"); err != nil {
		return err
	}
	if _, err := s.Transition("seed", steam.ID, domain.BoilerStatusRunning, "system"); err != nil {
		return err
	}
	if _, err := s.Transition("seed", hot.ID, domain.BoilerStatusStarting, "system"); err != nil {
		return err
	}
	if _, err := s.Transition("seed", hot.ID, domain.BoilerStatusRunning, "system"); err != nil {
		return err
	}

	// 回填最近 3 天运行数据（每 2 小时一条）。
	for day := 2; day >= 0; day-- {
		dayStart := time.Date(now.Year(), now.Month(), now.Day(), 6, 0, 0, 0, now.Location()).AddDate(0, 0, -day)
		for h := 0; h < 11; h++ {
			ts := dayStart.Add(time.Duration(h*2) * time.Hour)
			if ts.After(now) {
				break
			}
			if _, err := s.IngestRunData("seed", steam.ID, RunIngestInput{
				FuelAmount:      1180 + float64(h%3)*20,
				SteamOutput:     8.2 + float64(h%4)*0.3,
				FeedWaterFlow:   9.0,
				FlueGasTemp:     148 + float64(h%5)*2,
				OxygenContent:   7.5 + float64(h%6)*0.2,
				SteamPressure:   1.25,
				WaterLevel:      62,
				IntervalMinutes: 120,
				Timestamp:       ts,
				Operator:        "system",
			}); err != nil {
				return err
			}
			if _, err := s.IngestRunData("seed", hot.ID, RunIngestInput{
				FuelAmount:      734 + float64(h%3)*15,
				SteamOutput:     0,
				FeedWaterFlow:   130 + float64(h%2)*3,
				FlueGasTemp:     136 + float64(h%5)*2,
				OxygenContent:   8.2 + float64(h%4)*0.15,
				SteamPressure:   0,
				WaterLevel:      55,
				SupplyWaterTemp: 82,
				ReturnWaterTemp: 58,
				IntervalMinutes: 120,
				Timestamp:       ts,
				Operator:        "system",
			}); err != nil {
				return err
			}
		}
	}
	// 生成最近三天日报。
	for day := 2; day >= 0; day-- {
		date := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -day).Format(dailyReportLayout)
		if _, err := s.GenerateAllDailyReports(date); err != nil {
			return err
		}
	}
	slog.Info("演示数据写入完成")
	return nil
}
