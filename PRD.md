# PREJUDGE — 24-Hour Hackathon PRD
**GarudaHacks 7.0 · Track: Safety**

---

## 1. Project Brief

**Product name:** PREJUDGE
**Tagline:** *Block the domain before it does damage — not after it goes viral.*

PREJUDGE is a Chrome extension backed by a two-layer detection pipeline that identifies Indonesian online gambling (*judol*) domains — preemptively via infrastructure fingerprinting against TrustPositif-confirmed domains, and reactively via vision-API screenshot analysis. Every confirmed domain propagates to all installed devices in real time, so one user's exposure protects everyone else.

---

## 2. Problem Statement

**Core problem:** Judol blocking in Indonesia is whack-a-mole. Authorities block domains *after* they're live and reported; bandar operators register new ones in bulk faster than takedowns land. Individual users have no defense that updates itself.

**Evidence (from source doc — verify each before pitch):**
- PPATK: Rp286.84T in judol transaction flow in 2025, across 422.1M transactions
- Player population dropped to 3.1M (1.11%) mid-2025 after mass blocking, rebounded to 12.3M (4.39%) by late 2025 — *this rebound is the whole argument*
- ~77–80% of players bet under Rp100k/transaction; concentrated in lower-middle income: students, laborers, homemakers

**Why existing solutions fail:**

| Solution | Gap |
|---|---|
| Kominfo AI + crawler | Reactive — acts only after content is live |
| GATE (Unila) | Targets financial network, not personal access prevention |
| Stop Judol (STMIK Banjarbaru) | Rehab-focused, no blocking, no detection |
| Gamban / BlockSite / Chrome native | Generic manual lists, not Indonesia-specific, not auto-updating |

**The gap:** no Indonesian solution combines infrastructure-based preemptive detection + automatic device-level blocking + a self-updating collective database.

---

## 3. Target Users

**Primary (B2C) — "Rina, 41, Bekasi"**
Homemaker. Her husband started playing judol on his phone after seeing a Telegram link. She's not technical, doesn't know which sites to block, and by the time she finds one, he's on a different one. She installs PREJUDGE on the family laptop in two clicks and never touches it again. She needs to *see why* something was blocked — a black box makes her think the extension is broken.

**Secondary (B2G) — Komdigi verification team**
Receives a stream of pre-packaged reports (URL + screenshot + reason + confidence) instead of raw citizen complaints. Reduces triage load. They retain blocking authority; PREJUDGE never claims to replace them.

**Tertiary (B2B) — ISPs, banks/e-wallets, .id registrars**
Consume the domain feed as an additional signal.

---

## 4. Solution Overview & Business Model

### How it works

**Layer 1 — Preemptive (cron):** Generate candidate domain names (keyword combos + cheap TLDs, name-pattern variation from known bandar naming, public mentions) → bulk-check against TrustPositif (≤100 domains/request) → for confirmed hits, extract infrastructure fingerprint (registrar, hosting IP/ASN, nameserver, TLD, registration date, bulk-registration pattern) → query *other* domains sharing that fingerprint → flag as candidates.

**Layer 2 — Reactive-but-fast:** User hits a domain not on the blocklist. Page opens briefly. Extension fires a screenshot → vision API (Gemini) with a simple prompt: *is this an online gambling site, yes/no, brief reason* → if yes, write to DB, extract its infrastructure fingerprint, feed back into Layer 1.

**Blocking:** Chrome Manifest V3 `declarativeNetRequest` + Supabase realtime subscription. Confirmation on one device broadcasts to all devices within seconds.

**Methodological grounding:** PREDATOR (Hao et al., ACM CCS 2016 — Princeton/UC Berkeley/Google): 70% detection rate, 0.35% FPR from registration-time features alone, days-to-weeks before conventional blacklists. Netcraft's Preemptive Domain Disruption (2026) validates the same correlation approach commercially — for phishing, so we cite it as *methodological analogy, not proof for judol*.

### Business model

| Segment | Model |
|---|---|
| B2C | Free forever. This is the data engine — every user is a Layer 2 sensor. Not monetized. |
| B2B (ISP) | Licensed domain feed API, subscription per subscriber-tier — "Internet Sehat" feature |
| B2B (bank/e-wallet) | Fraud-signal feed, per-query or flat annual |
| B2G | Non-revenue. Free report pipeline to Komdigi — credibility and policy access, not income |

The honest pitch line: *B2C is free because users are the sensor network. The asset is the feed, and the feed is what ISPs and banks pay for.*

---

## 5. Judging Criteria Alignment

**Rubric (GarudaHacks 7.0, Gavel pairwise comparison — no published weights):**

| Category | Criterion | Question |
|---|---|---|
| Universal | Presentation & Communication | Can a non-expert understand what was built? |
| Universal | Problem Definition | Is the problem real and well-scoped? |
| Universal | Impact & Feasibility | Could this actually work, does it solve something meaningful? |
| Non-technical | User Experience & Design | Is the product usable, polished, accessible? |
| Non-technical | Market Fit / Viability | Would people actually use/pay for this? |
| Technical | Technical Implementation | Architecture and complexity of what was built |
| Technical | Innovation & Novelty | Does the approach bring something new, or just follow existing design? |

**Structural note:** Gavel scores by pairwise comparison, not weighted percentages. Implications:

1. **All 7 criteria weigh roughly equally.** No "innovation is 40%" to optimize toward.
2. **Memorability beats completeness.** A project with one unforgettable moment beats a uniformly-good-but-forgettable one. → **Our moment is the two-device live block.**
3. **"Can a non-expert understand what was built?" is scored first.** Never say "infrastructure fingerprinting" without immediately saying "same landlord, same street, same paperwork."

| Criterion | Our play | Owner |
|---|---|---|
| Presentation & Communication | 2-min demo, zero jargon, live block on 2 screens | D |
| Problem Definition | PPATK rebound stat: 3.1M → 12.3M. Blocking works, then fails. | D |
| Impact & Feasibility | Framed as measurable hypothesis, never as achieved fact | D |
| UX & Design | Block page shows score + plain-language *reason*. Transparent, not black box. | C |
| Market Fit / Viability | Complement to Komdigi, not competitor. Feed licensing to ISP/banks. | D |
| Technical Implementation | Two-layer pipeline, real public data sources, realtime propagation | A + B |
| Innovation & Novelty | Preemptive fingerprinting + reactive vision + collective realtime blocking — combination is new for judol ID | A + B |

**Weakest criterion to defend:** Innovation & Novelty. Blocking isn't new, vision APIs aren't new. Rehearsed answer: *the novelty is not any component, it's the feedback loop — Layer 2 detections feed Layer 1 fingerprints, so the system gets better at preempting the more people use it.*

---

## 6. MVP Features

### Must-have (demo-blocking — if these don't work, there's no demo)
1. Candidate generator + TrustPositif bulk-checker — script, ≤100 domains/request, produces confirmed ground-truth set
2. Fingerprint extractor — registrar, hosting IP/ASN, nameserver, TLD, registration date, from WHOIS/DNS
3. Layer 1 matcher — given a new domain, score similarity against known bandar fingerprint clusters
4. Layer 2 screenshot + vision API — capture page, send to Gemini, boolean + reason + confidence
5. Database — domains table (domain, detected_at, confidence, source, status) + fingerprints
6. Chrome extension — `declarativeNetRequest` blocking + Supabase realtime subscription + block page showing score & reason
7. Dashboard — detected domains list, confidence, source, and the historical validation view

### Nice-to-have (only if ahead of schedule at T+16)
- Auto-generate report package → send to WA Chatbot Stop Judi Online (0811-1001-5080)
- Affiliate network graph visualization (**mock data — must be labeled mock on screen**)
- Confidence-score explainability breakdown (which fingerprint fields matched)

### Explicitly NOT building (and never promised in pitch)
- Live monitoring of all new .id/global registrations (needs enterprise API)
- Auto-submit to aduankonten.id (CAPTCHA — do not bypass)
- Auto pre-block of bank accounts via OJK/PPATK (not how PPATK works; due-process risk)
- Android VpnService app (roadmap only)

### Demo validation methodology (honest framing — rehearse this)
Hold out a subset of TrustPositif-confirmed domains as a test set. Treat them as unseen. Show that infrastructure fingerprint alone matches them to known bandar clusters. Report the hit rate honestly, including misses.

> **Say:** "We proved the method works on historical data."
> **Never say:** "We're monitoring Indonesian internet in real time."

---

## 7. User Flow & Wireframes

### Flow A — Preemptive block (Layer 1 hit)
```
User clicks judol link (WA/Telegram/ads)
   → Extension intercepts, checks local blocklist
   → MATCH (Layer 1 preemptive)
   → BLOCKED. Page never renders.
   → Block page: "Blocked. Confidence 92%. Reason: same
     registrar + hosting IP as 47 confirmed judol domains."
```

### Flow B — Reactive block + collective propagation (**this is the demo money shot**)
```
Device 1: user opens unknown domain
   → Not in blocklist → page opens briefly
   → Extension screenshots → Gemini vision API
   → "YES, online gambling — slot game UI, deposit CTA"
   → Write to Supabase → extract fingerprint → feed Layer 1
   → Device 1 blocked immediately
   → Supabase realtime broadcast
Device 2 (never visited it): blocklist updated in <5s
   → Device 2 opens same domain → BLOCKED instantly
```

### Flow C — Dashboard
```
Open dashboard → detected domains list (domain, confidence,
source L1/L2, timestamp) → click row → fingerprint match
detail + screenshot → historical validation tab (test-set
hit rate)
```

### Lo-fi wireframes (ASCII — Frontend to redraw in Figma if time)

**Block page (extension)**
```
┌──────────────────────────────────────────┐
│              🛡  PREJUDGE                 │
│                                          │
│      This site was blocked                │
│      gacor88x.xyz                         │
│                                          │
│   Confidence: ███████████░ 92%            │
│                                          │
│   Why?                                    │
│   • Same hosting IP as 47 confirmed       │
│     gambling sites                        │
│   • Registered in bulk, 3 days ago        │
│   • Registrar matches known cluster       │
│                                          │
│   [ Report as mistake ]  [ Learn more ]   │
└──────────────────────────────────────────┘
```

**Dashboard**
```
┌────────────────────────────────────────────────────────┐
│ PREJUDGE  │ Detected  │ Validation  │ Network          │
├────────────────────────────────────────────────────────┤
│  Detected today: 128    L1: 91    L2: 37   ● live      │
├──────────────┬───────┬────────┬────────────────────────┤
│ DOMAIN       │ CONF  │ SOURCE │ TIME                   │
├──────────────┼───────┼────────┼────────────────────────┤
│ gacor88x.xyz │  92%  │  L1    │ 14:02                  │
│ slotjp7.top  │  88%  │  L2    │ 14:01                  │
│ maxwin4d.cc  │  95%  │  L1    │ 13:58                  │
└──────────────┴───────┴────────┴────────────────────────┘
     ▲ click row → fingerprint detail + screenshot
```

**Validation tab**
```
┌────────────────────────────────────────────────────────┐
│ Historical validation — held-out TrustPositif set       │
│                                                         │
│  Test set: 200 confirmed domains                        │
│  Matched to known cluster: 141  (70.5%)                 │
│  Missed:                     59  (29.5%)                │
│                                                         │
│  ⚠ Historical data. Not live internet monitoring.       │
└────────────────────────────────────────────────────────┘
```
> Numbers above are **placeholders**. Fill with actual results. Do not ship the mockup figures.

---

## 8. Tech Stack & Tools

**Locked. No debates after T+0.**

| Layer | Choice | Rationale |
|---|---|---|
| Extension | Chrome Manifest V3, vanilla JS + `declarativeNetRequest` | No build step. React in an extension is a time sink. |
| Dashboard | Next.js 14 (App Router) + Tailwind + shadcn/ui | Fast, deploys to Vercel in one command |
| Backend / Pipeline | Python 3.11 + FastAPI | Team's strongest language; best WHOIS/DNS libs |
| Database + Realtime | Supabase (Postgres + Realtime + Auth) | Realtime subscription is a hosted primitive — this is the demo's core, don't build it |
| Vision | Gemini 2.x Flash vision API | Cheap, fast, no self-trained model (not our innovation) |
| Screenshot | Chrome `captureVisibleTab` (extension-side) | Zero server infra. Playwright is fallback only. |
| WHOIS/DNS | `python-whois`, `dnspython`, RDAP | Free, no key |
| Cron | Supabase scheduled function *or* manual trigger | Don't burn time on scheduling infra in 24h |
| Hosting | Vercel (dashboard), Supabase (DB/API), Render (FastAPI if needed) | All free tier |
| Repo | GitHub monorepo, branch-per-person, PR to `main` | |
| Coordination | WhatsApp group + shared `.env` in pinned message | |

**Hard rules:**
- No Docker. No Kubernetes. No self-hosted anything.
- No new library after **T+14** without full-team agreement.
- All API keys in a shared `.env` distributed at T+0. Nobody hunts for keys at 3am.

---

## 9. Database Schema (ERD)

```
┌─────────────────────────┐
│ fingerprint_clusters    │
├─────────────────────────┤
│ id            uuid PK   │
│ registrar     text      │
│ hosting_ip    inet      │
│ asn           text      │
│ nameserver    text      │
│ tld           text      │
│ domain_count  int       │
│ created_at    timestamptz│
└───────────┬─────────────┘
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
│    blocked|false_pos)   │        │ screenshot_url text    │
│ source        enum      │        │ raw_response jsonb     │
│   (L1|L2|trustpositif)  │        │ detected_at  timestamptz│
│ confidence    float     │        └────────────────────────┘
│ first_seen    timestamptz│
│ registered_at date      │
│ blocked_at    timestamptz│
└───────────┬─────────────┘
            │ 1
            │ N
┌───────────▼─────────────┐        ┌────────────────────────┐
│ whois_records           │        │ validation_runs        │
├─────────────────────────┤        ├────────────────────────┤
│ id            uuid PK   │        │ id          uuid PK    │
│ domain_id     uuid FK   │        │ run_at      timestamptz│
│ registrar     text      │        │ test_set_size int      │
│ nameservers   text[]    │        │ matched     int        │
│ created_date  date      │        │ missed      int        │
│ raw           jsonb     │        │ notes       text       │
│ fetched_at    timestamptz│       └────────────────────────┘
└─────────────────────────┘
```

**Contract for parallel work — agree at T+0, do not change after T+4:**
- Extension reads **only** `domains` where `status = 'blocked'` → fields: `domain`, `confidence`, `reason` (denormalize `reason` onto `domains` if the join costs time)
- Realtime channel: `postgres_changes` on `domains`, filter `status=eq.blocked`
- Dashboard reads `domains` JOIN `detections` JOIN `fingerprint_clusters`

**Seed the DB with 20 fake rows at T+1** so Frontend isn't blocked waiting for Backend.

---

## 10. API Endpoint Plan

Base: `/api/v1`

| Method | Endpoint | Request | Response | Owner |
|---|---|---|---|---|
| `GET` | `/blocklist` | `?since=<ISO>` | `{domains:[{domain,confidence,reason}], updated_at}` | A |
| `POST` | `/check` | `{domain}` | `{status, confidence, source, reason}` — extension pre-flight | A |
| `POST` | `/analyze` | `{domain, screenshot_b64}` | `{is_judol:bool, confidence, reason, domain_id}` — Layer 2 | B |
| `POST` | `/fingerprint` | `{domain}` | `{cluster_id, registrar, ip, ns, tld, match_score}` | A |
| `GET` | `/domains` | `?limit&offset&source&status` | `{items:[...], total}` — dashboard list | A |
| `GET` | `/domains/{id}` | — | `{domain, detections[], whois, cluster, screenshot_url}` | A |
| `POST` | `/report-false-positive` | `{domain_id, note}` | `{ok:true}` | A |
| `GET` | `/validation/latest` | — | `{test_set_size, matched, missed, rate}` | A |
| `POST` | `/trustpositif/bulk-check` | `{domains:[...≤100]}` | `{results:[{domain, is_blocked}]}` — internal | A |

**Realtime (not REST):** extension subscribes to Supabase channel `domains:status=eq.blocked`. This is the propagation mechanism — **build and test this first, before anything else.**

**Contract-first:** paste this table into the repo README at T+0. Backend returns hardcoded stub responses matching these shapes by **T+2** so Frontend is never blocked.

---

## 11. Suggested Project Structure

```
prejudge/
├── README.md                 ← API contract + .env template + demo script
├── .env.example
├── extension/                ← OWNER: C
│   ├── manifest.json
│   ├── background.js         ← DNR rules + Supabase realtime sub
│   ├── content.js            ← screenshot trigger
│   ├── blocked.html          ← block page (score + reason)
│   ├── blocked.css
│   └── popup/
│       ├── popup.html
│       └── popup.js
├── dashboard/                ← OWNER: C
│   ├── app/
│   │   ├── page.tsx              ← detected list
│   │   ├── domain/[id]/page.tsx  ← detail
│   │   └── validation/page.tsx   ← historical validation
│   ├── components/
│   └── lib/supabase.ts
├── backend/                  ← OWNERS: A (layer1) / B (layer2)
│   ├── main.py               ← FastAPI app + routes
│   ├── layer1/               ← A ONLY
│   │   ├── candidates.py     ← name generator
│   │   ├── trustpositif.py   ← bulk checker
│   │   ├── fingerprint.py    ← WHOIS/DNS extract
│   │   └── matcher.py        ← cluster similarity
│   ├── layer2/               ← B ONLY
│   │   ├── vision.py         ← Gemini client
│   │   └── decide.py         ← threshold logic
│   ├── db/
│   │   ├── schema.sql
│   │   └── seed.sql          ← 20 fake rows, ship at T+1
│   └── requirements.txt
├── scripts/
│   └── validation_run.py     ← held-out test set evaluation
└── pitch/                    ← OWNER: D
    ├── deck.pdf
    ├── demo_script.md
    └── sources.md            ← every stat + its URL
```

**Merge-conflict rules:**
- One owner per top-level directory. Cross-directory edits require a WA message first.
- Branch naming: `a/layer1-fingerprint`, `c/extension-block-page`
- PR to `main`, no direct pushes. Reviewer = anyone awake.
- **`schema.sql` frozen at T+4.** Changes after that need all four to agree.

---

## 12. Roles & Responsibilities (4 members)

| | Role | Owns | Primary deliverable |
|---|---|---|---|
| **A** | Backend / Data Lead | Layer 1, DB schema, all REST endpoints | Fingerprint extraction + cluster matcher + `/blocklist` |
| **B** | Backend / AI | Layer 2, vision integration, validation script | Screenshot→Gemini→verdict→DB writeback + fingerprint feedback loop |
| **C** | Frontend (Extension + Dashboard) | `extension/`, `dashboard/` | Working two-device realtime block + dashboard |
| **D** | Pitch, Research & QA | `pitch/`, source verification, demo rehearsal | 2-min deck + demo script + verified `sources.md` |

**Load-balancing notes:**
- **C carries the heaviest load** (extension + dashboard). At **T+12** (roughly the halfway mark of a 24h build), if the extension isn't blocking end-to-end, **D drops deck work and pairs on the dashboard** — the dashboard is the simpler half.
- **D is not idle.** D owns: source verification (§16 checklist), the demo script, *and* being the person who actually clicks through the demo 5+ times to find where it breaks. D is the only one testing like a judge.
- **D pitches.** D has heard the least code and will therefore explain it in the least jargon — which is literally criterion #1.
- **A owns the schema.** If two people write schema, you get two schemas.

---

## 13. 24-Hour Execution Timeline

Blocks of ~4h. `T+0` = start.

| Block | Window | A (Layer 1) | B (Layer 2) | C (Frontend) | D (Pitch/QA) |
|---|---|---|---|---|---|
| **1** | T+0 → T+2 | Supabase project, `schema.sql`, seed data, **stub all endpoints** | Gemini key working, one screenshot → verdict by hand | `manifest.json`, extension loads, **realtime sub prints to console** | Lock problem statement, start `sources.md` |
| **2** | T+2 → T+5 | TrustPositif bulk-check + candidate generator running | Screenshot capture from extension → `/analyze` roundtrip | **Hardcoded blocklist blocks a real domain** ← must work by T+5 | Deck skeleton, verify every PPATK/Netcraft/PREDATOR figure |
| **3** | T+5 → T+8 | Fingerprint extractor (WHOIS/DNS) + cluster writes | Vision verdict → DB write → fingerprint feedback to L1 | Block page w/ score+reason; dashboard list from real DB | Deck v1 done. **Start demo script.** |
| **4** | T+8 → T+9.5 | Cluster matcher + `/fingerprint` scoring | Threshold tuning, `raw_response` logging | **Two-device realtime propagation working** | 🚩 **CHECKPOINT T+9.5** |
| — | **T+9.5** | **GO/NO-GO.** If two-device block doesn't work, cut everything else and all four fix it. This is the demo. | | | |
| **5** | T+9.5 → T+12 | `/blocklist`, `/domains`, `/validation` real | `validation_run.py` on held-out set | Dashboard detail + validation tab | Rehearse demo #1. Find breakage. |
| **6** | T+12 → T+14 | **Nice-to-haves only if green.** Otherwise: bugfix + record fallback video | Bugfix, record fallback video of Layer 2 | Polish block page, dashboard styling | Deck v2, record **fallback demo video** |
| — | **T+14** | 🛑 **FEATURE FREEZE.** No new features. Bugfix only. | | | |
| **7** | T+14 → T+17 | **SLEEP (A, B)** — 3h, staggered | | Bugfix + polish, then sleep | Rehearse #2, #3 |
| **8** | T+17 → T+21 | Final integration test on the actual demo laptop, both devices, demo wifi | | | Rehearse #4, #5 — full run, timed |
| **9** | T+21 → T+24 | Buffer. Charge everything. Deck exported to PDF. Fallback video on the desktop. Do nothing new. | | | |

**Non-negotiables:**
- **Everyone sleeps ≥3h.** A team that hasn't slept loses on "can a non-expert understand what you built" because they can't speak.
- **Feature freeze at T+14 is real.** Most hackathon losses are a feature merged in the last 2 hours.
- **Demo runs on ONE designated laptop** + one second device. Test on that exact hardware from T+21.

---

## 14. Risks & Fallback Plan

| # | Risk | Likelihood | Fallback |
|---|---|---|---|
| 1 | **TrustPositif rate-limits or blocks us** (no official API; scraping a form) | **High** | Cache every result to JSON from T+2. By T+5 have ≥500 confirmed domains on disk. Demo runs off cache. **Do this first — it's the single biggest technical risk.** |
| 2 | **TrustPositif IP-restricted to Indonesia** | Low (team is in-country) | If demo venue routes through a foreign VPN/proxy: cached JSON. Verify at T+21 on venue wifi. |
| 3 | **Gemini API quota / latency** | Medium | Cache verdicts per domain. Pre-run Layer 2 on 5 demo domains at T+14 and cache. Live call in demo reads cache on failure. |
| 4 | **Supabase realtime doesn't fire** (the demo's core) | Medium | Fallback: extension polls `/blocklist` every 3s. Visually identical to a judge. **Build the poller at T+8 as insurance, not at T+22 in a panic.** |
| 5 | **Venue wifi dies / captive portal** | **High** | Everything local: Supabase local, cached data, fallback video on desktop. Also: one phone hotspot, tested at T+21. |
| 6 | **Chrome blocks the unpacked extension / MV3 quirk** | Medium | Load unpacked in dev mode, pre-loaded and pinned on the demo laptop from T+21. Never install live in front of judges. |
| 7 | **WHOIS returns garbage / redacted (GDPR-ish redaction on many TLDs)** | **High** | Don't depend on registrant fields. Fingerprint on **hosting IP + ASN + nameserver + TLD + registration date** — these survive redaction. Decide this at T+5, not T+14. |
| 8 | **Validation hit rate is embarrassing (e.g. 20%)** | Medium | **Report it honestly.** "70% is PREDATOR's published number on a full dataset; we got X% on 200 domains in 24 hours." Judges reward honest measurement over suspicious perfection. Do not tune the number. |
| 9 | **A judge asks "isn't this just BlockSite?"** | **High** | Rehearsed answer (D): "BlockSite blocks a list you type in. We *generate* the list — and Layer 2 detections feed back into Layer 1's fingerprints, so it gets better at preempting the more people use it. The blocker isn't the product; the pipeline is." |
| 10 | **False positive on a legit site during demo** | Medium | Demo domains are fixed and pre-tested. Also: the "Report as mistake" button *is* the answer — show it. |
| 11 | **Someone breaks `main` at T+22** | Medium | Feature freeze T+14. Tag `demo-ready` at T+21. Demo runs from the tag, not from `main`. |

**Fallback video:** recorded at T+12–14, showing the full two-device block. If everything fails live, D plays the video and says so plainly. Recorded early, while it works — not at T+23 when it doesn't.

---

## 15. Demo Flow & Pitch Scenario

**Target: 2 minutes.** Must-have features only. Two screens visible to judges (laptop + phone/second laptop, both with extension installed).

| Time | Beat | Script (D speaks) |
|---|---|---|
| 0:00–0:20 | **The rebound** | "Indonesia blocked millions of gambling sites in 2025. Players dropped to 3.1 million. Six months later: 12.3 million. Blocking works — until the next domain. That's the whole problem: we're always one domain late." |
| 0:20–0:35 | **What we built** | "PREJUDGE catches the next domain before it matters. Two ways." |
| 0:35–1:00 | **Layer 1 live** *(dashboard on screen)* | "Bandar don't buy one domain. They buy hundreds — same registrar, same hosting, same week. Once TrustPositif confirms one, we fingerprint the whole neighborhood." *→ click a confirmed domain → 47 siblings appear → click a sibling → **blocked, never seen before**.* |
| 1:00–1:40 | **Layer 2 + propagation** ⭐ **the moment** | "But new operators exist. Watch." *→ Device 1 opens unknown judol site, it loads for a second → screenshot → verdict appears: "gambling — slot UI, deposit button" → Device 1 blocks.* "Now — this phone has never visited that site." *→ hold up Device 2 → open same domain → **instantly blocked**.* "One person's exposure protected everyone. And that site's infrastructure just fed back into Layer 1 — so its siblings are already flagged." |
| 1:40–1:50 | **Honesty beat** *(validation tab)* | "We proved this on historical data — 200 confirmed domains held out. We matched X%. That's not the whole internet. It's the method, measured." |
| 1:50–2:00 | **Close** | "Free for families forever — they're the sensor network. The feed is what ISPs and banks pay for. And it complements Komdigi — they still decide what gets officially blocked. We just get them there faster." |

**Rehearsal rules:**
- Demo domains **fixed and pre-tested**. Never type a URL you haven't run 5 times.
- If something hangs >3s: keep talking, cut to the fallback video, say "let me show the recorded run" without apology.
- **Never say:** "we monitor Indonesian internet in real time," "we blocked X million domains," "we work with Kominfo," or any unverified number.
- The Q&A killer is #9 above. D must have it word-perfect.

---

## 16. Post-Hackathon Roadmap

**Phase 1 — Harden (0–3 months)**
- Android app with `VpnService` (system-wide blocking — the actual use case, since judol is mobile-first)
- False-positive review queue + community reporting
- Proper cron infra + WHOIS/RDAP rate-limit handling
- Chrome Web Store submission

**Phase 2 — Legitimize (3–6 months)**
- Formal engagement with Komdigi: propose the report pipeline as a verified complaint channel
- Publish validation methodology + hit rates openly — credibility is the moat
- Partner with a `.id` registrar for at-registration risk flagging (closest to true PREDATOR-style preemption)
- Independent replication of PREDATOR-style features on Indonesian judol data — *this is a publishable paper*

**Phase 3 — Monetize (6–12 months)**
- Domain feed API GA: ISP tier (per-subscriber), bank/e-wallet tier (fraud signal)
- Pilot with one ISP under "Internet Sehat"
- B2C stays free, permanently

**Public release strategy**
- Open-source the extension. Keep the feed as the commercial asset. (Open blocker builds trust; the pipeline is the business.)
- Distribution: partner with Komdigi's existing anti-judol campaign, family/parenting communities, and university student orgs — not paid ads.

**Do not build:** anything touching bank accounts, anything auto-submitting to CAPTCHA-protected government forms, anything claiming authority to block officially.

---

### ⚠️ Pre-submission verification checklist (D owns)

- [ ] Every PPATK figure traced to a primary source URL in `sources.md`
- [ ] Netcraft launch/award dates re-confirmed — put the link in `sources.md` anyway, judges may ask
- [ ] PREDATOR citation exact: Hao, Kantchelian, Miller, Paxson, Feamster — ACM CCS 2016 (**not** University of Houston)
- [ ] NXDomain side-channel (99%), DGA reverse-engineering (99.93% AUC), WhoisXML "300k/day" — **cut from deck unless independently verified**
- [ ] No claim of existing partnership with Kominfo/Komdigi/OJK/PPATK anywhere in deck or demo
- [ ] Validation numbers on the dashboard are **real output**, not the placeholder figures in §7
- [ ] Mock affiliate graph (if built) labeled "SIMULATED DATA" on screen
