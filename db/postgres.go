// Direct Postgres access (not the Supabase REST API) — needs DATABASE_URL,
// the connection string from Supabase → Settings → Database → Connection string.
// This is the project's first non-stdlib dependency (team-agreed).
package db

import (
	"context"
	"errors"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DSNFromEnv reads DATABASE_URL and fails clearly if it's unset, rather than
// letting pgx return an opaque parse error against an empty string.
func DSNFromEnv() (string, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return "", errors.New("DATABASE_URL is not set (see .env.example)")
	}
	return dsn, nil
}

func Connect(ctx context.Context) (*pgxpool.Pool, error) {
	dsn, err := DSNFromEnv()
	if err != nil {
		return nil, err
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	// Supabase's connection pooler (pgbouncer, transaction mode) doesn't
	// support session-level prepared statements shared across backend
	// connections — disable pgx's statement cache and use the simple
	// protocol instead.
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	return pgxpool.NewWithConfig(ctx, cfg)
}

// DirectDSNFromEnv reads DATABASE_DIRECT_URL — Supabase's non-pooled
// connection string (port 5432), distinct from DATABASE_URL's pgbouncer
// pooler. LISTEN/NOTIFY needs a single session-scoped connection; pgbouncer
// in transaction mode recycles the underlying server connection between
// statements, which silently drops a LISTEN registered on it.
func DirectDSNFromEnv() (string, error) {
	dsn := os.Getenv("DATABASE_DIRECT_URL")
	if dsn == "" {
		return "", errors.New("DATABASE_DIRECT_URL is not set (see .env.example) — required for realtime LISTEN/NOTIFY")
	}
	return dsn, nil
}

// ListenDomainBlocked opens a dedicated connection, LISTENs on
// domain_blocked (see the trigger in schema.sql), and calls onNotify with
// each payload as it arrives. Blocks until ctx is cancelled or the
// connection breaks — callers reconnect-loop this (api/realtime.go does).
func ListenDomainBlocked(ctx context.Context, onNotify func(payload string)) error {
	dsn, err := DirectDSNFromEnv()
	if err != nil {
		return err
	}
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx, "LISTEN domain_blocked"); err != nil {
		return err
	}

	for {
		n, err := conn.WaitForNotification(ctx)
		if err != nil {
			return err
		}
		onNotify(n.Payload)
	}
}
