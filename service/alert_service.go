package service

import (
	"fmt"
	"time"

	"example.com/boiler-energy-efficiency-service/domain"
	"example.com/boiler-energy-efficiency-service/store"
)

// ListAlerts 查询告警列表。
func (s *Services) ListAlerts(filter store.AlertFilter) ([]*domain.RunAlert, error) {
	return s.Store.ListAlerts(filter)
}

// AckAlert 确认告警（待确认/已升级状态均可确认）。
func (s *Services) AckAlert(traceID, alertID, operator, note string) (*domain.RunAlert, error) {
	a, err := s.Store.GetAlert(alertID)
	if err != nil {
		return nil, fmt.Errorf("查询告警失败: %w", err)
	}
	if err := a.Acknowledge(operator, note, s.now()); err != nil {
		return nil, err
	}
	if err := s.Store.UpdateAlert(a); err != nil {
		return nil, err
	}
	_ = s.Audit(traceID, domain.ActionAckAlert, "alert", a.ID, operator,
		fmt.Sprintf("确认告警 %s：%s", a.Type.Label(), note))
	return a, nil
}

// EscalateAlert 人工升级告警。
func (s *Services) EscalateAlert(traceID, alertID, operator string) (*domain.RunAlert, error) {
	a, err := s.Store.GetAlert(alertID)
	if err != nil {
		return nil, fmt.Errorf("查询告警失败: %w", err)
	}
	if err := a.Escalate(s.now()); err != nil {
		return nil, err
	}
	if err := s.Store.UpdateAlert(a); err != nil {
		return nil, err
	}
	_ = s.Audit(traceID, domain.ActionEscalateAlert, "alert", a.ID, operator,
		fmt.Sprintf("升级告警 %s 为严重", a.Type.Label()))
	return a, nil
}

// ResolveAlert 处置关闭告警。
func (s *Services) ResolveAlert(traceID, alertID, operator, note string) (*domain.RunAlert, error) {
	a, err := s.Store.GetAlert(alertID)
	if err != nil {
		return nil, fmt.Errorf("查询告警失败: %w", err)
	}
	if err := a.Resolve(operator, note, s.now()); err != nil {
		return nil, err
	}
	if err := s.Store.UpdateAlert(a); err != nil {
		return nil, err
	}
	_ = s.Audit(traceID, domain.ActionResolveAlert, "alert", a.ID, operator,
		fmt.Sprintf("处置告警 %s：%s", a.Type.Label(), note))
	return a, nil
}

// EscalateDue 扫描全部待确认告警，超过升级时限的自动升级。
// 返回本次升级数量；供定时任务与测试调用。
func (s *Services) EscalateDue(now time.Time) (int, error) {
	alerts, err := s.Store.ListAlerts(store.AlertFilter{Status: domain.AlertOpen})
	if err != nil {
		return 0, err
	}
	escalated := 0
	for _, a := range alerts {
		if a.IsDueEscalation(now, s.Cfg.AlertEscalationAfter) {
			if err := a.Escalate(now); err != nil {
				continue // 已被并发处置的跳过
			}
			if err := s.Store.UpdateAlert(a); err != nil {
				return escalated, err
			}
			_ = s.Audit("sweeper", domain.ActionEscalateAlert, "alert", a.ID, "system",
				fmt.Sprintf("告警超过 %v 未确认自动升级", s.Cfg.AlertEscalationAfter))
			escalated++
		}
	}
	return escalated, nil
}
