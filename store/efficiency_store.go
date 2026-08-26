package store

import (
	"example.com/boiler-energy-efficiency-service/domain"
)

// EfficiencyStore 能效记录仓储接口。
type EfficiencyStore interface {
	CreateEfficiency(e *domain.EfficiencyRecord) error
	GetEfficiency(id string) (*domain.EfficiencyRecord, error)
	ListEfficiencyByBoiler(boilerID string, limit int) ([]*domain.EfficiencyRecord, error)
	LatestEfficiencyByBoiler(boilerID string) (*domain.EfficiencyRecord, error)
	CountEfficiency() int
}

// CreateEfficiency 保存能效记录并分配 ID。
func (s *Store) CreateEfficiency(e *domain.EfficiencyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.ID == "" {
		e.ID = s.newIDLocked("eff")
	}
	if _, exists := s.efficiency[e.ID]; exists {
		return domain.NewError(domain.KindConflict, "能效记录 %s 已存在", e.ID)
	}
	s.efficiency[e.ID] = e
	s.efficiencyOrder = append(s.efficiencyOrder, e.ID)
	s.efficiencyByBoiler[e.BoilerID] = append(s.efficiencyByBoiler[e.BoilerID], e.ID)
	return s.maybeSaveLocked()
}

// GetEfficiency 按 ID 获取能效记录。
func (s *Store) GetEfficiency(id string) (*domain.EfficiencyRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, exists := s.efficiency[id]
	if !exists {
		return nil, domain.NewError(domain.KindNotFound, "能效记录 %s 不存在", id)
	}
	return cloneEfficiency(e), nil
}

// ListEfficiencyByBoiler 返回某锅炉能效记录，limit<=0 表示不限制。
func (s *Store) ListEfficiencyByBoiler(boilerID string, limit int) ([]*domain.EfficiencyRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := s.efficiencyByBoiler[boilerID]
	if limit > 0 && len(ids) > limit {
		ids = ids[len(ids)-limit:]
	}
	out := make([]*domain.EfficiencyRecord, 0, len(ids))
	for _, id := range ids {
		out = append(out, cloneEfficiency(s.efficiency[id]))
	}
	return out, nil
}

// LatestEfficiencyByBoiler 返回某锅炉最近一条能效记录。
func (s *Store) LatestEfficiencyByBoiler(boilerID string) (*domain.EfficiencyRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := s.efficiencyByBoiler[boilerID]
	if len(ids) == 0 {
		return nil, nil
	}
	return cloneEfficiency(s.efficiency[ids[len(ids)-1]]), nil
}

// CountEfficiency 返回能效记录总量。
func (s *Store) CountEfficiency() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.efficiency)
}
