package service

import (
	"fmt"
	"time"

	"example.com/boiler-energy-efficiency-service/domain"
)

// ExecuteBlowdownInput 排污执行入参。
type ExecuteBlowdownInput struct {
	Operator    string  `json:"operator"`
	DurationMin float64 `json:"duration_min"`
	Note        string  `json:"note"`
}

// ExecuteBlowdown 执行排污：登记记录、重置排污计时并返回最新计划。
// 每次执行都写审计日志。
func (s *Services) ExecuteBlowdown(traceID, boilerID string, in ExecuteBlowdownInput) (*domain.BlowdownRecord, *domain.BlowdownPlan, error) {
	b, err := s.Store.GetBoiler(boilerID)
	if err != nil {
		return nil, nil, err
	}
	now := s.now()
	duration := in.DurationMin
	if duration <= 0 {
		duration = 5 // 默认 5 分钟
	}
	no := s.Store.CountBlowdownByBoiler(boilerID) + 1
	rec := domain.NewBlowdownRecord(boilerID, in.Operator, in.Note, duration, no, now)
	if err := s.Store.CreateBlowdown(rec); err != nil {
		return nil, nil, err
	}
	b.ResetBlowdownClock(now)
	if err := s.Store.UpdateBoiler(b); err != nil {
		return nil, nil, err
	}
	plan := s.PlanForBoiler(b)
	_ = s.Audit(traceID, domain.ActionBlowdown, "blowdown", rec.ID, in.Operator,
		fmt.Sprintf("锅炉 %s 第 %d 次排污，时长 %v 分钟，备注：%s", b.Name, no, duration, in.Note))
	return rec, plan, nil
}

// PlanForBoiler 计算某锅炉当前排污计划。
func (s *Services) PlanForBoiler(b *domain.Boiler) *domain.BlowdownPlan {
	lastAt := time.Time{}
	if last, err := s.Store.LastBlowdownByBoiler(b.ID); err == nil {
		lastAt = last.ExecutedAt
	}
	return domain.BuildBlowdownPlan(s.Cfg, b, lastAt, s.now())
}

// GetBlowdownDetail 返回某锅炉排污计划与执行记录。
func (s *Services) GetBlowdownDetail(boilerID string) (*domain.BlowdownPlan, []*domain.BlowdownRecord, error) {
	b, err := s.Store.GetBoiler(boilerID)
	if err != nil {
		return nil, nil, err
	}
	records, err := s.Store.ListBlowdownByBoiler(boilerID, 0)
	if err != nil {
		return nil, nil, err
	}
	return s.PlanForBoiler(b), records, nil
}

// ListBlowdownPlans 返回全部锅炉排污计划。
func (s *Services) ListBlowdownPlans() ([]*domain.BlowdownPlan, error) {
	boilers, err := s.Store.ListBoilers()
	if err != nil {
		return nil, err
	}
	out := make([]*domain.BlowdownPlan, 0, len(boilers))
	for _, b := range boilers {
		out = append(out, s.PlanForBoiler(b))
	}
	return out, nil
}
