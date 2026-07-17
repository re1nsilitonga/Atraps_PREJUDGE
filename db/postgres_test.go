package db

import "testing"

func TestDSNFromEnvMissing(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	_, err := DSNFromEnv()
	if err == nil {
		t.Fatal("expected error when DATABASE_URL is unset")
	}
}

func TestDSNFromEnvPresent(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@host:5432/db")
	dsn, err := DSNFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dsn != "postgres://user:pass@host:5432/db" {
		t.Fatalf("unexpected dsn: %s", dsn)
	}
}

func TestDirectDSNFromEnvMissing(t *testing.T) {
	t.Setenv("DATABASE_DIRECT_URL", "")
	_, err := DirectDSNFromEnv()
	if err == nil {
		t.Fatal("expected error when DATABASE_DIRECT_URL is unset")
	}
}

func TestDirectDSNFromEnvPresent(t *testing.T) {
	t.Setenv("DATABASE_DIRECT_URL", "postgres://user:pass@host:5432/db")
	dsn, err := DirectDSNFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dsn != "postgres://user:pass@host:5432/db" {
		t.Fatalf("unexpected dsn: %s", dsn)
	}
}
