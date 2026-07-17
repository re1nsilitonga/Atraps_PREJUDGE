// PJ-505: polling adapter — second transport behind the same onDomainBlocked
// callback the realtime adapter uses, so background.js can't tell them apart.
// Single-source-access follow-up: hits the Go API's own /blocklist?since=
// (already supported the cursor), not Supabase REST directly.
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
      // silent — next tick retries (PJ-202 pattern: failure invisible to the user)
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
