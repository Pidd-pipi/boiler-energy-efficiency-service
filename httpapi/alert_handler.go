package httpapi

import (
	"fmt"
	"net/http"

	"example.com/boiler-energy-efficiency-service/domain"
	"example.com/boiler-energy-efficiency-service/middleware"
	"example.com/boiler-energy-efficiency-service/service"
	"example.com/boiler-energy-efficiency-service/store"
)

// AlertHandler 运行告警。
type AlertHandler struct {
	Svc *service.Services
}

// List GET /api/alerts?status=&boiler_id=&type=&limit=&offset= 告警列表。
func (h *AlertHandler) List(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()
	status := domain.AlertStatus(q.Get("status"))
	typ := domain.AlertType(q.Get("type"))
	if status != "" && !status.Valid() {
		return domain.NewError(domain.KindInvalidInput, "非法告警状态: %q", status)
	}
	if typ != "" && !typ.Valid() {
		return domain.NewError(domain.KindInvalidInput, "非法告警类型: %q", typ)
	}
	limit, offset, err := parseLimitOffset(r, 100, 500)
	if err != nil {
		return err
	}
	filter := store.AlertFilter{
		BoilerID: q.Get("boiler_id"),
		Status:   status,
		Type:     typ,
	}
	list, err := h.Svc.ListAlerts(filter)
	if err != nil {
		return err
	}
	items, total := paginateTail(list, limit, offset)
	setPageHeaders(w, total, limit, offset)
	respondOK(w, r, items)
	return nil
}

// ackRequest 确认告警请求体。
type ackRequest struct {
	Operator string `json:"operator"`
	Note     string `json:"note"`
}

// Ack POST /api/alerts/{id}/ack 确认告警。
func (h *AlertHandler) Ack(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	if id == "" {
		return domain.NewError(domain.KindInvalidInput, "缺少路径参数 id")
	}
	var req ackRequest
	if err := decodeJSON(r, &req); err != nil {
		return err
	}
	if req.Operator == "" {
		req.Operator = r.Header.Get("X-Operator")
	}
	a, err := h.Svc.AckAlert(middleware.TraceID(r.Context()), id, req.Operator, req.Note)
	if err != nil {
		return fmt.Errorf("确认告警失败: %v", err)
	}
	respondMessage(w, r, "告警已确认", a)
	return nil
}

// Escalate POST /api/alerts/{id}/escalate 升级告警。
func (h *AlertHandler) Escalate(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	if id == "" {
		return domain.NewError(domain.KindInvalidInput, "缺少路径参数 id")
	}
	var req ackRequest
	if err := decodeJSON(r, &req); err != nil {
		return err
	}
	if req.Operator == "" {
		req.Operator = r.Header.Get("X-Operator")
	}
	a, err := h.Svc.EscalateAlert(middleware.TraceID(r.Context()), id, req.Operator)
	if err != nil {
		return err
	}
	respondMessage(w, r, "告警已升级", a)
	return nil
}

// Resolve POST /api/alerts/{id}/resolve 处置关闭告警。
func (h *AlertHandler) Resolve(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	if id == "" {
		return domain.NewError(domain.KindInvalidInput, "缺少路径参数 id")
	}
	var req ackRequest
	if err := decodeJSON(r, &req); err != nil {
		return err
	}
	if req.Operator == "" {
		req.Operator = r.Header.Get("X-Operator")
	}
	a, err := h.Svc.ResolveAlert(middleware.TraceID(r.Context()), id, req.Operator, req.Note)
	if err != nil {
		return fmt.Errorf("处置告警失败: %v", err)
	}
	respondMessage(w, r, "告警已处置", a)
	return nil
}
