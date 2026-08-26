// 状态徽章组件：锅炉状态 / 告警状态 / 诊断结论 / 告警级别。
import { BoilerStatus, AlertStatus, DiagnosisResult, AlertLevel } from '../enums.js';

export function statusBadge(status, kind = 'boiler') {
  const map = {
    boiler: BoilerStatus,
    alert: AlertStatus,
    diagnosis: DiagnosisResult,
    level: AlertLevel,
  }[kind] || {};
  const item = map[status] || { label: status || '-', cls: '' };
  return `<span class="badge ${item.cls || ''}">${escapeHtml(item.label)}</span>`;
}

export function escapeHtml(v) {
  if (v === undefined || v === null) return '';
  return String(v)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;');
}

export function fmtTime(t) {
  if (!t) return '-';
  const d = new Date(t);
  if (Number.isNaN(d.getTime())) return String(t);
  const pad = (n) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

export function fmtNum(v, digits = 2) {
  if (v === undefined || v === null || Number.isNaN(Number(v))) return '-';
  return Number(v).toFixed(digits);
}
