<div align="center">
  <img src="https://openmoji.org/data/color/svg/1F6D1.svg" width="72" alt="stop sign" />
  <h1>PRIME</h1>
  <p>Block the domain before it does damage — not after it goes viral.</p>

  ![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square)
  ![PostgreSQL](https://img.shields.io/badge/PostgreSQL-Supabase-336791?style=flat-square)
  ![Chrome Extension](https://img.shields.io/badge/Chrome-MV3-4285F4?style=flat-square)
  ![Gemini](https://img.shields.io/badge/Vision-Gemini%202.x-8E44AD?style=flat-square)
  ![License](https://img.shields.io/badge/License-Unreleased-lightgrey?style=flat-square)
  ![Track](https://img.shields.io/badge/GarudaHacks%207.0-Safety-22c55e?style=flat-square)
</div>

---

Judol (online gambling) blocking in Indonesia is whack-a-mole: authorities block a domain only after it is live and reported, while operators register replacements in bulk faster than takedowns land. PRIME starts with an empty database and builds its own blocklist — one vision-confirmed site yields an infrastructure fingerprint, and that fingerprint preemptively blocks its siblings, domains nobody has ever visited. Every confirmation propagates to every connected device in real time, so one person's exposure protects everyone, and the system gets more preemptive the more it is used.

PRIME is a GarudaHacks 7.0 (Track: Safety) hackathon project, built by a four-person team over 24 hours.

---

### Table of Contents

- [What is PRIME?](#what-is-prime)
- [Features](#features)
- [Tech Stack](#tech-stack)
- [Architecture](#architecture)
- [API Reference](#api-reference)
- [Module Boundaries](#module-boundaries)
- [Getting Started](#getting-started)
- [Environment Variables](#environment-variables)
- [Security](#security)

---

### <div id="what-is-prime">What is PRIME?</div>

PRIME (the name of the API module is `prime`) is a two-layer detection system for online gambling domains, delivered today as a Go API plus a Chrome MV3 extension. The core idea is a compounding loop rather than a static list:

- **Layer 2 (reactive, the bootstrap path)** — a domain nobody has classified yet gets a screenshot sent to a vision model (Gemini). If it looks like gambling, the domain is confirmed and blocked.
- **Layer 1 (preemptive, grows from Layer 2)** — every Layer 2 confirmation is fingerprinted (hosting IP, ASN, nameserver, registrar, TLD, registration date) and grouped into clusters. Bulk-registration bursts — many domains registered by the same registrar in a narrow time window — are scored deterministically. New, never-visited domains that match a cluster are blocked before anyone opens them.

There is no machine learning in Layer 1 by design: deterministic field-correlation scoring is what lets the block page name the exact fields that matched (`matched_fields`), instead of handing a user an unexplainable confidence number. The only AI in the system is the Layer 2 vision call, and the project treats that as a means, not the innovation — the innovation is the feedback loop that turns one confirmation into many preemptive blocks.

See [docs/PRD.md](docs/PRD.md) for the full product brief, and [docs/EPICS.md](docs/EPICS.md) / [docs/TASKS.md](docs/TASKS.md) for how the work was broken down.

---

### <div id="features">Features</div>

- **Cold-start detection** — the database starts empty; every Layer 1 lookup returns no-match until Layer 2 seeds the first cluster
- **Layer 2 vision bootstrap** — page evidence in, verdict out: `{is_judol, confidence, reason}`, backed by Gemini
- **Layer 1 fingerprint matcher** — scores an unseen domain against known clusters and reports which fields matched
- **Bulk-registration burst detection** — flags domains registered in the same narrow window by the same registrar, the strongest preemptive signal
- **Feedback loop (L2 → L1)** — a confirmed detection automatically seeds or updates a fingerprint cluster; this loop is what makes the system preemptive at all
- **Realtime propagation** — a WebSocket relay (`GET /realtime`) built on Postgres `LISTEN/NOTIFY` pushes every new block to all connected clients within seconds
- **Chrome extension (Manifest V3)** — blocks in-browser via `declarativeNetRequest`, with a realtime adapter and a 3-second polling fallback behind the same contract
- **Explainable block page** — every block shows a confidence score and plain-language reasons derived from `matched_fields`, never a bare number
- **Cold-start proof, not a marketing number** — `GET /bootstrap/latest` reports live counters: how many Layer 2 confirmations produced how many Layer 1 preemptive catches
- **False-positive reporting** — a domain can be unblocked client-side immediately and flagged server-side, no waiting on the next sync

---

### <div id="tech-stack">Tech Stack</div>

| Layer | Technology |
|---|---|
| Core detection | Go 1.25+ (stdlib only — no HTTP framework, no Chrome, no realtime imports) |
| API transport | Go `net/http` (stdlib `ServeMux`) via [pgx/v5](https://github.com/jackc/pgx) |
| Vision (Layer 2) | Gemini 2.x Flash vision API |
| WHOIS / DNS / RDAP | Go `net` + `net/http` + raw WHOIS over `net.Dial` — no third-party client |
| Database | PostgreSQL (Supabase) — pooler connection for queries, direct connection for `LISTEN/NOTIFY` |
| Realtime transport | WebSocket ([nhooyr.io/websocket](https://github.com/nhooyr/websocket)) fed by a Postgres trigger + `NOTIFY` |
| Blocker | Chrome Extension, Manifest V3, vanilla JS, `declarativeNetRequest` |
| Presentation | Plain HTML/CSS block page, shipped inside the extension bundle |

---

### <div id="architecture">Architecture</div>

#### Module split

The system is deliberately split into three modules so the browser is only one delivery surface — the roadmap target is Android (`VpnService`), where the Blocker and Presentation layers change completely and the Core Engine does not.

```
Core Engine (core/)            Detection only. Emits a Verdict.
    │                          Knows nothing about Chrome, realtime, or UI.
    ▼
Blocker Service (blocker/, api/) Enforcement + verdict transport.
    │                          declarativeNetRequest + realtime/polling adapters.
    ▼
Presentation Layer              Block page + dashboard surfaces.
                                 Reads verdicts, never writes them.
```

#### The compounding loop

```
Empty database
   └── fingerprint_clusters is EMPTY
       └── every Layer 1 lookup returns no-match, by construction
           └── every first visit to a domain falls through to Layer 2

Layer 2 confirms a domain (screenshot → Gemini → verdict)
   └── verdict written: status='blocked', source='L2'
       └── feedback loop extracts a fingerprint → seeds/updates a cluster
           └── Layer 1 now has something to correlate against

Layer 1 scores an unvisited domain against clusters
   └── match: same hosting IP, same registration-window burst
       └── domain blocked before the page ever renders
           └── block page explains why, using matched_fields directly
```

#### Realtime propagation

```
Device 1 confirms domain X (Layer 2)
   └── trigger on domains fires pg_notify('domain_blocked', ...)
       └── Go API's GET /realtime relay (LISTEN on DATABASE_DIRECT_URL)
           └── fans out to every connected WebSocket client
               └── Device 2, never having visited X, blocks it within seconds
```

The Blocker never talks to Postgres directly. `/blocklist`, `/check`, and `/realtime` are the single access surface — this keeps database credentials out of the extension entirely.

---

### <div id="api-reference">API Reference</div>

Base URL: `/api/v1` · JSON for every route except `GET /realtime`, which is a WebSocket upgrade · no auth in the MVP (see [Security](#security)).

```
GET  /blocklist               ?since=<ISO>            -> {domains:[...], updated_at}
POST /check                   {domain}                 -> {status, confidence, source, reason}
POST /analyze                 {domain, evidence_b64}   -> {is_judol, confidence, reason, domain_id}
POST /fingerprint              {domain}                 -> {cluster_id, registrar, ip, ns, tld, match_score, matched_fields}
GET  /domains                 ?limit&offset&source&status -> {items:[...], total}
GET  /domains/{id}            —                         -> {domain, detections[], whois, cluster, siblings[], evidence_url}
POST /report-false-positive   {domain_id, note}         -> {ok:true}
GET  /bootstrap/latest        —                         -> {l2_confirmations, l1_preemptive_catches, l1_misses, ratio}
POST /trustpositif/verify     {domain}                  -> {domain, is_blocked}   (permanent stub, always false)
GET  /realtime                —                         -> WebSocket upgrade, pushes one block event per message
```

**Example — bootstrap a domain through Layer 2:**
```bash
curl -X POST http://localhost:8000/api/v1/analyze \
  -H "Content-Type: application/json" \
  -d '{"domain":"gacor88x.xyz","evidence_b64":"<base64 JPEG>","evidence_type":"screenshot"}'
```

The decision (`layer2.Decide`) and the Layer 1 feedback loop (`core.Feedback`, `core.MatchSiblings`) run in a background goroutine after the response returns — a `200` here does not mean the blocklist is updated yet. Poll `GET /blocklist` or connect to `GET /realtime` to observe the result.

`POST /trustpositif/verify` is a permanent stub: `trustpositif.komdigi.go.id`'s search form requires solving a reCAPTCHA token client-side, which this project will not automate. It exists only so callers that expect it in the contract never need special-casing — never present it as real corroboration.

Full route-by-route reference, request/response bodies, and edge cases: [docs/API_DOCS.md](docs/API_DOCS.md).

---

### <div id="module-boundaries">Module Boundaries</div>

Ownership is enforced by import discipline, not tooling — `core/` importing anything Chrome-shaped, realtime-shaped, or UI-shaped breaks the Android portability the whole seam exists for.

| Module | Owns | May import | Endpoints |
|---|---|---|---|
| **Core** | Detection logic (Layer 1 + Layer 2 + feedback loop) | stdlib only | `/analyze`, `/fingerprint` |
| **Blocker** | Chrome adapter, verdict transport | Core's verdict contract | `/blocklist`, `/check`, `/realtime` |
| **Presentation** | Block page, dashboard | Blocker/API read surface | `/domains`, `/domains/{id}`, `/bootstrap/latest` |

> `core/contract.go` is the seam. The test before any Core commit: *would this file still compile if the extension didn't exist?* If no, it belongs somewhere else.

---

### <div id="getting-started">Getting Started</div>

**Prerequisites:** Go 1.25+, a PostgreSQL/Supabase project, a Gemini API key.

**1. Clone the repo and configure the environment:**
```bash
git clone <this-repo-url>
cd Atraps_PREJUDGE
cp .env.example .env
# fill in SUPABASE_URL, DATABASE_URL, DATABASE_DIRECT_URL, GEMINI_API_KEY
```

**2. Apply the schema** (no migration tool — one idempotent `schema.sql`, safe to re-run):
```bash
psql "$DATABASE_URL" -f db/schema.sql
psql "$DATABASE_URL" -f db/fixtures.sql   # optional, demo data — purge before any real demo
```

**3. Build and run the API:**
```bash
go build -o prime-api ./api
./prime-api
# or, without a separate build step:
go run ./api
```

The server listens on `:8000` (hardcoded). Verify it came up:
```bash
curl http://localhost:8000/api/v1/blocklist
```
An empty `domains` array is expected on a fresh database.

**4. Load the Chrome extension:**
```
chrome://extensions -> Developer mode -> Load unpacked -> select blocker/
```

Full deploy notes (two database URLs, running behind a tunnel, exposing this beyond a laptop): [docs/API_DEPLOY_DOCS.md](docs/API_DEPLOY_DOCS.md).

**Run the test suite:**
```bash
go build ./...
go vet ./...
go test ./...
```
No test requires a live database or a real Gemini key.

---

### <div id="environment-variables">Environment Variables</div>

Copy `.env.example` to `.env`. `.env` is gitignored — never commit real values.

| Variable | Required | Used for |
|---|---|---|
| `DATABASE_URL` | Yes | Every normal query — Supabase pooler connection string, transaction mode |
| `DATABASE_DIRECT_URL` | For realtime | Session-scoped connection required for `LISTEN/NOTIFY`; without it `GET /realtime` accepts connections but never pushes anything |
| `GEMINI_API_KEY` | For `/analyze` | Layer 2 vision calls. If unset, `/analyze` degrades to a stub response instead of failing |
| `GEMINI_MODEL` | No | Overrides the vision model, defaults to `gemini-2.0-flash` |
| `GEMINI_CACHE_DIR` | No | Writable directory for fallback verdicts if a live Gemini call fails |
| `SUPABASE_URL` | No | Not read by the Go API directly; kept for parity with the Supabase dashboard |
| `SUPABASE_ANON_KEY` | No | Never shipped to the extension — see [Security](#security) |
| `SUPABASE_SERVICE_ROLE_KEY` | No | Never leaves `api/`/`db/` |

Missing `DATABASE_URL` does not crash the server — `/fingerprint` and the domain endpoints report empty state and log a warning instead.

---

### <div id="security">Security</div>

| Aspect | Current state |
|---|---|
| Authentication | None in the MVP — every route under `/api/v1` is open |
| CORS | Wildcard (`Access-Control-Allow-Origin: *`) |
| Credential isolation | The extension holds zero Supabase credentials; `SUPABASE_URL`/`SUPABASE_ANON_KEY`/`SUPABASE_SERVICE_ROLE_KEY` never leave `api/`/`db/` — the API is the single point of database access, for both reads and the realtime push |
| Row-level security | Disabled on all tables for the MVP — a locked RLS policy silently breaking realtime mid-demo was judged the worse failure mode |
| Explainability | Layer 1 is deterministic field correlation, not ML, specifically so every block can show the exact `matched_fields` that triggered it instead of an unexplainable score |
| WebSocket origin | The `GET /realtime` upgrade currently allows any origin so the unpacked extension can connect during development; this is a known gap to close before any public deployment (see CSWSH note in `api/realtime.go`) |

**Before exposing this beyond a laptop:** put an API key or token check in front of every route. None of the above is production-hardened — it reflects the tradeoffs of a 24-hour build running on a single demo machine, documented honestly rather than silently. Full detail: [docs/API_DEPLOY_DOCS.md](docs/API_DEPLOY_DOCS.md).

---

<div align="center">
  <sub>Built for GarudaHacks 7.0 — Track: Safety</sub>
</div>
