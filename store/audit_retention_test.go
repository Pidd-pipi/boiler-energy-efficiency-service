package store

import (
	"fmt"
	"testing"

	"example.com/boiler-energy-efficiency-service/domain"
)

// TestAuditLogBoundedAfterAppend 审计日志追加超过上限后自动淘汰最旧记录。
func TestAuditLogBoundedAfterAppend(t *testing.T) {
	s, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	now := fixedTime()
	for i := 0; i < 6000; i++ {
		e := domain.NewAuditEntry(fmt.Sprintf("t%d", i), domain.ActionCreateBoiler, "boiler", fmt.Sprintf("b%d", i), "op", "create", now)
		if err := s.AppendAudit(e); err != nil {
			t.Fatal(err)
		}
	}
	if n := s.CountAudit(); n > auditRetentionMax {
		t.Fatalf("审计日志应被限制在上限内，实际 %d", n)
	}
}

// TestAuditFloodDeduplicated 连续相同的接口请求审计应去重，不刷屏。
func TestAuditFloodDeduplicated(t *testing.T) {
	s, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	now := fixedTime()
	for i := 0; i < 100; i++ {
		e := domain.NewAuditEntry("t", domain.ActionAPIRequest, "http", "/api/boilers/b1/run", "op", "POST /api/boilers/b1/run", now)
		if err := s.AppendAudit(e); err != nil {
			t.Fatal(err)
		}
	}
	if n := s.CountAudit(); n > 5 {
		t.Fatalf("连续相同接口请求应去重，实际 %d 条", n)
	}
}
