import { API_BASE } from "./config.js";
import { syncRules } from "./rules.js";

const CACHE_KEY = "blocklistCache";

function normalize(row) {
  return {
    id: row.id,
    domain: row.domain,
    confidence: row.confidence ?? 0,
    reason: row.reason ?? "",
    matchedFields: row.matched_fields ?? [],
  };
}

export async function fetchFull() {
  const res = await fetch(`${API_BASE}/blocklist`);
  if (!res.ok) return [];
  const body = await res.json();
  return (body.domains ?? []).map(normalize);
}

export async function loadCache() {
  const { [CACHE_KEY]: cache = {} } = await chrome.storage.local.get(CACHE_KEY);
  return Object.values(cache);
}

async function saveCache(entries) {
  const cache = Object.fromEntries(entries.map((e) => [e.domain, e]));
  await chrome.storage.local.set({ [CACHE_KEY]: cache });
}

export async function applyFull(entries) {
  await saveCache(entries);
  await syncRules(entries);
}

export async function applyOne(row) {
  const entry = normalize(row);
  const entries = await loadCache();
  const next = entries.filter((e) => e.domain !== entry.domain);
  next.push(entry);
  await saveCache(next);
  await syncRules(next);
}

export async function removeDomain(domain) {
  const entries = await loadCache();
  const next = entries.filter((e) => e.domain !== domain);
  await saveCache(next);
  await syncRules(next);
}
