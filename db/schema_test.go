package db

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func readSchema(t *testing.T) string {
	t.Helper()
	src, err := os.ReadFile("schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	return string(src)
}

func TestFiveTablesPresent(t *testing.T) {
	schema := readSchema(t)
	for _, table := range []string{"domains", "fingerprint_clusters", "detections", "whois_records", "bootstrap_runs"} {
		pattern := regexp.MustCompile(`CREATE TABLE IF NOT EXISTS ` + table)
		if !pattern.MatchString(schema) {
			t.Fatalf("missing table %s", table)
		}
	}
}

func TestEnumsPresent(t *testing.T) {
	schema := readSchema(t)
	if !strings.Contains(schema, "domain_status") {
		t.Fatal("missing domain_status enum")
	}
	for _, value := range []string{"'candidate'", "'confirmed'", "'blocked'", "'false_pos'"} {
		if !strings.Contains(schema, value) {
			t.Fatalf("missing enum value %s", value)
		}
	}
	if !strings.Contains(schema, "detection_source") {
		t.Fatal("missing detection_source enum")
	}
	for _, value := range []string{"'L1'", "'L2'", "'trustpositif'"} {
		if !strings.Contains(schema, value) {
			t.Fatalf("missing enum value %s", value)
		}
	}
}

func tableBlock(t *testing.T, schema, name string) string {
	t.Helper()
	pattern := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS ` + name + ` \((.*?)\);`)
	match := pattern.FindStringSubmatch(schema)
	if match == nil {
		t.Fatalf("could not find table block for %s", name)
	}
	return match[1]
}

func TestBurstFieldsOnCluster(t *testing.T) {
	block := tableBlock(t, readSchema(t), "fingerprint_clusters")
	for _, field := range []string{
		"first_registration_date",
		"last_registration_date",
		"registration_window_hours",
		"registration_burst_score",
	} {
		if !strings.Contains(block, field) {
			t.Fatalf("missing %s on fingerprint_clusters", field)
		}
	}
}

func TestDomainsDenormalizedFields(t *testing.T) {
	block := tableBlock(t, readSchema(t), "domains")
	for _, field := range []string{"matched_fields", "reason", "source_masked_pattern"} {
		if !strings.Contains(block, field) {
			t.Fatalf("missing %s on domains", field)
		}
	}
	if !strings.Contains(block, "UNIQUE") {
		t.Fatal("expected domain column to be UNIQUE")
	}
}

func TestDetectionsUsesEvidenceURLNotScreenshot(t *testing.T) {
	block := tableBlock(t, readSchema(t), "detections")
	if !strings.Contains(block, "evidence_url") {
		t.Fatal("missing evidence_url")
	}
	if strings.Contains(block, "screenshot_url") {
		t.Fatal("detections must not use screenshot_url — Android has no pixels")
	}
}

func TestIdempotentEnumCreation(t *testing.T) {
	schema := readSchema(t)
	if strings.Count(schema, "EXCEPTION") < 2 {
		t.Fatal("CREATE TYPE statements must be wrapped in DO blocks for idempotency")
	}
}

func TestStatusIndexExists(t *testing.T) {
	if !strings.Contains(readSchema(t), "idx_domains_status") {
		t.Fatal("missing idx_domains_status index")
	}
}

func TestDomainBlockedNotifyTriggerPresent(t *testing.T) {
	schema := readSchema(t)
	if !strings.Contains(schema, "pg_notify('domain_blocked'") {
		t.Fatal("missing pg_notify('domain_blocked', ...) — api/realtime.go's LISTEN depends on this")
	}
	if !strings.Contains(schema, "CREATE TRIGGER domains_notify_blocked") {
		t.Fatal("missing domains_notify_blocked trigger")
	}
	if !strings.Contains(schema, "DROP TRIGGER IF EXISTS domains_notify_blocked") {
		t.Fatal("trigger creation must be idempotent (DROP TRIGGER IF EXISTS first)")
	}
}
