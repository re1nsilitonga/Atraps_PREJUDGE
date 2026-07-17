import { ruleCount } from "./lib/rules.js";

const modeLabel = { realtime: "Realtime", polling: "Polling" };

async function render() {
  document.getElementById("blockedCount").textContent = await ruleCount();
  const { mode } = await chrome.runtime.sendMessage({ type: "GET_ADAPTER_MODE" });
  document.getElementById("adapterMode").textContent = modeLabel[mode] ?? mode;
  document.getElementById("toggleAdapter").textContent =
    mode === "realtime" ? "Ganti ke polling" : "Ganti ke realtime";
}

document.getElementById("toggleAdapter").addEventListener("click", async () => {
  const { mode } = await chrome.runtime.sendMessage({ type: "GET_ADAPTER_MODE" });
  const next = mode === "realtime" ? "polling" : "realtime";
  await chrome.runtime.sendMessage({ type: "SET_ADAPTER_MODE", mode: next });
  render();
});

render();
