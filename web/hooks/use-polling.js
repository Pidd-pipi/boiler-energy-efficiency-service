// 通用轮询 hook：包装 setInterval，支持立即执行、启动与停止。
export function usePolling(fn, intervalMs = 10000, { immediate = true } = {}) {
  let timer = null;
  let stopped = false;

  async function tick() {
    if (stopped) return;
    try {
      await fn();
    } catch (err) {
      console.error('[usePolling] 轮询失败', err);
    }
  }

  function start() {
    if (timer !== null) return;
    stopped = false;
    if (immediate) tick();
    timer = setInterval(tick, intervalMs);
  }

  function stop() {
    stopped = true;
    if (timer !== null) {
      clearInterval(timer);
      timer = null;
    }
  }

  return { start, stop, tick };
}
