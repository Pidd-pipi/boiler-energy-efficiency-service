// Package store 提供内存仓储（带读写锁）与 JSON 文件持久化。
// 所有仓储接口与实现都集中在 store 包内，service 只依赖接口语义。
package store

import (
	"fmt"
	"sync"
	"time"

	"example.com/boiler-energy-efficiency-service/domain"
)

// Store 是全部仓储的聚合实现。
// 内部使用单一读写锁保护所有集合，保证并发安全；
// 每个实体集合的增删改查方法见对应 *_store.go 文件。
type Store struct {
	mu sync.RWMutex

	boilers         map[string]*domain.Boiler
	boilerOrder     []string
	runData         map[string]*domain.RunData
	runDataOrder    []string
	runDataByBoiler map[string][]string

	efficiency         map[string]*domain.EfficiencyRecord
	efficiencyOrder    []string
	efficiencyByBoiler map[string][]string

	combustion         map[string]*domain.CombustionStatus
	combustionOrder    []string
	combustionByBoiler map[string][]string

	alerts         map[string]*domain.RunAlert
	alertOrder     []string
	alertsByBoiler map[string][]string

	blowdown         map[string]*domain.BlowdownRecord
	blowdownOrder    []string
	blowdownByBoiler map[string][]string

	reports     map[string]*domain.DailyReport // key: boilerID|date
	reportOrder []string

	audit []*domain.AuditEntry

	seq map[string]int64

	persister *JSONPersister

	// now 时间来源，测试可注入。
	now func() time.Time
}

// Options 控制仓储的创建方式。
type Options struct {
	// DataDir 非空时启用 JSON 文件持久化，数据保存在 <DataDir>/store.json。
	DataDir string
	// Now 时间来源，便于测试注入。
	Now func() time.Time
}

// New 创建空仓储；若 DataDir 非空则尝试从持久化文件恢复。
func New(opts Options) (*Store, error) {
	s := &Store{
		boilers:            make(map[string]*domain.Boiler),
		runData:            make(map[string]*domain.RunData),
		runDataByBoiler:    make(map[string][]string),
		efficiency:         make(map[string]*domain.EfficiencyRecord),
		efficiencyByBoiler: make(map[string][]string),
		combustion:         make(map[string]*domain.CombustionStatus),
		combustionByBoiler: make(map[string][]string),
		alerts:             make(map[string]*domain.RunAlert),
		alertsByBoiler:     make(map[string][]string),
		blowdown:           make(map[string]*domain.BlowdownRecord),
		blowdownByBoiler:   make(map[string][]string),
		reports:            make(map[string]*domain.DailyReport),
		seq:                make(map[string]int64),
		now:                time.Now,
	}
	if opts.Now != nil {
		s.now = opts.Now
	}
	if opts.DataDir != "" {
		p, err := NewJSONPersister(opts.DataDir)
		if err != nil {
			return nil, err
		}
		s.persister = p
		if err := s.loadFromDisk(); err != nil {
			if herr := s.handleLoadError(err); herr != nil {
				return nil, herr
			}
		}
	}
	return s, nil
}

// Now 返回仓储时间源。
func (s *Store) Now() time.Time { return s.now() }

// NewID 生成自增业务 ID，前缀标识实体类型。
func (s *Store) NewID(kind string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.newIDLocked(kind)
}

// newIDLocked 在调用方已持锁时生成 ID。
func (s *Store) newIDLocked(kind string) string {
	s.seq[kind]++
	return fmt.Sprintf("%s_%06d", kind, s.seq[kind])
}

// maybeSaveLocked 在调用方已持有写锁时落盘（无持久化则跳过）。
func (s *Store) maybeSaveLocked() error {
	if s.persister == nil {
		return nil
	}
	return s.persister.Write(s.snapshotLocked())
}

// Save 将当前全部数据快照写入持久化文件（若无持久化则跳过）。
func (s *Store) Save() error {
	if s.persister == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.persister.Write(s.snapshotLocked())
}

// Close 关闭仓储并做最后一次落盘。
func (s *Store) Close() (err error) {
	if s.persister == nil {
		return nil
	}
	defer func() {
		// 落盘失败时保持返回 nil，避免关闭流程被错误打断。
		err = nil
	}()
	return s.Save()
}
