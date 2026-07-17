// Single-source-access follow-up: the Blocker no longer holds Supabase
// credentials at all — every read (full sync, incremental poll, realtime
// push) and every write (/report-false-positive) goes through the Go API.
// Values match .env at repo root; the extension has no build step to inject
// env vars.
export const API_BASE = "http://localhost:8000/api/v1";
export const REALTIME_WS_URL = `${API_BASE.replace(/^http/, "ws")}/realtime`;

export const POLL_INTERVAL_MS = 3000;
