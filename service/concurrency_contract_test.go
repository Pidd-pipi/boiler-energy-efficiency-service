package service

import (
	"sync"
	"testing"
)

// TestConcurrentIngestNoDataRace 两个 goroutine 并发向同一锅炉上报运行数据，
// 同时有一个只读 goroutine 持续读取台账快照，依赖 -race 检测锁外共享对象改写。
func TestConcurrentIngestNoDataRace(t *testing.T) {
	svc, st := newTestServices(t)
	b := mustCreateSteamBoiler(t, svc)

	list, err := st.ListBoilers()
	if err != nil || len(list) != 1 {
		t.Fatalf("锅炉列表异常: %v", err)
	}
	shared := list[0]

	stop := make(chan struct{})
	var rwg sync.WaitGroup
	rwg.Add(1)
	go func() {
		defer rwg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				_ = shared.RunSecondsTotal
				_ = shared.RunSecondsSinceBlowdown
				_ = shared.Status
			}
		}
	}()

	const workers = 2
	const iters = 120
	start := make(chan struct{})
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for i := 0; i < iters; i++ {
				if _, err := svc.IngestRunData("conc", b.ID, RunIngestInput{
					FuelAmount:     900,
					SteamOutput:    8.5,
					FeedWaterFlow:  9,
					FlueGasTemp:    150,
					OxygenContent:  8,
					SteamPressure:  1.2,
					WaterLevel:     60,
					IntervalMinutes: 5,
				}); err != nil {
					t.Errorf("IngestRunData: %v", err)
					return
				}
			}
		}()
	}
	close(start)
	wg.Wait()
	close(stop)
	rwg.Wait()
	if _, err := st.GetBoiler(b.ID); err != nil {
		t.Fatalf("读取锅炉失败: %v", err)
	}
}
