package domain

import (
	"testing"
	"time"
)

// TestBoilerRunningDirectStopRejected 运行中的锅炉不能直接停炉。
func TestBoilerRunningDirectStopRejected(t *testing.T) {
	b := NewBoiler("1#锅炉", BoilerTypeSteam, 10, 2, time.Now())
	b.Status = BoilerStatusRunning
	if err := b.TransitionTo(BoilerStatusStopped, time.Now()); err == nil {
		t.Fatalf("运行中直接停炉应被状态机拒绝")
	}
}

// TestBoilerAllStatusesNoPhantom 全部状态枚举不应包含无法迁移的伪状态。
func TestBoilerAllStatusesNoPhantom(t *testing.T) {
	for _, s := range AllBoilerStatuses() {
		if s == "idle" {
			t.Fatalf("全部状态枚举不应包含伪状态 idle")
		}
	}
}
