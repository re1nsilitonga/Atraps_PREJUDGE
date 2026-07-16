# PREJUDGE MVP — Epics / Main Task Groups

Derived from PRD.md §6 (Must-have), §8 (Tech Stack), §9 (Schema), §10 (API Plan), §14 (Risks). Nice-to-haves and pitch content (deck, script, sources) excluded — this is build scope. Demo reliability *is* build scope (Epic 7).

---

## Epic 1: Data Foundation (Database + Contracts)
Everything else blocks on this existing first.
- Stand up Supabase project, implement `schema.sql` (domains, fingerprint_clusters, detections, whois_records, validation_runs)
- Seed 20 fake rows so Frontend can build against real shapes immediately
- Freeze schema (per PRD: no changes after early build phase)
- Publish API contract (endpoint shapes) in README before any real logic is written

**Maps to:** §9 (ERD), §10 (contract-first note)
**Owner:** A

---

## Epic 2: Layer 1 — Preemptive Detection Pipeline
- Candidate domain name generator (keyword + TLD combos, bandar naming-pattern variants)
- TrustPositif bulk-checker (≤100 domains/request), with local JSON caching (biggest single technical risk per §14)
- Fingerprint extractor: registrar, hosting IP/ASN, nameserver, TLD, registration date via WHOIS/DNS/RDAP
- Cluster matcher: score a new domain against known fingerprint clusters

**Maps to:** Must-haves #1–3 · Endpoints `/fingerprint`, `/trustpositif/bulk-check`
**Owner:** A

---

## Epic 3: Layer 2 — Reactive Content Detection
- Screenshot capture on unknown-domain visit (extension-triggered)
- Vision API integration (Gemini) with yes/no + reason + confidence prompt
- Decision logic + threshold tuning
- Feedback loop: confirmed Layer 2 hit → extract its fingerprint → write into Layer 1's cluster data

**Maps to:** Must-have #4 · Endpoint `/analyze`
**Owner:** B

---

## Epic 4: Blocking Extension
- Manifest V3 scaffold + `declarativeNetRequest` rule engine
- Supabase realtime subscription (`domains: status=eq.blocked`) — this is the core demo mechanism, build/test first
- Block page UI: confidence score + plain-language reason
- Polling fallback in case realtime doesn't fire (insurance, not afterthought)

**Maps to:** Must-have #6 · Endpoints `/blocklist`, `/check`
**Owner:** C

---

## Epic 5: Dashboard
- Detected-domains list view (confidence, source, timestamp)
- Domain detail view (fingerprint match, screenshot, detections)
- Historical validation view (test-set hit rate)

**Maps to:** Must-have #7 · Endpoints `/domains`, `/domains/{id}`, `/validation/latest`
**Owner:** C

---

## Epic 6: Validation & Ground Truth
- Held-out test set from TrustPositif-confirmed domains
- Script to run Layer 1 matching against the held-out set and report hit/miss rate honestly
- This is what backs the "we proved the method on historical data" claim — not optional, it's the credibility mechanism for the whole pitch

**Maps to:** §6 validation methodology · Endpoint `/validation/latest`
**Owner:** A/B shared

---

## Epic 7: Demo Reliability & Fallbacks
The demo working on the actual hardware, on the actual network, from a known-good commit. Fails silently if unowned.
- **Fallback video** — record the full two-device block at T+12–14, while it works. Not at T+23 when it doesn't.
- **Cached-data demo path** — verify the demo runs end-to-end with TrustPositif cache + pre-cached Gemini verdicts, no live API calls required (§14 risks #1, #3)
- **Offline/degraded rehearsal** — confirm the polling fallback (Epic 4) is visually indistinguishable from realtime (§14 risk #4)
- **Demo hardware lock** — designated laptop + second device, extension pre-loaded unpacked and pinned, tested from T+21 (§14 risk #6)
- **Venue network test** — run on actual venue wifi; phone hotspot as backup (§14 risks #2, #5)
- **`demo-ready` tag** — cut at T+21; demo runs from the tag, never from `main` (§14 risk #11)
- **Timed rehearsals** — 5+ full runs, fixed pre-tested demo domains, find breakage before judges do

**Maps to:** §14 (Risks & Fallback Plan), §13 (T+21 integration test, feature freeze)
**Owner:** D — with the caveat below

> **Surge clause (§12):** at the T+9.5 GO/NO-GO, if Epic 4 isn't blocking end-to-end, D drops deck work and pairs on Epic 5 (dashboard — the simpler half). Epic 7 then compresses into the T+17–21 window.

---

## Dependency Order (Critical Path)

```
Epic 1 (schema + stubs)
   ↓
Epic 2 (Layer 1) ──┐
Epic 3 (Layer 2) ──┼──→ Epic 4 (extension realtime block) ← THE DEMO MOMENT
                    │         ↓
                    └──→ Epic 5 (dashboard)
                              ↓
                        Epic 6 (validation)
                              ↓
                        Epic 7 (demo reliability) ← gates submission
```

**Note:** Epic 4's realtime propagation is the single feature the whole pitch hinges on (§15's "the moment"). If sequencing gets tight, everything else should yield to getting Epic 1 → Epic 4 working end-to-end first, even with stubbed/fake data from Epics 2–3.

**Note on Epic 7:** it depends on Epic 4 existing, but must not wait for Epics 5–6 to finish. The fallback video is recorded at T+12–14 — *before* feature freeze — because it needs to capture the system while it demonstrably works. Treating Epic 7 as "the last thing" is how it doesn't happen.
