// Package service 实现全部业务用例：采集联动、能效、诊断、告警、排污、日报、审计。
package service

import (
	"time"

	"example.com/boiler-energy-efficiency-service/config"
	"example.com/boiler-energy-efficiency-service/store"
)

// Services 聚合所有业务服务的依赖。
type Services struct {
	Cfg   *config.Config
	Store *store.Store
	// Now 时间来源，测试可注入。
	Now func() time.Time
}

// NewServices 构造业务服务集合。
func NewServices(cfg *config.Config, st *store.Store) *Services {
	return &Services{
		Cfg:   cfg,
		Store: st,
		Now:   time.Now,
	}
}

// now 返回当前时间（走可注入时钟）。
func (s *Services) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}
