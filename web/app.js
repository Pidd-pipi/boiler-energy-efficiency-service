// 应用入口：SPA 路由 + 页面渲染，全部数据来自后端 REST API。
import API, { qs } from './api.js';
import { boilerCard } from './components/boiler-card.js';
import { alertTable, bindAlertActions } from './components/alert-table.js';
import { efficiencyChart } from './components/efficiency-chart.js';
import { statusBadge, escapeHtml, fmtNum, fmtTime } from './components/status-badge.js';
import { toast, toastError, toastSuccess } from './components/toast.js';
import { useBoilers } from './hooks/use-boilers.js';
import { useAlerts } from './hooks/use-alerts.js';
import { DiagnosisResult, AlertType, BoilerType } from './enums.js';

const main = document.getElementById('main');

// ---------- 工具 ----------
function setLoading(html = '<div class="loading">加载中…</div>') {
  main.innerHTML = html;
}

function pageHead(title, sub = '') {
  return `<div class="page-head">
    <div>
      <div class="page-title">${title}</div>
      ${sub ? `<div class="page-sub">${sub}</div>` : ''}
    </div>
  </div>`;
}

function statCard(label, value, extra = '') {
  return `<div class="stat-card"><div class="stat-label">${label}</div>
    <div class="stat-value">${value}</div>${extra ? `<div class="muted">${extra}</div>` : ''}</div>`;
}


// ---------- 模态输入（替代 window.prompt，适配受限浏览器环境） ----------
function promptNote(title, placeholder = '') {
  return new Promise((resolve) => {
    const overlay = document.createElement('div');
    overlay.style.cssText = 'position:fixed;inset:0;background:rgba(0,0,0,.55);z-index:998;display:flex;align-items:center;justify-content:center;';
    const box = document.createElement('div');
    box.style.cssText = 'background:#1e293b;border:1px solid #334155;border-radius:10px;padding:18px;width:320px;';
    box.innerHTML = `
      <div style="font-weight:600;margin-bottom:12px">${title}</div>
      <input id="modal-input" type="text" placeholder="${placeholder}"
        style="width:100%;padding:8px 10px;border-radius:8px;border:1px solid #334155;background:#1a2536;color:#e2e8f0;font-size:13px" />
      <div style="display:flex;gap:8px;margin-top:14px;justify-content:flex-end">
        <button class="btn btn-sm" data-cancel>取消</button>
        <button class="btn btn-sm btn-primary" data-ok>确定</button>
      </div>`;
    overlay.appendChild(box);
    document.body.appendChild(overlay);
    const input = box.querySelector('#modal-input');
    input.focus();
    const done = (value) => { overlay.remove(); resolve(value); };
    box.querySelector('[data-ok]').addEventListener('click', () => done(input.value.trim()));
    box.querySelector('[data-cancel]').addEventListener('click', () => done(''));
    input.addEventListener('keydown', (e) => { if (e.key === 'Enter') done(input.value.trim()); });
  });
}

// ---------- 页面：锅炉房总览 ----------
async function renderOverview() {
  setLoading();
  const data = await API.get('/api/overview');
  const stats = data.stats || {};
  const html = `
    ${pageHead('锅炉房总览', '锅炉运行状态 · 实时热效率 · 未确认告警')}
    <div class="stat-grid">
      ${statCard('锅炉总数', stats.total_boilers ?? 0)}
      ${statCard('运行中', stats.running_boilers ?? 0)}
      ${statCard('未确认告警', `<span class="${(stats.open_alerts || 0) > 0 ? 'danger-text' : 'ok-text'}">${stats.open_alerts ?? 0}</span>`)}
      ${statCard('平均热效率', `${fmtNum(stats.avg_efficiency)}%`)}
      ${statCard('今日产汽', `${fmtNum(stats.today_steam)} t`)}
      ${statCard('今日煤耗', `${fmtNum(stats.today_coal)} kg`)}
    </div>
    <div class="cards mb">${(data.boilers || []).map(boilerCard).join('') || '<div class="empty">暂无锅炉，请先创建锅炉</div>'}</div>
    <div class="panel">
      <div class="panel-title">未确认告警</div>
      <div id="overview-alerts">${alertTable(data.alerts || [], { showBoiler: true, onAction: true })}</div>
    </div>`;
  main.innerHTML = html;
  bindAlertActions(document.getElementById('overview-alerts'), onAlertAction);
}

// ---------- 页面：锅炉详情 ----------
async function renderBoilerDetail(id) {
  setLoading();
  const [ov, runData, efficiency, diagnosis, blowdown, transitions] = await Promise.all([
    API.get(`/api/boilers/${id}`),
    API.get(`/api/boilers/${id}/run?limit=60`),
    API.get(`/api/boilers/${id}/efficiency?limit=60`),
    API.get(`/api/boilers/${id}/diagnosis?limit=30`),
    API.get(`/api/boilers/${id}/blowdown`),
    API.get(`/api/boilers/${id}/transitions`),
  ]);
  const b = ov.boiler;
  const type = BoilerType[b.type] || { label: b.type };
  const eff = ov.latest_efficiency;
  const transBtns = (transitions || []).map((t) =>
    `<button class="btn btn-sm" data-trans="${t}">→ ${t} (${t === 'running' ? '运行' : t === 'stopped' ? '停炉' : t === 'starting' ? '启动' : '压火'})</button>`).join(' ');

  const html = `
    ${pageHead('锅炉详情', `${escapeHtml(b.name)} · ${type.label} · ${fmtNum(b.rated_capacity, 1)} t/h`)}
    <div class="detail-head">
      ${statusBadge(b.status, 'boiler')}
      <span class="muted">累计运行 ${fmtNum(b.run_seconds_total / 3600, 1)} h</span>
      <span class="muted">距上次排污 ${fmtNum(b.run_seconds_since_blowdown / 3600, 1)} h</span>
      <span>${ov.open_alert_count > 0 ? `<span class="badge lv-critical">未确认告警 ${ov.open_alert_count}</span>` : '<span class="badge st-ack">无未确认告警</span>'}</span>
    </div>
    <div class="panel">
      <div class="panel-title">状态控制</div>
      <div class="filter-bar">
        ${transBtns || '<span class="muted">当前状态无可执行迁移</span>'}
      </div>
    </div>
    <div class="panel">
      <div class="panel-title">运行数据上报（触发能效计算 · 工况诊断 · 告警判定）</div>
      <div class="form-grid">
        <div class="field"><label>燃料量 kg/h</label><input id="f-fuel" type="number" step="0.1" value="1180" /></div>
        <div class="field"><label>蒸汽量 t/h</label><input id="f-steam" type="number" step="0.1" value="8.5" /></div>
        <div class="field"><label>给水流量 t/h</label><input id="f-flow" type="number" step="0.1" value="9" /></div>
        <div class="field"><label>排烟温度 ℃</label><input id="f-flue" type="number" step="0.1" value="152" /></div>
        <div class="field"><label>氧含量 %</label><input id="f-o2" type="number" step="0.1" value="8" /></div>
        <div class="field"><label>蒸汽压力 MPa</label><input id="f-pressure" type="number" step="0.01" value="1.2" /></div>
        <div class="field"><label>水位 %</label><input id="f-level" type="number" step="1" value="60" /></div>
        <div class="field"><label>出水温度 ℃（热水）</label><input id="f-supply" type="number" step="0.1" value="82" /></div>
        <div class="field"><label>回水温度 ℃（热水）</label><input id="f-return" type="number" step="0.1" value="58" /></div>
        <div class="field"><label>采样周期 min</label><input id="f-interval" type="number" step="1" value="5" /></div>
      </div>
      <div class="mt"><button id="btn-ingest" class="btn btn-primary">上报运行数据</button>
        <button id="btn-ingest-spike" class="btn">上报排烟突升(180℃)</button>
        <button id="btn-ingest-under" class="btn">上报缺氧(O2=2.5%)</button>
        <button id="btn-ingest-excess" class="btn">上报过剩(O2=19%)</button>
      </div>
      <div id="ingest-result" class="mt"></div>
    </div>
    <div class="grid-2">
      <div class="panel">
        <div class="panel-title">运行趋势（排烟温度 / 氧含量）</div>
        <div class="chart-box"><canvas id="chart-run"></canvas></div>
      </div>
      <div class="panel">
        <div class="panel-title">能效曲线（热效率 / 单位煤耗）</div>
        <div class="chart-box"><canvas id="chart-eff"></canvas></div>
      </div>
    </div>
    <div class="grid-2">
      <div class="panel">
        <div class="panel-title">实时能效</div>
        ${eff ? `<div class="card-row"><span>热效率</span><b>${fmtNum(eff.efficiency)}%</b></div>
                 <div class="card-row"><span>单位煤耗</span><b>${fmtNum(eff.unit_coal_consumption)} kg/t</b></div>
                 <div class="card-row"><span>过量空气系数</span><b>${fmtNum(eff.excess_air_coefficient)}</b></div>` : '<div class="empty">暂无能效记录</div>'}
      </div>
      <div class="panel">
        <div class="panel-title">工况诊断（最近 ${diagnosis.length} 条）</div>
        ${diagnosis.slice().reverse().slice(0, 8).map((d) => `
          <div class="card-row">
            <span>${fmtTime(d.timestamp)} · α=${fmtNum(d.excess_air_coefficient)}</span>
            <b>${statusBadge(d.result, 'diagnosis')}</b>
          </div>`).join('') || '<div class="empty">暂无诊断</div>'}
      </div>
    </div>
    <div class="panel">
      <div class="panel-title">排污管理</div>
      <div class="card-row"><span>排污周期</span><b>${fmtNum(blowdown.plan.interval_hours)} h</b></div>
      <div class="card-row"><span>已累计运行</span><b>${fmtNum(blowdown.plan.accumulated_run_hours)} h</b></div>
      <div class="card-row"><span>排污状态</span><b>${blowdown.plan.needs_attention ? '<span class="badge lv-critical">需关注（超期未排污）</span>' : blowdown.plan.due ? '<span class="badge lv-warning">已到期</span>' : '<span class="badge st-ack">未到期</span>'}</b></div>
      <div class="mt filter-bar">
        <div class="field" style="width:120px"><input id="f-bd-duration" type="number" value="5" placeholder="时长min" /></div>
        <div class="field" style="width:160px"><input id="f-bd-note" type="text" placeholder="备注" /></div>
        <button id="btn-blowdown" class="btn btn-primary">执行排污</button>
      </div>
      <div class="mt">
        <h4 style="margin-bottom:8px">排污记录</h4>
        ${(blowdown.records || []).slice().reverse().map((r) =>
          `<div class="card-row"><span>#${r.blowdown_no} ${fmtTime(r.executed_at)}</span><b>${r.duration_min} min · ${escapeHtml(r.operator || '-')}</b></div>`).join('') || '<div class="muted">暂无排污记录</div>'}
      </div>
    </div>`;
  main.innerHTML = html;

  // 图表。
  const runLabels = (runData || []).map((r) => {
    const d = new Date(r.timestamp);
    return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
  });
  efficiencyChart(document.getElementById('chart-run'), [
    { name: '排烟温度℃', color: '#f87171', labels: runLabels, data: (runData || []).map((r) => r.flue_gas_temp) },
    { name: '氧含量%', color: '#38bdf8', labels: runLabels, data: (runData || []).map((r) => r.oxygen_content) },
  ]);
  const effLabels = (efficiency || []).map((e) => {
    const d = new Date(e.timestamp);
    return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
  });
  efficiencyChart(document.getElementById('chart-eff'), [
    { name: '热效率%', color: '#34d399', labels: effLabels, data: (efficiency || []).map((e) => e.efficiency) },
    { name: '单位煤耗kg/t', color: '#fbbf24', labels: effLabels, data: (efficiency || []).map((e) => e.unit_coal_consumption) },
  ]);

  // 事件绑定。
  document.querySelectorAll('[data-trans]').forEach((btn) => {
    btn.addEventListener('click', async () => {
      try {
        await API.post(`/api/boilers/${id}/transition`, { target: btn.dataset.trans, operator: 'web' });
        toastSuccess('状态迁移成功');
        await renderBoilerDetail(id);
      } catch (err) { toastError(err); }
    });
  });
  const ingest = async (overrides = {}) => {
    const body = {
      fuel_amount: parseFloat(document.getElementById('f-fuel').value),
      steam_output: parseFloat(document.getElementById('f-steam').value),
      feed_water_flow: parseFloat(document.getElementById('f-flow').value),
      flue_gas_temp: parseFloat(document.getElementById('f-flue').value),
      oxygen_content: parseFloat(document.getElementById('f-o2').value),
      steam_pressure: parseFloat(document.getElementById('f-pressure').value),
      water_level: parseFloat(document.getElementById('f-level').value),
      supply_water_temp: parseFloat(document.getElementById('f-supply').value),
      return_water_temp: parseFloat(document.getElementById('f-return').value),
      interval_minutes: parseFloat(document.getElementById('f-interval').value),
      operator: 'web',
      ...overrides,
    };
    const result = await API.post(`/api/boilers/${id}/run`, body);
    const parts = [];
    if (result.efficiency) {
      parts.push(`热效率 ${fmtNum(result.efficiency.efficiency)}%`);
    } else if (result.efficiency_rejected) {
      parts.push(`<span class="hint">能效计算被拒绝：${escapeHtml(result.reject_reason)}</span>`);
    }
    if (result.combustion) {
      parts.push(`工况：${statusBadge(result.combustion.result, 'diagnosis')} (α=${fmtNum(result.combustion.excess_air_coefficient)})`);
    }
    parts.push(`告警 ${(result.alerts || []).length} 条`);
    document.getElementById('ingest-result').innerHTML = `<div class="card-row"><span>上报成功</span><b>${parts.join(' · ')}</b></div>`;
    toastSuccess('运行数据已上报');
    await renderBoilerDetail(id);
  };
  document.getElementById('btn-ingest').addEventListener('click', () => ingest().catch(toastError));
  document.getElementById('btn-ingest-spike').addEventListener('click', () => ingest({ flue_gas_temp: 200, oxygen_content: 8 }).catch(toastError));
  document.getElementById('btn-ingest-under').addEventListener('click', () => ingest({ oxygen_content: 2.5 }).catch(toastError));
  document.getElementById('btn-ingest-excess').addEventListener('click', () => ingest({ oxygen_content: 19 }).catch(toastError));
  document.getElementById('btn-blowdown').addEventListener('click', async () => {
    try {
      await API.post(`/api/boilers/${id}/blowdown`, {
        operator: 'web',
        duration_min: parseFloat(document.getElementById('f-bd-duration').value) || 5,
        note: document.getElementById('f-bd-note').value,
      });
      toastSuccess('排污执行成功');
      await renderBoilerDetail(id);
    } catch (err) { toastError(err); }
  });
}

// ---------- 页面：燃烧工况 ----------
async function renderDiagnosis() {
  setLoading();
  const data = await API.get(`/api/diagnosis${qs({ limit: 100 })}`);
  const summary = data.summary || {};
  const items = (data.items || []).slice().reverse();
  const html = `
    ${pageHead('燃烧工况', '过量空气系数诊断 · 调整建议')}
    <div class="stat-grid">
      ${statCard('缺氧燃烧', summary.under_air ?? 0)}
      ${statCard('燃烧正常', summary.normal ?? 0)}
      ${statCard('过剩空气', summary.excess_air ?? 0)}
    </div>
    <div class="panel">
      <div class="panel-title">诊断列表（最新 ${items.length} 条）</div>
      ${items.length === 0 ? '<div class="empty">暂无诊断记录，请先上报运行数据</div>' : `
      <table>
        <thead><tr><th>时间</th><th>锅炉</th><th>氧含量%</th><th>过量空气系数</th><th>结论</th><th>调整建议</th></tr></thead>
        <tbody>${items.map((d) => `<tr>
          <td>${fmtTime(d.timestamp)}</td>
          <td>${escapeHtml(d.boiler_id)}</td>
          <td>${fmtNum(d.oxygen_content)}</td>
          <td>${fmtNum(d.excess_air_coefficient)}</td>
          <td>${statusBadge(d.result, 'diagnosis')}</td>
          <td class="muted">${escapeHtml(d.suggestion)}</td>
        </tr>`).join('')}</tbody>
      </table>`}
    </div>`;
  main.innerHTML = html;
}

// ---------- 页面：运行告警 ----------
async function renderAlerts() {
  setLoading();
  const status = new URLSearchParams(location.search).get('status') || '';
  const data = await API.get(`/api/alerts${qs({ status, limit: 200 })}`);
  const html = `
    ${pageHead('运行告警', '告警列表 · 确认处置')}
    <div class="filter-bar">
      <button class="btn btn-sm ${!status ? 'btn-primary' : ''}" data-link href="/alerts">全部</button>
      <button class="btn btn-sm ${status === 'open' ? 'btn-primary' : ''}" data-link href="/alerts?status=open">待确认</button>
      <button class="btn btn-sm ${status === 'escalated' ? 'btn-primary' : ''}" data-link href="/alerts?status=escalated">已升级</button>
      <button class="btn btn-sm ${status === 'acknowledged' ? 'btn-primary' : ''}" data-link href="/alerts?status=acknowledged">已确认</button>
      <button class="btn btn-sm ${status === 'resolved' ? 'btn-primary' : ''}" data-link href="/alerts?status=resolved">已处置</button>
    </div>
    <div class="panel">
      <div id="alerts-body">${alertTable(data || [], { showBoiler: true, onAction: true })}</div>
    </div>`;
  main.innerHTML = html;
  bindAlertActions(document.getElementById('alerts-body'), onAlertAction);
}

// 告警操作统一处理（确认/升级/处置）。
async function onAlertAction(action, id, btn) {
  try {
    if (action === 'ack') {
      const note = await promptNote('确认说明（可选）');
      await API.post(`/api/alerts/${id}/ack`, { operator: 'web', note });
      toastSuccess('告警已确认');
    } else if (action === 'escalate') {
      await API.post(`/api/alerts/${id}/escalate`, { operator: 'web' });
      toastSuccess('告警已升级');
    } else if (action === 'resolve') {
      const note = await promptNote('处置说明（可选）');
      await API.post(`/api/alerts/${id}/resolve`, { operator: 'web', note });
      toastSuccess('告警已处置');
    }
    await renderAlerts();
  } catch (err) {
    toastError(err);
    await renderAlerts();
  }
}

// ---------- 页面：排污管理 ----------
async function renderBlowdown() {
  setLoading();
  const plans = await API.get('/api/blowdown');
  const html = `
    ${pageHead('排污管理', '按累计运行时长与水质硬度提示排污时机')}
    <div class="cards">
      ${(plans || []).map((p) => `
        <div class="card">
          <div class="card-title">${escapeHtml(p.boiler_id)}
            ${p.needs_attention ? '<span class="badge lv-critical">需关注</span>' : p.due ? '<span class="badge lv-warning">已到期</span>' : '<span class="badge st-ack">正常</span>'}
          </div>
          <div class="card-row"><span>硬度</span><b>${fmtNum(p.hardness)} mmol/L</b></div>
          <div class="card-row"><span>排污周期</span><b>${fmtNum(p.interval_hours)} h</b></div>
          <div class="card-row"><span>已累计运行</span><b>${fmtNum(p.accumulated_run_hours)} h</b></div>
          <div class="card-row"><span>剩余时间</span><b>${fmtNum(p.remaining_hours)} h</b></div>
          <div class="card-row"><span>预计下次</span><b>${fmtTime(p.next_due_at)}</b></div>
          <div class="mt"><a class="btn btn-sm" href="/boilers/${encodeURIComponent(p.boiler_id)}" data-link>前往详情执行排污</a></div>
        </div>`).join('') || '<div class="empty">暂无锅炉</div>'}
    </div>`;
  main.innerHTML = html;
}

// ---------- 页面：运行日报 ----------
async function renderReports() {
  setLoading();
  const date = new URLSearchParams(location.search).get('date') || todayStr();
  let reports = await API.get(`/api/daily-reports${qs({ date })}`);
  if (!reports || reports.length === 0) {
    // 首次进入自动为全部锅炉生成当日日报。
    try {
      reports = await API.post(`/api/daily-reports/generate${qs({ date })}`);
    } catch (_) { /* 无数据时保持空列表 */ }
  }
  const boilers = await API.get('/api/boilers');
  const firstBoiler = (boilers || [])[0];
  let chartHtml = '<div class="empty">无锅炉可选择</div>';
  if (firstBoiler) {
    const eff = await API.get(`/api/boilers/${firstBoiler.id}/efficiency?limit=60`);
    chartHtml = `<div class="chart-box"><canvas id="chart-report"></canvas></div>`;
    setTimeout(() => {
      const labels = (eff || []).map((e) => {
        const d = new Date(e.timestamp);
        return `${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')} ${String(d.getHours()).padStart(2, '0')}:00`;
      });
      efficiencyChart(document.getElementById('chart-report'), [
        { name: '热效率%', color: '#34d399', labels, data: (eff || []).map((e) => e.efficiency) },
        { name: '单位煤耗kg/t', color: '#fbbf24', labels, data: (eff || []).map((e) => e.unit_coal_consumption) },
      ]);
    }, 0);
  }
  const html = `
    ${pageHead('运行日报', '每日按锅炉聚合产汽量 · 煤耗 · 平均热效率 · 告警次数，用于月度能效考核')}
    <div class="filter-bar">
      <input type="date" id="report-date" value="${date}" />
      <button id="btn-report" class="btn btn-primary">查询</button>
      <button id="btn-generate" class="btn">为全部锅炉生成日报</button>
    </div>
    <div class="panel">
      <div class="panel-title">日报列表（${date}）</div>
      ${(reports || []).length === 0 ? '<div class="empty">该日期暂无日报</div>' : `
      <table>
        <thead><tr><th>锅炉</th><th>产汽量 t</th><th>煤耗 kg</th><th>平均热效率</th><th>告警次数</th><th>数据条数</th></tr></thead>
        <tbody>${(reports || []).map((r) => `<tr>
          <td>${escapeHtml(r.boiler_name)}</td>
          <td>${fmtNum(r.steam_output_total)}</td>
          <td>${fmtNum(r.coal_consumption_total)}</td>
          <td>${fmtNum(r.avg_efficiency)}%</td>
          <td>${r.alert_count}</td>
          <td>${r.run_data_count}</td>
        </tr>`).join('')}</tbody>
      </table>`}
    </div>
    <div class="panel">
      <div class="panel-title">能效曲线（${firstBoiler ? escapeHtml(firstBoiler.name) : ''} 最近 60 条）</div>
      ${chartHtml}
    </div>`;
  main.innerHTML = html;
  document.getElementById('btn-report').addEventListener('click', () => {
    navigate(`/reports?date=${encodeURIComponent(document.getElementById('report-date').value)}`);
  });
  document.getElementById('btn-generate').addEventListener('click', async () => {
    try {
      await API.post(`/api/daily-reports/generate${qs({ date: document.getElementById('report-date').value })}`);
      toastSuccess('日报生成完成');
      await renderReports();
    } catch (err) { toastError(err); }
  });
}

// ---------- 页面：审计日志 ----------
async function renderAudit() {
  setLoading();
  const data = await API.get(`/api/audit-logs${qs({ limit: 200 })}`);
  const items = (data || []).slice().reverse();
  const html = `
    ${pageHead('审计日志', '状态迁移 · 告警确认 · 排污执行等关键操作全程留痕')}
    <div class="panel">
      ${items.length === 0 ? '<div class="empty">暂无审计日志</div>' : `
      <table>
        <thead><tr><th>时间</th><th>动作</th><th>对象</th><th>操作人</th><th>Trace</th><th>详情</th></tr></thead>
        <tbody>${items.map((e) => `<tr>
          <td>${fmtTime(e.created_at)}</td>
          <td>${escapeHtml(e.action)}</td>
          <td>${escapeHtml(e.entity_type === 'http' ? 'http → ' + e.entity_id : e.entity_type + ':' + e.entity_id)}</td>
          <td>${escapeHtml(e.operator || '-')}</td>
          <td class="muted">${escapeHtml(e.trace_id || '-')}</td>
          <td class="muted">${escapeHtml(e.detail)}</td>
        </tr>`).join('')}</tbody>
      </table>`}
    </div>`;
  main.innerHTML = html;
}

// ---------- 路由 ----------
function todayStr() {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

const routes = [
  { pattern: /^\/$/, render: renderOverview },
  { pattern: /^\/boilers\/([^/]+)$/, render: (m) => renderBoilerDetail(decodeURIComponent(m[1])) },
  { pattern: /^\/diagnosis$/, render: renderDiagnosis },
  { pattern: /^\/alerts$/, render: renderAlerts },
  { pattern: /^\/blowdown$/, render: renderBlowdown },
  { pattern: /^\/reports$/, render: renderReports },
  { pattern: /^\/audit$/, render: renderAudit },
];

export async function navigate(path) {
  history.pushState({}, '', path);
  await render(path);
}

async function render(path) {
  for (const route of routes) {
    const m = path.match(route.pattern);
    if (m) {
      try {
        await route.render(m);
      } catch (err) {
        toastError(err);
        main.innerHTML = `<div class="panel"><div class="danger-text">页面加载失败：${escapeHtml(err.message)}</div></div>`;
      }
      updateNav();
      return;
    }
  }
  main.innerHTML = '<div class="empty">页面不存在</div>';
}

function updateNav() {
  const path = location.pathname;
  document.querySelectorAll('.nav-item').forEach((a) => {
    const href = a.getAttribute('href');
    const active = href === '/' ? path === '/' : path.startsWith(href);
    a.classList.toggle('active', active);
  });
}

// 导航点击拦截（history 路由）。
document.addEventListener('click', (e) => {
  const link = e.target.closest('[data-link]');
  if (!link) return;
  const href = link.getAttribute('href');
  if (!href || href.startsWith('http')) return;
  e.preventDefault();
  navigate(href);
});

window.addEventListener('popstate', () => render(location.pathname + location.search));

// 连接状态轮询。
async function checkHealth() {
  const el = document.getElementById('conn-status');
  try {
    const res = await fetch('/healthz');
    el.textContent = res.ok ? '● 服务正常' : '● 服务异常';
    el.className = `conn-status ${res.ok ? 'ok' : 'bad'}`;
  } catch (_) {
    el.textContent = '● 连接失败';
    el.className = 'conn-status bad';
  }
}
setInterval(checkHealth, 8000);
checkHealth();

// 启动。
render(location.pathname + location.search);
