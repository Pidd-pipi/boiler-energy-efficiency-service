package httpapi

import (
	"context"
	"net/http"
	"testing"
	"testing/fstest"
	"time"

	"example.com/boiler-energy-efficiency-service/config"
	"example.com/boiler-energy-efficiency-service/service"
	"example.com/boiler-energy-efficiency-service/store"
)

func newCtxRouter(t *testing.T) http.Handler {
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

func createCtxBoiler2(t *testing.T, h http.Handler) string {
	t.Helper()
	rec := doJSON(t, h, http.MethodPost, "/api/boilers", map[string]any{
		"name": "上下文锅炉", "type": "steam", "rated_capacity": 10, "water_hardness": 2,
	})
	boiler := decodeBody[struct {
		ID string `json:"id"`
	}](t, rec)
	if rec.Code != http.StatusCreated {
		t.Fatalf("创建锅炉失败: %d %s", rec.Code, rec.Body.String())
	}
	return boiler.ID
}

// TestAPI_IngestAbortsOnCanceledRequest 请求已取消时上报应中止，不再继续处理。
func TestAPI_IngestAbortsOnCanceledRequest(t *testing.T) {
	h := newCtxRouter(t)
	id := createCtxBoiler2(t, h)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := newCanceledRequest(t, http.MethodPost, "/api/boilers/"+id+"/run", ctx)
	rec := doJSONRequest(t, h, req)
	if rec.Code == http.StatusCreated {
		t.Fatalf("已取消的请求不应继续处理上报，实际 201")
	}
}

// TestAPI_RunListAbortsOnCanceledRequest 请求已取消时运行数据列表查询应中止。
func TestAPI_RunListAbortsOnCanceledRequest(t *testing.T) {
	h := newCtxRouter(t)
	id := createCtxBoiler2(t, h)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := newCanceledRequest(t, http.MethodGet, "/api/boilers/"+id+"/run", ctx)
	rec := doJSONRequest(t, h, req)
	if rec.Code == http.StatusOK {
		t.Fatalf("已取消的请求不应继续返回列表，实际 200")
	}
}
