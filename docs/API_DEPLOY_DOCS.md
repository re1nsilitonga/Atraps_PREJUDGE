# API Deploy Docs

How to run the Go API (`api/`) somewhere other than a developer's laptop, and what `.env` needs to contain. For the route reference itself, see API_DOCS.md. For the database schema, see `db/schema.sql`.

## 1. Prerequisites

- Go 1.25 or newer (`go.mod` pins the module to `go 1.25.0`).
- A Supabase project (or any Postgres instance, but the schema and the pooler-vs-direct split below assume Supabase).
- A Gemini API key (Layer 2 vision calls).

## 2. Set up `.env`

Copy the template and fill it in:

```bash
cp .env.example .env
```

| Variable | Where it comes from | Required for |
|---|---|---|
| `DATABASE_URL` | Supabase project, Settings, Database, Connection string (pooler, transaction mode) | Every normal query (`db.Connect`) |
| `DATABASE_DIRECT_URL` | Same page, session-scoped: either the pooler host on port 5432, or `db.<ref>.supabase.co:5432` if your network has IPv6 egress | `GET /realtime` (Postgres `LISTEN/NOTIFY`, see below) |
| `GEMINI_API_KEY` | Google AI Studio | `POST /analyze` |
| `GEMINI_MODEL` | optional, defaults to `gemini-2.0-flash` | overriding the vision model |
| `GEMINI_CACHE_DIR` | optional, a writable directory | fallback verdicts if a live Gemini call fails mid-demo |
| `SUPABASE_URL`, `SUPABASE_ANON_KEY`, `SUPABASE_SERVICE_ROLE_KEY` | Supabase project settings | not read by the Go API directly today, kept for parity with the Supabase dashboard and any tooling that expects them |

`.env` is gitignored. Never commit real values; the team has historically distributed them over a private channel (see `docs/PRD.md` §8).

### Why two database URLs

Supabase's pooler runs in transaction mode by default. Transaction-mode pgbouncer recycles the underlying server connection between statements, which silently drops a session-scoped `LISTEN`. `DATABASE_DIRECT_URL` is a connection that keeps one dedicated session alive for exactly that purpose. If you only set `DATABASE_URL`, the API still starts and every route still works except realtime push: the server logs `realtime listener disabled: DATABASE_DIRECT_URL is not set` once and moves on, it does not crash.

## 3. Apply the schema

There is no migration tool in this repo (see `docs/TASKS.md` for the reasoning: one idempotent `schema.sql`, re-run by hand). If `psql` is available:

```bash
psql "$DATABASE_URL" -f db/schema.sql
psql "$DATABASE_URL" -f db/fixtures.sql   # optional, demo data
```

If `psql` is not available in your environment, run it through `pgx` instead. A minimal one-off:

```go
package main

import (
    "context"
    "os"
    "prime/db"
)

func main() {
    pool, err := db.Connect(context.Background())
    if err != nil { panic(err) }
    defer pool.Close()
    sql, _ := os.ReadFile(os.Args[1])
    if _, err := pool.Exec(context.Background(), string(sql)); err != nil { panic(err) }
}
```

`schema.sql` is safe to re-run against a database that already has it applied; every `CREATE` is guarded (`IF NOT EXISTS`, `CREATE OR REPLACE`, or a `DO $$ ... EXCEPTION WHEN duplicate_object` block).

## 4. Build and run

```bash
go build -o prime-api ./api
./prime-api
```

Or without a separate build step:

```bash
go run ./api
```

The server listens on `:8000` (hardcoded, see `api/main.go`). There is no flag or env var to change the port today.

Verify it came up correctly:

```bash
curl http://localhost:8000/api/v1/blocklist
```

An empty `domains` array is the expected state on a fresh database. Check the process's stdout for two specific log lines that indicate a degraded (not failed) start:

- `db connect failed, /fingerprint and domain endpoints will report empty state: ...` means `DATABASE_URL` is missing or unreachable. The server still serves stub responses instead of crashing.
- `realtime listener disabled: DATABASE_DIRECT_URL is not set` means `GET /realtime` will accept connections but never push anything.

## 5. Exposing it beyond localhost

Nothing in this repo deploys the binary anywhere; that part is manual. Two things worth knowing if you do:

**WebSocket behind a tunnel.** `GET /realtime` works fine behind Cloudflare Tunnel or a similar reverse proxy: WebSocket upgrade is proxied transparently at the HTTP layer, no special config needed beyond pointing the tunnel at `http://localhost:8000`. Client should then connect to `wss://` on the tunnel's hostname. The server's 25-second heartbeat (see API_DOCS.md's `GET /realtime` section) exists specifically to survive this kind of proxy's idle timeout.

```bash
cloudflared tunnel --url http://localhost:8000
```

**Before exposing this publicly.** Every route is unauthenticated. That is an acceptable MVP tradeoff for a demo running on a laptop, not for anything reachable from the open internet. At minimum, put an API key or token check in front of every route before deploying somewhere with a public hostname. The extension's own Supabase credentials were removed from the client entirely for a related reason (see the "single-source access" note in README.md): this API is meant to be the one place that holds real credentials, so it is also the one place that needs to gate access if this ever becomes a service used outside the team.

## 6. Running the Go test suite before deploying

```bash
go build ./...
go vet ./...
go test ./...
```

None of the tests require a live database or a real Gemini key. Database-backed logic is tested against a mocked pool or, for the schema itself, against the raw SQL text (`db/schema_test.go` checks for specific `CREATE TABLE`/`CREATE TRIGGER` statements rather than connecting to Postgres). A green test run does not confirm the schema was actually applied to your target database; do that verification separately, for example with the `curl` check in step 4.
