package store

import (
	"sort"

	"example.com/boiler-energy-efficiency-service/domain"
)

// BoilerStore 锅炉台账仓储接口。
type BoilerStore interface {
	CreateBoiler(b *domain.Boiler) error
	UpdateBoiler(b *domain.Boiler) error
	MutateBoiler(id string, fn func(b *domain.Boiler) error) (*domain.Boiler, error)
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
	// 存入副本，断开调用方持有的指针与仓储内部指针的别名，
	// 避免调用方事后修改外部对象污染内部状态。
	s.boilers[b.ID] = cloneBoiler(b)
	s.boilerOrder = append(s.boilerOrder, b.ID)
	return s.maybeSaveLocked()
}

// UpdateBoiler 更新锅炉信息，锅炉不存在时返回未找到。
// 入参会先拷贝再入库，调用方持有的指针与仓储内部互不影响。
func (s *Store) UpdateBoiler(b *domain.Boiler) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.boilers[b.ID]; !exists {
		return domain.NewError(domain.KindNotFound, "锅炉 %s 不存在", b.ID)
	}
	s.boilers[b.ID] = cloneBoiler(b)
	return s.maybeSaveLocked()
}

// MutateBoiler 在单次写锁内执行“读取 -> 回调修改 -> 写回”，
// 保证并发读改写（如运行时长累计、状态迁移、排污重置）原子完成，
// 不会出现两次加锁之间的丢失更新。
// 回调收到的 b 是独立副本，可直接修改；fn 返回非 nil 时整体回滚不入库。
// 返回值为最终入库状态的副本。
func (s *Store) MutateBoiler(id string, fn func(b *domain.Boiler) error) (*domain.Boiler, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, exists := s.boilers[id]
	if !exists {
		return nil, domain.NewError(domain.KindNotFound, "锅炉 %s 不存在", id)
	}
	b := cloneBoiler(cur)
	if err := fn(b); err != nil {
		return nil, err
	}
	s.boilers[id] = cloneBoiler(b)
	if err := s.maybeSaveLocked(); err != nil {
		return nil, err
	}
	return cloneBoiler(b), nil
}

// GetBoiler 按 ID 获取锅炉（返回副本，调用方修改不影响仓储内部）。
func (s *Store) GetBoiler(id string) (*domain.Boiler, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, exists := s.boilers[id]
	if !exists {
		return nil, domain.NewError(domain.KindNotFound, "锅炉 %s 不存在", id)
	}
	return cloneBoiler(b), nil
}

// ListBoilers 返回全部锅炉（按创建顺序，均为副本）。
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
