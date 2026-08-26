// 前后端共享枚举定义（与 backend domain 保持一致）。
// 修改任意枚举时必须同步 backend/domain/*.go 与 README。
export const BoilerStatus = {
  stopped:     { label: '停炉',   cls: 'st-stopped' },
  starting:    { label: '启动中', cls: 'st-starting' },
  running:     { label: '运行中', cls: 'st-running' },
  firing_down: { label: '压火',   cls: 'st-firing' },
};

export const BoilerType = {
  steam:     { label: '蒸汽锅炉' },
  hot_water: { label: '热水锅炉' },
};

export const AlertType = {
  flue_temp_spike: { label: '排烟温度突升' },
  oxygen_abnormal: { label: '氧含量异常' },
  pressure_high:   { label: '压力过高' },
  water_low:       { label: '水位过低' },
};

export const AlertLevel = {
  warning:  { label: '一般', cls: 'lv-warning' },
  critical: { label: '严重', cls: 'lv-critical' },
};

export const AlertStatus = {
  open:         { label: '待确认', cls: 'st-open' },
  acknowledged: { label: '已确认', cls: 'st-ack' },
  escalated:    { label: '已升级', cls: 'st-escalated' },
  resolved:     { label: '已处置', cls: 'st-resolved' },
};

export const DiagnosisResult = {
  under_air:  { label: '缺氧燃烧', cls: 'dg-under' },
  normal:     { label: '燃烧正常', cls: 'dg-normal' },
  excess_air: { label: '过剩空气', cls: 'dg-excess' },
};

export const AuditAction = {
  create_boiler:  { label: '创建锅炉' },
  update_boiler:  { label: '更新锅炉' },
  transition:     { label: '状态迁移' },
  run_ingest:     { label: '运行数据上报' },
  ack_alert:      { label: '确认告警' },
  escalate_alert: { label: '升级告警' },
  resolve_alert:  { label: '处置告警' },
  blowdown:       { label: '排污执行' },
  api_request:    { label: '接口请求' },
};
