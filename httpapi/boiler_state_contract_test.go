package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"

	"example.com/boiler-energy-efficiency-service/domain"
)

func createRunningBoiler(t *testing.T, h http.Handler) string {
	t.Helper()
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
	return boiler.ID
}

// TestBoilerPhantomStatusRejected400 不存在的状态作为迁移目标应返回 400。
func TestBoilerPhantomStatusRejected400(t *testing.T) {
	h := newTestRouter(t)
	id := createRunningBoiler(t, h)
	rec := doJSON(t, h, http.MethodPost, "/api/boilers/"+id+"/transition", map[string]any{"target": "idle"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("伪状态迁移应返回 400，实际 %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestAPI_TransitionsExcludeIllegalTarget 运行中锅炉的迁移列表不应包含直接停炉。
func TestAPI_TransitionsExcludeIllegalTarget(t *testing.T) {
	h := newTestRouter(t)
	id := createRunningBoiler(t, h)
	rec := doJSON(t, h, http.MethodGet, "/api/boilers/"+id+"/transitions", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("迁移列表应返回 200，实际 %d", rec.Code)
	}
	var body struct {
		Data []string `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	for _, s := range body.Data {
		if s == string(domain.BoilerStatusStopped) {
			t.Fatalf("运行中锅炉迁移列表不应包含直接停炉: %v", body.Data)
		}
	}
}

// TestAPI_OverviewTransitionsExcludeIllegal 总览中运行锅炉的可迁移状态不应包含直接停炉。
func TestAPI_OverviewTransitionsExcludeIllegal(t *testing.T) {
	h := newTestRouter(t)
	id := createRunningBoiler(t, h)
	rec := doJSON(t, h, http.MethodGet, "/api/overview", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("总览应返回 200，实际 %d", rec.Code)
	}
	var body struct {
		Data struct {
			Boilers []struct {
				Boiler      struct {
					ID string `json:"id"`
				} `json:"boiler"`
				Transitions []string `json:"transitions"`
			} `json:"boilers"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	for _, b := range body.Data.Boilers {
		if b.Boiler.ID != id {
			continue
		}
		for _, s := range b.Transitions {
			if s == string(domain.BoilerStatusStopped) {
				t.Fatalf("总览中运行锅炉可迁移状态不应包含直接停炉: %v", b.Transitions)
			}
		}
		return
	}
	t.Fatalf("总览中未找到锅炉 %s", id)
}
