package httpapi

import (
	"net/http"
	"time"
)

// HealthHandler 健康检查。
type HealthHandler struct{}

// healthStatus 健康检查响应。
type healthStatus struct {
	Status    string    `json:"status"`
	Service   string    `json:"service"`
	Timestamp time.Time `json:"timestamp"`
}

// Healthz GET /healthz 与 GET /api/healthz。
func (h *HealthHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	respondOK(w, r, healthStatus{
		Status:    "ok",
		Service:   "boiler-energy-efficiency-service",
		Timestamp: time.Now(),
	})
}

// Readyz GET /readyz 就绪检查。
func (h *HealthHandler) Readyz(w http.ResponseWriter, r *http.Request) {
	respondOK(w, r, healthStatus{
		Status:    "ok",
		Service:   "boiler-energy-efficiency-service",
		Timestamp: time.Now(),
	})
}
