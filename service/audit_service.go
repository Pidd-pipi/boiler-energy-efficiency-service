package service

import (
	"example.com/boiler-energy-efficiency-service/domain"
	"example.com/boiler-energy-efficiency-service/store"
)

// Audit 写入一条操作审计日志。
// 状态迁移、告警确认、排污执行等关键动作全部经由该方法留痕。
func (s *Services) Audit(traceID string, action domain.AuditAction, entityType, entityID, operator, detail string) error {
	entry := domain.NewAuditEntry(traceID, action, entityType, entityID, operator, detail, s.now())
	return s.Store.AppendAudit(entry)
}

// ListAudit 查询审计日志。
func (s *Services) ListAudit(filter store.AuditFilter) ([]*domain.AuditEntry, error) {
	return s.Store.ListAudit(filter)
}
