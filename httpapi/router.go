package httpapi

import (
	"io/fs"
	"net/http"
	"strings"

	"example.com/boiler-energy-efficiency-service/middleware"
	"example.com/boiler-energy-efficiency-service/service"
)

// NewRouter 组装全部路由与中间件链。
// webFS 为内嵌前端静态资源（web/ 目录，由 main.go 注入）。
func NewRouter(svc *service.Services, webFS fs.FS) http.Handler {
	mux := http.NewServeMux()

	// 健康检查。
	health := &HealthHandler{}
	mux.HandleFunc("GET /healthz", health.Healthz)
	mux.HandleFunc("GET /readyz", health.Readyz)
	mux.HandleFunc("GET /api/healthz", health.Healthz)

	// 锅炉台账。
	boiler := &BoilerHandler{Svc: svc}
	mux.HandleFunc("POST /api/boilers", handle(boiler.Create))
	mux.HandleFunc("GET /api/boilers", handle(boiler.List))
	mux.HandleFunc("GET /api/boilers/{id}", handle(boiler.Get))
	mux.HandleFunc("GET /api/boilers/{id}/transitions", handle(boiler.AllowedTransitions))
	mux.HandleFunc("POST /api/boilers/{id}/transition", handle(boiler.Transition))

	// 运行数据上报 + 查询。
	run := &RunHandler{Svc: svc}
	mux.HandleFunc("POST /api/boilers/{id}/run", handle(run.Ingest))
	mux.HandleFunc("GET /api/boilers/{id}/run", handle(run.List))

	// 能效记录。
	eff := &EfficiencyHandler{Svc: svc}
	mux.HandleFunc("GET /api/boilers/{id}/efficiency", handle(eff.List))

	// 燃烧工况诊断。
	diag := &DiagnosisHandler{Svc: svc}
	mux.HandleFunc("GET /api/boilers/{id}/diagnosis", handle(diag.ListByBoiler))
	mux.HandleFunc("GET /api/diagnosis", handle(diag.ListAll))

	// 运行告警。
	alert := &AlertHandler{Svc: svc}
	mux.HandleFunc("GET /api/alerts", handle(alert.List))
	mux.HandleFunc("POST /api/alerts/{id}/ack", handle(alert.Ack))
	mux.HandleFunc("POST /api/alerts/{id}/escalate", handle(alert.Escalate))
	mux.HandleFunc("POST /api/alerts/{id}/resolve", handle(alert.Resolve))

	// 排污管理。
	blow := &BlowdownHandler{Svc: svc}
	mux.HandleFunc("GET /api/boilers/{id}/blowdown", handle(blow.Get))
	mux.HandleFunc("POST /api/boilers/{id}/blowdown", handle(blow.Execute))
	mux.HandleFunc("GET /api/blowdown", handle(blow.ListPlans))

	// 运行日报。
	report := &ReportHandler{Svc: svc}
	mux.HandleFunc("GET /api/boilers/{id}/daily-report", handle(report.GetByBoiler))
	mux.HandleFunc("GET /api/daily-reports", handle(report.List))
	mux.HandleFunc("POST /api/daily-reports/generate", handle(report.GenerateAll))

	// 总览。
	overview := &OverviewHandler{Svc: svc}
	mux.HandleFunc("GET /api/overview", handle(overview.Get))

	// 审计日志。
	audit := &AuditHandler{Svc: svc}
	mux.HandleFunc("GET /api/audit-logs", handle(audit.List))

	// 前端静态资源（SPA）。
	mux.Handle("/", serveWeb(webFS))

	// 中间件链：requestID -> securityHeaders -> audit(审计+访问日志) -> errorHandler -> mux。
	// errorHandler 放在最内层，确保 handler 的 panic 被恢复后外层仍能记录状态码。
	chain := middleware.Chain(
		mux,
		middleware.RequestID,
		middleware.SecurityHeaders,
		middleware.AuditLogger(svc),
		middleware.ErrorHandler,
	)
	return chain
}

// serveWeb 提供内嵌静态资源，并对未知路径回退到 index.html（SPA 路由）。
func serveWeb(webFS fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(webFS))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			p = "index.html"
		}
		if _, err := fs.Stat(webFS, p); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}
		// SPA 回退：保证 /boilers/xxx 等前端路由直开可访问。
		http.ServeFileFS(w, r, webFS, "index.html")
	})
}
