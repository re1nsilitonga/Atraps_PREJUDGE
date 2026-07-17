# API Docs

Reference for every route the Go API (`api/`) exposes under `/api/v1`. See [PRD.md](PRD.md) §10 for the design rationale and [README.md](../README.md) for the module boundary table.

**Base URL (local):** `http://localhost:8000/api/v1`
**Content-Type:** `application/json` for every request/response body except `GET /realtime`, which is a WebSocket upgrade.
**Auth:** none in the MVP. Every route is open. Do not treat this as production-ready without adding a key/token layer in front of it (see API_DEPLOY_DOCS.md, "Before exposing this publicly").
**CORS:** wildcard (`Access-Control-Allow-Origin: *`), preflight `OPTIONS` handled for every route.

Module ownership, per README.md:
- **Core:** `/analyze`, `/fingerprint`
- **Blocker:** `/blocklist`, `/check`, `/realtime`
- **Presentation:** `/domains`, `/domains/{id}`, `/bootstrap/latest`

---

## GET /blocklist

Full or incremental read of every domain at `status='blocked'`. This is what `blocker/lib/blocklist.js` calls on service-worker boot (full) and what the polling adapter calls on an interval (incremental, via `since`).

**Query params**
| Name | Type | Required | Notes |
|---|---|---|---|
| `since` | RFC3339 timestamp | no | Only rows with `blocked_at` after this value. Unparseable or absent means everything. |

**Response `200`**
```json
{
  "domains": [
    {
      "id": "c7e5c074-69aa-435d-aafe-e741819fdfa0",
      "domain": "gacor88x.xyz",
      "confidence": 0.92,
      "reason": "Menampilkan elemen khas judi online seperti jackpot slot dan tombol daftar.",
      "matched_fields": ["hosting_ip", "nameserver"]
    }
  ],
  "updated_at": "2026-07-17T14:20:18Z"
}
```

`matched_fields` is empty (`[]`) for a Layer 2 verdict and populated for a Layer 1 match (see `docs/PRD.md` §5 for the field-to-Indonesian-label mapping used on the block page).

---

## POST /check

One-shot status lookup for a single domain. Used by the Blocker for a synchronous check outside the cached blocklist path.

**Request**
```json
{ "domain": "gacor88x.xyz" }
```

**Response `200`**
```json
{
  "status": "blocked",
  "confidence": 0.92,
  "source": "L2",
  "reason": "Menampilkan elemen khas judi online..."
}
```

A domain the system has never seen is not an error. It reports `status: "candidate"` with `confidence`, `source`, and `reason` all `null`, same shape as a domain that was analyzed but never crossed the threshold.

---

## POST /analyze

Sends page evidence to Layer 2 (Gemini vision) and returns a verdict. This is the only endpoint that calls an external AI service. Called automatically by `blocker/evidence.js` on a completed navigation to a domain not already in the local blocklist.

**Request**
```json
{
  "domain": "gacor88x.xyz",
  "evidence_b64": "<base64 JPEG>",
  "evidence_type": "screenshot"
}
```

`evidence_type` is accepted but not yet branched on server-side; every call is currently treated as a screenshot (`core.EvidenceScreenshot`). Sent anyway so the field exists for the day a non-visual evidence type is added.

**Response `200`**
```json
{
  "is_judol": true,
  "confidence": 0.95,
  "reason": "Situs menampilkan elemen khas judi online...",
  "domain_id": "c7e5c074-69aa-435d-aafe-e741819fdfa0"
}
```

`domain_id` is always populated, even when `is_judol` is `false` or Gemini fails, because the row is created synchronously (`EnsureDomain`) before the vision call. If `GEMINI_API_KEY` is unset, the response degrades to a stub (`is_judol: false, reason: "stub"`) instead of failing.

The verdict decision (`layer2.Decide`) and the Layer 1 feedback loop (`core.Feedback` and `core.MatchSiblings`) run in a background goroutine after this response is sent. A `200` here does not mean the blocklist has been updated yet; poll `/blocklist` or connect to `/realtime` to observe that.

---

## POST /fingerprint

Scores one domain's infrastructure fingerprint (hosting IP, nameserver, registrar, TLD, registration burst) against the current cluster set. Read-only: it reports a match score, it does not write `status='blocked'` on its own. The actual auto-block path is `core.MatchSiblings`, triggered from `/analyze`'s background flow, not from this endpoint.

**Request**
```json
{ "domain": "slotjp7.top" }
```

**Response `200`**
```json
{
  "cluster_id": "b2a1...",
  "registrar": "Namecheap",
  "ip": "203.0.113.10",
  "ns": "ns1.example-hosting.com",
  "tld": "top",
  "match_score": 0.65,
  "matched_fields": ["hosting_ip", "nameserver", "tld"]
}
```

No match (or an extraction failure, e.g. an unreachable domain) returns `match_score: 0, matched_fields: []`, not an error.

---

## GET /domains

Paginated domain list for the dashboard.

**Query params:** `limit`, `offset`, `source` (`L1`/`L2`/`trustpositif`), `status` (`candidate`/`confirmed`/`blocked`/`false_pos`). All optional; omitted filters mean no filter.

**Response `200`**
```json
{
  "items": [
    { "id": "...", "domain": "gacor88x.xyz", "status": "blocked", "source": "L2", "confidence": 0.92, "detected_at": "2026-07-17T14:20:18Z" }
  ],
  "total": 1
}
```

---

## GET /domains/{id}

Full detail view for one domain: its detection history, WHOIS record, cluster, and sibling domains.

**Response `200`**
```json
{
  "domain": "gacor88x.xyz",
  "detections": [{ "layer": 2, "confidence": 0.92, "reason": "...", "evidence_url": null, "detected_at": "..." }],
  "whois": { "registrar": "Namecheap" },
  "cluster": { "hosting_ip": "203.0.113.10", "domain_count": 6 },
  "siblings": ["slotjp7.top", "maxwin4d.cc"],
  "evidence_url": null
}
```

---

## POST /report-false-positive

Backs the block page's "Laporkan salah" button. No auth in the MVP; an invalid or unknown `domain_id` still answers `ok: true` (a stuck-looking error here is worse than a silent no-op).

**Request**
```json
{ "domain_id": "c7e5c074-69aa-435d-aafe-e741819fdfa0", "note": "Dilaporkan dari block page" }
```

**Response `200`**
```json
{ "ok": true }
```

Sets `status='false_pos'` server-side. The Blocker also unblocks the domain client-side immediately (`chrome.runtime.sendMessage`, see `blocker/blocked.js`) rather than waiting for the next sync cycle.

---

## GET /bootstrap/latest

The cold-start proof counter: how many Layer 2 confirmations produced how many Layer 1 preemptive catches, from an empty database.

**Response `200`**
```json
{ "l2_confirmations": 12, "l1_preemptive_catches": 34, "l1_misses": 3, "ratio": 2.83 }
```

---

## POST /trustpositif/verify

Permanent stub. `trustpositif.komdigi.go.id` requires solving a reCAPTCHA client-side, which this project will not automate (see `docs/PRD.md` §6). This endpoint exists only so callers that expect it in the contract do not need special-casing.

**Request**
```json
{ "domain": "gacor88x.xyz" }
```

**Response `200`**
```json
{ "domain": "gacor88x.xyz", "is_blocked": false }
```

`is_blocked` is always `false`. Never present this as TrustPositif corroboration in a pitch or dashboard.

---

## GET /realtime

WebSocket upgrade, not a REST route. This is the Blocker's single realtime transport (see the "single-source access" note in README.md's Environment Variables section). No request body; the server pushes a JSON text message for every domain that flips to `status='blocked'`.

**Message shape (server to client)**
```json
{
  "id": "c7e5c074-69aa-435d-aafe-e741819fdfa0",
  "domain": "gacor88x.xyz",
  "confidence": 0.92,
  "reason": "Menampilkan elemen khas judi online...",
  "matched_fields": []
}
```

Same shape as one entry in `GET /blocklist`'s `domains` array, so `blocker/lib/blocklist.js`'s `normalize()` handles both without a branch.

The server sends a ping every 25 seconds to keep the connection alive behind an idle-timeout proxy (Cloudflare Tunnel's default is around 100 seconds). The client does not need to respond; the browser's WebSocket implementation answers pings at the protocol level automatically. On disconnect, `blocker/lib/realtime.js` reconnects on a fixed 2-second backoff.

This endpoint requires `DATABASE_DIRECT_URL` to be configured server-side (see API_DEPLOY_DOCS.md). Without it, the server still accepts connections but never pushes anything, since there is nothing listening for `pg_notify` on the database side.
