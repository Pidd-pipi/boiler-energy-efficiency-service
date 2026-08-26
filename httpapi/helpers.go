package httpapi

import "example.com/boiler-energy-efficiency-service/store"

// storeAlertFilterOpen 返回"全部未处置告警"的过滤条件。
func storeAlertFilterOpen() store.AlertFilter {
	return store.AlertFilter{OpenOnly: true, Limit: 20}
}
