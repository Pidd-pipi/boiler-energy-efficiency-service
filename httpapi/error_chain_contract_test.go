package httpapi

import (
	"net/http"
	"testing"
)

// TestAPI_TransitionMissingBoiler404 不存在的锅炉执行状态迁移应返回 404 而非 500。
func TestAPI_TransitionMissingBoiler404(t *testing.T) {
	h := newTestRouter(t)
	rec := doJSON(t, h, http.MethodPost, "/api/boilers/blr_missing/transition", map[string]any{
		"target": "running",
	})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("迁移不存在锅炉应返回 404，实际 %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestAPI_AllowedTransitionsMissingBoiler404 不存在的锅炉查询迁移目标应返回 404。
func TestAPI_AllowedTransitionsMissingBoiler404(t *testing.T) {
	h := newTestRouter(t)
	rec := doJSON(t, h, http.MethodGet, "/api/boilers/blr_missing/transitions", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("查询不存在锅炉迁移目标应返回 404，实际 %d", rec.Code)
	}
}

// TestAPI_AckAlertMissing404 不存在的告警确认应返回 404。
func TestAPI_AckAlertMissing404(t *testing.T) {
	h := newTestRouter(t)
	rec := doJSON(t, h, http.MethodPost, "/api/alerts/alt_missing/ack", map[string]any{
		"operator": "tester",
		"note":     "test",
	})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("确认不存在告警应返回 404，实际 %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestAPI_ResolveAlertMissing404 不存在的告警处置应返回 404。
func TestAPI_ResolveAlertMissing404(t *testing.T) {
	h := newTestRouter(t)
	rec := doJSON(t, h, http.MethodPost, "/api/alerts/alt_missing/resolve", map[string]any{
		"operator": "tester",
		"note":     "test",
	})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("处置不存在告警应返回 404，实际 %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestAPI_EscalateAlertMissing404 不存在的告警升级应返回 404。
func TestAPI_EscalateAlertMissing404(t *testing.T) {
	h := newTestRouter(t)
	rec := doJSON(t, h, http.MethodPost, "/api/alerts/alt_missing/escalate", map[string]any{
		"operator": "tester",
	})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("升级不存在告警应返回 404，实际 %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestAPI_TransitionIllegal409 运行中锅炉直接停炉等非法迁移应返回 409 而非 500。
func TestAPI_TransitionIllegal409(t *testing.T) {
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
	// 运行中直接停炉：应被状态机拒绝为 409。
	rec = doJSON(t, h, http.MethodPost, "/api/boilers/"+boiler.ID+"/transition", map[string]any{"target": "stopped"})
	if rec.Code != http.StatusConflict {
		t.Fatalf("非法迁移应返回 409，实际 %d body=%s", rec.Code, rec.Body.String())
	}
}
