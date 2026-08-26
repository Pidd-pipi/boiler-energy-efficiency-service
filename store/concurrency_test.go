package store

import (
	"sync"
	"testing"

	"example.com/boiler-energy-efficiency-service/domain"
)

// TestStoreConcurrentAccess 并发执行创建/读取/更新/列表操作，
// 与 go test -race 配合验证 Store 无数据竞争。
func TestStoreConcurrentAccess(t *testing.T) {
	s, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	const goroutines = 12
	const iterations = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(g int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				b := domain.NewBoiler("并发锅炉", domain.BoilerTypeSteam, 10, 2, fixedTime())
				if err := s.CreateBoiler(b); err != nil {
					t.Errorf("CreateBoiler: %v", err)
					return
				}
				got, err := s.GetBoiler(b.ID)
				if err != nil {
					t.Errorf("GetBoiler: %v", err)
					return
				}
				got.Status = domain.BoilerStatusStarting
				if err := s.UpdateBoiler(got); err != nil {
					t.Errorf("UpdateBoiler: %v", err)
					return
				}
				if _, err := s.ListBoilers(); err != nil {
					t.Errorf("ListBoilers: %v", err)
					return
				}
			}
		}(g)
	}
	wg.Wait()
	if got := s.CountBoilers(); got != goroutines*iterations {
		t.Fatalf("锅炉数量错误: %d", got)
	}
}
