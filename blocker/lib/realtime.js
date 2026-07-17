// PJ-502: Supabase Realtime adapter — vanilla WebSocket implementing the
// Phoenix channel protocol Supabase Realtime speaks. No supabase-js bundle:
// MV3 forbids remotely-fetched code, and this is a handful of messages.
//
// ponytail: reconnect is a fixed 2s backoff, not exponential. Fine for a
// 2-minute demo window; add exponential backoff if this ships past the demo.
import { SUPABASE_WS_URL, SUPABASE_ANON_KEY } from "./config.js";

const RECONNECT_MS = 2000;
const HEARTBEAT_MS = 25000;
const TOPIC = "realtime:public:domains";

export function createRealtimeAdapter(onDomainBlocked) {
  let ws = null;
  let heartbeatTimer = null;
  let reconnectTimer = null;
  let stopped = false;
  let ref = 1;

  function send(msg) {
    if (ws?.readyState === WebSocket.OPEN) ws.send(JSON.stringify(msg));
  }

  function join() {
    send({
      topic: TOPIC,
      event: "phx_join",
      payload: {
        config: {
          postgres_changes: [
            { event: "UPDATE", schema: "public", table: "domains", filter: "status=eq.blocked" },
            { event: "INSERT", schema: "public", table: "domains", filter: "status=eq.blocked" },
          ],
        },
      },
      ref: String(ref++),
    });
  }

  function handleMessage(raw) {
    let msg;
    try {
      msg = JSON.parse(raw);
    } catch {
      return;
    }
    if (msg.event !== "postgres_changes") return;
    const changes = msg.payload?.data?.record ? [msg.payload.data] : msg.payload?.data ?? [];
    for (const change of Array.isArray(changes) ? changes : [changes]) {
      const record = change?.record;
      if (record?.status === "blocked") onDomainBlocked(record);
    }
  }

  function connect() {
    if (stopped) return;
    const url = `${SUPABASE_WS_URL}?apikey=${SUPABASE_ANON_KEY}&vsn=1.0.0`;
    ws = new WebSocket(url);

    ws.onopen = () => {
      join();
      heartbeatTimer = setInterval(() => {
        send({ topic: "phoenix", event: "heartbeat", payload: {}, ref: String(ref++) });
      }, HEARTBEAT_MS);
    };
    ws.onmessage = (event) => handleMessage(event.data);
    ws.onclose = scheduleReconnect;
    ws.onerror = () => ws?.close();
  }

  function scheduleReconnect() {
    clearInterval(heartbeatTimer);
    if (stopped) return;
    clearTimeout(reconnectTimer);
    reconnectTimer = setTimeout(connect, RECONNECT_MS);
  }

  return {
    start() {
      stopped = false;
      connect();
    },
    stop() {
      stopped = true;
      clearInterval(heartbeatTimer);
      clearTimeout(reconnectTimer);
      ws?.close();
      ws = null;
    },
  };
}
