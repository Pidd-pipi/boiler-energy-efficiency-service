package service

import (
	"example.com/boiler-energy-efficiency-service/domain"
)

// ListEfficiencyByBoiler 返回某锅炉能效记录列表。
func (s *Services) ListEfficiencyByBoiler(boilerID string, limit int) ([]*domain.EfficiencyRecord, error) {
	return s.Store.ListEfficiencyByBoiler(boilerID, limit)
}

// LatestEfficiencyByBoiler 返回某锅炉最新能效记录。
func (s *Services) LatestEfficiencyByBoiler(boilerID string) (*domain.EfficiencyRecord, error) {
	return s.Store.LatestEfficiencyByBoiler(boilerID)
}

// AverageEfficiencyByBoiler 返回某锅炉最近 limit 条能效记录的平均热效率。
func (s *Services) AverageEfficiencyByBoiler(boilerID string, limit int) (float64, error) {
	records, err := s.Store.ListEfficiencyByBoiler(boilerID, limit)
	if err != nil {
		return 0, err
	}
	if len(records) == 0 {
		return 0, nil
	}
	var sum float64
	for _, r := range records {
		sum += r.Efficiency
	}
	return round2(sum / float64(len(records))), nil
}

// round2 保留两位小数（与 domain 内实现一致，供聚合统计使用）。
func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100.0
}
