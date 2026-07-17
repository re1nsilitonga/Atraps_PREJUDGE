import { API_BASE } from "./lib/config.js";
import { loadCache } from "./lib/blocklist.js";

const analyzedThisSession = new Set();

function isAnalyzableURL(url) {
  try {
    return ["http:", "https:"].includes(new URL(url).protocol);
  } catch {
    return false;
  }
}

function hostnameOf(url) {
  return new URL(url).hostname.replace(/^www\./, "");
}

export async function maybeCapture(tabId, url) {
  if (!isAnalyzableURL(url)) return;
  const domain = hostnameOf(url);
  if (!domain || analyzedThisSession.has(domain)) return;

  const cache = await loadCache();
  const known = cache.some((e) => domain === e.domain || domain.endsWith(`.${e.domain}`));
  if (known) return;

  analyzedThisSession.add(domain);

  try {
    const tab = await chrome.tabs.get(tabId);
    if (!tab.active) return;

    const dataUrl = await chrome.tabs.captureVisibleTab(tab.windowId, {
      format: "jpeg",
      quality: 50,
    });
    const evidenceB64 = dataUrl.split(",")[1];
    if (!evidenceB64) return;

    await fetch(`${API_BASE}/analyze`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ domain, evidence_b64: evidenceB64, evidence_type: "screenshot" }),
    });
  } catch {
  }
}
