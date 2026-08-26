package httpapi

import (
	"fmt"
	"net/http"

	"example.com/boiler-energy-efficiency-service/domain"
	"example.com/boiler-energy-efficiency-service/service"
)

// ReportHandler 运行日报。
type ReportHandler struct {
	Svc *service.Services
}

// GetByBoiler GET /api/boilers/{id}/daily-report?date=YYYY-MM-DD 单锅炉日报。
func (h *ReportHandler) GetByBoiler(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	if id == "" {
		return domain.NewError(domain.KindInvalidInput, "缺少路径参数 id")
	}
	date := r.URL.Query().Get("date")
	if date == "" {
		date = h.Svc.DateOf(h.Svc.Now())
	}
	if _, err := service.ParseReportDate(date); err != nil {
		return err
	}
	report, err := h.Svc.GetDailyReport(id, date)
	if err != nil {
		return fmt.Errorf("获取日报失败: %v", err)
	}
	respondOK(w, r, report)
	return nil
}

// List GET /api/daily-reports?date=YYYY-MM-DD&limit=&offset= 日报列表。
func (h *ReportHandler) List(w http.ResponseWriter, r *http.Request) error {
	date := r.URL.Query().Get("date")
	if date == "" {
		date = h.Svc.DateOf(h.Svc.Now())
	}
	if _, err := service.ParseReportDate(date); err != nil {
		return err
	}
	limit, offset, err := parseLimitOffset(r, 100, 500)
	if err != nil {
		return err
	}
	list, err := h.Svc.ListDailyReports(date)
	if err != nil {
		return err
	}
	items, total := paginateTail(list, limit, offset)
	setPageHeaders(w, total, limit, offset)
	respondOK(w, r, items)
	return nil
}

// GenerateAll POST /api/daily-reports/generate?date=YYYY-MM-DD 为全部锅炉生成日报。
func (h *ReportHandler) GenerateAll(w http.ResponseWriter, r *http.Request) error {
	date := r.URL.Query().Get("date")
	if date == "" {
		date = h.Svc.DateOf(h.Svc.Now())
	}
	if _, err := service.ParseReportDate(date); err != nil {
		return err
	}
	list, err := h.Svc.GenerateAllDailyReports(date)
	if err != nil {
		return err
	}
	respondOK(w, r, list)
	return nil
}
