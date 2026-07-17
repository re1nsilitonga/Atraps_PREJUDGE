// PJ-508: catch judol domains before the user even clicks through from
// Google search results — same blocked.html used for direct-nav blocks,
// just triggered by scanning result links instead of a DNR redirect.
const CACHE_KEY = "blocklistCache"; // matches lib/blocklist.js

function redirectUrl(entry) {
  const params = new URLSearchParams({
    d: entry.domain,
    c: String(entry.confidence ?? 0),
    r: entry.reason ?? "",
    m: JSON.stringify(entry.matchedFields ?? []),
    id: entry.id ?? "",
  });
  return `blocked.html?${params.toString()}`;
}

function matchEntry(hostname, cache) {
  for (const domain of Object.keys(cache)) {
    if (hostname === domain || hostname.endsWith(`.${domain}`)) return cache[domain];
  }
  return null;
}

async function scan() {
  const { [CACHE_KEY]: cache = {} } = await chrome.storage.local.get(CACHE_KEY);
  if (Object.keys(cache).length === 0) return;

  for (const a of document.querySelectorAll("a[href]")) {
    let hostname;
    try {
      hostname = new URL(a.href).hostname.replace(/^www\./, "");
    } catch {
      continue;
    }
    const entry = matchEntry(hostname, cache);
    if (entry) {
      location.replace(chrome.runtime.getURL(redirectUrl(entry)));
      return;
    }
  }
}

scan();
