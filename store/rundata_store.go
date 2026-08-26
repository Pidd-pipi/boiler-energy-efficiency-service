package store

import (
	"example.com/boiler-energy-efficiency-service/domain"
)

// RunDataStore 运行数据仓储接口。
type RunDataStore interface {
	CreateRunData(rd *domain.RunData) error
	GetRunData(id string) (*domain.RunData, error)
	ListRunDataByBoiler(boilerID string, limit int) ([]*domain.RunData, error)
	RecentRunDataByBoiler(boilerID string, n int) ([]*domain.RunData, error)
	CountRunData() int
}

// CreateRunData 保存运行数据并分配 ID。
func (s *Store) CreateRunData(rd *domain.RunData) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rd.ID == "" {
		rd.ID = s.newIDLocked("run")
	}
	if _, exists := s.runData[rd.ID]; exists {
		return domain.NewError(domain.KindConflict, "运行数据 %s 已存在", rd.ID)
	}
	s.runData[rd.ID] = rd
	s.runDataOrder = append(s.runDataOrder, rd.ID)
	s.runDataByBoiler[rd.BoilerID] = append(s.runDataByBoiler[rd.BoilerID], rd.ID)
	return s.maybeSaveLocked()
}

// GetRunData 按 ID 获取运行数据。
func (s *Store) GetRunData(id string) (*domain.RunData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rd, exists := s.runData[id]
	if !exists {
		return nil, domain.NewError(domain.KindNotFound, "运行数据 %s 不存在", id)
	}
	return cloneRunData(rd), nil
}

// ListRunDataByBoiler 返回某锅炉运行数据，limit<=0 表示不限制（按时间正序）。
func (s *Store) ListRunDataByBoiler(boilerID string, limit int) ([]*domain.RunData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := s.runDataByBoiler[boilerID]
	if limit > 0 && len(ids) > limit {
		ids = ids[len(ids)-limit:]
	}
	out := make([]*domain.RunData, 0, len(ids))
	for _, id := range ids {
		out = append(out, cloneRunData(s.runData[id]))
	}
	return out, nil
}

// RecentRunDataByBoiler 返回最近 n 条运行数据（用于基线计算）。
func (s *Store) RecentRunDataByBoiler(boilerID string, n int) ([]*domain.RunData, error) {
	if n <= 0 {
		return []*domain.RunData{}, nil
	}
	return s.ListRunDataByBoiler(boilerID, n)
}

// CountRunData 返回运行数据总量。
func (s *Store) CountRunData() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.runData)
}
