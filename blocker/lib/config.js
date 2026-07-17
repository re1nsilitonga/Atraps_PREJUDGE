export const API_BASE = "http://localhost:8000/api/v1";
export const REALTIME_WS_URL = `${API_BASE.replace(/^http/, "ws")}/realtime`;

export const POLL_INTERVAL_MS = 3000;
