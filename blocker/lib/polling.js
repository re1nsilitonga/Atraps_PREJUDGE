import { API_BASE, POLL_INTERVAL_MS } from "./config.js";

export function createPollingAdapter(onDomainBlocked) {
  let timer = null;
  let since = new Date(0).toISOString();

  async function poll() {
    try {
      const url = `${API_BASE}/blocklist?since=${encodeURIComponent(since)}`;
      const res = await fetch(url);
      if (!res.ok) return;
      const body = await res.json();
      for (const row of body.domains ?? []) {
        onDomainBlocked(row);
      }
      if (body.updated_at) since = body.updated_at;
    } catch {
    }
  }

  return {
    start() {
      since = new Date(0).toISOString();
      poll();
      timer = setInterval(poll, POLL_INTERVAL_MS);
    },
    stop() {
      clearInterval(timer);
      timer = null;
    },
  };
}
