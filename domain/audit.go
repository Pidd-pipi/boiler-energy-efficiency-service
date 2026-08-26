package domain

import "time"

// AuditAction 审计动作类型。
type AuditAction string

const (
	// ActionCreateBoiler 创建锅炉。
	ActionCreateBoiler AuditAction = "create_boiler"
	// ActionUpdateBoiler 更新锅炉。
	ActionUpdateBoiler AuditAction = "update_boiler"
	// ActionTransition 状态迁移。
	ActionTransition AuditAction = "transition"
	// ActionRunIngest 运行数据上报。
	ActionRunIngest AuditAction = "run_ingest"
	// ActionAckAlert 确认告警。
	ActionAckAlert AuditAction = "ack_alert"
	// ActionEscalateAlert 升级告警。
	ActionEscalateAlert AuditAction = "escalate_alert"
	// ActionResolveAlert 处置告警。
	ActionResolveAlert AuditAction = "resolve_alert"
	// ActionBlowdown 排污执行。
	ActionBlowdown AuditAction = "blowdown"
	// ActionAPIRequest 接口请求留痕（中间件写入）。
	ActionAPIRequest AuditAction = "api_request"
)

// Valid 校验审计动作。
func (a AuditAction) Valid() bool {
	switch a {
	case ActionCreateBoiler, ActionUpdateBoiler, ActionTransition, ActionRunIngest,
		ActionAckAlert, ActionEscalateAlert, ActionResolveAlert, ActionBlowdown, ActionAPIRequest:
		return true
	}
	return false
}

// Label 返回审计动作中文名。
func (a AuditAction) Label() string {
	switch a {
	case ActionCreateBoiler:
		return "创建锅炉"
	case ActionUpdateBoiler:
		return "更新锅炉"
	case ActionTransition:
		return "状态迁移"
	case ActionRunIngest:
		return "运行数据上报"
	case ActionAckAlert:
		return "确认告警"
	case ActionEscalateAlert:
		return "升级告警"
	case ActionResolveAlert:
		return "处置告警"
	case ActionBlowdown:
		return "排污执行"
	case ActionAPIRequest:
		return "接口请求"
	}
	return string(a)
}

// AuditEntry 操作审计日志实体。
// 状态迁移、告警确认、排污执行等关键动作全部留痕。
type AuditEntry struct {
	ID         string      `json:"id"`
	TraceID    string      `json:"trace_id"`
	Action     AuditAction `json:"action"`
	EntityType string      `json:"entity_type"` // boiler / alert / blowdown / http
	EntityID   string      `json:"entity_id"`
	Operator   string      `json:"operator"`
	Detail     string      `json:"detail"`
	CreatedAt  time.Time   `json:"created_at"`
}

// NewAuditEntry 构造审计条目。
func NewAuditEntry(traceID string, action AuditAction, entityType, entityID, operator, detail string, now time.Time) *AuditEntry {
	return &AuditEntry{
		TraceID:    traceID,
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		Operator:   operator,
		Detail:     detail,
		CreatedAt:  now,
	}
}
