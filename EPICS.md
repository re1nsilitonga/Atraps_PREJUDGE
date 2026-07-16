# PREJUDGE MVP — Epics / Main Task Groups

Derived from PRD.md §4 (Architecture), §6 (Must-have), §8 (Tech Stack), §9 (Schema), §10 (API Plan), §14 (Risks). Nice-to-haves and pitch content (deck, script, sources) excluded — this is build scope. Demo reliability _is_ build scope (Epic 8).

**Two architectural facts drive this breakdown:**

1. **The system starts with an empty database.** There is no bulk-imported ground truth. Layer 2 bootstraps Layer 1. Epic ordering reflects this — Layer 2 is not the fallback, it is the _seed_.
2. **Core Engine / Blocker Service / Presentation Layer are separable modules.** The Chrome extension is one delivery surface; Android VpnService is next. Epics are drawn along that seam, not along "frontend vs backend."

---

## Epic 1: Foundation — Schema + Core Contract

Everything blocks on this. Contains the seam that makes the Android roadmap real.

- Supabase project, credential distribution
- `schema.sql`: domains, fingerprint_clusters (incl. registration-burst fields), detections, whois_records, bootstrap_runs
- **`core/contract.py` — the verdict contract.** Single emit point: `{domain, is_judol, confidence, reason, matched_fields}`. This is the Core/Blocker seam.
- Fixture data (clearly marked, purgeable) so Presentation can build at T+1 against an intentionally-empty production DB
- API contract published before any logic
- Schema frozen T+4, contract frozen T+2

**Maps to:** §4 (architecture table), §9 (ERD), §10 (contract-first)
**Owner:** A

---

## Epic 2: Core — Layer 2 Bootstrap (Vision)

**This is the seed, not the fallback.** With an empty DB, every first access lands here. If Layer 2 doesn't work, the system has no data source at all.

- Gemini vision client: evidence → `{is_judol, confidence, reason}`
- Evidence capture path (Chrome-specific, isolated in `blocker/`)
- Verdict decision logic + threshold
- Verdict → DB write (`status='blocked'`, `source='L2'`)

**Maps to:** Must-have #2 · Endpoint `/analyze`
**Owner:** B

---

## Epic 3: Core — The Feedback Loop (L2 → L1)

**The single most important epic in the project.** Without it there is no Layer 1 at all — not a weaker Layer 1, _none_, because there is no other data source. It is also the entire Innovation & Novelty defense.

- L2 confirmation → fingerprint extraction → cluster seed/update
- Sibling domains become Layer 1 candidates without ever being visited
- Async — must not block the verdict response

**Maps to:** Must-have #4 · §5 (Innovation defense) · §14 risk #10
**Owner:** B

---

## Epic 4: Core — Layer 1 Preemptive Detection

Grows from Epic 3's output. Deterministic arithmetic — **no ML, deliberately** (§4: explainability is what the block page depends on).

- Fingerprint extractor: hosting IP/ASN, nameserver, registrar, TLD, `registered_at` (WHOIS/DNS/RDAP)
- **WHOIS fill-rate report at T+5** ← hard gate
- Cluster builder (GROUP BY, not clustering ML)
- **Registration-burst detection** — domains registered in the same narrow window by the same registrar. _This is what makes Layer 1 preemptive rather than merely correlative._
- Matcher: score unseen domain vs clusters → emit `matched_fields`

**Maps to:** Must-haves #3, #5, #6 · Endpoint `/fingerprint`
**Owner:** A

---

## Epic 5: Blocker Service (Chrome Adapter)

Enforcement + verdict transport. **One adapter, not the architecture** — realtime and polling are two transports behind one Core contract; VpnService will be a third.

- MV3 scaffold + `declarativeNetRequest` rule engine
- Realtime adapter (Supabase subscription) — build first, T+2 proof-of-life
- **Polling adapter (flag-swappable)** — build T+8, not T+22
- Blocklist sync, cache surviving service-worker restart

**Maps to:** Must-have #8 · Endpoints `/blocklist`, `/check` · §14 risks #5, #16
**Owner:** C

---

## Epic 6: Presentation Layer

- Block page: confidence + `matched_fields` rendered as plain-language Indonesian bullets
- Dashboard: detected list, domain detail + cluster siblings
- **Cold Start tab**: N confirmations → M preemptive catches (live counters)

**Maps to:** Must-haves #9, #10 · Endpoints `/domains`, `/domains/{id}`, `/bootstrap/latest`
**Owner:** C

---

## Epic 7: Cold-Start Proof

Replaces the dead held-out-validation plan (bulk ground truth is unavailable — TrustPositif results are masked).

- Bootstrap script: empty DB → N Layer 2 confirmations → M Layer 1 preemptive catches
- **Leakage assertion**: a Layer 2-confirmed domain can never count as a Layer 1 catch
- TrustPositif per-domain verifier (full string in → boolean out). **Verifier, not seed source.**
- Write results to `bootstrap_runs`

**Maps to:** §6 (validation methodology) · Must-have #11 · Endpoints `/bootstrap/latest`, `/trustpositif/verify`
**Owner:** A/B shared

---

## Epic 8: Demo Reliability & Fallbacks

The demo working on the actual hardware, on the actual network, from a known-good commit. Fails silently if unowned.

- **Demo cluster bootstrap** — deliberately run Layer 2 over same-network judol domains until ≥1 cluster has ≥5 siblings (§14 risk #1: the top risk under cold start)
- Fallback video recorded T+12–14, while it works
- Cached-data demo path verified (Gemini + TrustPositif both unreachable)
- Polling adapter verified indistinguishable from realtime
- **Fixture purge** — the opening beat is an empty database
- **Layer 2 demo domain verified absent from blocklist** — a rehearsal-cached verdict silently converts the bootstrap beat into a Layer 1 beat
- Demo hardware lock, venue network test, `demo-ready` tag, timed rehearsals

**Maps to:** §14 (Risks), §13 (T+21 integration), §15 (demo script)
**Owner:** D — with the caveat below

> **Surge clause (§12):** at the T+9.5 GO/NO-GO, if Epic 5 isn't blocking end-to-end, D drops deck work and pairs on Epic 6 (the simpler half). Epic 8 then compresses into T+17–21.

---

## Dependency Order (Critical Path)

```
Epic 1 (schema + core contract)
   ↓
Epic 2 (Layer 2 bootstrap) ← THE SEED. Empty DB means everything starts here.
   ↓
Epic 3 (feedback loop) ← WITHOUT THIS, EPIC 4 HAS NO INPUT
   ↓
Epic 4 (Layer 1 preemptive)
   ↓
Epic 5 (blocker adapter) ← THE DEMO MOMENT
   ↓
Epic 6 (presentation)
   ↓
Epic 7 (cold-start proof)
   ↓
Epic 8 (demo reliability) ← gates submission
```

**This ordering inverted under cold start.** Layer 1 (Epic 4) used to be seeded independently from TrustPositif and could be built in parallel with Layer 2. It can't anymore: with masking, there is no bulk import, so **Layer 4 depends on Layer 3 depends on Layer 2**. The chain is now serial where it used to be parallel. That is the single biggest scheduling consequence of the cold-start decision — and it's why Epic 3 is the most important epic in the project rather than a nice architectural touch.

**Epic 5 is still the demo moment** (§15's two-device block). If sequencing gets tight, get Epic 1 → 2 → 5 working end-to-end first, even with a stubbed Layer 1.

**Epic 8 is not last chronologically.** The fallback video records at T+12–14, before feature freeze, because it must capture the system while it demonstrably works. Treating Epic 8 as "the last thing" is how it doesn't happen.

---

## Two hard gates

| Gate                  | When | Test                                                | If it fails                                                                                         |
| --------------------- | ---- | --------------------------------------------------- | --------------------------------------------------------------------------------------------------- |
| **Fill rate**         | T+5  | Is `registered_at` populated often enough in WHOIS? | Burst detection dies → **cut "preemptive" claims from the deck now**, fall back to correlation-only |
| **Cluster formation** | T+14 | Does ≥1 cluster have ≥5 siblings?                   | **Cut the Layer 1 beat from the demo** → demo becomes Layer 2 + propagation only                    |

Both can force cutting a pitch claim. Both are better discovered on schedule than on stage.

---

## Scope Guard

Not epics. Do not create them. (§6)

- **Bulk import of TrustPositif** — no bulk export, results masked (`a*****gacor.biz`), cannot be WHOIS'd or fingerprinted
- Masked-pattern-constrained candidate generation — nice-to-have, not a dependency
- WA Chatbot report packaging — nice-to-have
- Affiliate network graph — nice-to-have
- Android VpnService app — roadmap (but the Core seam is built for it now)
- Live monitoring of all new registrations — needs enterprise API
- aduankonten.id auto-submit — CAPTCHA, do not bypass
- Bank account pre-blocking — not how PPATK works, legal risk
- Any self-trained vision model — explicitly not the innovation
- **ML in Layer 1** — deliberate; deterministic scoring is what produces explainable `matched_fields`
