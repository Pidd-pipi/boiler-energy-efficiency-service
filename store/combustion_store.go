package store

import (
	"example.com/boiler-energy-efficiency-service/domain"
)

// CombustionStore 燃烧工况诊断仓储接口。
type CombustionStore interface {
	CreateCombustion(c *domain.CombustionStatus) error
	GetCombustion(id string) (*domain.CombustionStatus, error)
	ListCombustionByBoiler(boilerID string, limit int) ([]*domain.CombustionStatus, error)
	ListCombustion(limit int) ([]*domain.CombustionStatus, error)
	CountCombustion() int
}

// CreateCombustion 保存诊断记录并分配 ID。
func (s *Store) CreateCombustion(c *domain.CombustionStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c.ID == "" {
		c.ID = s.newIDLocked("cmb")
	}
	if _, exists := s.combustion[c.ID]; exists {
		return domain.NewError(domain.KindConflict, "诊断记录 %s 已存在", c.ID)
	}
	s.combustion[c.ID] = c
	s.combustionOrder = append(s.combustionOrder, c.ID)
	s.combustionByBoiler[c.BoilerID] = append(s.combustionByBoiler[c.BoilerID], c.ID)
	return s.maybeSaveLocked()
}

// GetCombustion 按 ID 获取诊断记录。
func (s *Store) GetCombustion(id string) (*domain.CombustionStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, exists := s.combustion[id]
	if !exists {
		return nil, domain.NewError(domain.KindNotFound, "诊断记录 %s 不存在", id)
	}
	return cloneCombustion(c), nil
}

// ListCombustionByBoiler 返回某锅炉诊断列表。
func (s *Store) ListCombustionByBoiler(boilerID string, limit int) ([]*domain.CombustionStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := s.combustionByBoiler[boilerID]
	if limit > 0 && len(ids) > limit {
		ids = ids[len(ids)-limit:]
	}
	out := make([]*domain.CombustionStatus, 0, len(ids))
	for _, id := range ids {
		out = append(out, cloneCombustion(s.combustion[id]))
	}
	return out, nil
}

// ListCombustion 返回全部诊断记录（按时间正序）。
func (s *Store) ListCombustion(limit int) ([]*domain.CombustionStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := s.combustionOrder
	if limit > 0 && len(ids) > limit {
		ids = ids[len(ids)-limit:]
	}
	out := make([]*domain.CombustionStatus, 0, len(ids))
	for _, id := range ids {
		out = append(out, cloneCombustion(s.combustion[id]))
	}
	return out, nil
}

// CountCombustion 返回诊断记录总量。
func (s *Store) CountCombustion() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.combustion)
}
