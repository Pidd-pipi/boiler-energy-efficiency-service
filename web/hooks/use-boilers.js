// useBoilers(poll)：锅炉数据轮询 hook，被总览与详情页共用。
import API from '../api.js';
import { usePolling } from './use-polling.js';

export function useBoilers(poll = true, intervalMs = 10000) {
  let boilers = [];
  const listeners = new Set();

  function notify() {
    listeners.forEach((fn) => fn(boilers));
  }

  async function load() {
    const data = await API.get('/api/boilers');
    boilers = Array.isArray(data) ? data : [];
    notify();
    return boilers;
  }

  const poller = usePolling(load, intervalMs, { immediate: true });

  return {
    get list() { return boilers; },
    async refresh() { return load(); },
    subscribe(fn) {
      listeners.add(fn);
      return () => listeners.delete(fn);
    },
    start() { if (poll) poller.start(); },
    stop() { poller.stop(); },
  };
}
