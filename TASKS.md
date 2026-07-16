# PREJUDGE — Task Backlog

Granular tickets derived from EPICS.md. Scope is strictly MVP must-haves + demo reliability. Nothing here is a nice-to-have.

**ID scheme:** `PJ-{epic}{seq}` — e.g. `PJ-101` = Epic 1, task 1.
**Sizing:** S ≈ <45min · M ≈ 45–90min · L ≈ 90min–2h. Anything L is a candidate for splitting.

---

# EPIC 1 — Data Foundation

---

## PJ-101 · Stand up Supabase project + distribute credentials
**Owner:** A · **Size:** S · **Blocks:** everything

**Description**
Create the Supabase project, enable Realtime on the public schema, and get keys into everyone's hands before anyone writes a line of code that needs them.

**Acceptance Criteria**
- [ ] Supabase project exists, region closest to venue
- [ ] Realtime enabled for `public` schema
- [ ] `.env.example` committed to repo root with every key name (no values)
- [ ] Real `.env` values pinned in the team WA group
- [ ] All 4 members confirm they can connect (one-line script or dashboard login)

**Technical Notes**
- Files: `.env.example`, `README.md`
- Keys needed: `SUPABASE_URL`, `SUPABASE_ANON_KEY`, `SUPABASE_SERVICE_ROLE_KEY`, `GEMINI_API_KEY`
- Anon key goes in the extension (public by design). Service role key **never** ships in `extension/` — backend only.
- Per PRD §8: keys distributed at T+0. Nobody hunts for keys at 3am.

---

## PJ-102 · Implement schema.sql
**Owner:** A · **Size:** M · **Depends:** PJ-101

**Description**
Write and apply the full DDL for all five tables per PRD §9 ERD.

**Acceptance Criteria**
- [ ] All 5 tables created: `domains`, `fingerprint_clusters`, `detections`, `whois_records`, `validation_runs`
- [ ] Enums defined: `domain_status` (candidate|confirmed|blocked|false_pos), `detection_source` (L1|L2|trustpositif)
- [ ] FKs wired: `domains.cluster_id → fingerprint_clusters.id`, `detections.domain_id → domains.id`, `whois_records.domain_id → domains.id`
- [ ] `domains.domain` has UNIQUE constraint
- [ ] Index on `domains(status)` — the extension filters on this every subscribe
- [ ] Script is idempotent (re-runnable without error)

**Technical Notes**
- File: `backend/db/schema.sql`
- Types per ERD: `hosting_ip inet`, `raw_response jsonb`, `nameservers text[]`, timestamps `timestamptz`
- **Denormalize `reason` onto `domains`** — PRD §9 explicitly permits this and the extension needs `domain`/`confidence`/`reason` without a join. Do it now, not at T+20.
- RLS: disable for MVP or add a permissive read policy on `domains`. A locked-down RLS policy silently breaking realtime at T+22 is a real failure mode.

---

## PJ-103 · Seed database with fake rows
**Owner:** A · **Size:** S · **Depends:** PJ-102 · **Unblocks:** C

**Description**
20 fake domain rows with realistic shapes so Frontend builds against real data from T+1 instead of waiting on Layer 1.

**Acceptance Criteria**
- [ ] 20 rows in `domains` — mixed `status`, mixed `source` (L1/L2), varied `confidence` (0.6–0.98)
- [ ] At least 3 `fingerprint_clusters` with domains attached, one cluster having ≥5 domains (for the "47 siblings" demo view)
- [ ] Matching `detections` rows with populated `reason` text
- [ ] Seed script is re-runnable (truncate + insert)
- [ ] C confirms dashboard can read it

**Technical Notes**
- File: `backend/db/seed.sql`
- Use plausible judol-style names — the dashboard screenshots end up in the deck.
- Fake rows must be gone or clearly distinguishable before demo. Add a `-- SEED` comment marker.

---

## PJ-104 · Publish API contract in README
**Owner:** A · **Size:** S · **Depends:** none · **Unblocks:** B, C

**Description**
Paste the PRD §10 endpoint table into the repo README as the binding contract, before any implementation.

**Acceptance Criteria**
- [ ] All 9 endpoints listed with method, path, request shape, response shape
- [ ] Realtime channel spec documented: `postgres_changes` on `domains`, filter `status=eq.blocked`
- [ ] `.env` variable names documented
- [ ] Committed to `main` at T+0

**Technical Notes**
- File: `README.md`
- Base path `/api/v1`
- This is the contract. Changes after T+4 require all four to agree (PRD §11).

---

## PJ-105 · Stub all API endpoints
**Owner:** A · **Size:** M · **Depends:** PJ-104 · **Unblocks:** B, C

**Description**
FastAPI app returning hardcoded responses matching the contract exactly. Frontend is never blocked on backend logic.

**Acceptance Criteria**
- [ ] FastAPI runs locally, all 9 routes respond 200
- [ ] Every response matches the README shape exactly (keys and types, values fake)
- [ ] CORS enabled for extension origin + `localhost:3000`
- [ ] `/docs` (Swagger) reachable
- [ ] Done by T+2

**Technical Notes**
- Files: `backend/main.py`, `backend/requirements.txt`
- `fastapi`, `uvicorn`, `pydantic`
- Define Pydantic response models now — they become the real return types later, no rewrite.
- CORS: `chrome-extension://*` is not a valid origin match. Use `allow_origins=["*"]` for the hackathon and move on.

---

## PJ-106 · Freeze schema
**Owner:** A · **Size:** S · **Depends:** PJ-102, PJ-103

**Description**
Declare the schema frozen at T+4 per PRD §11. Ceremony, but the kind that prevents 3am migration chaos.

**Acceptance Criteria**
- [ ] Announced in WA group at T+4
- [ ] `schema.sql` header comment: `-- FROZEN T+4. Changes require all 4 to agree.`
- [ ] Any later change is a WA thread, not a silent push

**Technical Notes**
- File: `backend/db/schema.sql`

---

# EPIC 2 — Layer 1: Preemptive Detection

---

## PJ-201 · Candidate domain name generator
**Owner:** A · **Size:** M · **Depends:** none

**Description**
Generate plausible judol domain candidates from keyword+TLD combinations and pattern variation off known bandar naming (per PRD §4: "gacor88.xyz" → "gacor89.xyz").

**Acceptance Criteria**
- [ ] Produces ≥1000 unique candidates from a seed keyword list
- [ ] Keyword list externalized (not hardcoded in the function)
- [ ] Numeric-suffix variation: given a known domain, emits neighbors (±1..N on trailing digits)
- [ ] TLD permutation across cheap-TLD list (.xyz, .top, .cc, .site, etc.)
- [ ] Output: plain list of strings, deduped
- [ ] Callable as CLI: `python -m layer1.candidates --out candidates.json`

**Technical Notes**
- File: `backend/layer1/candidates.py`
- Pure stdlib — `itertools.product` is the whole job. No library needed.
- Keyword seed file: `backend/layer1/keywords.txt`
- **Scope guard:** no ML, no DGA modeling. Combinatorics only. PRD §3 explicitly parks DGA claims as unverified.

---

## PJ-202 · TrustPositif bulk-checker with cache
**Owner:** A · **Size:** L · **Depends:** PJ-201 · **⚠️ HIGHEST RISK TASK**

**Description**
Submit candidate domains to TrustPositif in batches of ≤100 and record which are confirmed blocked. Cache every single result to disk on receipt.

**Acceptance Criteria**
- [ ] Batches at ≤100 domains/request per PRD §5
- [ ] **Every response cached to JSON immediately on receipt** — cache write happens before any parsing
- [ ] Re-run reads cache first, only queries uncached domains
- [ ] Rate limiting: delay between requests, configurable
- [ ] Graceful failure: a blocked/timed-out request does not lose already-cached results
- [ ] **≥500 confirmed judol domains on disk by T+5** ← this is the gate
- [ ] Confirmed domains written to `domains` table with `source='trustpositif'`, `status='confirmed'`

**Technical Notes**
- Files: `backend/layer1/trustpositif.py`, cache at `backend/db/cache/trustpositif_*.json`
- `requests` + `beautifulsoup4` (form scrape — no official API per PRD §5)
- Site is IP-restricted to Indonesia (PRD §5) — team is in-country, but do not route through a VPN while developing.
- **PRD §14 risk #1: this is the single biggest technical risk in the project.** Cache-first is not optimization, it is the fallback plan. If TrustPositif rate-limits at T+15, the cache is the demo.
- Do not parallelize. Getting IP-banned at T+3 ends the project.

---

## PJ-203 · Infrastructure fingerprint extractor
**Owner:** A · **Size:** L · **Depends:** PJ-202

**Description**
For a confirmed domain, pull registrar, hosting IP, ASN, nameservers, TLD, and registration date via WHOIS/RDAP/DNS.

**Acceptance Criteria**
- [ ] Given a domain, returns: `registrar`, `hosting_ip`, `asn`, `nameserver`, `tld`, `registered_at`
- [ ] Handles redacted WHOIS without crashing — missing fields return `None`, not exceptions
- [ ] Raw response stored to `whois_records.raw` (jsonb) for later debugging
- [ ] Runs on ≥100 confirmed domains, reports field-level fill rate
- [ ] Results cached — never re-query a domain already fetched

**Technical Notes**
- Files: `backend/layer1/fingerprint.py`
- `python-whois`, `dnspython`, RDAP via `requests`
- **PRD §14 risk #7 (High):** WHOIS registrant fields are widely redacted. **Do not build the matcher on registrant data.** Fingerprint on hosting IP + ASN + nameserver + TLD + registration date — these survive redaction.
- **Decide this at T+5, not T+14.** Run the fill-rate report early; if `registrar` fill rate is <50%, drop it from the match weights immediately.
- ASN lookup: IP → ASN via RDAP or a free lookup. If it costs more than 30min, drop ASN and use IP /24 prefix instead.

---

## PJ-204 · Fingerprint cluster builder
**Owner:** A · **Size:** M · **Depends:** PJ-203

**Description**
Group confirmed domains sharing infrastructure into clusters. This is what "same landlord, same street" means concretely.

**Acceptance Criteria**
- [ ] Confirmed domains grouped by shared hosting IP / nameserver / registrar
- [ ] Each cluster written to `fingerprint_clusters` with accurate `domain_count`
- [ ] `domains.cluster_id` populated
- [ ] **At least one cluster with ≥5 member domains exists** — the demo needs a visible neighborhood
- [ ] Re-runnable without duplicating clusters

**Technical Notes**
- File: `backend/layer1/fingerprint.py` or new `backend/layer1/cluster.py`
- Grouping is a `GROUP BY`, not clustering ML. Do not import sklearn.
- If no cluster reaches 5 members, loosen the grouping key (IP → /24 prefix) before assuming the data is bad.

---

## PJ-205 · Cluster similarity matcher
**Owner:** A · **Size:** L · **Depends:** PJ-204

**Description**
Given an unseen domain, score how strongly its fingerprint matches known bandar clusters. This is the Layer 1 detection claim.

**Acceptance Criteria**
- [ ] Input: domain string. Output: `{cluster_id, match_score (0–1), matched_fields[]}`
- [ ] Weighted scoring across available fields — weights defined in one named constant, not scattered magic numbers
- [ ] Returns no-match cleanly when nothing scores above threshold
- [ ] `matched_fields` is human-readable — it feeds the block page "Why?" text directly
- [ ] Threshold defined as a named constant, tunable in one place

**Technical Notes**
- File: `backend/layer1/matcher.py`
- Suggested starting weights (tune with real data, do not treat as gospel):
  `hosting_ip 0.4 · nameserver 0.3 · registrar 0.15 · tld 0.1 · reg_date_proximity 0.05`
- Define `MATCH_THRESHOLD = 0.6` as a module constant.
- **`matched_fields` is a UX deliverable, not a debug field.** PRD §5 scores "transparent, not black box" — the block page needs "same hosting IP as 47 confirmed sites", which comes from here.

---

## PJ-206 · Endpoint: POST /fingerprint
**Owner:** A · **Size:** S · **Depends:** PJ-205, PJ-105

**Description**
Replace the stub with real fingerprint extraction + cluster matching.

**Acceptance Criteria**
- [ ] `POST /api/v1/fingerprint` with `{domain}` returns `{cluster_id, registrar, ip, ns, tld, match_score}`
- [ ] Response shape unchanged from the stub
- [ ] Unknown domain returns a valid response with null cluster, not a 500
- [ ] Responds <5s or returns a timeout error (WHOIS is slow)

**Technical Notes**
- File: `backend/main.py`
- Reuse the Pydantic model from PJ-105.

---

## PJ-207 · Endpoint: POST /trustpositif/bulk-check
**Owner:** A · **Size:** S · **Depends:** PJ-202, PJ-105

**Description**
Internal endpoint wrapping the bulk-checker.

**Acceptance Criteria**
- [ ] `POST /api/v1/trustpositif/bulk-check` with `{domains:[...]}` returns `{results:[{domain, is_blocked}]}`
- [ ] Rejects >100 domains with a 400
- [ ] Reads cache first

**Technical Notes**
- File: `backend/main.py`
- Internal only — not called by the extension. No auth needed for MVP.

---

# EPIC 3 — Layer 2: Reactive Content Detection

---

## PJ-301 · Gemini vision client
**Owner:** B · **Size:** M · **Depends:** PJ-101

**Description**
Wrap the Gemini vision API: take a screenshot, return a judol verdict with reason and confidence.

**Acceptance Criteria**
- [ ] Function: `analyze(image_b64) → {is_judol: bool, confidence: float, reason: str}`
- [ ] Prompt returns structured output, parsed safely (malformed response → `is_judol=False`, logged, no crash)
- [ ] Full raw response retained for `detections.raw_response`
- [ ] Verified by hand on one real judol screenshot **by T+2**
- [ ] `reason` is one plain-language sentence — it renders on the block page verbatim

**Technical Notes**
- File: `backend/layer2/vision.py`
- Gemini 2.x Flash vision per PRD §8. Model name in a constant, not inline.
- Prompt per PRD §4, roughly: *"Is this an online gambling site? Answer yes/no with a brief reason."* Request JSON-only output, strip markdown fences before parsing.
- **Not our innovation** (PRD §4). No fine-tuning, no custom model, no prompt-engineering rabbit hole. Get a verdict and move on.
- `reason` is user-facing Indonesian-context text. Prompt for plain language, not model jargon.

---

## PJ-302 · Screenshot capture path
**Owner:** B + C · **Size:** M · **Depends:** PJ-401

**Description**
Extension captures the visible tab on an unknown-domain visit and POSTs it to `/analyze`.

**Acceptance Criteria**
- [ ] Fires only when domain is **not** already in the local blocklist
- [ ] Uses `chrome.tabs.captureVisibleTab`, base64 encoded
- [ ] POSTs `{domain, screenshot_b64}` to `/api/v1/analyze`
- [ ] Payload size handled — downscale/compress if over limit
- [ ] Failure is silent to the user (no error toast on a page that isn't judol)

**Technical Notes**
- Files: `extension/content.js`, `extension/background.js`, `backend/layer2/vision.py`
- `captureVisibleTab` needs `activeTab` or `<all_urls>` permission + it only works from the background service worker, not a content script. Message-pass from content → background.
- **PRD §8: extension-side capture, zero server infra.** Playwright is fallback only — do not reach for it first.
- JPEG quality ~50 is plenty. A slot-machine UI is not subtle.
- Cross-owner task: agree the message shape between B and C before either starts.

---

## PJ-303 · Verdict decision logic + DB write
**Owner:** B · **Size:** M · **Depends:** PJ-301, PJ-102

**Description**
Turn a vision verdict into a database state change: confirmed judol → `status='blocked'`, which is what triggers realtime propagation.

**Acceptance Criteria**
- [ ] Confidence threshold as a named constant
- [ ] Above threshold → upsert `domains` with `status='blocked'`, `source='L2'`, populated `reason`
- [ ] `detections` row written with `layer=2`, confidence, reason, `raw_response`
- [ ] Below threshold → `detections` row logged, `domains.status` unchanged
- [ ] Upsert on `domain` unique constraint — no duplicate rows on repeat visits

**Technical Notes**
- File: `backend/layer2/decide.py`
- `L2_CONFIDENCE_THRESHOLD = 0.8` as a module constant.
- **The `status='blocked'` write is the realtime trigger.** Epic 4's subscription fires off this exact update. Coordinate with C before changing anything about how this write happens.
- Use the service role key (bypasses RLS). Never the anon key here.

---

## PJ-304 · Layer 2 → Layer 1 feedback loop
**Owner:** B · **Size:** M · **Depends:** PJ-303, PJ-203

**Description**
When Layer 2 confirms a domain, extract its infrastructure fingerprint and feed it into Layer 1's cluster data. **This is the novelty claim — the thing that answers "isn't this just BlockSite?"**

**Acceptance Criteria**
- [ ] L2-confirmed domain triggers `fingerprint.extract()`
- [ ] Fingerprint written; domain joins an existing cluster or creates a new one
- [ ] Sibling domains sharing that fingerprint become Layer 1 candidates
- [ ] **Demonstrable end-to-end:** L2 confirms domain X → X's cluster gains a member → a sibling Y is now flagged without ever being visited
- [ ] Feedback runs async — must not block the block-page response

**Technical Notes**
- Files: `backend/layer2/decide.py` → calls `backend/layer1/fingerprint.py`, `backend/layer1/cluster.py`
- FastAPI `BackgroundTasks` is sufficient. No Celery, no queue.
- **PRD §5 + §14 risk #9:** the pitch's entire Innovation & Novelty defense rests on this loop existing. If Epic 3 gets cut for time, cut PJ-302 polish — not this.
- Must be visible in the demo (§15, 1:00–1:40 beat: *"its siblings are already flagged"*).

---

## PJ-305 · Endpoint: POST /analyze
**Owner:** B · **Size:** S · **Depends:** PJ-301, PJ-303, PJ-105

**Description**
Wire the real Layer 2 pipeline behind the stubbed endpoint.

**Acceptance Criteria**
- [ ] `POST /api/v1/analyze` with `{domain, screenshot_b64}` returns `{is_judol, confidence, reason, domain_id}`
- [ ] Response shape unchanged from stub
- [ ] Returns <8s or times out cleanly
- [ ] Feedback loop (PJ-304) fires in background, does not delay response

**Technical Notes**
- File: `backend/main.py`
- Accept base64 in JSON body, not multipart. Simpler from the extension.

---

## PJ-306 · Pre-cache verdicts for demo domains
**Owner:** B · **Size:** S · **Depends:** PJ-301, PJ-303 · **Do at T+14**

**Description**
Run Layer 2 on the 5 fixed demo domains ahead of time and cache the verdicts, so a Gemini outage during judging is invisible.

**Acceptance Criteria**
- [ ] 5 demo domains have cached verdicts on disk
- [ ] `vision.analyze()` checks cache first, returns cached result on API failure
- [ ] Verified by simulating an API failure (bad key) — demo still works

**Technical Notes**
- Files: `backend/layer2/vision.py`, cache at `backend/db/cache/vision_*.json`
- PRD §14 risk #3.
- Cache-on-failure, not cache-always — the live call should still happen when the API is up. Judges may ask if it's live.

---

# EPIC 4 — Blocking Extension

> **This epic is the demo.** PRD §15's money shot is here. Everything else yields to it.

---

## PJ-401 · Manifest V3 scaffold
**Owner:** C · **Size:** S · **Depends:** none

**Description**
Minimal MV3 extension that loads unpacked without errors.

**Acceptance Criteria**
- [ ] Loads unpacked in Chrome, zero console errors
- [ ] Background service worker registered and running
- [ ] Permissions declared: `declarativeNetRequest`, `declarativeNetRequestWithHostAccess`, `tabs`, `storage`, `host_permissions: <all_urls>`
- [ ] Popup opens

**Technical Notes**
- Files: `extension/manifest.json`, `extension/background.js`, `extension/popup/popup.html`, `extension/popup/popup.js`
- `"manifest_version": 3`. Vanilla JS, no build step (PRD §8 — React in an extension is a time sink).
- MV3 service workers terminate when idle. Do not hold state in module-level variables; use `chrome.storage.local`.

---

## PJ-402 · Realtime subscription proof-of-life
**Owner:** C · **Size:** M · **Depends:** PJ-401, PJ-102 · **⚠️ DO THIS FIRST**

**Description**
Extension subscribes to Supabase realtime and logs any `domains` change to console. No blocking logic yet — just prove the pipe exists.

**Acceptance Criteria**
- [ ] Extension subscribes to `postgres_changes` on `domains`, filter `status=eq.blocked`
- [ ] Manual row update in Supabase dashboard → message appears in extension console
- [ ] Reconnects after network drop
- [ ] **Working by T+2**

**Technical Notes**
- File: `extension/background.js`
- Supabase JS client via CDN bundle or vendored file — **no npm build step in the extension.**
- Anon key only. Service role key never ships client-side.
- **PRD §10 explicitly: "build and test this first, before anything else."** If realtime doesn't work, the demo doesn't exist and you need to know at T+2, not T+12.
- MV3 service worker termination will kill a websocket. Test that the subscription survives idle — this is the most likely silent failure.

---

## PJ-403 · declarativeNetRequest blocking engine
**Owner:** C · **Size:** L · **Depends:** PJ-401

**Description**
Block domains from a list using DNR dynamic rules, redirecting to the block page.

**Acceptance Criteria**
- [ ] Hardcoded list of 3 domains → all blocked, redirect to `blocked.html`
- [ ] Rules added/removed at runtime without reload
- [ ] Block passes domain + confidence + reason to the block page
- [ ] Non-listed domains completely unaffected
- [ ] **Blocking a real domain by T+5** ← gate

**Technical Notes**
- File: `extension/background.js`
- `chrome.declarativeNetRequest.updateDynamicRules()` — dynamic, not static rulesets.
- Redirect: `{type: "redirect", redirect: {extensionPath: "/blocked.html?d=..."}}`
- Rule IDs must be unique integers. Keep a counter in `chrome.storage.local`, not memory — the service worker will die and reset it.
- Dynamic rule cap is finite (~5k). Not an MVP concern, but do not attempt to load 500k domains.

---

## PJ-404 · Blocklist sync from Supabase
**Owner:** C · **Size:** M · **Depends:** PJ-402, PJ-403

**Description**
Populate DNR rules from the database — full fetch on startup, incremental updates via realtime.

**Acceptance Criteria**
- [ ] On startup: fetch all `domains` where `status='blocked'` → DNR rules
- [ ] Realtime event → new rule added within 5s, no reload
- [ ] Blocklist cached in `chrome.storage.local`, survives service worker restart
- [ ] `domain`, `confidence`, `reason` all available to the block page

**Technical Notes**
- Files: `extension/background.js`
- Fetch via `GET /api/v1/blocklist?since=` (PJ-406) or direct Supabase client read. Direct read is fewer moving parts.
- This is where the denormalized `reason` (PJ-102) pays off — no join client-side.

---

## PJ-405 · Block page UI
**Owner:** C · **Size:** M · **Depends:** PJ-403

**Description**
The page a user actually sees. Per PRD §5, transparency here is a scored criterion — "user knows why a site was blocked, not a black box."

**Acceptance Criteria**
- [ ] Renders per PRD §7 wireframe: shield, domain, confidence bar, "Why?" bullets, two buttons
- [ ] Confidence rendered as a visual bar, not a bare number
- [ ] Reason bullets populated from `matched_fields` / L2 `reason` — real data, not placeholder
- [ ] "Report as mistake" button present and wired (PJ-407)
- [ ] Readable, not styled like a browser error page — this is a product surface

**Technical Notes**
- Files: `extension/blocked.html`, `extension/blocked.css`
- Params from query string or `chrome.storage.local` lookup by domain.
- Copy in Indonesian — the user is Rina from Bekasi (PRD §3), not a developer.
- **PRD §14 risk #10:** the "Report as mistake" button *is* the answer to a false positive in front of judges. It must be visible and clickable.

---

## PJ-406 · Endpoint: GET /blocklist + POST /check
**Owner:** A · **Size:** S · **Depends:** PJ-105, PJ-102

**Description**
Real implementations of the two endpoints the extension consumes.

**Acceptance Criteria**
- [ ] `GET /api/v1/blocklist?since=<ISO>` → `{domains:[{domain,confidence,reason}], updated_at}`
- [ ] `since` param actually filters (incremental sync)
- [ ] `POST /api/v1/check` with `{domain}` → `{status, confidence, source, reason}`
- [ ] `/check` on an unknown domain returns a clean "not found" state, not a 404 the extension has to catch

**Technical Notes**
- File: `backend/main.py`
- Reads denormalized `domains` — no joins on this path. It is called constantly.

---

## PJ-407 · Endpoint: POST /report-false-positive
**Owner:** A · **Size:** S · **Depends:** PJ-105, PJ-405

**Description**
Back the block page's "Report as mistake" button.

**Acceptance Criteria**
- [ ] `POST /api/v1/report-false-positive` with `{domain_id, note}` → `{ok:true}`
- [ ] Sets `domains.status='false_pos'`
- [ ] Domain disappears from blocklist on next sync
- [ ] No auth (MVP)

**Technical Notes**
- File: `backend/main.py`
- Status change → realtime event → the extension should *unblock*. Worth verifying once; a stuck block after clicking "mistake" is an ugly demo moment.

---

## PJ-408 · Polling fallback
**Owner:** C · **Size:** M · **Depends:** PJ-404 · **Build at T+8, not T+22**

**Description**
If realtime fails, poll `/blocklist` every 3s instead. Must be visually indistinguishable from realtime to a judge.

**Acceptance Criteria**
- [ ] Poller pulls `/blocklist?since=` on a 3s interval
- [ ] Single feature flag switches realtime ↔ polling
- [ ] With realtime disabled, the two-device demo still works and still looks instant
- [ ] Tested by deliberately breaking the realtime subscription

**Technical Notes**
- File: `extension/background.js`
- MV3: `setInterval` dies with the service worker. Use `chrome.alarms` — minimum period is 1 minute, which is too slow, so for the demo window keep the worker alive via an active subscription or accept `setInterval` and confirm it survives the 2-minute demo.
- **PRD §14 risk #4 — "Build the poller at T+8 as insurance, not at T+22 in a panic."** This ticket is dated on purpose.

---

# EPIC 5 — Dashboard

---

## PJ-501 · Next.js scaffold + Supabase client
**Owner:** C · **Size:** S · **Depends:** PJ-101, PJ-103

**Description**
Next.js 14 App Router project, Tailwind, shadcn/ui, connected to Supabase, reading seed data.

**Acceptance Criteria**
- [ ] `npm run dev` serves without error
- [ ] Tailwind + shadcn configured
- [ ] Supabase client reads seed rows and renders them raw
- [ ] Deploys to Vercel (do this once, early — not at T+22)

**Technical Notes**
- Files: `dashboard/app/page.tsx`, `dashboard/lib/supabase.ts`
- Anon key in `NEXT_PUBLIC_SUPABASE_ANON_KEY`.
- Deploy early. A first-time Vercel deploy failing at T+22 is a known way to lose.

---

## PJ-502 · Detected domains list view
**Owner:** C · **Size:** M · **Depends:** PJ-501

**Description**
The main dashboard table per PRD §7 wireframe.

**Acceptance Criteria**
- [ ] Table: domain, confidence, source (L1/L2), timestamp
- [ ] Header counters: detected today, L1 count, L2 count, live indicator
- [ ] Sorted newest first
- [ ] Rows clickable → detail view
- [ ] Renders real DB data, not seed, by T+8

**Technical Notes**
- Files: `dashboard/app/page.tsx`, `dashboard/components/`
- `GET /api/v1/domains?limit&offset&source&status` or direct Supabase read.
- The live indicator should reflect actual subscription state. A hardcoded green dot is a lie a judge might catch.

---

## PJ-503 · Domain detail view
**Owner:** C · **Size:** M · **Depends:** PJ-502

**Description**
Per-domain drill-down: fingerprint match detail, screenshot, detection history. This is the "47 siblings" screen from the demo.

**Acceptance Criteria**
- [ ] Route `/domain/[id]` renders
- [ ] Shows cluster info + matched fields + sibling domains in the same cluster
- [ ] Shows Layer 2 screenshot if one exists
- [ ] Shows detection history
- [ ] **Sibling list is the demo beat (§15, 0:35–1:00)** — must render a cluster with ≥5 members legibly

**Technical Notes**
- File: `dashboard/app/domain/[id]/page.tsx`
- `GET /api/v1/domains/{id}` → domain, detections[], whois, cluster, screenshot_url.
- Screenshot storage: Supabase Storage bucket, or skip persistence and show only the reason text. Decide by T+8 — do not discover at T+16 that no bucket exists.

---

## PJ-504 · Validation view
**Owner:** C · **Size:** M · **Depends:** PJ-501, PJ-603

**Description**
Render the held-out test set results. Per PRD §15 this is the honesty beat of the pitch.

**Acceptance Criteria**
- [ ] Route `/validation` renders test set size, matched, missed, rate
- [ ] **Disclaimer rendered on screen: "Historical data. Not live internet monitoring."**
- [ ] Reads real `validation_runs` data
- [ ] Placeholder numbers from PRD §7 (200/141/59) are **not** present in the shipped build

**Technical Notes**
- File: `dashboard/app/validation/page.tsx`
- `GET /api/v1/validation/latest`.
- **PRD §16 checklist:** validation numbers on the dashboard must be real output. The wireframe figures are illustrative and shipping them is a fabrication.
- The disclaimer is not decoration. PRD §6 draws a hard line between "we proved the method on historical data" and "we monitor Indonesian internet in real time."

---

## PJ-505 · Endpoints: GET /domains, GET /domains/{id}
**Owner:** A · **Size:** S · **Depends:** PJ-105, PJ-102

**Description**
Real implementations behind the dashboard's two read endpoints.

**Acceptance Criteria**
- [ ] `GET /api/v1/domains?limit&offset&source&status` → `{items:[...], total}`
- [ ] All filters work
- [ ] `GET /api/v1/domains/{id}` → `{domain, detections[], whois, cluster, screenshot_url}`
- [ ] Unknown id → 404

**Technical Notes**
- File: `backend/main.py`
- Joins `domains` × `detections` × `fingerprint_clusters` per PRD §9.

---

# EPIC 6 — Validation & Ground Truth

---

## PJ-601 · Hold out test set
**Owner:** A/B · **Size:** S · **Depends:** PJ-202

**Description**
Split confirmed TrustPositif domains into train/test. Test domains must be excluded from cluster building or the result is meaningless.

**Acceptance Criteria**
- [ ] ~200 confirmed domains held out
- [ ] **Test domains excluded from `fingerprint_clusters` construction** — verified, not assumed
- [ ] Split is deterministic (fixed seed) and reproducible
- [ ] Split recorded to disk

**Technical Notes**
- File: `scripts/validation_run.py`
- Fixed random seed.
- **Leakage check is the whole ticket.** If a test domain helped build the cluster it's later "matched" to, the number is fake and the honesty beat becomes a lie told confidently to judges. Assert explicitly.

---

## PJ-602 · Validation run script
**Owner:** A/B · **Size:** M · **Depends:** PJ-601, PJ-205

**Description**
Run the Layer 1 matcher over the held-out set and report the honest hit rate.

**Acceptance Criteria**
- [ ] Each test domain scored against clusters built from train-only data
- [ ] Reports: test set size, matched, missed, rate
- [ ] Misses logged individually (they're the interesting part)
- [ ] Result written to `validation_runs`
- [ ] Re-runnable

**Technical Notes**
- File: `scripts/validation_run.py`
- **PRD §14 risk #8: do not tune the number.** If it's 20%, ship 20% and say PREDATOR's 70% came from a full dataset over months. Judges reward honest measurement over suspicious perfection.
- Recording misses gives D a real answer to "where does this fail?"

---

## PJ-603 · Endpoint: GET /validation/latest
**Owner:** A · **Size:** S · **Depends:** PJ-602, PJ-105

**Description**
Serve the latest validation run to the dashboard.

**Acceptance Criteria**
- [ ] `GET /api/v1/validation/latest` → `{test_set_size, matched, missed, rate}`
- [ ] Returns the most recent `validation_runs` row
- [ ] No runs yet → clean empty state, not a 500

**Technical Notes**
- File: `backend/main.py`

---

# EPIC 7 — Demo Reliability & Fallbacks

> Owner D throughout. **Surge clause (PRD §12):** at the T+9.5 GO/NO-GO, if Epic 4 isn't blocking end-to-end, D drops these and pairs on Epic 5. Epic 7 then compresses into T+17–21.

---

## PJ-701 · Fix and pre-test demo domains
**Owner:** D · **Size:** S · **Depends:** PJ-403

**Description**
Choose the exact domains used in the demo and test them repeatedly. Nothing gets typed live that hasn't been run.

**Acceptance Criteria**
- [ ] 5 domains fixed: ≥2 for the Layer 1 beat, ≥1 unknown for the Layer 2 beat, ≥1 sibling for the propagation beat
- [ ] Each run ≥5 times without failure
- [ ] Recorded in `pitch/demo_script.md`
- [ ] No live URL typing during the demo, ever

**Technical Notes**
- File: `pitch/demo_script.md`
- PRD §15: "Never type a URL you haven't run 5 times."
- The Layer 2 demo domain must be genuinely absent from the blocklist at demo time. Verify immediately before — a previously-cached verdict silently converts the Layer 2 beat into a Layer 1 beat and the demo's whole point evaporates.

---

## PJ-702 · Record fallback video
**Owner:** D · **Size:** M · **Depends:** PJ-404, PJ-701 · **DO AT T+12–14**

**Description**
Record the full two-device block while it works. Not at T+23 when it doesn't.

**Acceptance Criteria**
- [ ] Video shows the complete §15 flow: Layer 1 block → Layer 2 detection → Device 2 propagation
- [ ] Both devices visible
- [ ] Under 2 minutes
- [ ] Stored **locally on the demo laptop desktop** — not cloud, not a link
- [ ] Playable with no network

**Technical Notes**
- File: `pitch/fallback_demo.mp4`
- **PRD §14: "Recorded early, while it works — not at T+23 when it doesn't."** The date on this ticket is the ticket.
- If it fails live, D plays this and says so plainly. No apology.

---

## PJ-703 · Verify cached-data demo path
**Owner:** D · **Size:** M · **Depends:** PJ-202, PJ-306

**Description**
Prove the demo runs with zero live external API calls — TrustPositif and Gemini both unreachable.

**Acceptance Criteria**
- [ ] Full demo runs with Gemini key deliberately invalidated → cached verdicts serve
- [ ] Full demo runs with TrustPositif unreachable → cached JSON serves
- [ ] Neither failure is visible to a viewer
- [ ] Failure modes documented in `demo_script.md`

**Technical Notes**
- PRD §14 risks #1 and #3.
- Test by breaking things deliberately. A path assumed to work is a path that doesn't.

---

## PJ-704 · Verify polling fallback is indistinguishable
**Owner:** D · **Size:** S · **Depends:** PJ-408

**Description**
Confirm that with realtime disabled, the two-device demo still looks instant.

**Acceptance Criteria**
- [ ] Realtime disabled via feature flag → demo still works
- [ ] Device 2 blocks within ~3s — a judge cannot tell the difference
- [ ] Flag-flip procedure documented in `demo_script.md`

**Technical Notes**
- PRD §14 risk #4.
- D flips the flag, not C — C may be asleep or heads-down when it matters.

---

## PJ-705 · Demo hardware lock + extension preload
**Owner:** D · **Size:** S · **Depends:** PJ-404 · **T+21**

**Description**
Designate the exact laptop and second device. Extension pre-loaded and pinned. Never install in front of judges.

**Acceptance Criteria**
- [ ] Demo laptop + second device designated by name
- [ ] Extension loaded unpacked and pinned to toolbar on **both**
- [ ] Full demo run on this exact hardware ≥1 time
- [ ] Both devices charged, chargers packed
- [ ] Notifications/updates disabled on the demo laptop

**Technical Notes**
- PRD §14 risk #6.
- "Load unpacked" is a dev-mode action. Chrome may prompt or disable it on restart — verify after a full reboot, not just after a sleep.

---

## PJ-706 · Venue network test
**Owner:** D · **Size:** S · **Depends:** PJ-705 · **T+21**

**Description**
Run the demo on the actual venue wifi. Confirm hotspot backup.

**Acceptance Criteria**
- [ ] Full demo on venue wifi ≥1 time
- [ ] Supabase realtime confirmed working through venue network (captive portals and firewalls kill websockets)
- [ ] TrustPositif reachable (IP restriction — PRD §5) or cache confirmed as the path
- [ ] Phone hotspot tested as fallback
- [ ] Both fallbacks documented

**Technical Notes**
- PRD §14 risks #2 and #5.
- Websockets through a captive portal is the classic silent killer here. If realtime fails on venue wifi, PJ-408's poller is the answer — which is why it exists at T+8.

---

## PJ-707 · Cut demo-ready tag
**Owner:** D · **Size:** S · **Depends:** all · **T+21**

**Description**
Tag the known-good commit. Demo runs from the tag, not `main`.

**Acceptance Criteria**
- [ ] `git tag demo-ready` on a verified-working commit
- [ ] Demo laptop checked out at the tag
- [ ] Announced in WA: `main` is now irrelevant to the demo
- [ ] Anyone pushing after this knows it does not reach the demo

**Technical Notes**
- PRD §14 risk #11.
- Feature freeze is T+14. This tag makes it enforceable rather than aspirational.

---

## PJ-708 · Timed rehearsals
**Owner:** D · **Size:** M · **Depends:** PJ-701 · **Throughout: T+12, T+15, T+17, T+19, T+21**

**Description**
Five full timed runs against the §15 script. D is the only person testing like a judge.

**Acceptance Criteria**
- [ ] ≥5 complete runs
- [ ] Each ≤2:00
- [ ] Breakage found in rehearsal is filed and fixed, not noted and forgotten
- [ ] The "isn't this just BlockSite?" answer (§14 risk #9) delivered word-perfect
- [ ] Final run at T+21 on demo hardware, on venue wifi, from the `demo-ready` tag

**Technical Notes**
- File: `pitch/demo_script.md`
- PRD §12: D pitches — D has heard the least code and will explain it with the least jargon, which is literally scored criterion #1.
- The rehearsals are a QA pass, not a performance warm-up. Their output is bugs.

---

## Scope Guard

Not tickets. Do not create them. (PRD §6)

- WA Chatbot report packaging — nice-to-have
- Affiliate network graph — nice-to-have
- Confidence explainability breakdown — nice-to-have
- Android VpnService app — roadmap
- Live monitoring of all new registrations — needs enterprise API
- aduankonten.id auto-submit — CAPTCHA, do not bypass
- Bank account pre-blocking — not how PPATK works, legal risk
- Any self-trained vision model — explicitly not the innovation
