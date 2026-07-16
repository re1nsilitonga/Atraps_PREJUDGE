# PREJUDGE — Task Backlog

Granular tickets derived from EPICS.md. Scope is strictly MVP must-haves + demo reliability.

**ID scheme:** `PJ-{epic}{seq}` — e.g. `PJ-101` = Epic 1, task 1.
**Sizing:** S ≈ <45min · M ≈ 45–90min · L ≈ 90min–2h.

**Two facts that shape every ticket below:**

1. **The DB starts empty.** No bulk ground truth exists (TrustPositif results are masked: `a*****gacor.biz`). Layer 2 seeds Layer 1. The chain is serial: Epic 2 → 3 → 4.
2. **Core / Blocker / Presentation are separable.** `core/` imports nothing Chrome-shaped, nothing realtime-shaped, nothing UI-shaped. That seam is the Android roadmap's only insurance.

---

# EPIC 1 — Foundation: Schema + Core Contract

---

## PJ-101 · Stand up Supabase + distribute credentials

**Owner:** A · **Size:** S · **Blocks:** everything

**Description**
Create the Supabase project, enable Realtime, get keys into everyone's hands before anyone needs them.

**Acceptance Criteria**

- [ ] Supabase project exists, region closest to venue
- [ ] Realtime enabled for `public` schema
- [ ] `.env.example` at repo root with every key name (no values)
- [ ] Real `.env` values pinned in team WA group
- [ ] All 4 members confirm they can connect

**Technical Notes**

- Files: `.env.example`, `README.md`
- Keys: `SUPABASE_URL`, `SUPABASE_ANON_KEY`, `SUPABASE_SERVICE_ROLE_KEY`, `GEMINI_API_KEY`
- Anon key ships in the Blocker (public by design). Service role key **never** leaves `core/` + `api/`.
- PRD §8: keys distributed at T+0. Nobody hunts for keys at 3am.

---

## PJ-102 · Define `core/contract.go` — the verdict seam

**Owner:** A · **Size:** M · **Depends:** none · **⚠️ FREEZE AT T+2**

**Description**
The single data structure Core emits and every Blocker adapter consumes. This is the Core/Blocker seam — the thing that makes Android a port instead of a rewrite.

**Acceptance Criteria**

- [ ] `Verdict` dataclass: `domain, is_judol, confidence, reason, matched_fields, source (L1|L2), detected_at`
- [ ] `Evidence` dataclass: `domain, evidence_b64, evidence_type` — **generic, not "screenshot"**
- [ ] File imports **nothing** beyond the Go standard library. No HTTP framework, no supabase client, no third-party HTTP client.
- [ ] Documented in README as the binding contract
- [ ] Frozen at T+2. Changes require all four.

**Technical Notes**

- File: `core/contract.go`
- **The seam test:** would this file compile if the extension didn't exist? If no, it's wrong.
- `EvidenceType` matters: Chrome sends a screenshot, Android VpnService will send DNS/SNI. Core must not care. Naming this field `Screenshot` now costs a rename during Phase 1.
- PRD §14 risk #16: at T+20 someone will want to `import chrome` into Core to fix a bug. This file is the line.

---

## PJ-103 · Implement schema.sql

**Owner:** A · **Size:** M · **Depends:** PJ-101

**Description**
Full DDL per PRD §9 ERD, including the registration-burst fields that burst detection needs.

**Acceptance Criteria**

- [ ] 5 tables: `domains`, `fingerprint_clusters`, `detections`, `whois_records`, `bootstrap_runs`
- [ ] Enums: `domain_status` (candidate|confirmed|blocked|false_pos), `detection_source` (L1|L2|trustpositif)
- [ ] **`fingerprint_clusters` includes:** `first_registration_date`, `last_registration_date`, `registration_window_hours`, `registration_burst_score`
- [ ] **`domains` includes:** `matched_fields jsonb`, `reason text` (denormalized), `source_masked_pattern text` (nullable)
- [ ] **`detections.evidence_url`** — not `screenshot_url`
- [ ] FKs wired, `domains.domain` UNIQUE, index on `domains(status)`
- [ ] Idempotent

**Technical Notes**

- File: `db/schema.sql`
- The four burst fields are new and load-bearing: PRD §7's block page promises "Didaftarkan massal, 3 hari lalu" and nothing else produces it.
- `evidence_url` not `screenshot_url` — Android has no pixels. Rename now; a migration mid-hackathon is worse.
- RLS: disable for MVP or add permissive read on `domains`. Locked RLS silently breaking realtime at T+22 is a real failure mode.

---

## PJ-104 · Fixture data (NOT seed data)

**Owner:** A · **Size:** S · **Depends:** PJ-103 · **Unblocks:** C

**Description**
Fake rows so Presentation can build at T+1. **Explicitly marked and purgeable** — production starts empty by design.

**Acceptance Criteria**

- [ ] ~20 fixture rows, mixed status/source/confidence
- [ ] ≥1 `fingerprint_clusters` row with ≥5 attached domains (so C can build the sibling view)
- [ ] Populated `matched_fields` and burst fields
- [ ] Every row carries a `-- FIXTURE` marker or identifiable flag
- [ ] **Purge script exists and is tested**
- [ ] C confirms dashboard reads it

**Technical Notes**

- File: `db/fixtures.sql` + purge in the same file
- **These are fixtures, not seeds.** PRD §4: the DB starts empty by design and §15's opening beat _is_ the empty database. PRD §14 risk #12 — leftover fixtures destroy that beat while looking like success.
- Name them obviously (`fixture-cluster-1.test`), not plausibly. Plausible fixtures are the ones that survive to demo.

---

## PJ-105 · Publish API contract in README

**Owner:** A · **Size:** S · **Unblocks:** B, C

**Description**
The PRD §10 endpoint table as binding contract, before implementation.

**Acceptance Criteria**

- [ ] All 9 endpoints: method, path, request, response
- [ ] **Module boundary documented:** `/analyze`, `/fingerprint` = Core. `/blocklist`, `/check` = Blocker read surface. `/domains`, `/bootstrap/latest` = Presentation.
- [ ] Realtime channel spec: `postgres_changes` on `domains`, filter `status=eq.blocked` — **labelled as one adapter, not the architecture**
- [ ] `.env` names documented
- [ ] Committed at T+0

**Technical Notes**

- File: `README.md`
- Base `/api/v1`
- Changes after T+4 require all four (PRD §11).

---

## PJ-106 · Stub all API endpoints

**Owner:** A · **Size:** M · **Depends:** PJ-105 · **Unblocks:** B, C

**Description**
Go `net/http` server returning hardcoded contract-shaped responses. Nobody is blocked on anyone.

**Acceptance Criteria**

- [ ] All 9 routes respond 200 with correct shapes (fake values)
- [ ] `api/main.go` is **thin** — no detection logic, calls into `core/`
- [ ] CORS enabled for extension + `localhost:3000`
- [ ] Done by T+2

**Technical Notes**

- Files: `api/main.go`, `api/models.go`, `go.mod`
- Standard library only: `net/http` (Go 1.22+ pattern-based `ServeMux`), `encoding/json`
- `api/models.go` structs mirror `core/contract.go` — they do not replace it. Core stays framework-free.
- CORS: `chrome-extension://*` isn't a valid origin match. Use a wildcard `Access-Control-Allow-Origin: *` middleware and move on.

---

## PJ-107 · Freeze schema + contract

**Owner:** A · **Size:** S · **Depends:** PJ-102, PJ-103, PJ-104

**Description**
Contract frozen T+2, schema frozen T+4. Ceremony, but the kind that prevents 3am chaos.

**Acceptance Criteria**

- [ ] `core/contract.go` header: `// FROZEN T+2. The seam. Changes require all 4.`
- [ ] `db/schema.sql` header: `-- FROZEN T+4. Changes require all 4.`
- [ ] Both announced in WA

**Technical Notes**

- If the contract churns, the modularity is theatre and the §16 Android claim isn't honest.

---

# EPIC 2 — Core: Layer 2 Bootstrap (Vision)

> **This is the seed, not the fallback.** Empty DB → every first access lands here. No Layer 2, no data, no system.

---

## PJ-201 · Gemini vision client

**Owner:** B · **Size:** M · **Depends:** PJ-101, PJ-102

**Description**
Evidence in → judol verdict out. Core's only AI call.

**Acceptance Criteria**

- [ ] `Analyze(evidence core.Evidence) (core.Verdict, error)` — signature matches `core/contract.go`
- [ ] Malformed response → `is_judol=False`, logged, no crash
- [ ] Raw response retained for `detections.raw_response`
- [ ] Verified by hand on one real judol screenshot **by T+2**
- [ ] `reason` is one plain-language **Indonesian** sentence — renders verbatim on the block page
- [ ] File imports no Chrome APIs, no supabase client

**Technical Notes**

- File: `core/layer2/vision.go`
- Gemini 2.x Flash vision (PRD §8). Model name in a constant.
- Prompt per §4: _"Apakah ini situs judi online? Ya/tidak, alasan singkat."_ Request JSON-only, strip markdown fences before parsing.
- **Not our innovation** (PRD §4). No fine-tuning. Get a verdict and move on.
- `reason` is for Rina from Bekasi (§3), not a developer. Prompt for plain language.
- PRD §14 risk #4: **Gemini down at demo = no bootstrap = no system.** Higher stakes than the original plan, where TrustPositif seeded L1 independently.

---

## PJ-202 · Evidence capture (Chrome-specific)

**Owner:** B + C · **Size:** M · **Depends:** PJ-301

**Description**
Blocker captures page evidence on unknown-domain visit, POSTs to `/analyze`.

**Acceptance Criteria**

- [ ] Fires only when domain is **not** in the local blocklist
- [ ] `chrome.tabs.captureVisibleTab`, base64
- [ ] POSTs `{domain, evidence_b64, evidence_type: "screenshot"}` to `/api/v1/analyze`
- [ ] Payload downscaled/compressed if oversized
- [ ] Failure silent to the user
- [ ] **Lives in `blocker/evidence.js`, not in `core/`**

**Technical Notes**

- Files: `blocker/evidence.js`, `blocker/background.js`
- `captureVisibleTab` requires `activeTab`/`<all_urls>` **and only works from the background service worker**, not a content script. Message-pass content → background.
- JPEG quality ~50 is plenty. A slot-machine UI is not subtle.
- **This file does not port to Android** — VpnService sees DNS/SNI, not pixels. That's expected and fine: it's isolated on purpose (PRD §4). It is _not_ a Core file and must never be imported by one.
- Cross-owner: B and C agree the message shape before either starts.

---

## PJ-203 · Verdict decision + DB write

**Owner:** B · **Size:** M · **Depends:** PJ-201, PJ-103

**Description**
Verdict → database state change. `status='blocked'` is what Blocker adapters observe.

**Acceptance Criteria**

- [ ] `L2_CONFIDENCE_THRESHOLD` as a named constant
- [ ] Above → upsert `domains`: `status='blocked'`, `source='L2'`, populated `reason` + `matched_fields`
- [ ] `detections` row: `layer=2`, confidence, reason, `evidence_url`, `raw_response`
- [ ] Below → `detections` logged, `domains.status` unchanged
- [ ] Upsert on unique `domain` — no dupes on repeat visits

**Technical Notes**

- File: `core/layer2/decide.go`
- `L2_CONFIDENCE_THRESHOLD = 0.8`
- **Core writes the verdict. Core does not know who's listening.** The original plan coupled this to "coordinate with C before changing how this write happens" — that coupling is now deleted by design. Blocker adapters observe `status='blocked'`; Core emits it and forgets.
- Service role key here. Never anon.

---

## PJ-204 · Endpoint: POST /analyze

**Owner:** B · **Size:** S · **Depends:** PJ-201, PJ-203, PJ-106

**Description**
Real Layer 2 behind the stub.

**Acceptance Criteria**

- [ ] `POST /api/v1/analyze` `{domain, evidence_b64}` → `{is_judol, confidence, reason, domain_id}`
- [ ] Shape unchanged from stub
- [ ] <8s or clean timeout
- [ ] Feedback loop (PJ-301) fires in background, does not delay response

**Technical Notes**

- File: `api/main.go` — thin, calls `core/layer2/`
- base64 in JSON body, not multipart. Simpler from the extension.

---

## PJ-205 · Pre-cache verdicts for demo domains

**Owner:** B · **Size:** S · **Depends:** PJ-201, PJ-203 · **DO AT T+14**

**Description**
Run Layer 2 on demo domains ahead of time so a Gemini outage during judging is invisible.

**Acceptance Criteria**

- [ ] All demo domains have cached verdicts on disk
- [ ] `vision.analyze()` checks cache first, returns cached on API failure
- [ ] Verified by simulating failure (bad key) — demo still works

**Technical Notes**

- Files: `core/layer2/vision.go`, cache `db/cache/vision_*.json`
- PRD §14 risk #4.
- Cache-on-failure, not cache-always — live call still happens when the API is up. Judges may ask if it's live.
- **This ticket doubles as demo cluster bootstrap** (see PJ-801). Same runs, two purposes.

---

# EPIC 3 — Core: The Feedback Loop (L2 → L1)

> **The most important epic in the project.** Without it Layer 1 has no input at all — not weaker, _none_. It is also the entire Innovation & Novelty defense.

---

## PJ-301 · L2 verdict → L1 cluster seeding

**Owner:** B · **Size:** M · **Depends:** PJ-203, PJ-401

**Description**
A Layer 2 confirmation extracts the domain's infrastructure fingerprint and seeds/updates a cluster. This is how Layer 1 comes into existence from nothing.

**Acceptance Criteria**

- [ ] L2-confirmed domain triggers `fingerprint.extract()`
- [ ] Fingerprint written; domain joins an existing cluster or **creates the first one**
- [ ] Sibling domains sharing the fingerprint become Layer 1 candidates
- [ ] **Demonstrable end-to-end from an empty DB:** confirm X → cluster appears where none existed → sibling Y flagged without ever being visited
- [ ] Runs async — must not block the verdict response

**Technical Notes**

- File: `core/feedback.go` → calls `core/layer1/fingerprint.go`, `core/layer1/cluster.go`
- A `go func() { ... }()` goroutine is enough. No queue, no worker system.
- **PRD §5 + §14 risk #10:** the entire Innovation defense rests on this loop. If Epic 2 gets cut for time, cut PJ-202 polish — never this.
- Must be visible in the demo (§15, 1:05–1:35 beat: _"Layer 1 emerges from nothing"_).
- **Under cold start this is on the critical path, not a nice-to-have.** The old plan let Layer 1 be seeded independently from TrustPositif, so this loop was an enhancement. Masking killed that. Now: no loop → no Layer 1 → no preemptive claim → no pitch.

---

# EPIC 4 — Core: Layer 1 Preemptive Detection

> Deterministic arithmetic. **No ML, deliberately** — explainability is what the block page depends on (§4).

---

## PJ-401 · Fingerprint extractor + fill-rate report

**Owner:** A · **Size:** L · **Depends:** PJ-102 · **🚩 GATE AT T+5**

**Description**
Given a domain, pull hosting IP, ASN, nameservers, registrar, TLD, `registered_at` via WHOIS/RDAP/DNS. Then report how often each field is actually populated.

**Acceptance Criteria**

- [ ] Returns `{registrar, hosting_ip, asn, nameserver, tld, registered_at}`
- [ ] Redacted WHOIS → `None`, not exceptions
- [ ] Raw response → `whois_records.raw`
- [ ] Cached — never re-query a fetched domain
- [ ] **Fill-rate report over ≥30 domains, per field, by T+5**
- [ ] Imports no Chrome, no supabase-realtime

**Technical Notes**

- File: `core/layer1/fingerprint.go`
- Go `net` package for DNS/nameserver lookups, `net/http` for RDAP, raw WHOIS via `net.Dial("tcp", "whois.iana.org:43")` — no third-party client needed
- **PRD §14 risk #8 (High).** Don't build on registrant fields — widely redacted. Use hosting IP + ASN + nameserver + TLD + `registered_at`.
- **The T+5 gate is specifically about `registered_at`.** If its fill rate is poor, **burst detection (PJ-403) is dead and "preemptive" becomes unsupportable** — cut the claim from the deck at T+5, not at T+20. The original plan weighted `registered_at` at 0.05 as if it didn't matter; under the Netcraft mechanism it's the whole signal.
- ASN: if it costs >30min, drop it and use IP /24 prefix.

---

## PJ-402 · Cluster builder

**Owner:** A · **Size:** M · **Depends:** PJ-401

**Description**
Group confirmed domains sharing infrastructure. "Same landlord, same street" made concrete.

**Acceptance Criteria**

- [ ] Grouped by shared hosting IP / nameserver / registrar
- [ ] `fingerprint_clusters` written with accurate `domain_count`
- [ ] `domains.cluster_id` populated
- [ ] Re-runnable without duplicating clusters
- [ ] **Handles the empty case cleanly** — zero clusters is the valid starting state, not an error

**Technical Notes**

- File: `core/layer1/cluster.go`
- Grouping is a `GROUP BY`, not clustering ML. **Do not import a clustering/ML library.**
- If no cluster reaches 5 members, loosen the key (IP → /24 prefix) before assuming the data is bad.
- Empty-case handling matters more than it used to: under cold start, empty is where the system _starts_, and the demo's opening beat depends on it not throwing.

---

## PJ-403 · Registration-burst detection

**Owner:** A · **Size:** M · **Depends:** PJ-402 · **Blocked by PJ-401 gate**

**Description**
Detect clusters of domains registered in the same narrow window by the same registrar. **This is the actual Netcraft/PREDATOR signal** — it exploits the gap between registration and campaign activation. Correlation alone is reactive-but-clustered; the burst is what makes Layer 1 genuinely _preemptive_.

**Acceptance Criteria**

- [ ] Per cluster: compute `first_registration_date`, `last_registration_date`, `registration_window_hours`
- [ ] `registration_burst_score` = f(domain_count, window_hours) — dense window + many domains = high score
- [ ] Score contributes to the matcher (PJ-404)
- [ ] **Produces the block page string: "Didaftarkan massal, N hari lalu"**
- [ ] Degrades cleanly when `registered_at` is null — burst score becomes null, other fields still match

**Technical Notes**

- File: `core/layer1/cluster.go`
- **This existed nowhere in the original backlog.** PRD §4 named it, §6 listed it, §7's wireframe promised the bullet — no ticket built it. It would have shipped as an empty or hardcoded bullet in front of judges.
- Suggested: `burst_score = domain_count / max(window_hours, 1)`, normalized. Tune with real data.
- **If the T+5 gate failed, this ticket is cut and §15's script drops the burst line.** Do not fake it.

---

## PJ-404 · Cluster similarity matcher

**Owner:** A · **Size:** L · **Depends:** PJ-402, PJ-403

**Description**
Score an unseen domain against known clusters. The Layer 1 detection claim.

**Acceptance Criteria**

- [ ] Input domain → `{cluster_id, match_score (0–1), matched_fields[]}`
- [ ] Weights in one named constant, not scattered magic numbers
- [ ] **No-match returns cleanly** — the normal case on an empty/young DB
- [ ] `matched_fields` is human-readable — feeds the block page "Kenapa?" bullets directly
- [ ] `MATCH_THRESHOLD` as a named constant

**Technical Notes**

- File: `core/layer1/matcher.go`
- Suggested starting weights (tune, don't treat as gospel):
  `hosting_ip 0.30 · nameserver 0.25 · registration_burst 0.25 · registrar 0.10 · tld 0.10`
- **Burst is weighted at 0.25, not 0.05.** The original weighting treated registration date as a rounding error; under the Netcraft mechanism it's a primary signal. If the T+5 gate failed, redistribute burst's 0.25 across IP and nameserver and say so.
- `MATCH_THRESHOLD = 0.6`
- **`matched_fields` is a UX deliverable, not a debug field.** PRD §5 scores transparency; §14 risk #11's "where's the AI" answer depends on it. This is _why_ Layer 1 has no ML.

---

## PJ-405 · Endpoint: POST /fingerprint

**Owner:** A · **Size:** S · **Depends:** PJ-404, PJ-106

**Description**
Real extraction + matching behind the stub.

**Acceptance Criteria**

- [ ] `POST /api/v1/fingerprint` `{domain}` → `{cluster_id, registrar, ip, ns, tld, match_score, matched_fields}`
- [ ] Shape unchanged from stub
- [ ] Unknown domain → valid response with null cluster, not a 500
- [ ] <5s or clean timeout

**Technical Notes**

- File: `api/main.go` — thin wrapper over `core/layer1/`

---

# EPIC 5 — Blocker Service (Chrome Adapter)

> **One adapter, not the architecture.** Realtime and polling are two transports behind one Core contract. VpnService will be a third.

---

## PJ-501 · MV3 scaffold

**Owner:** C · **Size:** S

**Description**
Minimal MV3 extension loading unpacked without errors.

**Acceptance Criteria**

- [ ] Loads unpacked, zero console errors
- [ ] Background service worker registered
- [ ] Permissions: `declarativeNetRequest`, `declarativeNetRequestWithHostAccess`, `tabs`, `storage`, `host_permissions: <all_urls>`
- [ ] Popup opens

**Technical Notes**

- Files: `blocker/manifest.json`, `blocker/background.js`
- MV3, vanilla JS, no build step (PRD §8).
- **Service workers terminate when idle.** No module-level state; use `chrome.storage.local`.

---

## PJ-502 · Realtime adapter — proof of life

**Owner:** C · **Size:** M · **Depends:** PJ-501, PJ-103 · **⚠️ DO THIS FIRST, T+2**

**Description**
Subscribe to Supabase realtime, log `domains` changes to console. No blocking logic — just prove the pipe exists.

**Acceptance Criteria**

- [ ] Subscribes to `postgres_changes` on `domains`, filter `status=eq.blocked`
- [ ] Manual row update in Supabase dashboard → console message
- [ ] Reconnects after network drop
- [ ] **Working by T+2**
- [ ] Isolated behind an adapter interface — swappable with PJ-505

**Technical Notes**

- File: `blocker/background.js`
- Supabase JS via CDN bundle or vendored file — **no npm build step in the extension.**
- Anon key only.
- PRD §10: build and test this first. If realtime doesn't work, know at T+2, not T+12.
- **MV3 service worker termination will kill a websocket.** Test that the subscription survives idle — most likely silent failure.

---

## PJ-503 · declarativeNetRequest blocking engine

**Owner:** C · **Size:** L · **Depends:** PJ-501

**Description**
Block domains from a list via DNR dynamic rules, redirect to block page.

**Acceptance Criteria**

- [ ] Hardcoded list of 3 domains → all blocked → redirect to `blocked.html`
- [ ] Rules added/removed at runtime without reload
- [ ] Passes domain + confidence + `matched_fields` to the block page
- [ ] Non-listed domains unaffected
- [ ] **Blocking a real domain by T+5** ← gate

**Technical Notes**

- File: `blocker/background.js`
- `chrome.declarativeNetRequest.updateDynamicRules()` — dynamic, not static rulesets.
- Redirect: `{type:"redirect", redirect:{extensionPath:"/blocked.html?d=..."}}`
- Rule IDs must be unique ints. Counter in `chrome.storage.local`, **not memory** — the worker dies and resets it.

---

## PJ-504 · Blocklist sync

**Owner:** C · **Size:** M · **Depends:** PJ-502, PJ-503

**Description**
Populate DNR rules from verdicts — full fetch on startup, incremental via adapter.

**Acceptance Criteria**

- [ ] Startup: fetch all `domains` where `status='blocked'` → DNR rules
- [ ] **Empty result is the normal cold-start state** — zero rules is not an error
- [ ] Verdict event → new rule within 5s, no reload
- [ ] Blocklist cached in `chrome.storage.local`, survives worker restart
- [ ] `domain`, `confidence`, `reason`, `matched_fields` available to block page

**Technical Notes**

- Files: `blocker/background.js`
- Via `GET /api/v1/blocklist?since=` or direct Supabase read. Direct is fewer moving parts.
- Denormalized `reason` + `matched_fields` (PJ-103) pay off here — no client-side joins.
- Empty-state handling: on a fresh install the blocklist _is_ empty. Don't render an error.

---

## PJ-505 · Polling adapter

**Owner:** C · **Size:** M · **Depends:** PJ-504 · **BUILD AT T+8, NOT T+22**

**Description**
Second transport adapter: poll `/blocklist` every 3s. Must be indistinguishable from realtime to a judge.

**Acceptance Criteria**

- [ ] Polls `/blocklist?since=` on 3s interval
- [ ] **Single feature flag switches realtime ↔ polling**
- [ ] With realtime disabled, two-device demo still works and still looks instant
- [ ] Tested by deliberately breaking the subscription

**Technical Notes**

- File: `blocker/background.js`
- MV3: `setInterval` dies with the worker; `chrome.alarms` min period is 1min (too slow). For the 2-min demo window, keep the worker alive or accept `setInterval` and verify it survives.
- **PRD §14 risk #5 — "Build at T+8, not T+22."** The date is the ticket.
- **This ticket is also the seam's proof.** Two adapters, one Core contract, zero Core changes. That's the argument that Android is a port. Worth saying to judges if asked about the roadmap.

---

## PJ-506 · Endpoints: GET /blocklist + POST /check

**Owner:** A · **Size:** S · **Depends:** PJ-106, PJ-103

**Description**
The two endpoints the Blocker consumes.

**Acceptance Criteria**

- [ ] `GET /api/v1/blocklist?since=<ISO>` → `{domains:[{domain,confidence,reason,matched_fields}], updated_at}`
- [ ] `since` actually filters
- [ ] `POST /api/v1/check` `{domain}` → `{status, confidence, source, reason}`
- [ ] Unknown domain → clean not-found state, not a 404 the Blocker must catch
- [ ] Empty blocklist → `{domains:[], ...}`, not an error

**Technical Notes**

- File: `api/main.go`
- Reads denormalized `domains`. No joins — called constantly.

---

## PJ-507 · Endpoint: POST /report-false-positive

**Owner:** A · **Size:** S · **Depends:** PJ-106, PJ-601

**Description**
Back the block page's "Laporkan salah" button.

**Acceptance Criteria**

- [ ] `POST /api/v1/report-false-positive` `{domain_id, note}` → `{ok:true}`
- [ ] Sets `domains.status='false_pos'`
- [ ] Domain disappears from blocklist on next sync
- [ ] No auth (MVP)

**Technical Notes**

- File: `api/main.go`
- Status change → adapter event → Blocker should _unblock_. Verify once; a stuck block after clicking "salah" is an ugly demo moment (PRD §14 risk #14).

---

# EPIC 6 — Presentation Layer

---

## PJ-601 · Block page

**Owner:** C · **Size:** M · **Depends:** PJ-503

**Description**
What Rina actually sees. PRD §5 scores transparency — "user knows why, not a black box."

**Acceptance Criteria**

- [ ] Renders per §7 wireframe: shield, domain, confidence bar, "Kenapa?" bullets, two buttons
- [ ] Confidence as a visual bar, not a bare number
- [ ] **Bullets rendered from `matched_fields`** — real data, no placeholders
- [ ] **Burst bullet ("Didaftarkan massal, N hari lalu") renders when burst data exists, and is absent (not blank) when it doesn't**
- [ ] "Laporkan salah" present and wired (PJ-507)
- [ ] Copy in Indonesian

**Technical Notes**

- Files: `presentation/blocked/blocked.html`, `blocked.css`
- Params via query string or `chrome.storage.local` lookup.
- **The burst bullet is conditional, not decorative.** If PJ-403 was cut at the T+5 gate, this bullet must not render at all. An empty bullet in front of judges is worse than three bullets.
- The user is Rina from Bekasi (§3), not a developer.

---

## PJ-602 · Dashboard scaffold + list view

**Owner:** C · **Size:** M · **Depends:** PJ-101, PJ-104

**Description**
Next.js 14 + Tailwind + shadcn, reading fixtures, then real data.

**Acceptance Criteria**

- [ ] `npm run dev` clean; Tailwind + shadcn configured
- [ ] Table: domain, confidence, source (L1/L2), timestamp
- [ ] Header counters: today, L1, L2, live indicator
- [ ] **Live indicator reflects actual adapter state** — a hardcoded green dot is a lie a judge might catch
- [ ] Rows clickable → detail
- [ ] Real DB data by T+8
- [ ] **Deploys to Vercel** (do this once, early)

**Technical Notes**

- Files: `presentation/dashboard/app/page.tsx`, `lib/supabase.ts`
- `NEXT_PUBLIC_SUPABASE_ANON_KEY`
- Deploy early. A first-time Vercel deploy failing at T+22 is a known way to lose.

---

## PJ-603 · Domain detail + siblings view

**Owner:** C · **Size:** M · **Depends:** PJ-602

**Description**
Cluster drill-down. **This is the §15 1:05–1:35 beat** — where Layer 1 visibly emerges from a single confirmation.

**Acceptance Criteria**

- [ ] Route `/domain/[id]`
- [ ] Cluster info + `matched_fields` + **sibling domains in the same cluster**
- [ ] Burst info shown when present: window, domain count
- [ ] Evidence shown if it exists
- [ ] Detection history
- [ ] **Renders a cluster with ≥5 siblings legibly** — the demo beat

**Technical Notes**

- File: `presentation/dashboard/app/domain/[id]/page.tsx`
- `GET /api/v1/domains/{id}`
- Evidence storage: Supabase Storage bucket, or skip persistence and show reason text only. **Decide by T+8** — do not discover at T+16 that no bucket exists.

---

## PJ-604 · Cold Start tab

**Owner:** C · **Size:** M · **Depends:** PJ-602, PJ-703

**Description**
The honesty beat (§15, 0:20–0:35 and 1:50–2:00). Replaces the dead validation tab.

**Acceptance Criteria**

- [ ] Route `/bootstrap`
- [ ] Shows: L2 confirmations (N), L1 preemptive catches (M), misses, ratio
- [ ] **Renders all-zeros cleanly** — this is the demo's opening frame
- [ ] **Disclaimer on screen: "Bukan monitoring internet real-time."**
- [ ] Reads real `bootstrap_runs` data — no invented figures

**Technical Notes**

- File: `presentation/dashboard/app/bootstrap/page.tsx`
- `GET /api/v1/bootstrap/latest`
- **The all-zeros state is a feature, not an edge case.** §15 opens on it: _"This is our database. It's empty."_ If it renders an error or a spinner on empty, the demo's first beat dies.
- PRD §16 checklist: live counters, not placeholders. The old wireframe's 200/141/59 were illustrative and shipping them would be fabrication.

---

## PJ-605 · Endpoints: GET /domains, GET /domains/{id}

**Owner:** A · **Size:** S · **Depends:** PJ-106, PJ-103

**Description**
Dashboard read endpoints.

**Acceptance Criteria**

- [ ] `GET /api/v1/domains?limit&offset&source&status` → `{items:[...], total}`
- [ ] All filters work
- [ ] `GET /api/v1/domains/{id}` → `{domain, detections[], whois, cluster, siblings[], evidence_url}`
- [ ] **`siblings[]` included** — the detail view needs it
- [ ] Unknown id → 404

**Technical Notes**

- File: `api/main.go`
- Joins `domains` × `detections` × `fingerprint_clusters` per §9.

---

# EPIC 7 — Cold-Start Proof

---

## PJ-701 · TrustPositif single-domain verifier

**Owner:** A · **Size:** M · **Depends:** PJ-102

**Description**
Submit one **full** domain, get a boolean. Verifier, not seed source.

**Acceptance Criteria**

- [ ] `verify(domain) → bool`
- [ ] Results cached to disk
- [ ] Rate limited, configurable delay
- [ ] Graceful failure — unavailability does not break anything downstream
- [ ] **No bulk-harvest code path exists**

**Technical Notes**

- File: `core/trustpositif.go`
- `net/http` + `golang.org/x/net/html` for scraping (no official API, §5) — or a minimal regex-based parser to avoid a new dependency
- **Masking is why this is verify-only.** Public results come back as `a*****gacor.biz` — unusable for WHOIS, DNS, or fingerprinting. Masking is irrelevant when _we_ supply the full string and only need yes/no back.
- IP-restricted to Indonesia (§5) — team is in-country; don't develop through a VPN.
- **Not on the critical path anymore.** Under the old plan this was the #1 risk (bulk-seeding L1). Now Layer 2 bootstraps L1 and TrustPositif is corroboration. Do not let it consume T+2–T+5 like the original PJ-202 did.
- Do not parallelize. Getting IP-banned still helps nobody.

---

## PJ-702 · Masked-pattern parser

**Owner:** A · **Size:** S · **Depends:** PJ-701 · **NICE-TO-HAVE**

**Description**
Extract structure from `a*****gacor.biz`: first char, masked length, suffix. Stores as audit trail; optionally narrows candidate guessing.

**Acceptance Criteria**

- [ ] Parses masked string → `{first_char, masked_len, suffix, tld}`
- [ ] Stored to `domains.source_masked_pattern` when a candidate derives from one
- [ ] **Does not block anything** — if cut, nothing downstream breaks

**Technical Notes**

- File: `core/trustpositif.go`
- The mask leaks real constraints: exact prefix char, exact segment length, exact suffix. Enough to narrow blind combinatorics to constrained guessing.
- **Nice-to-have (§6).** Build only if green at T+16. The system bootstraps from Layer 2 regardless.

---

## PJ-703 · Cold-start proof script

**Owner:** A/B · **Size:** M · **Depends:** PJ-301, PJ-404

**Description**
Empty DB → N Layer 2 confirmations → M Layer 1 preemptive catches. Replaces the dead held-out validation.

**Acceptance Criteria**

- [ ] Starts from a verified-empty DB
- [ ] Records N (L2 confirmations), M (L1 catches on never-visited domains), misses
- [ ] **Leakage assertion: an L2-confirmed domain can NEVER be counted as an L1 catch.** Asserted in code, not assumed.
- [ ] Writes to `bootstrap_runs`
- [ ] Misses logged individually
- [ ] Re-runnable from clean state

**Technical Notes**

- File: `scripts/bootstrap_run.go`
- **The leakage assertion is the whole ticket.** If a domain Layer 2 confirmed gets counted as a Layer 1 preemptive catch, the ratio is fake and §15's honesty beat becomes a confident lie told to judges. Worse than a bad number.
- **PRD §14 risk #9: do not tune the ratio.** If 5 confirmations bought 2 catches, ship that. "PREDATOR's 70% came from a full dataset over months; we started from an empty database 20 hours ago" is a strong answer. A suspiciously perfect number invites the question you don't want.
- Misses give D a real answer to "where does this fail?"

---

## PJ-704 · Endpoint: GET /bootstrap/latest

**Owner:** A · **Size:** S · **Depends:** PJ-703, PJ-106

**Description**
Serve the latest cold-start run.

**Acceptance Criteria**

- [ ] `GET /api/v1/bootstrap/latest` → `{l2_confirmations, l1_preemptive_catches, l1_misses, ratio}`
- [ ] Most recent `bootstrap_runs` row
- [ ] **No runs yet → all-zeros, not a 500** — the demo opens on this state

**Technical Notes**

- File: `api/main.go`
- The zero state is the demo's first frame. It must be a valid response, not an empty-state error.

---

# EPIC 8 — Demo Reliability & Fallbacks

> Owner D throughout. **Surge clause (§12):** at T+9.5, if Epic 5 isn't blocking end-to-end, D drops these and pairs on Epic 6. Epic 8 compresses into T+17–21.

---

## PJ-801 · Bootstrap the demo clusters

**Owner:** D + B · **Size:** M · **Depends:** PJ-301, PJ-402 · **T+12–14** · **🚩 TOP RISK**

**Description**
Deliberately run Layer 2 over a curated set of same-network judol domains until at least one cluster has enough siblings to demo Layer 1.

**Acceptance Criteria**

- [ ] Curated set of judol domains likely sharing infrastructure
- [ ] Layer 2 run over them → clusters form via the feedback loop
- [ ] **≥1 cluster reaches ≥5 sibling domains**
- [ ] ≥1 sibling is blockable by Layer 1 alone, never visited
- [ ] Verified in the dashboard detail view (PJ-603)
- [ ] **GATE: if no cluster forms by T+14, the Layer 1 beat is CUT from the demo**

**Technical Notes**

- **PRD §14 risk #1 — the top risk under cold start.** Bulk seeding is gone. If Layer 2 confirmations don't yield a cluster with siblings, there is no preemptive beat and the pitch loses its core claim.
- Curation matters: pick domains plausibly on shared hosting (similar naming, similar TLD). Random judol domains may not cluster at all.
- **This shares runs with PJ-205 (verdict pre-caching).** Same executions, two purposes — do them together.
- The T+14 decision is binary and must be made off-stage. Demo becomes Layer 2 + propagation only; deck updated to match (§16 checklist).

---

## PJ-802 · Fix and pre-test demo domains

**Owner:** D · **Size:** S · **Depends:** PJ-503, PJ-801

**Description**
Choose the exact domains for each beat and test them repeatedly.

**Acceptance Criteria**

- [ ] Domains fixed per beat: 1 unknown for L2 bootstrap, ≥1 sibling for the L1 beat, ≥1 for propagation
- [ ] Each run ≥5 times without failure
- [ ] Recorded in `pitch/demo_script.md`
- [ ] **No live URL typing during the demo, ever**

**Technical Notes**

- File: `pitch/demo_script.md`
- §15: "Never type a URL you haven't run 5 times."
- **PRD §14 risk #13:** the L2 demo domain must be genuinely absent from the blocklist at demo time. A rehearsal-cached verdict silently converts the bootstrap beat into a Layer 1 beat — the demo appears to work while its entire point evaporates. This is the subtlest failure in the whole plan.

---

## PJ-803 · Fixture purge + empty-state verification

**Owner:** D · **Size:** S · **Depends:** PJ-104 · **T+17–21**

**Description**
The demo opens on an empty database. Make sure it actually is one.

**Acceptance Criteria**

- [ ] Purge script run against the demo DB
- [ ] Dashboard renders all-zeros cleanly
- [ ] Cold Start tab shows 0/0/0 without error
- [ ] Blocklist is empty; extension does not error
- [ ] Verified as part of the `demo-ready` tag (PJ-808)

**Technical Notes**

- **PRD §14 risk #12.** §15's opening beat _is_ the empty database. Leftover fixtures destroy it while looking like success — the worst kind of failure, because nothing appears wrong.
- Test the empty state deliberately. "It works with data" does not imply "it works without."

---

## PJ-804 · Record fallback video

**Owner:** D · **Size:** M · **Depends:** PJ-504, PJ-801, PJ-802 · **DO AT T+12–14**

**Description**
Record the full run while it works. Not at T+23 when it doesn't.

**Acceptance Criteria**

- [ ] Shows the complete §15 flow: empty DB → L2 bootstrap → L1 emerges → Device 2 propagation
- [ ] Both devices visible
- [ ] Under 2 minutes
- [ ] **Stored locally on the demo laptop desktop** — not cloud, not a link
- [ ] Plays with no network

**Technical Notes**

- File: `pitch/fallback_demo.mp4`
- **PRD §14: "Recorded early, while it works."** The date is the ticket.
- Record _after_ PJ-801 confirms clusters exist — a fallback video missing the L1 beat is only half a fallback.

---

## PJ-805 · Verify cached-data demo path

**Owner:** D · **Size:** M · **Depends:** PJ-205, PJ-701

**Description**
Prove the demo runs with zero live external API calls.

**Acceptance Criteria**

- [ ] Gemini key deliberately invalidated → cached verdicts serve → demo works
- [ ] TrustPositif unreachable → demo unaffected
- [ ] Neither failure visible to a viewer
- [ ] Documented in `demo_script.md`

**Technical Notes**

- PRD §14 risks #3, #4.
- **Gemini matters more than TrustPositif now.** Under cold start, Layer 2 _is_ the bootstrap — Gemini down means no system, where TrustPositif down means only "no corroboration." Test the Gemini path harder.
- Break things deliberately. A path assumed to work is a path that doesn't.

---

## PJ-806 · Verify polling adapter indistinguishable

**Owner:** D · **Size:** S · **Depends:** PJ-505

**Description**
With realtime disabled, the two-device demo must still look instant.

**Acceptance Criteria**

- [ ] Flag flipped → demo works
- [ ] Device 2 blocks within ~3s — a judge cannot tell
- [ ] Flag-flip procedure in `demo_script.md`

**Technical Notes**

- PRD §14 risk #5.
- **D flips the flag, not C** — C may be asleep or heads-down when it matters.

---

## PJ-807 · Demo hardware lock + venue network test

**Owner:** D · **Size:** M · **Depends:** PJ-504 · **T+21**

**Description**
Designate exact devices, preload the extension, run on venue wifi.

**Acceptance Criteria**

- [ ] Demo laptop + second device designated by name
- [ ] Extension loaded unpacked and pinned on **both**
- [ ] Full demo on this exact hardware ≥1 time
- [ ] **Full demo on actual venue wifi ≥1 time**
- [ ] Realtime confirmed working through venue network — or polling confirmed as the path
- [ ] Phone hotspot tested as fallback
- [ ] Both devices charged, chargers packed, notifications disabled

**Technical Notes**

- PRD §14 risks #6, #7.
- "Load unpacked" is dev-mode. Chrome may prompt or disable on restart — **verify after a full reboot**, not just a sleep.
- **Websockets through a captive portal is the classic silent killer.** If realtime dies on venue wifi, PJ-505's poller is the answer — which is exactly why it exists at T+8.

---

## PJ-808 · Cut demo-ready tag

**Owner:** D · **Size:** S · **Depends:** all · **T+21**

**Description**
Tag the known-good commit. Demo runs from the tag, not `main`.

**Acceptance Criteria**

- [ ] `git tag demo-ready` on a verified-working commit
- [ ] **Fixtures purged (PJ-803) before tagging**
- [ ] Demo laptop checked out at the tag
- [ ] Announced in WA: `main` is now irrelevant to the demo

**Technical Notes**

- PRD §14 risk #15.
- Feature freeze is T+14. This tag makes it enforceable rather than aspirational.

---

## PJ-809 · Timed rehearsals

**Owner:** D · **Size:** M · **Depends:** PJ-802 · **T+12, T+15, T+17, T+19, T+21**

**Description**
Five full timed runs against the §15 script. D is the only person testing like a judge.

**Acceptance Criteria**

- [ ] ≥5 complete runs, each ≤2:00
- [ ] Breakage found is filed and fixed, not noted and forgotten
- [ ] **Both Q&A killers word-perfect:** §14 risk #10 (BlockSite → "the empty database is the answer") and **risk #11 ("where's the AI?" → deliberately absent, deterministic is why the block page can explain itself)**
- [ ] Final run at T+21 on demo hardware, venue wifi, from the `demo-ready` tag

**Technical Notes**

- File: `pitch/demo_script.md`
- §12: D pitches — D has heard the least code and will explain it with the least jargon, which is scored criterion #1.
- **Risk #11 is new and currently unrehearsed anywhere else.** Layer 1 has no ML by design; if a judge catches that as an absence rather than hearing it as a choice, Innovation & Novelty takes the hit. It must be delivered proactively, not defensively.
- Rehearsals are a QA pass, not a warm-up. Their output is bugs.

---

## Scope Guard

Not tickets. Do not create them. (§6)

- **Bulk import of TrustPositif** — no bulk export, results masked, cannot be WHOIS'd or fingerprinted
- WA Chatbot report packaging — nice-to-have
- Affiliate network graph — nice-to-have
- Android VpnService app — roadmap (Core seam is built for it now)
- Live monitoring of all new registrations — needs enterprise API
- aduankonten.id auto-submit — CAPTCHA, do not bypass
- Bank account pre-blocking — not how PPATK works, legal risk
- Any self-trained vision model — explicitly not the innovation
- **ML in Layer 1** — deliberate; deterministic scoring is what produces explainable `matched_fields`
