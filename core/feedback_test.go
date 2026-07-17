package core

import (
	"context"
	"errors"
	"testing"

	"prime/core/layer1"
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

type fakeCandidateDomains struct{ domains []string }

func (f fakeCandidateDomains) ListCandidates(ctx context.Context) ([]string, error) {
	return f.domains, nil
}

type fakeClusterLister struct{ clusters []layer1.Cluster }

func (f fakeClusterLister) ListClusters(ctx context.Context) ([]layer1.Cluster, error) {
	return f.clusters, nil
}

type fakeBlockedWriter struct{ blocked []Verdict }

func (f *fakeBlockedWriter) UpsertBlocked(ctx context.Context, v Verdict) error {
	f.blocked = append(f.blocked, v)
	return nil
}

func TestMatchSiblingsBlocksCandidateMatchingCluster(t *testing.T) {
	candidates := fakeCandidateDomains{domains: []string{"sibling.xyz"}}
	extractor := fakeExtractor{byDomain: map[string]layer1.Fingerprint{
		"sibling.xyz": {Domain: "sibling.xyz", HostingIP: ip("203.0.113.10"), Nameserver: ip("ns1.evil.test"), TLD: "xyz"},
	}}
	clusters := fakeClusterLister{clusters: []layer1.Cluster{
		{ID: "cluster-1", HostingIP: "203.0.113.10", Nameserver: "ns1.evil.test", TLD: "xyz"},
	}}
	writer := &fakeBlockedWriter{}

	if err := MatchSiblings(context.Background(), candidates, extractor, clusters, writer); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(writer.blocked) != 1 {
		t.Fatalf("expected 1 domain blocked, got %d", len(writer.blocked))
	}
	v := writer.blocked[0]
	if v.Domain != "sibling.xyz" || !v.IsJudol || v.Source != SourceL1 {
		t.Fatalf("unexpected verdict: %+v", v)
	}
	if len(v.MatchedFields) == 0 {
		t.Fatal("expected matched_fields to be populated")
	}
}

func TestMatchSiblingsLeavesNonMatchAlone(t *testing.T) {
	candidates := fakeCandidateDomains{domains: []string{"unrelated.com"}}
	extractor := fakeExtractor{byDomain: map[string]layer1.Fingerprint{
		"unrelated.com": {Domain: "unrelated.com", HostingIP: ip("198.51.100.1"), TLD: "com"},
	}}
	clusters := fakeClusterLister{clusters: []layer1.Cluster{
		{ID: "cluster-1", HostingIP: "203.0.113.10", Nameserver: "ns1.evil.test", TLD: "xyz"},
	}}
	writer := &fakeBlockedWriter{}

	if err := MatchSiblings(context.Background(), candidates, extractor, clusters, writer); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(writer.blocked) != 0 {
		t.Fatalf("expected no domains blocked for a non-matching candidate, got %d", len(writer.blocked))
	}
}

func TestMatchSiblingsNoClustersIsNoOp(t *testing.T) {
	candidates := fakeCandidateDomains{domains: []string{"whatever.com"}}
	extractor := fakeExtractor{byDomain: map[string]layer1.Fingerprint{}}
	clusters := fakeClusterLister{clusters: nil}
	writer := &fakeBlockedWriter{}

	if err := MatchSiblings(context.Background(), candidates, extractor, clusters, writer); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(writer.blocked) != 0 {
		t.Fatalf("expected no domains blocked when there are no clusters, got %d", len(writer.blocked))
	}
}

func TestMatchSiblingsSkipsExtractionFailures(t *testing.T) {
	candidates := fakeCandidateDomains{domains: []string{"unreachable.test"}}
	extractor := fakeExtractor{byDomain: map[string]layer1.Fingerprint{}}
	clusters := fakeClusterLister{clusters: []layer1.Cluster{
		{ID: "cluster-1", HostingIP: "203.0.113.10", Nameserver: "ns1.evil.test", TLD: "xyz"},
	}}
	writer := &fakeBlockedWriter{}

	if err := MatchSiblings(context.Background(), candidates, extractor, clusters, writer); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(writer.blocked) != 0 {
		t.Fatalf("expected no domains blocked when extraction fails, got %d", len(writer.blocked))
	}
}
