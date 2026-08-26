package httpapi

import (
	"net/http"

	"example.com/boiler-energy-efficiency-service/domain"
	"example.com/boiler-energy-efficiency-service/service"
	"example.com/boiler-energy-efficiency-service/store"
)

// AuditHandler 操作审计日志查询。
type AuditHandler struct {
	Svc *service.Services
}

// List GET /api/audit-logs?action=&entity_type=&entity_id=&limit=&offset= 审计日志。
func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()
	action := domain.AuditAction(q.Get("action"))
	if action != "" && !action.Valid() {
		return domain.NewError(domain.KindInvalidInput, "非法审计动作: %q", action)
	}
	limit, offset, err := parseLimitOffset(r, 0, 1000)
	if err != nil {
		return err
	}
	filter := store.AuditFilter{
		Action:     action,
		EntityType: q.Get("entity_type"),
		EntityID:   q.Get("entity_id"),
	}
	list, err := h.Svc.ListAudit(filter)
	if err != nil {
		return err
	}
	items, total := paginateTail(list, limit, offset)
	setPageHeaders(w, total, limit, offset)
	respondOK(w, r, items)
	return nil
}
