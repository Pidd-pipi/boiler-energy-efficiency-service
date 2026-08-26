package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestAPI_OverviewEmptyEfficiencyOK 锅炉暂无能效记录时总览接口应正常返回 200。
func TestAPI_OverviewEmptyEfficiencyOK(t *testing.T) {
	h := newTestRouter(t)
	rec := doJSON(t, h, http.MethodPost, "/api/boilers", map[string]any{
		"name": "空数据锅炉", "type": "steam", "rated_capacity": 10, "water_hardness": 2,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("创建锅炉失败: %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodGet, "/api/overview", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("空能效数据总览应返回 200，实际 %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestAPI_BlowdownDetailNoRecordOK 锅炉尚无排污记录时排污详情应正常返回 200。
func TestAPI_BlowdownDetailNoRecordOK(t *testing.T) {
	h := newTestRouter(t)
	rec := doJSON(t, h, http.MethodPost, "/api/boilers", map[string]any{
		"name": "空数据锅炉", "type": "steam", "rated_capacity": 10, "water_hardness": 2,
	})
	boiler := decodeBody[struct {
		ID string `json:"id"`
	}](t, rec)
	if rec.Code != http.StatusCreated {
		t.Fatalf("创建锅炉失败: %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodGet, "/api/boilers/"+boiler.ID+"/blowdown", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("无排污记录时应返回 200，实际 %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestAPI_DailyReportAutoGenerates 锅炉尚无日报时查询应自动生成并返回报告数据。
func TestAPI_DailyReportAutoGenerates(t *testing.T) {
	h := newTestRouter(t)
	rec := doJSON(t, h, http.MethodPost, "/api/boilers", map[string]any{
		"name": "空数据锅炉", "type": "steam", "rated_capacity": 10, "water_hardness": 2,
	})
	boiler := decodeBody[struct {
		ID string `json:"id"`
	}](t, rec)
	if rec.Code != http.StatusCreated {
		t.Fatalf("创建锅炉失败: %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodGet, "/api/boilers/"+boiler.ID+"/daily-report", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("查询日报应返回 200，实际 %d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Data *struct {
			BoilerID string `json:"boiler_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if body.Data == nil || body.Data.BoilerID == "" {
		t.Fatalf("日报应被自动生成并返回数据，实际 body=%s", rec.Body.String())
	}
}
