// Toast 轻提示组件。
export function toast(message, type = 'info', timeout = 3200) {
  const container = document.getElementById('toast-container');
  if (!container) return;
  const el = document.createElement('div');
  el.className = `toast ${type}`;
  el.textContent = message;
  container.appendChild(el);
  setTimeout(() => {
    el.style.opacity = '0';
    el.style.transition = 'opacity .3s';
    setTimeout(() => el.remove(), 300);
  }, timeout);
}

export const toastError = (err) => {
  const msg = (err && err.message) ? err.message : '操作失败';
  toast(msg, 'error');
};

export const toastSuccess = (msg) => toast(msg, 'success');
