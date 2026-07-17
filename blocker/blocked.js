import { API_BASE } from "./lib/config.js";

// matched_fields are machine keys from core/layer1/matcher.go — translated
// here into the "Kenapa?" bullets Rina from Bekasi actually reads.
const FIELD_LABELS = {
  hosting_ip: "Menggunakan alamat IP hosting yang sama dengan situs judi online lain yang telah dikonfirmasi.",
  nameserver: "Menggunakan name server yang sama dengan situs judi online lain.",
  registrar: "Didaftarkan lewat penyedia domain yang sama dengan situs judi online lain.",
  tld: "Menggunakan pola akhiran domain yang sama dengan situs judi online lain.",
  registration_burst: "Didaftarkan bersama sejumlah domain lain dalam waktu berdekatan, pola khas kampanye judi online massal.",
};

const params = new URLSearchParams(location.search);
const domain = params.get("d") || "";
const confidence = Number(params.get("c") || 0);
const reason = params.get("r") || "";
const domainId = params.get("id") || "";
let matchedFields = [];
try {
  matchedFields = JSON.parse(params.get("m") || "[]");
} catch {
  matchedFields = [];
}

document.getElementById("domainName").textContent = domain || "situs ini";
document.title = `${domain || "Situs"} diblokir, PREJUDGE`;

const pct = Math.round(confidence * 100);
document.getElementById("confidenceValue").textContent = `${pct}%`;
requestAnimationFrame(() => {
  document.getElementById("confidenceFill").style.width = `${pct}%`;
});

const reasonList = document.getElementById("reasonList");
const bullets = matchedFields.map((key) => FIELD_LABELS[key]).filter(Boolean);
if (bullets.length === 0 && reason) bullets.push(reason);
for (const text of bullets) {
  const li = document.createElement("li");
  li.textContent = text;
  reasonList.appendChild(li);
}
if (bullets.length === 0) reasonList.closest(".reasons").style.display = "none";

document.getElementById("backBtn").addEventListener("click", () => {
  if (history.length > 1) history.back();
  else location.href = "https://www.google.com";
});

const reportBtn = document.getElementById("reportBtn");
const statusMsg = document.getElementById("statusMsg");

reportBtn.addEventListener("click", async () => {
  reportBtn.disabled = true;
  statusMsg.textContent = "Mengirim laporan...";
  try {
    await fetch(`${API_BASE}/report-false-positive`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ domain_id: domainId, note: "Dilaporkan dari block page" }),
    });
    // Unblock immediately client-side too — a stuck block after "Laporkan
    // salah" is an ugly demo moment (PRD §14 risk #14); don't wait for the
    // next blocklist sync cycle.
    await chrome.runtime.sendMessage({ type: "UNBLOCK_DOMAIN", domain });
    statusMsg.textContent = "Laporan terkirim. Situs tidak lagi diblokir.";
    reportBtn.textContent = "Terlapor";
  } catch {
    statusMsg.textContent = "Gagal mengirim laporan, coba lagi.";
    reportBtn.disabled = false;
  }
});
