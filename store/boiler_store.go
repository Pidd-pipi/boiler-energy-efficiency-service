package store

import (
	"sort"

	"example.com/boiler-energy-efficiency-service/domain"
)

// BoilerStore 锅炉台账仓储接口。
type BoilerStore interface {
	CreateBoiler(b *domain.Boiler) error
	UpdateBoiler(b *domain.Boiler) error
	GetBoiler(id string) (*domain.Boiler, error)
	ListBoilers() ([]*domain.Boiler, error)
	CountBoilers() int
}

// CreateBoiler 新建锅炉并分配 ID。
func (s *Store) CreateBoiler(b *domain.Boiler) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if b.ID == "" {
		b.ID = s.newIDLocked("blr")
	}
	if _, exists := s.boilers[b.ID]; exists {
		return domain.NewError(domain.KindConflict, "锅炉 %s 已存在", b.ID)
	}
	s.boilers[b.ID] = b
	s.boilerOrder = append(s.boilerOrder, b.ID)
	return s.maybeSaveLocked()
}

// UpdateBoiler 更新锅炉信息，锅炉不存在时返回未找到。
func (s *Store) UpdateBoiler(b *domain.Boiler) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.boilers[b.ID]; !exists {
		return domain.NewError(domain.KindNotFound, "锅炉 %s 不存在", b.ID)
	}
	s.boilers[b.ID] = b
	return s.maybeSaveLocked()
}

// GetBoiler 按 ID 获取锅炉。
func (s *Store) GetBoiler(id string) (*domain.Boiler, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, exists := s.boilers[id]
	if !exists {
		return nil, domain.NewError(domain.KindNotFound, "锅炉 %s 不存在", id)
	}
	return cloneBoiler(b), nil
}

// ListBoilers 返回全部锅炉（按创建顺序）。
func (s *Store) ListBoilers() ([]*domain.Boiler, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*domain.Boiler, 0, len(s.boilerOrder))
	for _, id := range s.boilerOrder {
		out = append(out, cloneBoiler(s.boilers[id]))
	}
	return out, nil
}

// CountBoilers 返回锅炉数量。
func (s *Store) CountBoilers() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.boilers)
}

// BoilerSummaries 返回按名称排序的锅炉（供统计/展示）。
func (s *Store) BoilerSummaries() ([]*domain.Boiler, error) {
	bs, err := s.ListBoilers()
	if err != nil {
		return nil, err
	}
	sort.Slice(bs, func(i, j int) bool { return bs[i].Name < bs[j].Name })
	return bs, nil
}
