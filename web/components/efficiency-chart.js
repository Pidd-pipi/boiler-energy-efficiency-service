// EfficiencyChart 能效曲线组件：纯 Canvas 多序列折线图，无外部依赖。
// 被详情页与日报页共用。
import { fmtNum } from './status-badge.js';

export function efficiencyChart(canvas, series, { height = 260 } = {}) {
  if (!canvas) return;
  canvas.height = height;
  const ctx = canvas.getContext('2d');
  const dpr = window.devicePixelRatio || 1;
  const rect = canvas.getBoundingClientRect();
  const width = Math.max(rect.width, 300);
  canvas.width = width * dpr;
  canvas.height = height * dpr;
  ctx.scale(dpr, dpr);
  ctx.clearRect(0, 0, width, height);

  const pad = { top: 18, right: 18, bottom: 34, left: 48 };
  const plotW = width - pad.left - pad.right;
  const plotH = height - pad.top - pad.bottom;

  // 计算所有数值范围。
  let min = Infinity, max = -Infinity;
  let maxLen = 0;
  series.forEach((s) => {
    if (!s.data) return;
    maxLen = Math.max(maxLen, s.data.length);
    s.data.forEach((v) => {
      if (v === null || v === undefined || Number.isNaN(Number(v))) return;
      min = Math.min(min, Number(v));
      max = Math.max(max, Number(v));
    });
  });
  if (!Number.isFinite(min) || !Number.isFinite(max)) {
    ctx.fillStyle = '#94a3b8';
    ctx.font = '13px sans-serif';
    ctx.textAlign = 'center';
    ctx.fillText('暂无数据', width / 2, height / 2);
    return;
  }
  if (min === max) { min -= 1; max += 1; }
  const range = max - min;
  min -= range * 0.1;
  max += range * 0.1;

  const xAt = (i) => pad.left + (maxLen <= 1 ? plotW / 2 : (i / (maxLen - 1)) * plotW);
  const yAt = (v) => pad.top + plotH - ((Number(v) - min) / (max - min)) * plotH;

  // 网格与坐标轴。
  ctx.strokeStyle = '#334155';
  ctx.lineWidth = 1;
  const gridN = 5;
  for (let g = 0; g <= gridN; g++) {
    const y = pad.top + (g / gridN) * plotH;
    ctx.beginPath();
    ctx.moveTo(pad.left, y);
    ctx.lineTo(width - pad.right, y);
    ctx.stroke();
    const val = max - (g / gridN) * (max - min);
    ctx.fillStyle = '#94a3b8';
    ctx.font = '11px sans-serif';
    ctx.textAlign = 'right';
    ctx.fillText(fmtNum(val), pad.left - 6, y + 4);
  }
  ctx.strokeStyle = '#1e293b';
  ctx.strokeRect(pad.left, pad.top, plotW, plotH);

  // 序列折线。
  series.forEach((s) => {
    if (!s.data || s.data.length === 0) return;
    ctx.strokeStyle = s.color;
    ctx.lineWidth = 2;
    ctx.beginPath();
    let started = false;
    s.data.forEach((v, i) => {
      if (v === null || v === undefined || Number.isNaN(Number(v))) return;
      const x = xAt(i);
      const y = yAt(v);
      if (!started) { ctx.moveTo(x, y); started = true; } else { ctx.lineTo(x, y); }
    });
    ctx.stroke();
  });

  // 图例。
  let legendX = pad.left + 8;
  ctx.font = '12px sans-serif';
  ctx.textAlign = 'left';
  series.forEach((s) => {
    ctx.fillStyle = s.color;
    ctx.fillRect(legendX, 6, 10, 10);
    ctx.fillStyle = '#e2e8f0';
    ctx.fillText(s.name, legendX + 14, 15);
    legendX += 14 + ctx.measureText(s.name).width + 16;
  });

  // X 轴标签（时间）。
  if (series.length > 0 && series[0].labels && series[0].labels.length > 0) {
    const labels = series[0].labels;
    const step = Math.max(1, Math.floor(labels.length / 6));
    ctx.fillStyle = '#94a3b8';
    ctx.font = '10px sans-serif';
    ctx.textAlign = 'center';
    for (let i = 0; i < labels.length; i += step) {
      ctx.fillText(labels[i], xAt(i), height - 12);
    }
  }
}
