package main

import (
	"errors"
	"testing"

	"prejudge/core/layer1"
)

func fp(ip string) layer1.Fingerprint {
	return layer1.Fingerprint{HostingIP: &ip, TLD: "xyz"}
}

func TestRunCountsCatchesAndMisses(t *testing.T) {
	confirmed := []string{"seed1.test", "seed2.test"}
	candidates := []string{"sib1.test", "unrelated.test"}
	burst := 0.9
	ip := "203.0.113.10"
	clusters := []layer1.Cluster{{ID: "cluster-1", HostingIP: ip, TLD: "xyz", RegistrationBurstScore: &burst}}

	extract := func(domain string) (layer1.Fingerprint, error) {
		if domain == "sib1.test" {
			return fp(ip), nil // shares the cluster's hosting IP — should catch
		}
		return fp("198.51.100.1"), nil // different infrastructure — should miss
	}

	result, err := Run(confirmed, candidates, extract, clusters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.N != 2 {
		t.Fatalf("expected N=2, got %d", result.N)
	}
	if result.M != 1 {
		t.Fatalf("expected M=1 (sib1.test catches), got %d", result.M)
	}
	if len(result.Misses) != 1 || result.Misses[0] != "unrelated.test" {
		t.Fatalf("expected miss [unrelated.test], got %v", result.Misses)
	}
}

func TestRunLeakageAssertionFailsLoud(t *testing.T) {
	confirmed := []string{"seed1.test"}
	candidates := []string{"seed1.test"} // contaminated: a confirmed domain in the candidate pool

	_, err := Run(confirmed, candidates, func(string) (layer1.Fingerprint, error) {
		t.Fatal("extract must not be called once leakage is detected")
		return layer1.Fingerprint{}, nil
	}, nil)
	if err == nil {
		t.Fatal("expected a leakage error, got nil")
	}
}

func TestRunEmptyStateIsCleanNotError(t *testing.T) {
	result, err := Run(nil, nil, func(string) (layer1.Fingerprint, error) {
		t.Fatal("extract must not be called with no candidates")
		return layer1.Fingerprint{}, nil
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error on empty state: %v", err)
	}
	if result.N != 0 || result.M != 0 || len(result.Misses) != 0 {
		t.Fatalf("expected zero state, got %+v", result)
	}
}

func TestRunExtractionFailureCountsAsMiss(t *testing.T) {
	result, err := Run(nil, []string{"redacted.test"}, func(string) (layer1.Fingerprint, error) {
		return layer1.Fingerprint{}, errors.New("dns lookup failed")
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.M != 0 || len(result.Misses) != 1 {
		t.Fatalf("expected 1 miss, 0 catches, got %+v", result)
	}
}

func TestRunNoMatchingClusterCountsAsMiss(t *testing.T) {
	ip := "198.51.100.1"
	result, err := Run(nil, []string{"lonely.test"}, func(string) (layer1.Fingerprint, error) {
		return fp(ip), nil
	}, nil) // no clusters at all — the honest cold-start state
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.M != 0 || len(result.Misses) != 1 {
		t.Fatalf("expected 1 miss, 0 catches against empty clusters, got %+v", result)
	}
}
