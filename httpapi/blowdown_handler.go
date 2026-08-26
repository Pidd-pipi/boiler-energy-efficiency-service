package httpapi

import (
	"net/http"

	"example.com/boiler-energy-efficiency-service/domain"
	"example.com/boiler-energy-efficiency-service/middleware"
	"example.com/boiler-energy-efficiency-service/service"
)

// BlowdownHandler 排污管理。
type BlowdownHandler struct {
	Svc *service.Services
}

// Get GET /api/boilers/{id}/blowdown 排污计划 + 执行记录。
func (h *BlowdownHandler) Get(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	if id == "" {
		return domain.NewError(domain.KindInvalidInput, "缺少路径参数 id")
	}
	plan, records, err := h.Svc.GetBlowdownDetail(id)
	if err != nil {
		return err
	}
	respondOK(w, r, map[string]any{
		"plan":    plan,
		"records": records,
	})
	return nil
}

// Execute POST /api/boilers/{id}/blowdown 排污执行。
func (h *BlowdownHandler) Execute(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	if id == "" {
		return domain.NewError(domain.KindInvalidInput, "缺少路径参数 id")
	}
	var in service.ExecuteBlowdownInput
	if err := decodeJSON(r, &in); err != nil {
		return err
	}
	if in.DurationMin < 0 {
		return domain.NewError(domain.KindInvalidInput, "duration_min 不能为负数：%v", in.DurationMin)
	}
	if in.Operator == "" {
		in.Operator = r.Header.Get("X-Operator")
	}
	rec, plan, err := h.Svc.ExecuteBlowdown(middleware.TraceID(r.Context()), id, in)
	if err != nil {
		return err
	}
	respondCreated(w, r, map[string]any{
		"record": rec,
		"plan":   plan,
	})
	return nil
}

// ListPlans GET /api/blowdown?limit=&offset= 全部锅炉排污计划。
func (h *BlowdownHandler) ListPlans(w http.ResponseWriter, r *http.Request) error {
	limit, offset, err := parseLimitOffset(r, 200, 500)
	if err != nil {
		return err
	}
	plans, err := h.Svc.ListBlowdownPlans()
	if err != nil {
		return err
	}
	items, total := paginateTail(plans, limit, offset)
	setPageHeaders(w, total, limit, offset)
	respondOK(w, r, items)
	return nil
}
