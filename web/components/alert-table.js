// AlertTable 运行告警表格：被总览与告警页共用。
import { statusBadge, escapeHtml, fmtNum, fmtTime } from './status-badge.js';
import { AlertType } from '../enums.js';

export function alertTable(alerts, { showBoiler = false, onAction } = {}) {
  if (!alerts || alerts.length === 0) {
    return '<div class="empty">暂无告警记录</div>';
  }
  const rows = alerts.map((a) => {
    const type = AlertType[a.type] || { label: a.type };
    let actions = '';
    if (onAction) {
      if (a.status === 'open') {
        actions = `<button class="btn btn-sm btn-primary" data-action="ack" data-id="${a.id}">确认</button>
                   <button class="btn btn-sm" data-action="escalate" data-id="${a.id}">升级</button>`;
      } else if (a.status === 'escalated') {
        actions = `<button class="btn btn-sm btn-primary" data-action="ack" data-id="${a.id}">确认</button>
                   <button class="btn btn-sm" data-action="resolve" data-id="${a.id}">处置</button>`;
      } else if (a.status === 'acknowledged') {
        actions = `<button class="btn btn-sm" data-action="resolve" data-id="${a.id}">处置</button>`;
      } else {
        actions = '<span class="muted">已关闭</span>';
      }
    }
    return `<tr>
      <td>${a.id}</td>
      ${showBoiler ? `<td>${escapeHtml(a.boiler_id)}</td>` : ''}
      <td>${escapeHtml(type.label)}</td>
      <td>${statusBadge(a.level, 'level')}</td>
      <td>${statusBadge(a.status, 'alert')}</td>
      <td>${escapeHtml(a.message)}</td>
      <td>${fmtNum(a.value)}</td>
      <td>${fmtTime(a.created_at)}</td>
      <td>${escapeHtml(a.confirm_by || '-')}</td>
      <td>${actions}</td>
    </tr>`;
  }).join('');
  return `<table>
    <thead><tr>
      <th>编号</th>
      ${showBoiler ? '<th>锅炉</th>' : ''}
      <th>类型</th><th>级别</th><th>状态</th><th>内容</th><th>数值</th>
      <th>时间</th><th>确认人</th><th>操作</th>
    </tr></thead>
    <tbody>${rows}</tbody>
  </table>`;
}

// 为表格绑定告警操作事件（事件委托）。
export function bindAlertActions(container, onAction) {
  container.querySelectorAll('[data-action]').forEach((btn) => {
    btn.addEventListener('click', async () => {
      const action = btn.dataset.action;
      const id = btn.dataset.id;
      btn.disabled = true;
      try {
        await onAction(action, id, btn);
      } finally {
        btn.disabled = false;
      }
    });
  });
}
