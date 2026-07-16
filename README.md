# Atraps_PREJUDGE

## Roles & Responsibilities (4 members)

| | Role | Owns | Primary Deliverable |
|---|---|---|---|
| **A** | Backend / Data Lead | Layer 1 (preemptive detection), database schema, all REST endpoints | Fingerprint extraction + cluster matcher + `/blocklist` endpoint |
| **B** | Backend / AI | Layer 2 (reactive detection via vision API), Gemini integration, validation script | Screenshot → Gemini → verdict → DB write → fingerprint feedback loop |
| **C** | Frontend (Extension + Dashboard) | `extension/`, `dashboard/` | Working two-device realtime block + dashboard |
| **D** | Pitch, Research & QA | `pitch/`, source verification, demo rehearsal | 2-min deck + demo script + verified `sources.md` |

See [PRD.md](PRD.md) §12 for load-balancing notes and full context.