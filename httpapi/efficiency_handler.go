package httpapi

import (
	"net/http"

	"example.com/boiler-energy-efficiency-service/domain"
	"example.com/boiler-energy-efficiency-service/service"
)

// EfficiencyHandler 能效记录查询。
type EfficiencyHandler struct {
	Svc *service.Services
}

// List GET /api/boilers/{id}/efficiency?limit=N&offset=M 能效记录列表。
func (h *EfficiencyHandler) List(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	if id == "" {
		return domain.NewError(domain.KindInvalidInput, "缺少路径参数 id")
	}
	limit, offset, err := parseLimitOffset(r, 100, 500)
	if err != nil {
		return err
	}
	list, err := h.Svc.ListEfficiencyByBoiler(id, 0)
	if err != nil {
		return err
	}
	items, total := paginateTail(list, limit, offset)
	setPageHeaders(w, total, limit, offset)
	respondOK(w, r, items)
	return nil
}
