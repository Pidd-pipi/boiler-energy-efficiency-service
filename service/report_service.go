package service

import (
	"fmt"
	"time"

	"example.com/boiler-energy-efficiency-service/domain"
	"example.com/boiler-energy-efficiency-service/store"
)

// dailyReportLayout 日报日期格式。
const dailyReportLayout = "2006-01-02"

// DateOf 返回本地时区下的日期字符串。
func (s *Services) DateOf(t time.Time) string {
	return t.Format(dailyReportLayout)
}

// ParseReportDate 解析日报日期字符串。
func ParseReportDate(v string) (time.Time, error) {
	t, err := time.ParseInLocation(dailyReportLayout, v, time.Local)
	if err != nil {
		return time.Time{}, domain.NewError(domain.KindInvalidInput, "日期格式应为 YYYY-MM-DD：%q", v)
	}
	return t, nil
}

// GenerateDailyReport 生成（或刷新）某锅炉指定日期的运行日报。
// 聚合：产汽量、煤耗、平均热效率、告警次数、运行数据条数。
func (s *Services) GenerateDailyReport(boilerID, date string) (*domain.DailyReport, error) {
	b, err := s.Store.GetBoiler(boilerID)
	if err != nil {
		return nil, err
	}
	dayStart, err := ParseReportDate(date)
	if err != nil {
		return nil, err
	}
	dayEnd := dayStart.Add(24 * time.Hour)

	records, err := s.Store.ListEfficiencyByBoiler(boilerID, 0)
	if err != nil {
		return nil, err
	}
	alerts, err := s.Store.ListAlerts(store.AlertFilter{BoilerID: boilerID})
	if err != nil {
		return nil, err
	}

	var steamTotal, coalTotal, effSum float64
	var effCount, runCount, alertCount int
	for _, r := range records {
		if r.Timestamp.Before(dayStart) || !r.Timestamp.Before(dayEnd) {
			continue
		}
		hours := r.IntervalMinutes / 60.0
		if hours <= 0 {
			hours = 1
		}
		steamTotal += r.SteamOutput * hours
		coalTotal += r.FuelAmount * hours
		effSum += r.Efficiency
		effCount++
		runCount++
	}
	for _, a := range alerts {
		if a.CreatedAt.Before(dayStart) || !a.CreatedAt.Before(dayEnd) {
			continue
		}
		alertCount++
	}

	avgEff := 0.0
	if effCount > 0 {
		avgEff = round2(effSum / float64(effCount))
	}

	report := &domain.DailyReport{
		BoilerID:             boilerID,
		BoilerName:           b.Name,
		Date:                 date,
		RunDataCount:         runCount,
		SteamOutputTotal:     round2(steamTotal),
		CoalConsumptionTotal: round2(coalTotal),
		AvgEfficiency:        avgEff,
		AlertCount:           alertCount,
		CreatedAt:            s.now(),
	}
	if err := s.Store.UpsertDailyReport(report); err != nil {
		return nil, err
	}
	return report, nil
}

// GetDailyReport 获取某锅炉某日日报；未生成时自动生成。
func (s *Services) GetDailyReport(boilerID, date string) (*domain.DailyReport, error) {
	if r, err := s.Store.GetDailyReport(boilerID, date); err == nil {
		return r, nil
	}
	return s.GenerateDailyReport(boilerID, date)
}

// ListDailyReports 列出日报（可按日期过滤，date 为空返回全部）。
func (s *Services) ListDailyReports(date string) ([]*domain.DailyReport, error) {
	return s.Store.ListDailyReports(date)
}

// GenerateAllDailyReports 为全部锅炉生成指定日期日报。
func (s *Services) GenerateAllDailyReports(date string) ([]*domain.DailyReport, error) {
	boilers, err := s.Store.ListBoilers()
	if err != nil {
		return nil, err
	}
	out := make([]*domain.DailyReport, 0, len(boilers))
	for _, b := range boilers {
		r, err := s.GenerateDailyReport(b.ID, date)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

// OverviewStats 总览统计摘要。
type OverviewStats struct {
	TotalBoilers   int     `json:"total_boilers"`
	RunningBoilers int     `json:"running_boilers"`
	OpenAlerts     int     `json:"open_alerts"`
	AvgEfficiency  float64 `json:"avg_efficiency"`
	TodaySteam     float64 `json:"today_steam"`
	TodayCoal      float64 `json:"today_coal"`
}

// Overview 聚合总览数据（首页使用）。
func (s *Services) Overview() (*OverviewStats, error) {
	boilers, err := s.Store.ListBoilers()
	if err != nil {
		return nil, err
	}
	stats := &OverviewStats{TotalBoilers: len(boilers)}
	var effSum float64
	var effN int
	for _, b := range boilers {
		if b.Status == domain.BoilerStatusRunning {
			stats.RunningBoilers++
		}
		if e, err := s.Store.LatestEfficiencyByBoiler(b.ID); err == nil {
			effSum += e.Efficiency
			effN++
		}
	}
	if effN > 0 {
		stats.AvgEfficiency = round2(effSum / float64(effN))
	}
	stats.OpenAlerts = s.Store.CountOpenAlerts("")

	today := s.DateOf(s.now())
	reports, err := s.Store.ListDailyReports(today)
	if err != nil {
		return nil, err
	}
	for _, r := range reports {
		stats.TodaySteam += r.SteamOutputTotal
		stats.TodayCoal += r.CoalConsumptionTotal
	}
	stats.TodaySteam = round2(stats.TodaySteam)
	stats.TodayCoal = round2(stats.TodayCoal)
	return stats, nil
}

// BoilerOverview 单锅炉总览视图（详情页头部）。
type BoilerOverview struct {
	Boiler           *domain.Boiler           `json:"boiler"`
	LatestEfficiency *domain.EfficiencyRecord `json:"latest_efficiency,omitempty"`
	OpenAlertCount   int                      `json:"open_alert_count"`
	BlowdownPlan     *domain.BlowdownPlan     `json:"blowdown_plan,omitempty"`
	TodayReport      *domain.DailyReport      `json:"today_report,omitempty"`
}

// BoilerOverviewByID 聚合单锅炉总览信息。
func (s *Services) BoilerOverviewByID(boilerID string) (*BoilerOverview, error) {
	b, err := s.Store.GetBoiler(boilerID)
	if err != nil {
		return nil, err
	}
	ov := &BoilerOverview{Boiler: b}
	if e, err := s.Store.LatestEfficiencyByBoiler(boilerID); err == nil {
		ov.LatestEfficiency = e
	}
	ov.OpenAlertCount = s.Store.CountOpenAlerts(boilerID)
	ov.BlowdownPlan = s.PlanForBoiler(b)
	if r, err := s.GetDailyReport(boilerID, s.DateOf(s.now())); err == nil {
		ov.TodayReport = r
	}
	return ov, nil
}

// boilerNameForReport 供日报生成时回填锅炉名。
func boilerNameForReport(b *domain.Boiler) string {
	return fmt.Sprintf("%s(%s)", b.Name, b.Type.Label())
}
