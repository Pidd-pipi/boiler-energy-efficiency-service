package httpapi

import (
	"net/http"

	"example.com/boiler-energy-efficiency-service/domain"
	"example.com/boiler-energy-efficiency-service/middleware"
	"example.com/boiler-energy-efficiency-service/service"
)

// RunHandler 运行数据上报与查询。
type RunHandler struct {
	Svc *service.Services
}

// Ingest POST /api/boilers/{id}/run 运行数据上报。
// 主链路：采集 -> 能效计算 -> 工况诊断 -> 告警判定。
func (h *RunHandler) Ingest(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	if id == "" {
		return domain.NewError(domain.KindInvalidInput, "缺少路径参数 id")
	}
	var in service.RunIngestInput
	if err := decodeJSON(r, &in); err != nil {
		return err
	}
	if err := validateRunIngestInput(&in); err != nil {
		return err
	}
	if in.Operator == "" {
		in.Operator = r.Header.Get("X-Operator")
	}
	result, err := h.Svc.IngestRunData(middleware.TraceID(r.Context()), id, in)
	if err != nil {
		return err
	}
	respondCreated(w, r, result)
	return nil
}

// List GET /api/boilers/{id}/run?limit=N&offset=M 运行数据列表。
func (h *RunHandler) List(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	if id == "" {
		return domain.NewError(domain.KindInvalidInput, "缺少路径参数 id")
	}
	limit, offset, err := parseLimitOffset(r, 50, 200)
	if err != nil {
		return err
	}
	list, err := h.Svc.Store.ListRunDataByBoiler(id, 0)
	if err != nil {
		return err
	}
	items, total := paginateTail(list, limit, offset)
	setPageHeaders(w, total, limit, offset)
	respondOK(w, r, items)
	return nil
}

// validateRunIngestInput 校验上报入参的数值范围，非法值返回 400。
func validateRunIngestInput(in *service.RunIngestInput) error {
	bad := func(field string) error {
		return domain.NewError(domain.KindInvalidInput, "%s 不能为负数", field)
	}
	if in.FuelAmount < 0 {
		return bad("fuel_amount")
	}
	if in.SteamOutput < 0 {
		return bad("steam_output")
	}
	if in.FeedWaterFlow < 0 {
		return bad("feed_water_flow")
	}
	if in.FlueGasTemp < 0 {
		return bad("flue_gas_temp")
	}
	if in.OxygenContent < 0 || in.OxygenContent >= 21 {
		return domain.NewError(domain.KindInvalidInput, "oxygen_content 必须在 [0,21) 区间内：%v", in.OxygenContent)
	}
	if in.SteamPressure < 0 {
		return bad("steam_pressure")
	}
	if in.WaterLevel < 0 || in.WaterLevel > 100 {
		return domain.NewError(domain.KindInvalidInput, "water_level 必须在 [0,100] 区间内：%v", in.WaterLevel)
	}
	if in.SupplyWaterTemp < 0 {
		return bad("supply_water_temp")
	}
	if in.ReturnWaterTemp < 0 {
		return bad("return_water_temp")
	}
	if in.IntervalMinutes < 0 {
		return bad("interval_minutes")
	}
	return nil
}
