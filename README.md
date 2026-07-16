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
| POST | `/trustpositif/verify` | `{domain}` | `{domain, is_blocked}` |

**Module boundary:**

- **Core** (owns detection logic, no Chrome/realtime/UI imports): `/analyze`, `/fingerprint`
- **Blocker** (Chrome adapter read surface): `/blocklist`, `/check`
- **Presentation** (dashboard): `/domains`, `/domains/{id}`, `/bootstrap/latest`

**Realtime channel (one Blocker adapter, not the architecture):** Supabase `postgres_changes` on `domains`, filter `status=eq.blocked`.

## Environment Variables

See `.env.example`. Real values distributed via the team WhatsApp group, never committed.

- `SUPABASE_URL` — project URL
- `SUPABASE_ANON_KEY` — public, ships in the Blocker extension
- `SUPABASE_SERVICE_ROLE_KEY` — never leaves `core/` + `api/`
- `GEMINI_API_KEY` — Layer 2 vision calls