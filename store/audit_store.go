package store

import (
	"example.com/boiler-energy-efficiency-service/domain"
)

// auditRetentionMax 内存中审计日志保留上限，防止无限增长。
const auditRetentionMax = 5000

// AuditFilter 审计日志过滤条件。
type AuditFilter struct {
	Action     domain.AuditAction
	EntityType string
	EntityID   string
	Limit      int
}

// AuditStore 审计日志仓储接口。
type AuditStore interface {
	AppendAudit(e *domain.AuditEntry) error
	ListAudit(filter AuditFilter) ([]*domain.AuditEntry, error)
	CountAudit() int
}

// AppendAudit 追加审计日志并分配 ID。
func (s *Store) AppendAudit(e *domain.AuditEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.ID == "" {
		e.ID = s.newIDLocked("aud")
	}
	s.audit = append(s.audit, e)
	return s.maybeSaveLocked()
}

// ListAudit 按过滤条件返回审计日志（时间正序，limit 取末尾最新）。
func (s *Store) ListAudit(filter AuditFilter) ([]*domain.AuditEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*domain.AuditEntry, 0, len(s.audit))
	for _, e := range s.audit {
		if filter.Action != "" && e.Action != filter.Action {
			continue
		}
		if filter.EntityType != "" && e.EntityType != filter.EntityType {
			continue
		}
		if filter.EntityID != "" && e.EntityID != filter.EntityID {
			continue
		}
		out = append(out, cloneAudit(e))
	}
	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[len(out)-filter.Limit:]
	}
	return out, nil
}

// CountAudit 返回审计日志总量。
func (s *Store) CountAudit() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.audit)
}
