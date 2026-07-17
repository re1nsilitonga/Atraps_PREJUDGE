// PJ-502, single-source-access follow-up: realtime adapter connects to the
// Go API's own WebSocket relay (api/realtime.go) instead of Supabase
// Realtime directly. The Blocker never holds Supabase credentials now.
// Payload shape from the server is already {id, domain, confidence, reason,
// matched_fields} — the same row shape blocklist.js's normalize() expects
// from /blocklist — so no protocol envelope to unwrap here, unlike the old
// hand-rolled Phoenix channel join this replaces. Heartbeat/reconnect is the
// server's job now (api/realtime.go pings every 25s); the browser's native
// WebSocket answers pings at the protocol level on its own.
import { REALTIME_WS_URL } from "./config.js";

// ponytail: reconnect is a fixed 2s backoff, not exponential — same
// tradeoff the Supabase adapter this replaces documented. Fine for a demo,
// add exponential backoff if this runs unattended for real.
const RECONNECT_MS = 2000;

export function createRealtimeAdapter(onDomainBlocked) {
  let ws = null;
  let stopped = false;
  let reconnectTimer = null;

  function handleMessage(raw) {
    try {
      onDomainBlocked(JSON.parse(raw));
    } catch {
      // malformed payload — ignore; next event or a poll cycle catches up
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
