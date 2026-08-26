package store

import "example.com/boiler-energy-efficiency-service/domain"

// 本文件提供领域实体的浅拷贝，保证仓库 getter 返回副本而非共享指针，
// 从而避免调用方在持锁外修改对象造成数据竞争。

func cloneRunData(r *domain.RunData) *domain.RunData {
	if r == nil {
		return nil
	}
	c := *r
	return &c
}

func cloneEfficiency(e *domain.EfficiencyRecord) *domain.EfficiencyRecord {
	if e == nil {
		return nil
	}
	c := *e
	return &c
}

func cloneCombustion(c *domain.CombustionStatus) *domain.CombustionStatus {
	if c == nil {
		return nil
	}
	cp := *c
	return &cp
}

func cloneAlert(a *domain.RunAlert) *domain.RunAlert {
	if a == nil {
		return nil
	}
	c := *a
	return &c
}

func cloneBlowdown(b *domain.BlowdownRecord) *domain.BlowdownRecord {
	if b == nil {
		return nil
	}
	c := *b
	return &c
}

func cloneReport(r *domain.DailyReport) *domain.DailyReport {
	if r == nil {
		return nil
	}
	c := *r
	return &c
}

func cloneAudit(e *domain.AuditEntry) *domain.AuditEntry {
	if e == nil {
		return nil
	}
	c := *e
	return &c
}
