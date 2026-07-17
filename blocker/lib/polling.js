// PJ-505: polling adapter — second transport behind the same onDomainBlocked
// callback the realtime adapter uses, so background.js can't tell them apart.
import { SUPABASE_REST_URL, SUPABASE_ANON_KEY, POLL_INTERVAL_MS } from "./config.js";

export function createPollingAdapter(onDomainBlocked) {
  let timer = null;
  let since = new Date(0).toISOString();

  async function poll() {
    try {
      const url = `${SUPABASE_REST_URL}/domains?select=id,domain,confidence,reason,matched_fields,status,blocked_at&status=eq.blocked&blocked_at=gt.${encodeURIComponent(since)}&order=blocked_at.asc`;
      const res = await fetch(url, {
        headers: { apikey: SUPABASE_ANON_KEY, Authorization: `Bearer ${SUPABASE_ANON_KEY}` },
      });
      if (!res.ok) return;
      const rows = await res.json();
      for (const row of rows) {
        onDomainBlocked(row);
        if (row.blocked_at > since) since = row.blocked_at;
      }
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
