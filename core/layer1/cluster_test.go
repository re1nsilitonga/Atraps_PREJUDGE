package layer1

import (
	"fmt"
	"testing"
	"time"
)

func fp(ip, ns, registrar string, registeredAt *time.Time) Fingerprint {
	return Fingerprint{HostingIP: &ip, Nameserver: &ns, Registrar: &registrar, RegisteredAt: registeredAt, TLD: "xyz"}
}

func TestBuildClustersGroupsBySharedHostingIP(t *testing.T) {
	t0 := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	t1 := t0.Add(3 * time.Hour)
	records := []DomainRecord{
		{Domain: "sib1.test", Fingerprint: fp("203.0.113.10", "ns1.test", "Fixture Registrar", &t0)},
		{Domain: "sib2.test", Fingerprint: fp("203.0.113.10", "ns1.test", "Fixture Registrar", &t1)},
		{Domain: "solo.test", Fingerprint: fp("198.51.100.20", "ns2.test", "Other Registrar", nil)},
	}

	clusters := BuildClusters(records)
	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster (solo domain has no sibling), got %d", len(clusters))
	}
	c := clusters[0]
	if len(c.Domains) != 2 {
		t.Fatalf("expected 2 domains in cluster, got %d", len(c.Domains))
	}
	if c.RegistrationWindowHours != 3 {
		t.Fatalf("expected window 3h, got %d", c.RegistrationWindowHours)
	}
	if c.RegistrationBurstScore == nil {
		t.Fatal("expected a burst score")
	}
}

func TestBuildClustersEmptyInputIsCleanNotError(t *testing.T) {
	if clusters := BuildClusters(nil); len(clusters) != 0 {
		t.Fatalf("expected zero clusters for empty input, got %d", len(clusters))
	}
}

func TestBurstScoreNilWhenRegistrationDatesMissing(t *testing.T) {
	records := []DomainRecord{
		{Domain: "a.test", Fingerprint: fp("203.0.113.10", "ns1.test", "Registrar", nil)},
		{Domain: "b.test", Fingerprint: fp("203.0.113.10", "ns1.test", "Registrar", nil)},
	}
	clusters := BuildClusters(records)
	if clusters[0].RegistrationBurstScore != nil {
		t.Fatal("expected nil burst score when registration dates are missing")
	}
}

func TestBurstScoreDenseWindowManyDomainsIsHigh(t *testing.T) {
	t0 := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	records := make([]DomainRecord, 0, 40)
	for i := 0; i < 40; i++ {
		regAt := t0.Add(time.Duration(i) * time.Minute)
		records = append(records, DomainRecord{
			Domain:      fmt.Sprintf("d%d.test", i),
			Fingerprint: fp("203.0.113.10", "ns1.test", "Registrar", &regAt),
		})
	}
	clusters := BuildClusters(records)
	if *clusters[0].RegistrationBurstScore < 0.9 {
		t.Fatalf("expected high burst score for 40 domains in <1h, got %v", *clusters[0].RegistrationBurstScore)
	}
}
