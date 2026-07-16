# PREJUDGE — 24-Hour Hackathon PRD

**GarudaHacks 7.0 · Track: Safety**

---

## 1. Project Brief

**Product name:** PREJUDGE
**Tagline:** _Block the domain before it does damage — not after it goes viral._

PREJUDGE starts with an empty database and builds its own blocklist: one vision-confirmed judol site yields an infrastructure fingerprint, and that fingerprint preemptively blocks its siblings — domains nobody has ever visited. Every confirmation propagates to all devices in real time, so one user's exposure protects everyone, and the system gets more preemptive the more it's used.

**Architecture in one line:** a portable **Core Engine** (detection) behind swappable **Blocker** adapters (Chrome today, Android VpnService next) and **Presentation** surfaces (block page, dashboard).

---

## 2. Problem Statement

**Core problem:** Judol blocking in Indonesia is whack-a-mole. Authorities block domains _after_ they're live and reported; bandar operators register new ones in bulk faster than takedowns land. Individual users have no defense that updates itself.

**Evidence (from source doc — verify each before pitch):**

- PPATK: Rp286.84T in judol transaction flow in 2025, across 422.1M transactions
- Player population dropped to 3.1M (1.11%) mid-2025 after mass blocking, rebounded to 12.3M (4.39%) by late 2025 — _this rebound is the whole argument_
- ~77–80% of players bet under Rp100k/transaction; concentrated in lower-middle income: students, laborers, homemakers

**Why existing solutions fail:**

| Solution                           | Gap                                                             |
| ---------------------------------- | --------------------------------------------------------------- |
| Kominfo AI + crawler               | Reactive — acts only after content is live                      |
| GATE (Unila)                       | Targets financial network, not personal access prevention       |
| Stop Judol (STMIK Banjarbaru)      | Rehab-focused, no blocking, no detection                        |
| Gamban / BlockSite / Chrome native | Generic manual lists, not Indonesia-specific, not auto-updating |

**The gap:** no Indonesian solution combines infrastructure-based preemptive detection + automatic device-level blocking + a database that **builds itself from zero**. Every existing option depends on a list someone else maintains — Kominfo's crawler, BlockSite's manual entry, TrustPositif's registry. PREJUDGE starts empty and compounds: each confirmation makes the next block cheaper.

---

## 3. Target Users

**Primary (B2C) — "Rina, 41, Bekasi"**
Homemaker. Her husband started playing judol on his phone after seeing a Telegram link. She's not technical, doesn't know which sites to block, and by the time she finds one, he's on a different one. She installs PREJUDGE on the family laptop in two clicks and never touches it again. She needs to _see why_ something was blocked — a black box makes her think the extension is broken.

**Secondary (B2G) — Komdigi verification team**
Receives a stream of pre-packaged reports (URL + screenshot + reason + confidence) instead of raw citizen complaints. Reduces triage load. They retain blocking authority; PREJUDGE never claims to replace them.

**Tertiary (B2B) — ISPs, banks/e-wallets, .id registrars**
Consume the domain feed as an additional signal.

---

## 4. Solution Overview & Business Model

### Architecture — three separable modules

The MVP ships as a Chrome extension, but the system is deliberately split so the browser is only one delivery surface. Phase 1 roadmap is Android (VpnService), where the Blocker and Presentation layers change completely and the Core Engine does not.

| Module                 | Responsibility                                                                                                                                                                                                                         | MVP implementation                                               | Android (Phase 1)                 |
| ---------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------- | --------------------------------- |
| **Core Engine**        | Detection only. Consumes a domain (+ optional page evidence), emits a verdict: `{domain, is_judol, confidence, reason, matched_fields}`. Owns Layer 1 + Layer 2 + clustering. **Knows nothing about how blocking or display happens.** | `core/`                                                          | unchanged                         |
| **Blocker Service**    | Enforcement + verdict transport. Subscribes to verdicts, applies them.                                                                                                                                                                 | `blocker/` — `declarativeNetRequest` + realtime/polling adapters | VpnService + DNS/SNI interception |
| **Presentation Layer** | User-facing surfaces. Block page, dashboard.                                                                                                                                                                                           | `presentation/blocked/`, `presentation/dashboard/`               | native Android UI                 |

**The seam that must not leak:** the Core Engine never references Chrome APIs, realtime channels, or DNR rules. It writes a verdict; _adapters_ decide what that means. The polling fallback (§14 risk #4) is proof this works — realtime and polling are two adapters behind one Core Engine contract. On Android, VpnService becomes a third. No Core Engine code changes.

> Cost of ignoring this: `chrome.tabs.captureVisibleTab` does not exist on Android — VpnService sees DNS/SNI, not pixels. If screenshot capture is welded into the detection pipeline, Phase 1 is a rewrite, not a port.

### How it works — cold start by design

**The system starts with an empty database.** There is no pre-seeded blocklist and no bulk-imported ground truth. This is a design decision, not a limitation (see "Why not seed from TrustPositif" below).

**Bootstrap (T=0, empty DB):** `fingerprint_clusters` is empty → every Layer 1 lookup returns no-match by construction → the first access to any domain **always** falls through to Layer 2. This is the honest default state of an empty table, not a special case.

**Layer 2 — Reactive-but-fast (the bootstrap path):** User hits a domain not on the blocklist. Page opens briefly. Extension captures page evidence → vision API (Gemini) → _is this an online gambling site, yes/no, brief reason_ → if yes: write verdict, extract infrastructure fingerprint, seed Layer 1.

**Layer 1 — Preemptive (grows from Layer 2):** Once Layer 2 confirms domains, Layer 1 has fingerprints to correlate against:

- Extract infrastructure fingerprint (hosting IP/ASN, nameserver, registrar, TLD, registration date)
- Detect **bulk-registration bursts** — clusters of domains registered in the same narrow window by the same registrar. _This is the actual Netcraft/PREDATOR signal: it exploits the gap between registration and campaign activation._ Correlation alone is reactive-but-clustered; the registration-window burst is what makes it genuinely preemptive.
- Score unseen domains against known clusters → flag siblings **never visited by anyone**

**The loop is the product.** Layer 2 detections feed Layer 1 fingerprints. Every confirmation makes the system more preemptive. Detection N+1 costs less than detection N. This is the answer to _"isn't this just BlockSite?"_ — BlockSite blocks a list you type in; PREJUDGE generates the list and compounds.

**Preemptive is an algorithm, not AI — deliberately.** Layer 1 is weighted field correlation plus burst detection over registration timestamps. Deterministic arithmetic, no ML. This is a design choice with a defense: deterministic scoring yields `matched_fields`, which renders directly as the block page's "Why?" bullets. PRD §5 scores transparency; an ML classifier could not explain itself to Rina. The only AI in the system is Layer 2's vision call — and per §6, that is explicitly _not_ our innovation.

### Why not seed from TrustPositif

TrustPositif has no bulk export/API, and its public results are **masked**: `a*****gacor.biz`. That string cannot be WHOIS'd, DNS-resolved, or fingerprinted. Bulk-harvesting exact confirmed domains as ground truth is not available.

TrustPositif's real role is narrower and still valuable:

- **Verifier, not source.** Submit a _full_ candidate domain → get a boolean. Masking is irrelevant when you already know what you sent.
- **Pattern constraint.** `a*****gacor.biz` reveals first character, exact masked-segment length, and exact suffix. That narrows candidate generation from blind combinatorics to constrained guessing.

**Blocking:** Chrome Manifest V3 `declarativeNetRequest` + Supabase realtime subscription (one Blocker adapter). Confirmation on one device broadcasts to all devices within seconds.

**Methodological grounding:** PREDATOR (Hao et al., ACM CCS 2016 — Princeton/UC Berkeley/Google): 70% detection rate, 0.35% FPR from registration-time features alone, days-to-weeks before conventional blacklists. Netcraft's Preemptive Domain Disruption (2026) validates the same correlation approach commercially — for phishing, so we cite it as _methodological analogy, not proof for judol_.

### Business model

| Segment             | Model                                                                                    |
| ------------------- | ---------------------------------------------------------------------------------------- |
| B2C                 | Free forever. This is the data engine — every user is a Layer 2 sensor. Not monetized.   |
| B2B (ISP)           | Licensed domain feed API, subscription per subscriber-tier — "Internet Sehat" feature    |
| B2B (bank/e-wallet) | Fraud-signal feed, per-query or flat annual                                              |
| B2G                 | Non-revenue. Free report pipeline to Komdigi — credibility and policy access, not income |

The honest pitch line: _B2C is free because users are the sensor network. The asset is the feed, and the feed is what ISPs and banks pay for._

---

## 5. Judging Criteria Alignment

**Rubric (GarudaHacks 7.0, Gavel pairwise comparison — no published weights):**

| Category      | Criterion                    | Question                                                               |
| ------------- | ---------------------------- | ---------------------------------------------------------------------- |
| Universal     | Presentation & Communication | Can a non-expert understand what was built?                            |
| Universal     | Problem Definition           | Is the problem real and well-scoped?                                   |
| Universal     | Impact & Feasibility         | Could this actually work, does it solve something meaningful?          |
| Non-technical | User Experience & Design     | Is the product usable, polished, accessible?                           |
| Non-technical | Market Fit / Viability       | Would people actually use/pay for this?                                |
| Technical     | Technical Implementation     | Architecture and complexity of what was built                          |
| Technical     | Innovation & Novelty         | Does the approach bring something new, or just follow existing design? |

**Structural note:** Gavel scores by pairwise comparison, not weighted percentages. Implications:

1. **All 7 criteria weigh roughly equally.** No "innovation is 40%" to optimize toward.
2. **Memorability beats completeness.** A project with one unforgettable moment beats a uniformly-good-but-forgettable one. → **Our moment is the two-device live block.**
3. **"Can a non-expert understand what was built?" is scored first.** Never say "infrastructure fingerprinting" without immediately saying "same landlord, same street, same paperwork."

| Criterion                    | Our play                                                                                          | Owner |
| ---------------------------- | ------------------------------------------------------------------------------------------------- | ----- |
| Presentation & Communication | 2-min demo, zero jargon, empty DB → live block on 2 screens                                       | D     |
| Problem Definition           | PPATK rebound stat: 3.1M → 12.3M. Blocking works, then fails.                                     | D     |
| Impact & Feasibility         | Cold-start ratio as measured hypothesis, never as achieved fact                                   | D     |
| UX & Design                  | Block page shows score + `matched_fields` as plain-language reasons. Transparent by architecture. | C     |
| Market Fit / Viability       | Complement to Komdigi, not competitor. Feed licensing to ISP/banks.                               | D     |
| Technical Implementation     | Core/Blocker/Presentation seam + two-layer pipeline + registration-burst detection                | A + B |
| Innovation & Novelty         | **The compounding loop**: starts empty, every L2 confirmation makes L1 more preemptive            | A + B |

**Weakest criterion to defend:** Innovation & Novelty. Blocking isn't new, vision APIs aren't new, and Layer 1 has no ML at all. Two rehearsed answers required:

- _"Isn't this BlockSite?"_ → **the empty database is the answer.** BlockSite blocks a list you type in. We start from zero and the list builds itself — one confirmation seeds fingerprints that block siblings nobody visited.
- _"Where's the AI in the preemptive layer?"_ → **deliberately absent.** Layer 1 is weighted correlation + registration-burst detection. Deterministic arithmetic is _why_ the block page can name the exact fields that matched. A classifier gives Rina a number and no reason. See §14 risk #11 — this must land as a design choice, not an absence.

---

## 6. MVP Features

### Must-have (demo-blocking — if these don't work, there's no demo)

1. **Core Engine verdict contract** — a single emit point: `{domain, is_judol, confidence, reason, matched_fields}`. No Chrome APIs, no realtime references inside Core.
2. **Layer 2 bootstrap path** — page evidence capture → Gemini → verdict + reason + confidence. This is how the system gets its first data.
3. **Fingerprint extractor** — hosting IP/ASN, nameserver, registrar, TLD, registration date, from WHOIS/DNS/RDAP
4. **Feedback loop (L2 → L1)** — confirmed detection seeds/updates a fingerprint cluster. **Without this there is no preemptive layer at all**, because there is no other data source.
5. **Layer 1 matcher** — score an unseen domain against grown clusters, emit `matched_fields`
6. **Bulk-registration burst detection** — identify domains registered in the same narrow window by the same registrar. _This is what makes Layer 1 preemptive rather than merely correlative._
7. **Database** — domains, fingerprint_clusters (incl. registration-window fields), detections, whois_records, bootstrap_runs
8. **Blocker adapter (Chrome)** — `declarativeNetRequest` + Supabase realtime, consuming the Core verdict contract
9. **Presentation — block page** — score + plain-language reason from `matched_fields`
10. **Presentation — dashboard** — detected domains, confidence, source, cluster siblings, cold-start growth view
11. **TrustPositif verifier** — per-domain full-string boolean check. **Verifier, not seed source.**

### Nice-to-have (only if ahead of schedule at T+16)

- Masked-pattern-constrained candidate generator (`a*****gacor.biz` → constrained guesses → TrustPositif verify)
- Auto-generate report package → send to WA Chatbot Stop Judi Online (0811-1001-5080)
- Affiliate network graph visualization (**mock data — must be labeled mock on screen**)

### Explicitly NOT building (and never promised in pitch)

- **Bulk import of TrustPositif as ground truth** — no bulk export exists, and public results are masked (`a*****gacor.biz`). Cannot be WHOIS'd or fingerprinted.
- Live monitoring of all new .id/global registrations (needs enterprise API)
- Auto-submit to aduankonten.id (CAPTCHA — do not bypass)
- Auto pre-block of bank accounts via OJK/PPATK (not how PPATK works; due-process risk)
- Android VpnService app (roadmap only — but the Core Engine seam is built for it now)
- Any self-trained vision model (§4: explicitly not our innovation)
- ML for Layer 1 (deliberate — deterministic scoring is what produces explainable `matched_fields`)

### Demo validation methodology — cold-start proof

The old plan (hold out a TrustPositif corpus as a test set) is **dead**: it required bulk ground truth that masking makes unavailable.

The replacement is closer to what PREDATOR/Netcraft actually claim, and it reuses the feedback loop directly:

1. Start with an empty database. Show it empty.
2. Confirm N domains via Layer 2 (the bootstrap).
3. Show that Layer 1, using only fingerprints harvested from those N, correctly flags M sibling domains **that no one has ever visited**.
4. Report N, M, and the misses honestly.

> **Say:** "We started from zero. N confirmations bought us M preemptive blocks."
> **Never say:** "We're monitoring Indonesian internet in real time" or "we have a database of X million domains."

**Leakage rule:** a domain confirmed by Layer 2 cannot be counted as a Layer 1 preemptive catch. Assert this in code, do not assume it. A leaked number turns the honesty beat into a confident lie.

---

## 7. User Flow & Wireframes

### Flow 0 — Cold start (the honest opening state)

```
Fresh install, empty DB
   → fingerprint_clusters is EMPTY
   → EVERY Layer 1 lookup returns no-match (by construction)
   → EVERY first access falls through to Layer 2
   → System has zero preemptive power. This is correct.
```

### Flow A — Layer 2 bootstrap (how the system learns)

```
User opens unknown domain
   → Not in blocklist → page opens briefly
   → Blocker captures evidence → Core /analyze → Gemini
   → "YES — slot UI, deposit CTA"
   → Verdict written: status='blocked', source='L2'
   → feedback.py: extract fingerprint → seed/update cluster
   → Layer 1 now knows something it didn't 2 seconds ago
```

### Flow B — Layer 1 preemptive (the payoff, only possible after A)

```
Different domain, never visited by anyone, no screenshot taken
   → Blocker checks blocklist → matcher scores vs clusters
   → MATCH: same hosting IP + registered in same 6h burst
   → BLOCKED before the page renders
   → Block page: "IP hosting sama dengan 47 situs terkonfirmasi.
     Didaftarkan massal, 3 hari lalu."
```

### Flow C — Collective propagation (**the demo money shot**)

```
Device 1: Layer 2 confirms domain X  (Flow A)
   → Core writes verdict
   → Blocker adapter broadcasts (realtime | polling)
Device 2 (never visited X): blocklist updated <5s
   → Device 2 opens X → BLOCKED instantly
   → AND X's siblings are now Layer-1 flagged on BOTH devices
```

### Flow D — Dashboard

```
Open dashboard → detected list (domain, confidence, L1/L2, time)
→ click row → cluster detail + sibling domains + evidence
→ Cold Start tab → N confirmations → M preemptive catches
```

### Lo-fi wireframes (ASCII — Frontend to redraw in Figma if time)

**Block page (Presentation Layer)**

```
┌──────────────────────────────────────────┐
│              🛡  PREJUDGE                 │
│                                          │
│      Situs ini diblokir                   │
│      gacor88x.xyz                         │
│                                          │
│   Keyakinan: ███████████░ 92%             │
│                                          │
│   Kenapa?                                 │
│   • IP hosting sama dengan 47 situs judi  │
│     yang sudah dikonfirmasi               │
│   • Didaftarkan massal, 3 hari lalu       │
│   • Registrar cocok dengan kluster dikenal│
│                                          │
│   [ Laporkan salah ]     [ Selengkapnya ] │
└──────────────────────────────────────────┘
```

> Every bullet is rendered from `domains.matched_fields`. **Bullet 2 ("Didaftarkan massal") requires the registration-burst fields in §9.** Previously this wireframe promised a capability nothing produced — it would have shipped empty or hardcoded.

**Dashboard**

```
┌────────────────────────────────────────────────────────┐
│ PREJUDGE  │ Terdeteksi │ Cold Start │ Jaringan         │
├────────────────────────────────────────────────────────┤
│  Hari ini: 128    L1: 91    L2: 37       ● live        │
├──────────────┬───────┬────────┬────────────────────────┤
│ DOMAIN       │ CONF  │ SUMBER │ WAKTU                  │
├──────────────┼───────┼────────┼────────────────────────┤
│ gacor88x.xyz │  92%  │  L1    │ 14:02                  │
│ slotjp7.top  │  88%  │  L2    │ 14:01                  │
│ maxwin4d.cc  │  95%  │  L1    │ 13:58                  │
└──────────────┴───────┴────────┴────────────────────────┘
     ▲ klik baris → detail fingerprint + sibling domains
```

> The live dot must reflect actual adapter state. A hardcoded green dot is a lie a judge might catch.

**Cold-start tab (replaces the old validation tab)**

```
┌────────────────────────────────────────────────────────┐
│ Bukti cold start — database dimulai KOSONG              │
│                                                         │
│  Konfirmasi Layer 2 (dilihat manusia):   N              │
│  Tangkapan preemptive Layer 1:           M              │
│    └─ domain yang TIDAK PERNAH dikunjungi siapapun      │
│  Meleset:                                X              │
│                                                         │
│  Rasio: 1 konfirmasi → M/N blokir preemptive            │
│                                                         │
│  ⚠ Bukan monitoring internet real-time.                 │
└────────────────────────────────────────────────────────┘
```

> N, M, X are **live counters from `bootstrap_runs`**, not placeholders. Do not invent figures. The ratio is the whole claim: _one confirmation buys M/N free blocks._

---

## 8. Tech Stack & Tools

**Locked. No debates after T+0.**

| Module           | Layer              | Choice                                                    | Rationale                                                                                    |
| ---------------- | ------------------ | --------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| **Core**         | Detection pipeline | Python 3.11 (pure, no framework imports)                  | Portable. `core/` must not import FastAPI, Chrome, or Supabase-realtime.                     |
| **Core**         | Vision             | Gemini 2.x Flash vision API                               | Cheap, fast, no self-trained model (not our innovation)                                      |
| **Core**         | WHOIS/DNS          | `python-whois`, `dnspython`, RDAP                         | Free, no key                                                                                 |
| **API**          | Transport          | FastAPI                                                   | Thin wrapper over Core. Owns HTTP, not logic.                                                |
| **Blocker**      | Chrome adapter     | MV3, vanilla JS + `declarativeNetRequest`                 | No build step. React in an extension is a time sink.                                         |
| **Blocker**      | Verdict transport  | Supabase Realtime **or** 3s polling (flag)                | Two adapters, one Core contract. Proof the seam works.                                       |
| **Blocker**      | Evidence capture   | Chrome `captureVisibleTab`                                | **Chrome-specific. Does not port to Android.** Isolated in `blocker/evidence.js` on purpose. |
| **Presentation** | Block page         | Plain HTML/CSS                                            | Ships in the extension bundle                                                                |
| **Presentation** | Dashboard          | Next.js 14 + Tailwind + shadcn/ui                         | Deploys to Vercel in one command                                                             |
| **Shared**       | Database           | Supabase (Postgres + Realtime)                            | Realtime is a hosted primitive — don't build it                                              |
| **Shared**       | Hosting            | Vercel (dashboard), Supabase (DB), Render (API if needed) | All free tier                                                                                |
| **Shared**       | Repo               | GitHub monorepo, branch-per-person, PR to `main`          |                                                                                              |
| **Shared**       | Coordination       | WhatsApp group + shared `.env` in pinned message          |                                                                                              |

**Hard rules:**

- No Docker. No Kubernetes. No self-hosted anything.
- No new library after **T+14** without full-team agreement.
- All API keys in a shared `.env` distributed at T+0. Nobody hunts for keys at 3am.
- **`core/` imports nothing Chrome-shaped, nothing realtime-shaped, nothing UI-shaped.** This is the Android roadmap's only insurance policy.
- **No sklearn, no ML in Layer 1.** Deterministic by design (§4).

---

## 9. Database Schema (ERD)

```
┌──────────────────────────────────┐
│ fingerprint_clusters             │
├──────────────────────────────────┤
│ id                    uuid PK    │
│ registrar             text       │
│ hosting_ip            inet       │
│ asn                   text       │
│ nameserver            text       │
│ tld                   text       │
│ domain_count          int        │
│ ── registration burst (NEW) ──   │
│ first_registration_date  date    │
│ last_registration_date   date    │
│ registration_window_hours int    │
│ registration_burst_score float   │
│ created_at            timestamptz│
└───────────┬──────────────────────┘
            │ 1
            │
            │ N
┌───────────▼─────────────┐        ┌────────────────────────┐
│ domains                 │  1   N │ detections             │
├─────────────────────────┤────────├────────────────────────┤
│ id            uuid PK   │        │ id           uuid PK   │
│ domain        text UQ   │        │ domain_id    uuid FK   │
│ cluster_id    uuid FK   │        │ layer        int (1|2) │
│ status        enum      │        │ confidence   float     │
│   (candidate|confirmed| │        │ reason       text      │
│    blocked|false_pos)   │        │ evidence_url text      │
│ source        enum      │        │ raw_response jsonb     │
│   (L1|L2|trustpositif)  │        │ detected_at  timestamptz│
│ confidence    float     │        └────────────────────────┘
│ reason        text      │  ← denormalized for Blocker
│ matched_fields jsonb    │  ← NEW: feeds block page "Why?"
│ first_seen    timestamptz│
│ registered_at date      │
│ blocked_at    timestamptz│
│ source_masked_pattern text│ ← NEW: audit trail, nullable
└───────────┬─────────────┘
            │ 1
            │ N
┌───────────▼─────────────┐        ┌──────────────────────────┐
│ whois_records           │        │ bootstrap_runs (REPLACES │
├─────────────────────────┤        │  validation_runs)        │
│ id            uuid PK   │        ├──────────────────────────┤
│ domain_id     uuid FK   │        │ id            uuid PK    │
│ registrar     text      │        │ run_at        timestamptz│
│ nameservers   text[]    │        │ l2_confirmations int     │ ← N
│ created_date  date      │        │ l1_preemptive_catches int│ ← M
│ raw           jsonb     │        │ l1_misses     int        │
│ fetched_at    timestamptz│       │ notes         text       │
└─────────────────────────┘        └──────────────────────────┘
```

**Schema changes from the original ERD, and why:**

| Change                                                                                                       | Reason                                                                                                                                                                               |
| ------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `fingerprint_clusters.first/last_registration_date`, `registration_window_hours`, `registration_burst_score` | **Bulk-registration detection had no home.** PRD §4 named it, §7's wireframe promises "Registered in bulk, 3 days ago" — nothing aggregated it. These four fields are the whole gap. |
| `domains.matched_fields jsonb`                                                                               | The block page's "Why?" bullets need the matched field list. Previously implied but unstored.                                                                                        |
| `domains.source_masked_pattern`                                                                              | Audit trail: which TrustPositif masked string (`a*****gacor.biz`) produced this candidate. Nullable — only set for pattern-derived candidates.                                       |
| `detections.screenshot_url` → `evidence_url`                                                                 | Core Engine must not assume evidence is a screenshot. Android VpnService has no pixels. Rename now; renaming later is a migration during a hackathon.                                |
| `validation_runs` → `bootstrap_runs`                                                                         | Held-out validation is dead (no bulk ground truth). Replaced by cold-start proof: N confirmations → M preemptive catches.                                                            |

**Contract for parallel work — agree at T+0, do not change after T+4:**

- **Core Engine writes verdicts. Blocker adapters read them.** Core never subscribes, never references Chrome APIs.
- Blocker reads **only** `domains` where `status = 'blocked'` → fields: `domain`, `confidence`, `reason`, `matched_fields` (all denormalized onto `domains` — no joins on this path)
- Realtime channel (Chrome adapter): `postgres_changes` on `domains`, filter `status=eq.blocked`
- Dashboard reads `domains` JOIN `detections` JOIN `fingerprint_clusters`

**Fixture data, not seed data.** The DB starts empty by design (§4). Frontend still needs shapes to build against at T+1, so ship a **fixture set clearly marked as such** (`-- FIXTURE`) and **purge it before the demo** — the demo's opening beat is an empty database, and leftover fixture rows destroy that beat while looking like success.

---

## 10. API Endpoint Plan

Base: `/api/v1`

| Method | Endpoint                 | Request                       | Response                                                            | Owner |
| ------ | ------------------------ | ----------------------------- | ------------------------------------------------------------------- | ----- |
| `GET`  | `/blocklist`             | `?since=<ISO>`                | `{domains:[{domain,confidence,reason,matched_fields}], updated_at}` | A     |
| `POST` | `/check`                 | `{domain}`                    | `{status, confidence, source, reason}` — Blocker pre-flight         | A     |
| `POST` | `/analyze`               | `{domain, evidence_b64}`      | `{is_judol, confidence, reason, domain_id}` — Layer 2 bootstrap     | B     |
| `POST` | `/fingerprint`           | `{domain}`                    | `{cluster_id, registrar, ip, ns, tld, match_score, matched_fields}` | A     |
| `GET`  | `/domains`               | `?limit&offset&source&status` | `{items:[...], total}` — dashboard list                             | A     |
| `GET`  | `/domains/{id}`          | —                             | `{domain, detections[], whois, cluster, siblings[], evidence_url}`  | A     |
| `POST` | `/report-false-positive` | `{domain_id, note}`           | `{ok:true}`                                                         | A     |
| `GET`  | `/bootstrap/latest`      | —                             | `{l2_confirmations, l1_preemptive_catches, l1_misses, ratio}`       | A     |
| `POST` | `/trustpositif/verify`   | `{domain}`                    | `{domain, is_blocked}` — **single full domain, verifier only**      | A     |

**Removed from the original plan:**

- `POST /trustpositif/bulk-check` — bulk harvesting is dead (masked results, no bulk export). Replaced by `/trustpositif/verify`, single-domain, verifier role only.
- `GET /validation/latest` — replaced by `/bootstrap/latest` (cold-start proof, not held-out validation).

**Core Engine boundary:** `/analyze`, `/fingerprint` are Core. `/blocklist`, `/check` are the Blocker adapter's read surface. `/domains`, `/bootstrap/latest` are Presentation's. A Core function must never import a Chrome API or a realtime client.

**Realtime (not REST):** the Chrome Blocker adapter subscribes to Supabase channel `domains:status=eq.blocked`. **This is one adapter, not the architecture.** Polling (§14 risk #4) is a second adapter; VpnService will be a third. Build and test the subscription first — but do not let it leak into Core.

**Contract-first:** paste this table into the repo README at T+0. Backend returns hardcoded stub responses matching these shapes by **T+2** so Frontend is never blocked.

---

## 11. Suggested Project Structure

```
prejudge/
├── README.md                 ← Core verdict contract + API + .env template
├── .env.example
│
├── core/                     ← CORE ENGINE — OWNERS: A (layer1) / B (layer2)
│   │                            NO Chrome APIs. NO realtime. NO UI. Portable to Android as-is.
│   ├── contract.py           ← Verdict dataclass. The seam. Touch = all-4 discussion.
│   ├── layer1/               ← A ONLY
│   │   ├── fingerprint.py    ← WHOIS/DNS/RDAP extract
│   │   ├── cluster.py        ← grouping + registration-burst detection
│   │   └── matcher.py        ← cluster similarity → matched_fields
│   ├── layer2/               ← B ONLY
│   │   ├── vision.py         ← Gemini client (evidence → verdict)
│   │   └── decide.py         ← threshold logic → emit verdict
│   ├── feedback.py           ← L2 verdict → L1 cluster seeding (THE LOOP)
│   └── trustpositif.py       ← single-domain verifier + masked-pattern parser
│
├── api/                      ← TRANSPORT — OWNER: A
│   ├── main.py               ← FastAPI. Thin. Calls core/, returns verdicts.
│   └── models.py             ← Pydantic request/response
│
├── blocker/                  ← BLOCKER SERVICE (Chrome adapter) — OWNER: C
│   ├── manifest.json
│   ├── background.js         ← DNR rules + realtime adapter + polling adapter
│   └── evidence.js           ← captureVisibleTab — CHROME-SPECIFIC, does not port
│
├── presentation/             ← PRESENTATION LAYER — OWNER: C
│   ├── blocked/              ← block page (rides in the extension bundle)
│   │   ├── blocked.html
│   │   └── blocked.css
│   └── dashboard/            ← Next.js
│       ├── app/
│       │   ├── page.tsx              ← detected list
│       │   ├── domain/[id]/page.tsx  ← detail + siblings
│       │   └── bootstrap/page.tsx    ← cold-start growth proof
│       └── lib/supabase.ts
│
├── db/
│   ├── schema.sql
│   └── fixtures.sql          ← FIXTURE ONLY. Purge before demo.
│
├── scripts/
│   └── bootstrap_run.py      ← cold-start proof: N confirmations → M catches
│
└── pitch/                    ← OWNER: D
    ├── deck.pdf
    ├── demo_script.md
    └── sources.md
```

**Merge-conflict rules:**

- One owner per top-level directory. Cross-directory edits require a WA message first.
- Branch naming: `a/core-layer1-burst`, `c/blocker-realtime`
- PR to `main`, no direct pushes. Reviewer = anyone awake.
- **`db/schema.sql` frozen at T+4.** Changes after that need all four to agree.
- **`core/contract.py` frozen at T+2.** It is the seam. If it churns, the modularity is theatre.

**The seam test — run it mentally before every Core commit:** _would this file still compile if the extension didn't exist?_ If no, it's in the wrong directory. `core/` must have zero knowledge of Chrome, of Supabase realtime, of the block page. That is what makes the Android port a port instead of a rewrite.

---

## 12. Roles & Responsibilities (4 members)

|       | Role                    | Owns                                              | Primary deliverable                                                  |
| ----- | ----------------------- | ------------------------------------------------- | -------------------------------------------------------------------- |
| **A** | Core/Layer 1 + Schema   | `core/layer1/`, `core/contract.py`, `db/`, `api/` | Fingerprint + burst detection + matcher + `matched_fields`           |
| **B** | Core/Layer 2 + Feedback | `core/layer2/`, `core/feedback.py`                | Vision bootstrap + **the L2→L1 loop** (the novelty claim)            |
| **C** | Blocker + Presentation  | `blocker/`, `presentation/`                       | Two-device block (realtime + polling adapters) + dashboard           |
| **D** | Pitch, Research & QA    | `pitch/`, demo reliability (Epic 8)               | 2-min deck + demo script + cluster bootstrap + verified `sources.md` |

**Load-balancing notes:**

- **C carries the heaviest load** (Blocker + Presentation). At the **T+9.5 GO/NO-GO**, if the Blocker isn't blocking end-to-end, **D drops deck work and pairs on Presentation** — the dashboard is the simpler half.
- **B's Epic 3 (the feedback loop) is now on the critical path, not a nice-to-have.** Under cold start, no loop → no Layer 1 → no preemptive claim. If B is stuck at T+8, A should pair rather than polish the matcher — a perfect matcher with no clusters to match against is worth nothing.
- **D is not idle.** D owns: source verification (§16), the demo script, **the cluster bootstrap (§14 risk #1 — the top risk)**, and being the person who clicks through the demo 5+ times to find where it breaks. D is the only one testing like a judge.
- **D pitches.** D has heard the least code and will therefore explain it in the least jargon — which is literally criterion #1.
- **A owns the schema and the contract.** If two people write schema, you get two schemas. If two people edit `core/contract.py`, the seam is theatre.

**Scheduling consequence of cold start:** Layer 1 used to be seedable independently from TrustPositif, so A and B could work in parallel. Masking killed that. The chain is now **serial: L2 → feedback → L1**. A's Epic 4 depends on B's Epic 3 producing clusters. Plan for A to be partially blocked around T+5–T+8 and use that window for the fill-rate report and the API surface, not for waiting.

---

## 13. 24-Hour Execution Timeline

Blocks of ~4h. `T+0` = start.

| Block | Window       | A (Core/L1)                                                                                                                                    | B (Core/L2)                                            | C (Blocker/Presentation)                                                 | D (Pitch/QA)                                          |
| ----- | ------------ | ---------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------ | ------------------------------------------------------------------------ | ----------------------------------------------------- |
| **1** | T+0 → T+2    | Supabase, `schema.sql`, **`core/contract.py` — the seam**, stub API                                                                            | Gemini key working, one evidence → verdict by hand     | `manifest.json`, extension loads, **realtime adapter prints to console** | Lock problem statement, start `sources.md`            |
| **2** | T+2 → T+5    | Fingerprint extractor + **WHOIS fill-rate report** ← gate                                                                                      | `/analyze` roundtrip: evidence → Gemini → verdict → DB | **Hardcoded blocklist blocks a real domain** ← gate                      | Deck skeleton, verify PPATK/Netcraft/PREDATOR figures |
| —     | **T+5**      | 🚩 **Fill-rate gate.** If `registered_at` fill rate is poor, **burst detection is dead** — cut it from the pitch now, not at T+20.             |                                                        |                                                                          |                                                       |
| **3** | T+5 → T+8    | Cluster builder + **registration-burst detection**                                                                                             | **Feedback loop L2 → L1** ← the novelty claim          | Block page w/ score + `matched_fields`; dashboard list                   | Deck v1. Start demo script.                           |
| **4** | T+8 → T+9.5  | Matcher + `matched_fields` output                                                                                                              | Threshold tuning, `raw_response` logging               | **Two-device propagation working** + **polling adapter built**           | 🚩 **CHECKPOINT T+9.5**                               |
| —     | **T+9.5**    | **GO/NO-GO.** If two-device block doesn't work, all four fix it. This is the demo.                                                             |                                                        |                                                                          |                                                       |
| **5** | T+9.5 → T+12 | `/blocklist`, `/domains`, `/bootstrap` real                                                                                                    | `bootstrap_run.py` — cold-start proof                  | Dashboard detail + siblings + Cold Start tab                             | Rehearse #1. Find breakage.                           |
| **6** | T+12 → T+14  | Bugfix                                                                                                                                         | **Bootstrap the demo clusters** ← see gate below       | Polish block page, dashboard                                             | Deck v2, **record fallback video**                    |
| —     | **T+14**     | 🚩 **Cluster gate (§14 risk #1).** Does ≥1 cluster have ≥5 siblings? **If no: cut the Layer 1 beat from the demo now.** 🛑 **FEATURE FREEZE.** |                                                        |                                                                          |                                                       |
| **7** | T+14 → T+17  | **SLEEP (A, B)** — 3h, staggered                                                                                                               |                                                        | Bugfix, then sleep                                                       | Rehearse #2, #3                                       |
| **8** | T+17 → T+21  | Final integration on demo laptop, both devices, venue wifi. **Purge fixtures.**                                                                |                                                        |                                                                          | Rehearse #4, #5 — timed                               |
| **9** | T+21 → T+24  | Buffer. `demo-ready` tag. Charge everything. Deck → PDF. Fallback video on desktop. Do nothing new.                                            |                                                        |                                                                          |                                                       |

**Non-negotiables:**

- **Everyone sleeps ≥3h.** A team that hasn't slept loses on "can a non-expert understand what you built" because they can't speak.
- **Feature freeze at T+14 is real.** Most hackathon losses are a feature merged in the last 2 hours.
- **Two hard gates: T+5 (fill rate) and T+14 (cluster formation).** Both can force cutting a pitch claim. Both are better discovered on schedule than on stage.
- **Demo runs on ONE designated laptop** + one second device. Test on that exact hardware from T+21.

---

## 14. Risks & Fallback Plan

| #   | Risk                                                                                                                                                                                                                            | Likelihood              | Fallback                                                                                                                                                                                                                                                                                                                                                                                    |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Cold start produces too few clusters to demo Layer 1** — the new #1 risk. Bulk seeding is gone; if Layer 2 confirmations don't yield a cluster with siblings, there is no preemptive beat and the pitch loses its core claim. | **High**                | Bootstrap deliberately: run Layer 2 over a curated set of same-network judol domains at T+12–14 (PJ-306's cache work doubles as this). Verify ≥1 cluster reaches ≥5 members. **If no cluster forms by T+14, the demo becomes Layer 2 + propagation only and the Layer 1 beat is cut** — decide this at T+14, not on stage.                                                                  |
| 2   | **TrustPositif results are masked** (`a*****gacor.biz`)                                                                                                                                                                         | **Certain — confirmed** | Not a risk, a constraint. Verifier role only (full domain in → boolean out). Never claim bulk import. Masked-pattern-constrained generation is nice-to-have, not a dependency.                                                                                                                                                                                                              |
| 3   | **TrustPositif rate-limits / IP-restricts**                                                                                                                                                                                     | Medium                  | Reduced impact under cold start — it's no longer on the critical path. Cache all verify results. If unavailable entirely, the demo still works: Layer 2 is the bootstrap, not TrustPositif.                                                                                                                                                                                                 |
| 4   | **Gemini API quota / latency** — now higher stakes, since Layer 2 _is_ the bootstrap                                                                                                                                            | **High**                | Cache verdicts per domain. Pre-run Layer 2 on all demo domains at T+14 and cache. Live call reads cache on failure. **Gemini being down at demo = no bootstrap = no system.**                                                                                                                                                                                                               |
| 5   | **Supabase realtime doesn't fire**                                                                                                                                                                                              | Medium                  | Polling adapter (§8). Visually identical to a judge. **Build at T+8, not T+22.** Both are adapters behind the Core contract — swapping is a flag flip, not a rewrite.                                                                                                                                                                                                                       |
| 6   | **Venue wifi dies / captive portal**                                                                                                                                                                                            | **High**                | Everything local: cached data, fallback video on desktop. Phone hotspot tested at T+21. Captive portals kill websockets → polling adapter is the answer.                                                                                                                                                                                                                                    |
| 7   | **Chrome blocks the unpacked extension / MV3 quirk**                                                                                                                                                                            | Medium                  | Pre-loaded and pinned on the demo laptop from T+21. Never install live. Verify after a full reboot, not just a sleep.                                                                                                                                                                                                                                                                       |
| 8   | **WHOIS redacted → fingerprint fields empty**                                                                                                                                                                                   | **High**                | Don't depend on registrant fields. Use hosting IP + ASN + nameserver + TLD + `registered_at`. **Run the fill-rate report at T+5.** If `registered_at` fill rate is poor, **burst detection dies and "preemptive" becomes unsupportable** — say so honestly rather than weighting a field you don't have.                                                                                    |
| 9   | **Cold-start ratio is unimpressive (e.g. 5 confirmations → 2 catches)**                                                                                                                                                         | Medium                  | **Report it honestly.** "PREDATOR's 70% came from a full dataset over months. We started from an empty database 20 hours ago." Judges reward honest measurement over suspicious perfection. Do not tune the number.                                                                                                                                                                         |
| 10  | **"Isn't this just BlockSite?"**                                                                                                                                                                                                | **High**                | Rehearsed (D): "BlockSite blocks a list you type in. We started with an empty database — watch." _Show the cold-start tab._ "Every confirmation seeds fingerprints, and siblings get blocked without anyone visiting them. The blocker isn't the product; the loop is."                                                                                                                     |
| 11  | **"Where's the AI in your preemptive layer?"** ← **NEW, and currently unrehearsed**                                                                                                                                             | **High**                | Rehearsed (D): "There isn't any, deliberately. Layer 1 is weighted correlation plus registration-burst detection — arithmetic. That's why the block page can tell Rina _exactly_ which fields matched. A classifier would give her a number and no reason. The AI is in Layer 2, where the problem is actually visual." **Must be delivered as a design choice, not caught as an absence.** |
| 12  | **Fixture rows still in the DB at demo time**                                                                                                                                                                                   | Medium                  | The opening beat is an _empty database_. Leftover fixtures destroy it while looking like success. Purge + verify at T+21 as part of the `demo-ready` tag.                                                                                                                                                                                                                                   |
| 13  | **Layer 2 demo domain already cached from rehearsal**                                                                                                                                                                           | **High**                | Silently converts the Layer 2 beat into a Layer 1 beat — demo appears to work while its whole point evaporates. Verify the domain is absent from the blocklist immediately before demo.                                                                                                                                                                                                     |
| 14  | **False positive on a legit site during demo**                                                                                                                                                                                  | Medium                  | Demo domains fixed and pre-tested. The "Laporkan salah" button _is_ the answer — show it.                                                                                                                                                                                                                                                                                                   |
| 15  | **Someone breaks `main` at T+22**                                                                                                                                                                                               | Medium                  | Feature freeze T+14. Tag `demo-ready` at T+21. Demo runs from the tag.                                                                                                                                                                                                                                                                                                                      |
| 16  | **Core seam leaks under time pressure**                                                                                                                                                                                         | **High**                | At T+20 someone will want to `import chrome` into Core to fix a bug. The seam test (§11): _would this file compile if the extension didn't exist?_ A leak here doesn't break the demo — it breaks the Android roadmap claim in §16, silently, and only Phase 1 finds out.                                                                                                                   |

**Fallback video:** recorded at T+12–14, showing the full two-device block. If everything fails live, D plays the video and says so plainly. Recorded early, while it works — not at T+23 when it doesn't.

---

## 15. Demo Flow & Pitch Scenario

**Target: 2 minutes.** Must-have features only. Two screens visible to judges (laptop + phone/second laptop, both with extension installed).

| Time      | Beat                                    | Script (D speaks)                                                                                                                                                                                                                                                                                                                               |
| --------- | --------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 0:00–0:20 | **The rebound**                         | "Indonesia blocked millions of gambling sites in 2025. Players dropped to 3.1 million. Six months later: 12.3 million. Blocking works — until the next domain. We're always one domain late."                                                                                                                                                   |
| 0:20–0:35 | **The empty database** ⭐ _new opening_ | _→ show Cold Start tab: all zeros._ "This is our database. It's empty. We have no list. We didn't buy one, we didn't scrape one. Watch what happens."                                                                                                                                                                                           |
| 0:35–1:05 | **Layer 2 bootstrap**                   | _→ Device 1 opens an unknown judol site, loads ~1s → evidence captured → verdict appears: "judi online — slot UI, tombol deposit" → blocked._ "One site, seen once. Now here's the part that matters —"                                                                                                                                         |
| 1:05–1:35 | **Layer 1 emerges from nothing**        | _→ dashboard: cluster now exists, siblings listed._ "That one site told us its hosting IP, its nameserver, and that it was registered in a burst of 40 domains in the same six hours. Bandar don't buy one domain — they buy hundreds. Same landlord, same street, same paperwork." _→ click a sibling → **blocked, never visited by anyone**._ |
| 1:35–1:50 | **Propagation**                         | "And this phone has never seen any of it." _→ hold up Device 2 → open the sibling → **instantly blocked**._ "One person's exposure protected everyone."                                                                                                                                                                                         |
| 1:50–2:00 | **Close**                               | _→ Cold Start tab: N → M._ "We started at zero, twenty hours ago. N confirmations bought M preemptive blocks. Free for families — they're the sensor network. The feed is what ISPs and banks pay for. Komdigi still decides what's officially blocked; we just get them there faster."                                                         |

**Why this ordering beats the old one:** the old script opened on a pre-populated dashboard and never explained where the data came from — a judge asking "so you just imported a blocklist?" would have deflated the whole pitch. Starting from an empty DB turns the biggest constraint (no bulk ground truth) into the strongest narrative beat: _the system builds itself in front of them._

**Rehearsal rules:**

- Demo domains **fixed and pre-tested**. Never type a URL you haven't run 5 times.
- **Verify the Layer 2 domain is NOT in the blocklist immediately before demo** (§14 risk #13). A rehearsal-cached verdict silently turns the bootstrap beat into a Layer 1 beat and the demo's point evaporates while appearing to work.
- **Verify fixtures are purged** (§14 risk #12). The opening beat is an empty database.
- If something hangs >3s: keep talking, cut to the fallback video, say "let me show the recorded run" without apology.
- **Never say:** "we monitor Indonesian internet in real time," "we blocked X million domains," "we have a database of X domains," "we work with Kominfo," or any unverified number.
- Two rehearsed Q&A killers: #10 (BlockSite) and #11 (where's the AI). Both must be word-perfect.

---

## 16. Post-Hackathon Roadmap

**Phase 1 — Harden & port (0–3 months)**

- **Android app with `VpnService`** — system-wide blocking, the actual use case since judol is mobile-first. **New Blocker adapter + new Presentation layer. Core Engine unchanged.** This is what the §4 seam bought.
  - Note: Android has no page screenshot. The Layer 2 evidence path becomes DNS/SNI + optional fetch-and-render server-side. `blocker/evidence.js` is Chrome-only by design and is _expected_ to be replaced, not ported.
- Cold-start acceleration: cross-user cluster sharing means new installs inherit an already-warm Layer 1
- False-positive review queue + community reporting
- Chrome Web Store submission

**Phase 2 — Legitimize (3–6 months)**

- Formal engagement with Komdigi: propose the report pipeline as a verified complaint channel
- **Publish the cold-start methodology + real ratios openly** — credibility is the moat, and "we started from an empty DB" is a more auditable claim than any accuracy percentage
- Partner with a `.id` registrar for at-registration risk flagging — **this is where the registration-burst signal becomes true PREDATOR-style preemption**, because a registrar sees the burst as it happens rather than inferring it afterward
- Independent replication of PREDATOR-style features on Indonesian judol data — _this is a publishable paper_

**Phase 3 — Monetize (6–12 months)**

- Domain feed API GA: ISP tier (per-subscriber), bank/e-wallet tier (fraud signal)
- Pilot with one ISP under "Internet Sehat"
- B2C stays free, permanently

**Public release strategy**

- Open-source the Blocker and Presentation layers. **Keep the Core Engine's cluster data as the commercial asset.** Open blocker builds trust; the loop is the business.
- Distribution: partner with Komdigi's existing anti-judol campaign, family/parenting communities, and university student orgs — not paid ads.

**Do not build:** anything touching bank accounts, anything auto-submitting to CAPTCHA-protected government forms, anything claiming authority to block officially, any ML in Layer 1 that costs the explainability the block page depends on.

---

### ⚠️ Pre-submission verification checklist (D owns)

- [ ] Every PPATK figure traced to a primary source URL in `sources.md`
- [ ] Netcraft launch/award dates re-confirmed — put the link in `sources.md` anyway, judges may ask
- [ ] PREDATOR citation exact: Hao, Kantchelian, Miller, Paxson, Feamster — ACM CCS 2016 (**not** University of Houston)
- [ ] NXDomain side-channel (99%), DGA reverse-engineering (99.93% AUC), WhoisXML "300k/day" — **cut from deck unless independently verified**
- [ ] No claim of existing partnership with Kominfo/Komdigi/OJK/PPATK anywhere in deck or demo
- [ ] **No claim of bulk-importing TrustPositif.** Results are masked (`a*****gacor.biz`); we use it as a per-domain verifier only.
- [ ] **Cold-start numbers on the dashboard are live counters from `bootstrap_runs`**, not invented figures
- [ ] **Fixtures purged from the DB** — the opening beat is an empty database (§14 risk #12)
- [ ] **Layer 2 demo domain confirmed absent from blocklist** immediately before demo (§14 risk #13)
- [ ] If burst detection was cut at the T+5 fill-rate gate: **"preemptive" claims softened in deck and script** to correlation-only. Do not claim the registration-window mechanism if it isn't running.
- [ ] If no cluster reached ≥5 siblings at T+14: **Layer 1 beat cut from the demo**, deck updated to match
- [ ] Seam test passed: does anything in `core/` import Chrome, Supabase-realtime, or UI? If yes, the §16 Android roadmap claim is not honest.
- [ ] Mock affiliate graph (if built) labeled "SIMULATED DATA" on screen
