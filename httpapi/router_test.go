package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"example.com/boiler-energy-efficiency-service/config"
	"example.com/boiler-energy-efficiency-service/service"
	"example.com/boiler-energy-efficiency-service/store"
)

// newTestRouter 构造真实路由（内嵌空 web 目录）。
func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.SeedDemo = false
	st, err := store.New(store.Options{})
	if err != nil {
		t.Fatal(err)
	}
	svc := service.NewServices(cfg, st)
	svc.Now = func() time.Time {
		return time.Date(2026, 8, 25, 10, 0, 0, 0, time.Local)
	}
	webFS := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html>test</html>")},
	}
	return NewRouter(svc, webFS)
}

func doJSON(t *testing.T, h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeBody[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var out struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    T      `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("解析响应失败: %v body=%s", err, rec.Body.String())
	}
	return out.Data
}

func TestHealthz(t *testing.T) {
	h := newTestRouter(t)
	rec := doJSON(t, h, http.MethodGet, "/healthz", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz 状态码: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatalf("healthz 内容错误: %s", rec.Body.String())
	}
	// SPA 首页回退。
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec2.Code != http.StatusOK || !strings.Contains(rec2.Body.String(), "<html>test</html>") {
		t.Fatalf("首页回退失败: %d %s", rec2.Code, rec2.Body.String())
	}
}

func TestAPIFlow_CreateTransitionRunOverview(t *testing.T) {
	h := newTestRouter(t)

	// 1) 创建锅炉。
	rec := doJSON(t, h, http.MethodPost, "/api/boilers", map[string]any{
		"name": "1#蒸汽锅炉", "type": "steam", "rated_capacity": 10, "water_hardness": 2, "operator": "tester",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("创建锅炉失败: %d %s", rec.Code, rec.Body.String())
	}
	boiler := decodeBody[map[string]any](t, rec)
	id, _ := boiler["id"].(string)
	if id == "" {
		t.Fatal("锅炉 ID 为空")
	}

	// 2) 状态迁移：stopped -> running 应被状态机拒绝。
	rec = doJSON(t, h, http.MethodPost, "/api/boilers/"+id+"/transition", map[string]any{"target": "running", "operator": "tester"})
	if rec.Code != http.StatusConflict {
		t.Fatalf("直接迁移到 running 应 409: %d %s", rec.Code, rec.Body.String())
	}

	// 3) 合法迁移链：starting -> running。
	for _, target := range []string{"starting", "running"} {
		rec = doJSON(t, h, http.MethodPost, "/api/boilers/"+id+"/transition", map[string]any{"target": target, "operator": "tester"})
		if rec.Code != http.StatusOK {
			t.Fatalf("迁移到 %s 失败: %d %s", target, rec.Code, rec.Body.String())
		}
	}

	// 4) 上报运行数据。
	rec = doJSON(t, h, http.MethodPost, "/api/boilers/"+id+"/run", map[string]any{
		"fuel_amount": 900, "steam_output": 8.5, "feed_water_flow": 9,
		"flue_gas_temp": 152, "oxygen_content": 8, "steam_pressure": 1.2, "water_level": 60,
		"interval_minutes": 5, "operator": "tester",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("上报运行数据失败: %d %s", rec.Code, rec.Body.String())
	}
	ingest := decodeBody[map[string]any](t, rec)
	if ingest["efficiency"] == nil {
		t.Fatalf("应生成能效记录: %s", rec.Body.String())
	}

	// 5) 能效 / 诊断 / 日报接口。
	for _, path := range []string{
		"/api/boilers/" + id + "/efficiency",
		"/api/boilers/" + id + "/diagnosis",
		"/api/boilers/" + id + "/daily-report?date=2026-08-25",
		"/api/overview",
		"/api/blowdown",
	} {
		rec = doJSON(t, h, http.MethodGet, path, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s 失败: %d %s", path, rec.Code, rec.Body.String())
		}
	}

	// 6) 运行中直接停炉应 409（先压火）。
	rec = doJSON(t, h, http.MethodPost, "/api/boilers/"+id+"/transition", map[string]any{"target": "stopped", "operator": "tester"})
	if rec.Code != http.StatusConflict {
		t.Fatalf("运行中直接停炉应 409: %d %s", rec.Code, rec.Body.String())
	}

	// 7) 审计日志存在。
	rec = doJSON(t, h, http.MethodGet, "/api/audit-logs?limit=50", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("审计日志失败: %d", rec.Code)
	}
	audits := decodeBody[[]map[string]any](t, rec)
	if len(audits) == 0 {
		t.Fatal("审计日志应为空")
	}
}

func TestAPIFlow_AlertAck(t *testing.T) {
	h := newTestRouter(t)
	rec := doJSON(t, h, http.MethodPost, "/api/boilers", map[string]any{
		"name": "告警锅炉", "type": "steam", "rated_capacity": 10, "water_hardness": 2,
	})
	id, _ := decodeBody[map[string]any](t, rec)["id"].(string)
	for _, target := range []string{"starting", "running"} {
		doJSON(t, h, http.MethodPost, "/api/boilers/"+id+"/transition", map[string]any{"target": target})
	}
	// 基线。
	doJSON(t, h, http.MethodPost, "/api/boilers/"+id+"/run", map[string]any{
		"fuel_amount": 900, "steam_output": 8.5, "flue_gas_temp": 150, "oxygen_content": 8, "interval_minutes": 5,
	})
	// 异常：排烟突升 + 氧异常。
	rec = doJSON(t, h, http.MethodPost, "/api/boilers/"+id+"/run", map[string]any{
		"fuel_amount": 900, "steam_output": 8.5, "flue_gas_temp": 200, "oxygen_content": 2.5, "interval_minutes": 5,
	})
	ingest := decodeBody[map[string]any](t, rec)
	alerts, _ := ingest["alerts"].([]any)
	if len(alerts) == 0 {
		t.Fatal("应生成告警")
	}
	first := alerts[0].(map[string]any)
	alertID, _ := first["id"].(string)

	rec = doJSON(t, h, http.MethodPost, "/api/alerts/"+alertID+"/ack", map[string]any{"operator": "web", "note": "确认"})
	if rec.Code != http.StatusOK {
		t.Fatalf("确认告警失败: %d %s", rec.Code, rec.Body.String())
	}
	// 确认后告警列表该条状态为 acknowledged。
	rec = doJSON(t, h, http.MethodGet, "/api/alerts?status=acknowledged", nil)
	ackd := decodeBody[[]map[string]any](t, rec)
	found := false
	for _, a := range ackd {
		if a["id"] == alertID {
			found = true
		}
	}
	if !found {
		t.Fatal("确认后的告警应出现在已确认列表")
	}
}

func TestAPIFlow_BlowdownAndReport(t *testing.T) {
	h := newTestRouter(t)
	rec := doJSON(t, h, http.MethodPost, "/api/boilers", map[string]any{
		"name": "排污锅炉", "type": "steam", "rated_capacity": 10, "water_hardness": 2,
	})
	id, _ := decodeBody[map[string]any](t, rec)["id"].(string)
	for _, target := range []string{"starting", "running"} {
		doJSON(t, h, http.MethodPost, "/api/boilers/"+id+"/transition", map[string]any{"target": target})
	}
	// 累计运行 10 小时。
	for i := 0; i < 10; i++ {
		doJSON(t, h, http.MethodPost, "/api/boilers/"+id+"/run", map[string]any{
			"fuel_amount": 900, "steam_output": 8.5, "oxygen_content": 8, "interval_minutes": 60,
		})
	}
	// 执行排污。
	rec = doJSON(t, h, http.MethodPost, "/api/boilers/"+id+"/blowdown", map[string]any{
		"operator": "web", "duration_min": 8, "note": "定期排污",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("排污执行失败: %d %s", rec.Code, rec.Body.String())
	}
	// 计划累计运行归零。
	rec = doJSON(t, h, http.MethodGet, "/api/boilers/"+id+"/blowdown", nil)
	detail := decodeBody[map[string]any](t, rec)
	plan := detail["plan"].(map[string]any)
	if plan["accumulated_run_hours"].(float64) != 0 {
		t.Fatalf("排污后累计运行应归零: %v", plan["accumulated_run_hours"])
	}
	// 日报：先为全部锅炉生成，再查询列表。
	rec = doJSON(t, h, http.MethodPost, "/api/daily-reports/generate?date=2026-08-25", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("生成日报失败: %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodGet, "/api/daily-reports?date=2026-08-25", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("日报列表失败: %d %s", rec.Code, rec.Body.String())
	}
	reports := decodeBody[[]map[string]any](t, rec)
	if len(reports) == 0 {
		t.Fatal("应生成日报")
	}
}
