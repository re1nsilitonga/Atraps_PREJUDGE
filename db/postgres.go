package db

import (
	"context"
	"errors"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	return pgxpool.NewWithConfig(ctx, cfg)
}

func DirectDSNFromEnv() (string, error) {
	dsn := os.Getenv("DATABASE_DIRECT_URL")
	if dsn == "" {
		return "", errors.New("DATABASE_DIRECT_URL is not set (see .env.example) — required for realtime LISTEN/NOTIFY")
	}
	return dsn, nil
}

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
