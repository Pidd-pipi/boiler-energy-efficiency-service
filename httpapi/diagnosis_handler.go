package httpapi

import (
	"net/http"

	"example.com/boiler-energy-efficiency-service/domain"
	"example.com/boiler-energy-efficiency-service/service"
)

// DiagnosisHandler 燃烧工况诊断。
type DiagnosisHandler struct {
	Svc *service.Services
}

// ListByBoiler GET /api/boilers/{id}/diagnosis?limit=N&offset=M 某锅炉诊断列表。
func (h *DiagnosisHandler) ListByBoiler(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	if id == "" {
		return domain.NewError(domain.KindInvalidInput, "缺少路径参数 id")
	}
	limit, offset, err := parseLimitOffset(r, 50, 500)
	if err != nil {
		return err
	}
	list, err := h.Svc.ListDiagnosisByBoiler(id, 0)
	if err != nil {
		return err
	}
	items, total := paginateTail(list, limit, offset)
	setPageHeaders(w, total, limit, offset)
	respondOK(w, r, items)
	return nil
}

// diagnosisListResponse 全部诊断 + 结论统计。
type diagnosisListResponse struct {
	Items   []any `json:"items"`
	Summary any   `json:"summary"`
	Total   int   `json:"total"`
	Limit   int   `json:"limit"`
	Offset  int   `json:"offset"`
}

// ListAll GET /api/diagnosis?limit=N&offset=M 全部诊断与结论统计。
func (h *DiagnosisHandler) ListAll(w http.ResponseWriter, r *http.Request) error {
	limit, offset, err := parseLimitOffset(r, 100, 500)
	if err != nil {
		return err
	}
	list, err := h.Svc.ListDiagnosis(0)
	if err != nil {
		return err
	}
	window, total := paginateTail(list, limit, offset)
	summary, err := h.Svc.DiagnosisSummary(limit)
	if err != nil {
		return err
	}
	items := make([]any, 0, len(window))
	for _, c := range window {
		items = append(items, c)
	}
	setPageHeaders(w, total, limit, offset)
	respondOK(w, r, diagnosisListResponse{Items: items, Summary: summary, Total: total, Limit: limit, Offset: offset})
	return nil
}
