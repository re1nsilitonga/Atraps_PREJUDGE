const CACHE_KEY = "blocklistCache";

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
      a.href = chrome.runtime.getURL(redirectUrl(entry));
    }
  }
}

scan();
