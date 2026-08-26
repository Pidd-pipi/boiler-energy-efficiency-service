// useAlerts(filter)：告警列表 hook，被告警页与总览共用。
import API from '../api.js';
import { qs } from '../api.js';
import { usePolling } from './use-polling.js';

export function useAlerts(filter = {}, poll = true, intervalMs = 15000) {
  let alerts = [];
  const listeners = new Set();

  function notify() {
    listeners.forEach((fn) => fn(alerts));
  }

  async function load() {
    const data = await API.get(`/api/alerts${qs(filter)}`);
    alerts = Array.isArray(data) ? data : [];
    notify();
    return alerts;
  }

  const poller = usePolling(load, intervalMs, { immediate: true });

  return {
    get list() { return alerts; },
    async refresh() { return load(); },
    subscribe(fn) {
      listeners.add(fn);
      return () => listeners.delete(fn);
    },
    start() { if (poll) poller.start(); },
    stop() { poller.stop(); },
  };
}
