import { REALTIME_WS_URL } from "./config.js";

const RECONNECT_MS = 2000;

export function createRealtimeAdapter(onDomainBlocked) {
  let ws = null;
  let stopped = false;
  let reconnectTimer = null;

  function handleMessage(raw) {
    try {
      onDomainBlocked(JSON.parse(raw));
    } catch {
    }
  }

  function scheduleReconnect() {
    if (stopped) return;
    clearTimeout(reconnectTimer);
    reconnectTimer = setTimeout(connect, RECONNECT_MS);
  }

  function connect() {
    if (stopped) return;
    ws = new WebSocket(REALTIME_WS_URL);
    ws.onmessage = (event) => handleMessage(event.data);
    ws.onclose = scheduleReconnect;
    ws.onerror = () => ws?.close();
  }

  return {
    start() {
      stopped = false;
      connect();
    },
    stop() {
      stopped = true;
      clearTimeout(reconnectTimer);
      ws?.close();
      ws = null;
    },
  };
}
