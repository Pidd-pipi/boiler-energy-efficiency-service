// BoilerCard 锅炉状态卡片：被总览与详情页共用。
import { statusBadge, escapeHtml, fmtNum } from './status-badge.js';
import { BoilerType } from '../enums.js';

export function boilerCard(item) {
  const b = item.boiler;
  const eff = item.latest_efficiency;
  const type = BoilerType[b.type] || { label: b.type };
  const alertBadge = item.open_alert_count > 0
    ? `<span class="badge lv-critical">告警 ${item.open_alert_count}</span>`
    : `<span class="badge st-ack">无告警</span>`;
  const blowBadge = item.blowdown_due
    ? '<span class="badge lv-warning">排污到期</span>'
    : '';
  return `
  <a class="card" href="/boilers/${encodeURIComponent(b.id)}" data-link style="text-decoration:none;color:inherit;display:block">
    <div class="card-title">
      ${escapeHtml(b.name)}
      ${statusBadge(b.status, 'boiler')}
    </div>
    <div class="card-row"><span>类型</span><b>${type.label}</b></div>
    <div class="card-row"><span>额定蒸发量</span><b>${fmtNum(b.rated_capacity, 1)} t/h</b></div>
    <div class="card-row"><span>水质硬度</span><b>${fmtNum(b.water_hardness, 1)} mmol/L</b></div>
    <div class="card-row"><span>实时热效率</span><b>${eff ? fmtNum(eff.efficiency) + '%' : '--'}</b></div>
    <div class="card-row"><span>单位煤耗</span><b>${eff ? fmtNum(eff.unit_coal_consumption) + ' kg/t' : '--'}</b></div>
    <div class="card-row" style="align-items:center">
      <span>运行状态</span>
      <span>${alertBadge} ${blowBadge}</span>
    </div>
    <div class="card-row"><span>累计运行</span><b>${fmtNum(b.run_seconds_total / 3600, 1)} h</b></div>
  </a>`;
}
