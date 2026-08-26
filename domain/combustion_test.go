package domain

import (
	"testing"
	"time"

	"example.com/boiler-energy-efficiency-service/config"
)

func TestExcessAirCoefficient(t *testing.T) {
	cases := []struct {
		oxygen float64
		want   float64
	}{
		{8, 21.0 / 13.0},
		{0, 1},
		{5, 21.0 / 16.0},
	}
	for _, c := range cases {
		got, err := ExcessAirCoefficient(c.oxygen)
		if err != nil {
			t.Fatalf("氧含量 %v 计算失败: %v", c.oxygen, err)
		}
		if round2(got) != round2(c.want) {
			t.Fatalf("氧含量 %v: got=%v want=%v", c.oxygen, got, c.want)
		}
	}
	if _, err := ExcessAirCoefficient(21); err == nil {
		t.Fatal("氧含量 21 应报错")
	}
	if _, err := ExcessAirCoefficient(-1); err == nil {
		t.Fatal("负氧含量应报错")
	}
}

func TestDiagnoseCombustion(t *testing.T) {
	cfg := config.Default()
	cfg.ExcessAirLow = 1.2
	cfg.ExcessAirHigh = 1.8

	cases := []struct {
		oxygen float64
		want   DiagnosisResult
	}{
		{10, ResultExcessAir}, // α=21/11≈1.91
		{8, ResultNormal},     // α=21/13≈1.62
		{4, ResultNormal},     // α=21/17≈1.24
		{2, ResultUnderAir},   // α=21/19≈1.11
		{19, ResultExcessAir}, // α=21/2=10.5
	}
	for _, c := range cases {
		rd := NewRunData("blr_000001", time.Now())
		rd.OxygenContent = c.oxygen
		st, err := DiagnoseCombustion(cfg, rd)
		if err != nil {
			t.Fatalf("氧含量 %v 诊断失败: %v", c.oxygen, err)
		}
		if st.Result != c.want {
			t.Fatalf("氧含量 %v: got=%s want=%s", c.oxygen, st.Result, c.want)
		}
		if st.Suggestion == "" {
			t.Fatalf("氧含量 %v 缺少调整建议", c.oxygen)
		}
	}
}
