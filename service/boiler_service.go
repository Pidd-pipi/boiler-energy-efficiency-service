package service

import (
	"fmt"

	"example.com/boiler-energy-efficiency-service/domain"
)

// CreateBoilerInput 创建锅炉入参。
type CreateBoilerInput struct {
	Name          string            `json:"name"`
	Type          domain.BoilerType `json:"type"`
	RatedCapacity float64           `json:"rated_capacity"`
	WaterHardness float64           `json:"water_hardness"`
	Operator      string            `json:"operator"`
}

// CreateBoiler 创建锅炉台账（默认停炉状态）。
func (s *Services) CreateBoiler(traceID string, in CreateBoilerInput) (*domain.Boiler, error) {
	now := s.now()
	b := domain.NewBoiler(in.Name, in.Type, in.RatedCapacity, in.WaterHardness, now)
	if err := b.Validate(); err != nil {
		return nil, err
	}
	if err := s.Store.CreateBoiler(b); err != nil {
		return nil, err
	}
	detail := fmt.Sprintf("新增锅炉 %s（%s，额定 %v t/h，硬度 %v mmol/L）",
		b.Name, b.Type.Label(), b.RatedCapacity, b.WaterHardness)
	_ = s.Audit(traceID, domain.ActionCreateBoiler, "boiler", b.ID, in.Operator, detail)
	return b, nil
}

// GetBoiler 获取锅炉。
func (s *Services) GetBoiler(id string) (*domain.Boiler, error) {
	return s.Store.GetBoiler(id)
}

// ListBoilers 获取全部锅炉。
func (s *Services) ListBoilers() ([]*domain.Boiler, error) {
	return s.Store.ListBoilers()
}

// Transition 执行锅炉状态迁移（状态机规则见 domain.Boiler）。
// 每次迁移都会写审计日志。
func (s *Services) Transition(traceID, boilerID string, target domain.BoilerStatus, operator string) (*domain.Boiler, error) {
	b, err := s.Store.GetBoiler(boilerID)
	if err != nil {
		return nil, fmt.Errorf("获取锅炉失败: %w", err)
	}
	from := b.Status
	if err := b.TransitionTo(target, s.now()); err != nil {
		return nil, fmt.Errorf("状态迁移失败: %w", err)
	}
	if err := s.Store.UpdateBoiler(b); err != nil {
		return nil, err
	}
	detail := fmt.Sprintf("状态迁移 %s(%s) -> %s(%s)",
		from, from.Label(), target, target.Label())
	_ = s.Audit(traceID, domain.ActionTransition, "boiler", b.ID, operator, detail)
	return b, nil
}

// AllowedTargets 返回锅炉当前状态允许迁移的目标状态。
func (s *Services) AllowedTargets(id string) ([]domain.BoilerStatus, error) {
	b, err := s.Store.GetBoiler(id)
	if err != nil {
		return nil, fmt.Errorf("查询锅炉迁移目标失败: %w", err)
	}
	allowed := domain.AllowedTransitions()[b.Status]
	return allowed, nil
}
