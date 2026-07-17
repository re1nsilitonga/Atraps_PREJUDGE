// Anon key ships in the Blocker by design (PJ-101) — public, read-only RLS-open table.
// Values match .env at repo root; the extension has no build step to inject env vars.
const SUPABASE_PROJECT_URL = "https://rckecxocnwxnwjmqmgmn.supabase.co";
export const SUPABASE_REST_URL = `${SUPABASE_PROJECT_URL}/rest/v1`;
export const SUPABASE_WS_URL = `wss://${SUPABASE_PROJECT_URL.replace("https://", "")}/realtime/v1/websocket`;
export const SUPABASE_ANON_KEY =
  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6InJja2VjeG9jbnd4bndqbXFtZ21uIiwicm9sZSI6ImFub24iLCJpYXQiOjE3ODQyMTIzODAsImV4cCI6MjA5OTc4ODM4MH0.FgH1uXlNWM6o9pSm7wBqxPpH6A8eu2rhxJXRyY9BS0M";

// Go API — used only for /report-false-positive, which is not a plain table write.
export const API_BASE = "http://localhost:8000/api/v1";

export const POLL_INTERVAL_MS = 3000;
