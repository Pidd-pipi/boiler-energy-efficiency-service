package store

import (
	"sort"

	"example.com/boiler-energy-efficiency-service/domain"
)

// reportKey 日报主键：boilerID|date。
func reportKey(boilerID, date string) string { return boilerID + "|" + date }

// ReportStore 运行日报仓储接口。
type ReportStore interface {
	UpsertDailyReport(r *domain.DailyReport) error
	GetDailyReport(boilerID, date string) (*domain.DailyReport, error)
	ListDailyReports(date string) ([]*domain.DailyReport, error)
	CountReports() int
}

// UpsertDailyReport 写入或更新日报（同一锅炉同一天只保留一条）。
func (s *Store) UpsertDailyReport(r *domain.DailyReport) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := reportKey(r.BoilerID, r.Date)
	if existing, ok := s.reports[key]; ok {
		r.ID = existing.ID
	} else {
		if r.ID == "" {
			r.ID = s.newIDLocked("rpt")
		}
		s.reportOrder = append(s.reportOrder, key)
	}
	s.reports[key] = r
	return s.maybeSaveLocked()
}

// GetDailyReport 获取某锅炉某日日报。
func (s *Store) GetDailyReport(boilerID, date string) (*domain.DailyReport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, exists := s.reports[reportKey(boilerID, date)]
	if !exists {
		return nil, nil
	}
	return cloneReport(r), nil
}

// ListDailyReports 返回指定日期的全部日报（按锅炉名排序）；date 为空返回全部。
func (s *Store) ListDailyReports(date string) ([]*domain.DailyReport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*domain.DailyReport, 0, len(s.reports))
	for key, r := range s.reports {
		if date != "" && !keyHasDate(key, date) {
			continue
		}
		out = append(out, cloneReport(r))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Date != out[j].Date {
			return out[i].Date < out[j].Date
		}
		return out[i].BoilerName < out[j].BoilerName
	})
	return out, nil
}

// keyHasDate 判断报告主键是否属于指定日期。
func keyHasDate(key, date string) bool {
	for i := 0; i+len(date) <= len(key); i++ {
		if key[i:i+len(date)] == date {
			return true
		}
	}
	return false
}

// CountReports 返回日报总量。
func (s *Store) CountReports() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.reports)
}
