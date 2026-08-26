package middleware

import (
	"net/http"
	"strings"
)

// SecurityHeaders 为所有响应添加企业级安全头，并对 API 响应禁用缓存。
// 注意：不添加严格 CSP，以免破坏前端内联脚本。
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
			h.Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}
