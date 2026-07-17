// PJ-504: blocklist sync — full fetch on startup, cached in
// chrome.storage.local so it survives service-worker restart, DNR rules
// rebuilt from the cache immediately (no network wait before rules exist).
import { SUPABASE_REST_URL, SUPABASE_ANON_KEY } from "./config.js";
import { syncRules } from "./rules.js";

const CACHE_KEY = "blocklistCache"; // { [domain]: entry }

function normalize(row) {
  return {
    id: row.id,
    domain: row.domain,
    confidence: row.confidence ?? 0,
    reason: row.reason ?? "",
    matchedFields: row.matched_fields ?? [],
  };
}

// fetchFull is the cold-start / worker-restart path. Empty result is the
// normal state on a fresh install (PJ-504) — not an error.
export async function fetchFull() {
  const url = `${SUPABASE_REST_URL}/domains?select=id,domain,confidence,reason,matched_fields&status=eq.blocked`;
  const res = await fetch(url, {
    headers: { apikey: SUPABASE_ANON_KEY, Authorization: `Bearer ${SUPABASE_ANON_KEY}` },
  });
  if (!res.ok) return [];
  const rows = await res.json();
  return rows.map(normalize);
}

export async function loadCache() {
  const { [CACHE_KEY]: cache = {} } = await chrome.storage.local.get(CACHE_KEY);
  return Object.values(cache);
}

async function saveCache(entries) {
  const cache = Object.fromEntries(entries.map((e) => [e.domain, e]));
  await chrome.storage.local.set({ [CACHE_KEY]: cache });
}

// applyFull replaces the whole blocklist (startup) and rebuilds DNR rules.
export async function applyFull(entries) {
  await saveCache(entries);
  await syncRules(entries);
}

// applyOne merges a single incremental event (realtime/polling) into the
// cache and DNR rules without refetching everything.
export async function applyOne(row) {
  const entry = normalize(row);
  const entries = await loadCache();
  const next = entries.filter((e) => e.domain !== entry.domain);
  next.push(entry);
  await saveCache(next);
  await syncRules(next);
}

// removeDomain drops a domain's DNR rule immediately (PJ-507 / PRD §14 risk
// #14: a stuck block after "Laporkan salah" is an ugly demo moment).
export async function removeDomain(domain) {
  const entries = await loadCache();
  const next = entries.filter((e) => e.domain !== domain);
  await saveCache(next);
  await syncRules(next);
}
