package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"example.com/boiler-energy-efficiency-service/domain"
	"example.com/boiler-energy-efficiency-service/service"
)

// AuditLogger 返回操作审计日志中间件。
// 每个 HTTP 请求都会在审计存储中留痕（Action=api_request），
// 并使用 slog 输出结构化访问日志（method path status duration trace_id）。
func AuditLogger(svc *service.Services) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w}
			next.ServeHTTP(sw, r)

			if sw.status == 0 {
				sw.status = http.StatusOK
			}
			traceID := TraceID(r.Context())
			operator := r.Header.Get("X-Operator")
			if operator == "" {
				operator = "anonymous"
			}
			duration := time.Since(start)

			detail := fmt.Sprintf("%s %s (remote=%s, duration=%v)",
				r.Method, r.URL.Path, r.RemoteAddr, duration.Round(time.Millisecond))
			_ = svc.Audit(traceID, domain.ActionAPIRequest, "http", r.URL.Path, operator, detail)

			slog.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", sw.status,
				"duration_ms", duration.Milliseconds(),
				"request_id", traceID,
				"remote", r.RemoteAddr,
			)
		})
	}
}
