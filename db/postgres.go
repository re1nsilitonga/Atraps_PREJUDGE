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
