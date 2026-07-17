# Atraps_PREJUDGE

## Roles & Responsibilities (4 members)

| | Role | Owns | Primary Deliverable |
|---|---|---|---|
| **A** | Backend / Data Lead | Layer 1 (preemptive detection), database schema, all REST endpoints | Fingerprint extraction + cluster matcher + `/blocklist` endpoint |
| **B** | Backend / AI | Layer 2 (reactive detection via vision API), Gemini integration, validation script | Screenshot → Gemini → verdict → DB write → fingerprint feedback loop |
| **C** | Frontend (Extension + Dashboard) | `extension/`, `dashboard/` | Working two-device realtime block + dashboard |
| **D** | Pitch, Research & QA | `pitch/`, source verification, demo rehearsal | 2-min deck + demo script + verified `sources.md` |

See [PRD.md](PRD.md) §12 for load-balancing notes and full context.

## API Contract (`/api/v1`)

Binding contract per PRD.md §10. Backend returns hardcoded stub responses matching these shapes from T+2 (PJ-106) so Frontend is never blocked. Changes after T+4 require all four team members.

| Method | Endpoint | Request | Response |
| --- | --- | --- | --- |
| GET | `/blocklist` | `?since=<ISO>` | `{domains:[{domain,confidence,reason,matched_fields}], updated_at}` |
| POST | `/check` | `{domain}` | `{status, confidence, source, reason}` |
| POST | `/analyze` | `{domain, evidence_b64}` | `{is_judol, confidence, reason, domain_id}` |
| POST | `/fingerprint` | `{domain}` | `{cluster_id, registrar, ip, ns, tld, match_score, matched_fields}` |
| GET | `/domains` | `?limit&offset&source&status` | `{items:[...], total}` |
| GET | `/domains/{id}` | — | `{domain, detections[], whois, cluster, siblings[], evidence_url}` |
| POST | `/report-false-positive` | `{domain_id, note}` | `{ok:true}` |
| GET | `/bootstrap/latest` | — | `{l2_confirmations, l1_preemptive_catches, l1_misses, ratio}` |
| POST | `/trustpositif/verify` | `{domain}` | `{domain, is_blocked}` — **always `false`, permanent stub, see below** |

**Module boundary:**

- **Core** (owns detection logic, no Chrome/realtime/UI imports): `/analyze`, `/fingerprint`
- **Blocker** (Chrome adapter read surface): `/blocklist`, `/check`
- **Presentation** (dashboard): `/domains`, `/domains/{id}`, `/bootstrap/latest`

**Realtime channel (one Blocker adapter, not the architecture):** `GET /api/v1/realtime` — the Go API's own WebSocket relay. It LISTENs on Postgres `domain_blocked` NOTIFY (fired by a trigger on `domains`, see `db/schema.sql`) and fans out to connected clients. The Blocker no longer talks to Supabase directly at all — this is the single source of access, both for reads (`/blocklist`, `/check`) and for the realtime push. See `api/realtime.go`.

**TrustPositif verifier cut (team decision):** `trustpositif.komdigi.go.id`'s search form requires a Google reCAPTCHA token, which cannot be automated without a CAPTCHA bypass — forbidden by the same rule that already blocks auto-submitting to aduankonten.id (PRD §6). PJ-701/PJ-702 are cut; `/trustpositif/verify` stays in the contract as a permanent stub (`is_blocked: false`) so the Blocker/Presentation code paths that call it don't need special-casing. Never claim TrustPositif corroboration in the pitch or dashboard.

## Environment Variables

See `.env.example`. Real values distributed via the team WhatsApp group, never committed.

- `SUPABASE_URL` / `SUPABASE_ANON_KEY` / `SUPABASE_SERVICE_ROLE_KEY` — never leave `api/`/`db/` now. Single-source-access change: the Blocker extension used to ship the anon key and talk to Supabase directly; it now only knows `API_BASE` and holds zero Supabase credentials.
- `DATABASE_URL` — Supabase's pooler connection string (transaction mode), used for all normal queries.
- `DATABASE_DIRECT_URL` — a session-scoped connection (Supabase pooler on port 5432, or a direct `db.<ref>.supabase.co:5432` connection if your network has IPv6 egress). Required for `LISTEN/NOTIFY` — the transaction-mode pooler on `DATABASE_URL` recycles the underlying server connection between statements and silently drops a session-scoped `LISTEN`. Realtime push is disabled (logged once, not fatal) if this isn't set.
- `GEMINI_API_KEY` — Layer 2 vision calls