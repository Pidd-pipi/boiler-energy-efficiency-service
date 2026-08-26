package httpapi

import (
	"net/http"
	"testing"
)

// TestAPI_IllegalTransition409 非法状态迁移应返回 409 而非 500。
func TestAPI_IllegalTransition409(t *testing.T) {
	h := newTestRouter(t)
	rec := doJSON(t, h, http.MethodPost, "/api/boilers", map[string]any{
		"name": "1#蒸汽锅炉", "type": "steam", "rated_capacity": 10, "water_hardness": 2,
	})
	boiler := decodeBody[struct {
		ID string `json:"id"`
	}](t, rec)
	if rec.Code != http.StatusCreated {
		t.Fatalf("创建锅炉失败: %d %s", rec.Code, rec.Body.String())
	}
	for _, s := range []string{"starting", "running"} {
		r2 := doJSON(t, h, http.MethodPost, "/api/boilers/"+boiler.ID+"/transition", map[string]any{"target": s})
		if r2.Code != http.StatusOK {
			t.Fatalf("迁移到 %s 失败: %d %s", s, r2.Code, r2.Body.String())
		}
	}
	rec = doJSON(t, h, http.MethodPost, "/api/boilers/"+boiler.ID+"/transition", map[string]any{"target": "stopped"})
	if rec.Code != http.StatusConflict {
		t.Fatalf("非法迁移应返回 409，实际 %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestAPI_BlowdownMissingBoiler404 不存在的锅炉执行排污应返回 404。
func TestAPI_BlowdownMissingBoiler404(t *testing.T) {
	h := newTestRouter(t)
	rec := doJSON(t, h, http.MethodPost, "/api/boilers/blr_missing/blowdown", map[string]any{
		"operator": "tester", "duration_min": 5,
	})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("不存在锅炉排污应返回 404，实际 %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestAPI_AckMissingAlert404 不存在的告警确认应返回 404。
func TestAPI_AckMissingAlert404(t *testing.T) {
	h := newTestRouter(t)
	rec := doJSON(t, h, http.MethodPost, "/api/alerts/alt_missing/ack", map[string]any{
		"operator": "tester", "note": "test",
	})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("不存在告警确认应返回 404，实际 %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestAPI_ReportMissingBoiler404 不存在的锅炉查日报应返回 404。
func TestAPI_ReportMissingBoiler404(t *testing.T) {
	h := newTestRouter(t)
	rec := doJSON(t, h, http.MethodGet, "/api/boilers/blr_missing/daily-report", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("不存在锅炉日报应返回 404，实际 %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestAPI_ResolveMissingAlert404 不存在的告警处置应返回 404。
func TestAPI_ResolveMissingAlert404(t *testing.T) {
	h := newTestRouter(t)
	rec := doJSON(t, h, http.MethodPost, "/api/alerts/alt_missing/resolve", map[string]any{
		"operator": "tester", "note": "test",
	})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("不存在告警处置应返回 404，实际 %d body=%s", rec.Code, rec.Body.String())
	}
}
