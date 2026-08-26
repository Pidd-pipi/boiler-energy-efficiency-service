package store

import (
	"example.com/boiler-energy-efficiency-service/domain"
)

// BlowdownStore 排污记录仓储接口。
type BlowdownStore interface {
	CreateBlowdown(b *domain.BlowdownRecord) error
	GetBlowdown(id string) (*domain.BlowdownRecord, error)
	ListBlowdownByBoiler(boilerID string, limit int) ([]*domain.BlowdownRecord, error)
	LastBlowdownByBoiler(boilerID string) (*domain.BlowdownRecord, error)
	CountBlowdownByBoiler(boilerID string) int64
	CountBlowdown() int
}

// CreateBlowdown 保存排污记录并分配 ID。
func (s *Store) CreateBlowdown(b *domain.BlowdownRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if b.ID == "" {
		b.ID = s.newIDLocked("bld")
	}
	if _, exists := s.blowdown[b.ID]; exists {
		return domain.NewError(domain.KindConflict, "排污记录 %s 已存在", b.ID)
	}
	s.blowdown[b.ID] = b
	s.blowdownOrder = append(s.blowdownOrder, b.ID)
	s.blowdownByBoiler[b.BoilerID] = append(s.blowdownByBoiler[b.BoilerID], b.ID)
	return s.maybeSaveLocked()
}

// GetBlowdown 按 ID 获取排污记录。
func (s *Store) GetBlowdown(id string) (*domain.BlowdownRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, exists := s.blowdown[id]
	if !exists {
		return nil, domain.NewError(domain.KindNotFound, "排污记录 %s 不存在", id)
	}
	return cloneBlowdown(b), nil
}

// ListBlowdownByBoiler 返回某锅炉排污记录列表。
func (s *Store) ListBlowdownByBoiler(boilerID string, limit int) ([]*domain.BlowdownRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := s.blowdownByBoiler[boilerID]
	if limit > 0 && len(ids) > limit {
		ids = ids[len(ids)-limit:]
	}
	out := make([]*domain.BlowdownRecord, 0, len(ids))
	for _, id := range ids {
		out = append(out, cloneBlowdown(s.blowdown[id]))
	}
	return out, nil
}

// LastBlowdownByBoiler 返回某锅炉最近一次排污记录。
func (s *Store) LastBlowdownByBoiler(boilerID string) (*domain.BlowdownRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := s.blowdownByBoiler[boilerID]
	if len(ids) == 0 {
		return nil, domain.NewError(domain.KindNotFound, "锅炉 %s 尚无排污记录", boilerID)
	}
	return cloneBlowdown(s.blowdown[ids[len(ids)-1]]), nil
}

// CountBlowdownByBoiler 返回某锅炉排污次数。
func (s *Store) CountBlowdownByBoiler(boilerID string) int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return int64(len(s.blowdownByBoiler[boilerID]))
}

// CountBlowdown 返回排污记录总量。
func (s *Store) CountBlowdown() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.blowdown)
}
