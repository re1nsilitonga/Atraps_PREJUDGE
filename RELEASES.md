# Releases

Format follows [Keep a Changelog](https://keepachangelog.com/) and [Semantic Versioning](https://semver.org/).

## [v1.0.0] — 2026-07-17

First release. PRIME is a two-layer gambling-domain detection system: a Go API, a Postgres/Supabase backend, and a Chrome extension, working together to block gambling domains before they're ever visited.

### Added
- **Core detection engine** — Layer 2 vision bootstrap (screenshot → Gemini → verdict) confirms new domains, and Layer 1 fingerprint matching (hosting IP, ASN, nameserver, registrar, TLD, registration-burst scoring) preemptively blocks unvisited domains that share infrastructure with a confirmed one
- **Feedback loop (L2 → L1)** — every confirmed detection automatically seeds or updates a fingerprint cluster and auto-blocks matching sibling domains, so the system gets more preemptive the more it's used
- **Full API surface** (`/api/v1`) — `/blocklist`, `/check`, `/analyze`, `/fingerprint`, `/domains`, `/domains/{id}`, `/report-false-positive`, `/bootstrap/latest`, `/realtime` (WebSocket)
- **Realtime propagation** — a Postgres `LISTEN/NOTIFY`-backed WebSocket relay pushes every new block to all connected devices within seconds
- **Chrome extension (Manifest V3)** — `declarativeNetRequest` blocking, a realtime adapter with polling fallback, in-page Google search result blocking, and an explainable block page that shows the exact `matched_fields` behind each verdict
- **Cold-start transparency** — `GET /bootstrap/latest` reports live detection-to-preemptive-catch ratios
- **Gemini cache fallback** — verdicts degrade gracefully to a cached response if a live vision call fails, instead of breaking the pipeline
- Single idempotent database schema (`db/schema.sql`) with no migration tool required
- Full documentation set: `README.md`, `docs/PRD.md`, `docs/API_DOCS.md`, `docs/API_DEPLOY_DOCS.md`

### Changed
- Backend rebuilt in Go (`net/http`, stdlib-first) for a small, portable core that can extend to platforms beyond the browser
- All Supabase access consolidated behind the Go API — the extension never touches the database directly, keeping credentials out of client code entirely

### Fixed
- WebSocket handshake hardened against cross-site WebSocket hijacking (CSWSH) while still allowing the extension to connect
- Startup environment loading, so `DATABASE_URL`/`DATABASE_DIRECT_URL` are always picked up
- Immediate tab redirect on a positive Layer 2 verdict
- Extension rule-ID handling in `declarativeNetRequest`
- Search-result blocking scoped to flagged results only, not the whole page
- Layer 1 hosting-IP matching corrected to compare exact addresses instead of a leaked subnet mask
