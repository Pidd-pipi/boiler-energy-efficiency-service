package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"

	"example.com/boiler-energy-efficiency-service/domain"
)

// TestAuditHealthCheckNotLogged 健康检查请求不应写入审计日志。
func TestAuditHealthCheckNotLogged(t *testing.T) {
	_, _, svc, h := newRetentionRouter(t)
	rec := doJSON(t, h, http.MethodGet, "/healthz", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("健康检查应返回 200，实际 %d", rec.Code)
	}
	// 只允许业务动作的审计（如果健康检查也被记录，说明刷屏）。
	if n := svc.Store.CountAudit(); n != 0 {
		t.Fatalf("健康检查不应写审计日志，实际 %d 条", n)
	}
}

// TestAPI_AuditListDefaultLimited 不带 limit 的审计列表默认只返回有限条数。
func TestAPI_AuditListDefaultLimited(t *testing.T) {
	_, _, svc, h := newRetentionRouter(t)
	for i := 0; i < 3000; i++ {
		if err := svc.Audit("t", domain.ActionCreateBoiler, "boiler", "b", "op", "create"); err != nil {
			t.Fatal(err)
		}
	}
	rec := doJSON(t, h, http.MethodGet, "/api/audit-logs", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("审计列表应返回 200，实际 %d", rec.Code)
	}
	var body struct {
		Data []any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if len(body.Data) > 200 {
		t.Fatalf("不带 limit 的审计列表默认应不超过 200 条，实际 %d", len(body.Data))
	}
}
