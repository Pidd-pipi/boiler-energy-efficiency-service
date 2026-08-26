package httpapi

import (
	"net/http"

	"example.com/boiler-energy-efficiency-service/domain"
	"example.com/boiler-energy-efficiency-service/service"
)

// OverviewHandler 锅炉房总览。
type OverviewHandler struct {
	Svc *service.Services
}

// overviewBoiler 总览中的锅炉视图。
type overviewBoiler struct {
	Boiler           *domain.Boiler           `json:"boiler"`
	LatestEfficiency *domain.EfficiencyRecord `json:"latest_efficiency,omitempty"`
	OpenAlertCount   int                      `json:"open_alert_count"`
	BlowdownDue      bool                     `json:"blowdown_due"`
	Transitions      []domain.BoilerStatus    `json:"transitions"`
}

// overviewResponse 总览响应。
type overviewResponse struct {
	Stats            *service.OverviewStats `json:"stats"`
	Boilers          []*overviewBoiler      `json:"boilers"`
	Alerts           []*domain.RunAlert     `json:"alerts"`
	DiagnosisSummary any                    `json:"diagnosis_summary"`
}

// Get GET /api/overview 总览聚合：锅炉状态 + 实时热效率 + 未确认告警。
func (h *OverviewHandler) Get(w http.ResponseWriter, r *http.Request) error {
	stats, err := h.Svc.Overview()
	if err != nil {
		return err
	}
	boilers, err := h.Svc.ListBoilers()
	if err != nil {
		return err
	}
	resp := &overviewResponse{Stats: stats, Boilers: []*overviewBoiler{}}
	for _, b := range boilers {
		item := &overviewBoiler{Boiler: b}
		if e, err := h.Svc.LatestEfficiencyByBoiler(b.ID); err == nil {
			item.LatestEfficiency = e
		}
		item.OpenAlertCount = h.Svc.Store.CountOpenAlerts(b.ID)
		item.BlowdownDue = h.Svc.PlanForBoiler(b).Due
		item.Transitions = domain.AllBoilerStatuses()
		resp.Boilers = append(resp.Boilers, item)
	}
	alerts, err := h.Svc.ListAlerts(storeAlertFilterOpen())
	if err != nil {
		return err
	}
	resp.Alerts = alerts
	summary, err := h.Svc.DiagnosisSummary(50)
	if err != nil {
		return err
	}
	resp.DiagnosisSummary = summary
	respondOK(w, r, resp)
	return nil
}
