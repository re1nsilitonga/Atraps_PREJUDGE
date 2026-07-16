package layer1

import "testing"

func TestMatchAboveThresholdReturnsMatchedFields(t *testing.T) {
	burst := 0.9
	cluster := Cluster{ID: "cluster-1", HostingIP: "203.0.113.10", Nameserver: "ns1.test", Registrar: "Fixture Registrar", TLD: "xyz", RegistrationBurstScore: &burst}
	ip := "203.0.113.10"
	ns := "ns1.test"
	target := Fingerprint{HostingIP: &ip, Nameserver: &ns, TLD: "xyz"}

	result := Match(target, []Cluster{cluster})
	if result == nil {
		t.Fatal("expected a match")
	}
	if result.ClusterID != "cluster-1" {
		t.Fatalf("expected cluster-1, got %s", result.ClusterID)
	}
	for _, want := range []string{"hosting_ip", "nameserver", "tld"} {
		found := false
		for _, got := range result.MatchedFields {
			if got == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected matched field %s, got %v", want, result.MatchedFields)
		}
	}
}

func TestMatchBelowThresholdReturnsNilCleanly(t *testing.T) {
	cluster := Cluster{ID: "cluster-1", HostingIP: "203.0.113.10", TLD: "xyz"}
	otherIP := "198.51.100.1"
	target := Fingerprint{HostingIP: &otherIP, TLD: "com"}

	if result := Match(target, []Cluster{cluster}); result != nil {
		t.Fatalf("expected no match below threshold, got %+v", result)
	}
}

func TestMatchEmptyClustersReturnsNil(t *testing.T) {
	ip := "203.0.113.10"
	if Match(Fingerprint{HostingIP: &ip}, nil) != nil {
		t.Fatal("expected nil match against empty cluster list")
	}
}

func TestMatchPicksHighestScoringCluster(t *testing.T) {
	ip := "203.0.113.10"
	ns := "ns1.test"
	registrar := "Fixture Registrar"
	target := Fingerprint{HostingIP: &ip, Nameserver: &ns, Registrar: &registrar, TLD: "xyz"}

	weak := Cluster{ID: "weak", HostingIP: ip, TLD: "xyz"}
	strong := Cluster{ID: "strong", HostingIP: ip, Nameserver: ns, Registrar: registrar, TLD: "xyz"}

	result := Match(target, []Cluster{weak, strong})
	if result.ClusterID != "strong" {
		t.Fatalf("expected the higher-scoring cluster, got %s", result.ClusterID)
	}
}
