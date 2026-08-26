package service

import (
	"testing"

	"example.com/boiler-energy-efficiency-service/domain"
)

// TestIngestRunData_GeneratesAlert 上报触发告警条件的运行数据应生成告警。
func TestIngestRunData_GeneratesAlert(t *testing.T) {
	svc, _ := newTestServices(t)
	b := mustCreateSteamBoiler(t, svc)

	res, err := svc.IngestRunData("trace-alert", b.ID, RunIngestInput{
		FuelAmount:     900,
		SteamOutput:    8.5,
		FeedWaterFlow:  9,
		FlueGasTemp:    150,
		OxygenContent:  2.0,
		SteamPressure:  1.2,
		WaterLevel:     60,
		IntervalMinutes: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, a := range res.Alerts {
		if a.Type == domain.AlertOxygenAbnormal {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("氧含量偏低应生成告警，实际告警列表: %+v", res.Alerts)
	}
}
