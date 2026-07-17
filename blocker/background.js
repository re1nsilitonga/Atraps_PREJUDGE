import { fetchFull, applyFull, applyOne, removeDomain } from "./lib/blocklist.js";
import { createRealtimeAdapter } from "./lib/realtime.js";
import { createPollingAdapter } from "./lib/polling.js";

const ADAPTER_MODE_KEY = "adapterMode"; // "realtime" | "polling"

let activeAdapter = null;

async function getAdapterMode() {
  const { [ADAPTER_MODE_KEY]: mode = "realtime" } = await chrome.storage.local.get(ADAPTER_MODE_KEY);
  return mode;
}

// startAdapter is idempotent: stops whatever is running, starts the mode
// requested. Called on boot and whenever the popup flips the feature flag
// (PJ-505 — one flag switches transports, no reload needed).
async function startAdapter() {
  activeAdapter?.stop();
  const mode = await getAdapterMode();
  activeAdapter = mode === "polling" ? createPollingAdapter(applyOne) : createRealtimeAdapter(applyOne);
  activeAdapter.start();
}

// bootPromise dedupes concurrent boot() calls: top-level module code already
// runs on every service-worker start (install, browser startup, or waking
// from idle), so onInstalled/onStartup firing *again* on top of that raced
// two boots against each other — the second stopped the first's still-
// connecting WebSocket, logging "closed before the connection is established".
let bootPromise = null;

function boot() {
  if (!bootPromise) {
    bootPromise = (async () => {
      const entries = await fetchFull();
      await applyFull(entries);
      await startAdapter();
    })();
  }
  return bootPromise;
}

chrome.runtime.onInstalled.addListener(boot);
chrome.runtime.onStartup.addListener(boot);
boot();

chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
  if (message?.type === "SET_ADAPTER_MODE") {
    chrome.storage.local.set({ [ADAPTER_MODE_KEY]: message.mode }).then(startAdapter).then(() => sendResponse({ ok: true }));
    return true; // async response
  }
  if (message?.type === "GET_ADAPTER_MODE") {
    getAdapterMode().then((mode) => sendResponse({ mode }));
    return true;
  }
  if (message?.type === "UNBLOCK_DOMAIN") {
    removeDomain(message.domain).then(() => sendResponse({ ok: true }));
    return true;
  }
});
