import { API_BASE } from "./lib/config.js";
import { loadCache } from "./lib/blocklist.js";
import { redirectUrl } from "./lib/rules.js";

const analyzedThisSession = new Set();

const L2_CONFIDENCE_THRESHOLD = 0.8;

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

    const res = await fetch(`${API_BASE}/analyze`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ domain, evidence_b64: evidenceB64, evidence_type: "screenshot" }),
    });
    const result = await res.json();

    if (result.is_judol && result.confidence >= L2_CONFIDENCE_THRESHOLD) {
      const target = chrome.runtime.getURL(redirectUrl({
        domain,
        confidence: result.confidence,
        reason: result.reason,
        matchedFields: [],
        id: result.domain_id,
      }));
      await chrome.tabs.update(tabId, { url: target });
    }
  } catch {
  }
}
