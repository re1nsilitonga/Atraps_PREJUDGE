package db

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func readFixtures(t *testing.T) string {
	t.Helper()
	src, err := os.ReadFile("fixtures.sql")
	if err != nil {
		t.Fatal(err)
	}
	return string(src)
}

func domainsInsertBlock(t *testing.T, fixtures string) string {
	t.Helper()
	parts := strings.SplitN(fixtures, "INSERT INTO domains", 2)
	if len(parts) != 2 {
		t.Fatal("could not find INSERT INTO domains")
	}
	return strings.SplitN(parts[1], ";", 2)[0]
}

func TestMarkedAsFixture(t *testing.T) {
	if !strings.Contains(readFixtures(t), "-- FIXTURE") {
		t.Fatal("fixtures.sql must be marked with -- FIXTURE")
	}
}

func TestAtLeastTwentyDomainRows(t *testing.T) {
	block := domainsInsertBlock(t, readFixtures(t))
	rows := regexp.MustCompile(`\('fixture-`).FindAllString(block, -1)
	if len(rows) < 20 {
		t.Fatalf("expected >=20 fixture domain rows, found %d", len(rows))
	}
}

func TestClusterHasAtLeastFiveSiblings(t *testing.T) {
	block := domainsInsertBlock(t, readFixtures(t))
	siblings := strings.Count(block, "00000000-0000-0000-0000-000000000001")
	if siblings < 5 {
		t.Fatalf("expected >=5 siblings, found %d", siblings)
	}
}

func TestMixedStatusAndSource(t *testing.T) {
	block := domainsInsertBlock(t, readFixtures(t))
	for _, status := range []string{"'candidate'", "'confirmed'", "'blocked'", "'false_pos'"} {
		if !strings.Contains(block, status) {
			t.Fatalf("fixtures missing a row with status %s", status)
		}
	}
	for _, source := range []string{"'L1'", "'L2'", "'trustpositif'"} {
		if !strings.Contains(block, source) {
			t.Fatalf("fixtures missing a row with source %s", source)
		}
	}
}

func TestPurgeSectionTargetsOnlyFixtures(t *testing.T) {
	fixtures := readFixtures(t)
	if !strings.Contains(fixtures, "=== PURGE") {
		t.Fatal("missing purge section")
	}
	purge := strings.SplitN(fixtures, "=== PURGE", 2)[1]
	if !strings.Contains(purge, "fixture-%") {
		t.Fatal("purge must target fixture-% domains")
	}
	if !strings.Contains(purge, "DELETE FROM domains") {
		t.Fatal("purge must delete from domains")
	}
	if !strings.Contains(purge, "DELETE FROM fingerprint_clusters") {
		t.Fatal("purge must delete from fingerprint_clusters")
	}
}
