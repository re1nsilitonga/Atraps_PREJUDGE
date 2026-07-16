package core

import (
	"context"
	"errors"
	"testing"

	"prejudge/core/layer1"
)

type fakeConfirmedDomains struct{ domains []string }

func (f fakeConfirmedDomains) ListConfirmed(ctx context.Context) ([]string, error) {
	return f.domains, nil
}

type fakeExtractor struct{ byDomain map[string]layer1.Fingerprint }

func (f fakeExtractor) Extract(ctx context.Context, domain string) (layer1.Fingerprint, error) {
	fp, ok := f.byDomain[domain]
	if !ok {
		return layer1.Fingerprint{}, errors.New("no fixture for domain")
	}
	return fp, nil
}

type fakeClusterStore struct{ upserted []layer1.Cluster }

func (f *fakeClusterStore) Upsert(ctx context.Context, c layer1.Cluster) (string, error) {
	f.upserted = append(f.upserted, c)
	return "cluster-id", nil
}

func ip(s string) *string { return &s }

func TestFeedbackSeedsClusterFromConfirmedSiblings(t *testing.T) {
	domains := fakeConfirmedDomains{domains: []string{"a.xyz", "b.xyz"}}
	extractor := fakeExtractor{byDomain: map[string]layer1.Fingerprint{
		"a.xyz": {Domain: "a.xyz", HostingIP: ip("203.0.113.10"), TLD: "xyz"},
		"b.xyz": {Domain: "b.xyz", HostingIP: ip("203.0.113.10"), TLD: "xyz"},
	}}
	clusters := &fakeClusterStore{}

	if err := Feedback(context.Background(), domains, extractor, clusters); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clusters.upserted) != 1 {
		t.Fatalf("expected 1 cluster upserted, got %d", len(clusters.upserted))
	}
	if len(clusters.upserted[0].Domains) != 2 {
		t.Fatalf("expected 2 sibling domains in cluster, got %d", len(clusters.upserted[0].Domains))
	}
}

func TestFeedbackEmptyDBProducesNoClusters(t *testing.T) {
	domains := fakeConfirmedDomains{domains: nil}
	extractor := fakeExtractor{byDomain: map[string]layer1.Fingerprint{}}
	clusters := &fakeClusterStore{}

	if err := Feedback(context.Background(), domains, extractor, clusters); err != nil {
		t.Fatalf("unexpected error on empty DB: %v", err)
	}
	if len(clusters.upserted) != 0 {
		t.Fatalf("expected no clusters on empty DB, got %d", len(clusters.upserted))
	}
}

func TestFeedbackSkipsDomainsWhoseExtractionFails(t *testing.T) {
	domains := fakeConfirmedDomains{domains: []string{"a.xyz", "unreachable.test"}}
	extractor := fakeExtractor{byDomain: map[string]layer1.Fingerprint{
		"a.xyz": {Domain: "a.xyz", HostingIP: ip("203.0.113.10"), TLD: "xyz"},
	}}
	clusters := &fakeClusterStore{}

	if err := Feedback(context.Background(), domains, extractor, clusters); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only one domain has a fingerprint, so no cluster (needs >=2 siblings).
	if len(clusters.upserted) != 0 {
		t.Fatalf("expected no clusters when only one domain has a fingerprint, got %d", len(clusters.upserted))
	}
}

func TestFeedbackSoleDomainDoesNotClusterAlone(t *testing.T) {
	domains := fakeConfirmedDomains{domains: []string{"a.xyz"}}
	extractor := fakeExtractor{byDomain: map[string]layer1.Fingerprint{
		"a.xyz": {Domain: "a.xyz", HostingIP: ip("203.0.113.10"), TLD: "xyz"},
	}}
	clusters := &fakeClusterStore{}

	if err := Feedback(context.Background(), domains, extractor, clusters); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clusters.upserted) != 0 {
		t.Fatalf("expected no cluster for a lone domain, got %d", len(clusters.upserted))
	}
}
