package store

import (
	"example.com/boiler-energy-efficiency-service/domain"
)

// AlertFilter 告警列表过滤条件。
type AlertFilter struct {
	BoilerID string
	Status   domain.AlertStatus
	Type     domain.AlertType
	// OpenOnly 仅返回未处置（待确认/已升级）告警。
	OpenOnly bool
	Limit    int
}

// AlertStore 运行告警仓储接口。
type AlertStore interface {
	CreateAlert(a *domain.RunAlert) error
	UpdateAlert(a *domain.RunAlert) error
	GetAlert(id string) (*domain.RunAlert, error)
	ListAlerts(filter AlertFilter) ([]*domain.RunAlert, error)
	OpenAlertOfType(boilerID string, typ domain.AlertType) (*domain.RunAlert, error)
	CountOpenAlerts(boilerID string) int
	CountAlerts() int
}

// CreateAlert 保存告警并分配 ID。
func (s *Store) CreateAlert(a *domain.RunAlert) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a.ID == "" {
		a.ID = s.newIDLocked("alt")
	}
	if _, exists := s.alerts[a.ID]; exists {
		return domain.NewError(domain.KindConflict, "告警 %s 已存在", a.ID)
	}
	s.alerts[a.ID] = a
	s.alertOrder = append(s.alertOrder, a.ID)
	s.alertsByBoiler[a.BoilerID] = append(s.alertsByBoiler[a.BoilerID], a.ID)
	return s.maybeSaveLocked()
}

// UpdateAlert 更新告警（确认/升级/处置后落盘）。
func (s *Store) UpdateAlert(a *domain.RunAlert) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.alerts[a.ID]; !exists {
		return domain.NewError(domain.KindNotFound, "告警 %s 不存在", a.ID)
	}
	s.alerts[a.ID] = a
	return s.maybeSaveLocked()
}

// GetAlert 按 ID 获取告警。
func (s *Store) GetAlert(id string) (*domain.RunAlert, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, exists := s.alerts[id]
	if !exists {
		return nil, domain.NewError(domain.KindNotFound, "告警 %s 不存在", id)
	}
	return cloneAlert(a), nil
}

// ListAlerts 按过滤条件返回告警列表（时间正序，limit 取末尾最新）。
func (s *Store) ListAlerts(filter AlertFilter) ([]*domain.RunAlert, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := s.alertOrder
	if filter.BoilerID != "" {
		ids = s.alertsByBoiler[filter.BoilerID]
	}
	out := make([]*domain.RunAlert, 0, len(ids))
	for _, id := range ids {
		a := s.alerts[id]
		if filter.Status != "" && a.Status != filter.Status {
			continue
		}
		if filter.OpenOnly && !a.IsOpen() {
			continue
		}
		if filter.Type != "" && a.Type != filter.Type {
			continue
		}
		out = append(out, cloneAlert(a))
	}
	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[len(out)-filter.Limit:]
	}
	return out, nil
}

// OpenAlertOfType 返回某锅炉指定类型未处置的告警（用于去重，避免同类型刷屏）。
func (s *Store) OpenAlertOfType(boilerID string, typ domain.AlertType) (*domain.RunAlert, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, id := range s.alertsByBoiler[boilerID] {
		a := s.alerts[id]
		if a.Type == typ && a.IsOpen() {
			return cloneAlert(a), nil
		}
	}
	return nil, domain.NewError(domain.KindNotFound, "锅炉 %s 无 %s 类型未处置告警", boilerID, typ)
}

// CountOpenAlerts 返回某锅炉未处置告警数量；boilerID 为空时统计全部。
func (s *Store) CountOpenAlerts(boilerID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := s.alertOrder
	if boilerID != "" {
		ids = s.alertsByBoiler[boilerID]
	}
	n := 0
	for _, id := range ids {
		if s.alerts[id].IsOpen() {
			n++
		}
	}
	return n
}

// CountAlerts 返回告警总量。
func (s *Store) CountAlerts() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.alerts)
}
