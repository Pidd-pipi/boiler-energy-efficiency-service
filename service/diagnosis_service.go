package service

import (
	"example.com/boiler-energy-efficiency-service/domain"
)

// ListDiagnosisByBoiler 返回某锅炉工况诊断列表。
func (s *Services) ListDiagnosisByBoiler(boilerID string, limit int) ([]*domain.CombustionStatus, error) {
	return s.Store.ListCombustionByBoiler(boilerID, limit)
}

// ListDiagnosis 返回全部工况诊断（按时间正序）。
func (s *Services) ListDiagnosis(limit int) ([]*domain.CombustionStatus, error) {
	return s.Store.ListCombustion(limit)
}

// DiagnosisSummary 统计指定结果类别的诊断数量。
func (s *Services) DiagnosisSummary(limit int) (map[domain.DiagnosisResult]int, error) {
	list, err := s.Store.ListCombustion(limit)
	if err != nil {
		return nil, err
	}
	out := map[domain.DiagnosisResult]int{
		domain.ResultUnderAir:  0,
		domain.ResultNormal:    0,
		domain.ResultExcessAir: 0,
	}
	for _, c := range list {
		if c.Result.Valid() {
			out[c.Result]++
		}
	}
	return out, nil
}
