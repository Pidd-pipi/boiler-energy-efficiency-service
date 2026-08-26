package store

import (
	"testing"

	"example.com/boiler-energy-efficiency-service/domain"
)

// TestStoreGetBoilerReturnsCopy GetBoiler 应返回独立副本，修改返回值不污染库内对象。
func TestStoreGetBoilerReturnsCopy(t *testing.T) {
	s, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	b := domain.NewBoiler("副本锅炉", domain.BoilerTypeSteam, 10, 2, fixedTime())
	if err := s.CreateBoiler(b); err != nil {
		t.Fatal(err)
	}
	got1, err := s.GetBoiler(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	got1.RunSecondsTotal = 999
	got2, err := s.GetBoiler(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got2.RunSecondsTotal == 999 {
		t.Fatalf("GetBoiler 应返回独立副本，实际与库内对象共享引用")
	}
}

// TestStoreListBoilersReturnsCopies ListBoilers 应返回独立副本。
func TestStoreListBoilersReturnsCopies(t *testing.T) {
	s, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	b := domain.NewBoiler("副本锅炉", domain.BoilerTypeSteam, 10, 2, fixedTime())
	if err := s.CreateBoiler(b); err != nil {
		t.Fatal(err)
	}
	list1, err := s.ListBoilers()
	if err != nil {
		t.Fatal(err)
	}
	if len(list1) != 1 {
		t.Fatalf("锅炉数量错误: %d", len(list1))
	}
	list1[0].RunSecondsSinceBlowdown = 777
	list2, err := s.ListBoilers()
	if err != nil {
		t.Fatal(err)
	}
	if list2[0].RunSecondsSinceBlowdown == 777 {
		t.Fatalf("ListBoilers 应返回独立副本，实际与库内对象共享引用")
	}
}

// TestStoreUpdateBoilerStoresCopy UpdateBoiler 后调用方再改对象不得影响库内状态。
func TestStoreUpdateBoilerStoresCopy(t *testing.T) {
	s, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	b := domain.NewBoiler("副本锅炉", domain.BoilerTypeSteam, 10, 2, fixedTime())
	if err := s.CreateBoiler(b); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetBoiler(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	got.RunSecondsTotal = 500
	if err := s.UpdateBoiler(got); err != nil {
		t.Fatal(err)
	}
	got.RunSecondsTotal = 999
	after, err := s.GetBoiler(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.RunSecondsTotal == 999 {
		t.Fatalf("UpdateBoiler 后调用方再修改不应影响库内状态，实际被改到 %v", after.RunSecondsTotal)
	}
}
