package httpapi

import (
	"net/http"

	"example.com/boiler-energy-efficiency-service/domain"
	"example.com/boiler-energy-efficiency-service/middleware"
	"example.com/boiler-energy-efficiency-service/service"
)

// BoilerHandler 锅炉台账与状态迁移。
type BoilerHandler struct {
	Svc *service.Services
}

// Create POST /api/boilers 创建锅炉。
func (h *BoilerHandler) Create(w http.ResponseWriter, r *http.Request) error {
	var in service.CreateBoilerInput
	if err := decodeJSON(r, &in); err != nil {
		return err
	}
	if in.Operator == "" {
		in.Operator = r.Header.Get("X-Operator")
	}
	b, err := h.Svc.CreateBoiler(middleware.TraceID(r.Context()), in)
	if err != nil {
		return err
	}
	respondCreated(w, r, b)
	return nil
}

// List GET /api/boilers 锅炉列表（支持 limit/offset，total 见 X-Total-Count）。
func (h *BoilerHandler) List(w http.ResponseWriter, r *http.Request) error {
	limit, offset, err := parseLimitOffset(r, 100, 500)
	if err != nil {
		return err
	}
	list, err := h.Svc.ListBoilers()
	if err != nil {
		return err
	}
	items, total := paginateTail(list, limit, offset)
	setPageHeaders(w, total, limit, offset)
	respondOK(w, r, items)
	return nil
}

// Get GET /api/boilers/{id} 锅炉详情（含总览聚合）。
func (h *BoilerHandler) Get(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	if id == "" {
		return domain.NewError(domain.KindInvalidInput, "缺少路径参数 id")
	}
	ov, err := h.Svc.BoilerOverviewByID(id)
	if err != nil {
		return err
	}
	respondOK(w, r, ov)
	return nil
}

// AllowedTransitions GET /api/boilers/{id}/transitions 允许的迁移目标。
func (h *BoilerHandler) AllowedTransitions(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	if id == "" {
		return domain.NewError(domain.KindInvalidInput, "缺少路径参数 id")
	}
	targets, err := h.Svc.AllowedTargets(id)
	if err != nil {
		return err
	}
	respondOK(w, r, targets)
	return nil
}

// transitionRequest 状态迁移请求体。
type transitionRequest struct {
	Target   domain.BoilerStatus `json:"target"`
	Operator string              `json:"operator"`
}

// Transition POST /api/boilers/{id}/transition 状态迁移。
func (h *BoilerHandler) Transition(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	if id == "" {
		return domain.NewError(domain.KindInvalidInput, "缺少路径参数 id")
	}
	var req transitionRequest
	if err := decodeJSON(r, &req); err != nil {
		return err
	}
	if req.Operator == "" {
		req.Operator = r.Header.Get("X-Operator")
	}
	b, err := h.Svc.Transition(middleware.TraceID(r.Context()), id, req.Target, req.Operator)
	if err != nil {
		return err
	}
	respondMessage(w, r, "状态迁移成功", b)
	return nil
}
