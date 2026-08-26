package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestErrorHandlerRecoversPanic(t *testing.T) {
	h := ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("panic 应返回 500: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "服务器内部错误") {
		t.Fatalf("panic 响应体错误: %s", rec.Body.String())
	}
}

func TestRequestIDSetsHeader(t *testing.T) {
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if TraceID(r.Context()) == "" {
			t.Fatal("trace id 不应为空")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", "req_custom")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if got := rec.Header().Get("X-Request-Id"); got != "req_custom" {
		t.Fatalf("应透传请求头中的 request id: %s", got)
	}
}

func TestSecurityHeaders(t *testing.T) {
	h := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/alerts", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	checks := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "no-referrer",
		"Permissions-Policy":     "camera=(), microphone=(), geolocation=()",
		"Cache-Control":          "no-store",
	}
	for k, want := range checks {
		if got := rec.Header().Get(k); got != want {
			t.Fatalf("安全头 %s 错误: %q", k, got)
		}
	}
}
