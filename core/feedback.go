// The feedback loop (PJ-301, Epic 3): a Layer 2 confirmation triggers
// fingerprint extraction and cluster seeding, so Layer 1 has something to
// match against. Without this, Layer 1 has no input at all — not weaker,
// none — because under cold start there is no other data source
// (PRD.md §4, TASKS.md Epic 3).
package core

import (
	"context"

	"prejudge/core/layer1"
)

// ConfirmedDomains lists every domain Layer 2 has confirmed so far. The
// feedback loop rebuilds clusters from this whole set on every
// confirmation — cheap at hackathon scale, and it keeps ClusterStore.Upsert
// idempotent rather than needing incremental merge logic.
type ConfirmedDomains interface {
	ListConfirmed(ctx context.Context) ([]string, error)
}

// FingerprintExtractor pulls infrastructure signals for one domain.
// Satisfied by layer1.Extractor.
type FingerprintExtractor interface {
	Extract(ctx context.Context, domain string) (layer1.Fingerprint, error)
}

// ClusterStore persists clusters. Satisfied by db.ClusterRepository.
type ClusterStore interface {
	Upsert(ctx context.Context, c layer1.Cluster) (string, error)
}

// Feedback seeds/updates Layer 1 clusters from every currently-confirmed
// domain. Call it after a Layer 2 verdict flips a domain to blocked; run it
// in a goroutine — it must never delay the verdict response (PJ-301).
func Feedback(ctx context.Context, domains ConfirmedDomains, extractor FingerprintExtractor, clusters ClusterStore) error {
	confirmed, err := domains.ListConfirmed(ctx)
	if err != nil {
		return err
	}

	records := make([]layer1.DomainRecord, 0, len(confirmed))
	for _, domain := range confirmed {
		fp, err := extractor.Extract(ctx, domain)
		if err != nil {
			continue // PJ-401: a failed/redacted lookup degrades to no fingerprint, not a crash
		}
		records = append(records, layer1.DomainRecord{Domain: domain, Fingerprint: fp})
	}

	for _, cluster := range layer1.BuildClusters(records) {
		if _, err := clusters.Upsert(ctx, cluster); err != nil {
			return err
		}
	}
	return nil
}
