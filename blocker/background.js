import { fetchFull, applyFull, applyOne, removeDomain } from "./lib/blocklist.js";
import { createRealtimeAdapter } from "./lib/realtime.js";
import { createPollingAdapter } from "./lib/polling.js";
import { maybeCapture } from "./evidence.js";

const ADAPTER_MODE_KEY = "adapterMode";

let activeAdapter = null;

async function getAdapterMode() {
  const { [ADAPTER_MODE_KEY]: mode = "realtime" } = await chrome.storage.local.get(ADAPTER_MODE_KEY);
  return mode;
}

async function startAdapter() {
  activeAdapter?.stop();
  const mode = await getAdapterMode();
  activeAdapter = mode === "polling" ? createPollingAdapter(applyOne) : createRealtimeAdapter(applyOne);
  activeAdapter.start();
}

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

chrome.tabs.onUpdated.addListener((tabId, changeInfo, tab) => {
  if (changeInfo.status === "complete" && tab.url) {
    maybeCapture(tabId, tab.url);
  }
});

chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
  if (message?.type === "SET_ADAPTER_MODE") {
    chrome.storage.local.set({ [ADAPTER_MODE_KEY]: message.mode }).then(startAdapter).then(() => sendResponse({ ok: true }));
    return true;
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
