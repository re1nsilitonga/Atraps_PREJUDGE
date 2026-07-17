package core

import (
	"context"
	"time"

	"prime/core/layer1"
)

type ConfirmedDomains interface {
	ListConfirmed(ctx context.Context) ([]string, error)
}

type FingerprintExtractor interface {
	Extract(ctx context.Context, domain string) (layer1.Fingerprint, error)
}

type ClusterStore interface {
	Upsert(ctx context.Context, c layer1.Cluster) (string, error)
}

func Feedback(ctx context.Context, domains ConfirmedDomains, extractor FingerprintExtractor, clusters ClusterStore) error {
	confirmed, err := domains.ListConfirmed(ctx)
	if err != nil {
		return err
	}

	records := make([]layer1.DomainRecord, 0, len(confirmed))
	for _, domain := range confirmed {
		fp, err := extractor.Extract(ctx, domain)
		if err != nil {
			continue
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

type CandidateDomains interface {
	ListCandidates(ctx context.Context) ([]string, error)
}

type ClusterLister interface {
	ListClusters(ctx context.Context) ([]layer1.Cluster, error)
}

type BlockedDomainWriter interface {
	UpsertBlocked(ctx context.Context, v Verdict) error
}

func MatchSiblings(ctx context.Context, candidates CandidateDomains, extractor FingerprintExtractor, clusters ClusterLister, store BlockedDomainWriter) error {
	names, err := candidates.ListCandidates(ctx)
	if err != nil {
		return err
	}

	clusterList, err := clusters.ListClusters(ctx)
	if err != nil {
		return err
	}
	if len(clusterList) == 0 {
		return nil
	}

	for _, domain := range names {
		fp, err := extractor.Extract(ctx, domain)
		if err != nil {
			continue
		}
		result := layer1.Match(fp, clusterList)
		if result == nil {
			continue
		}

		v := Verdict{
			Domain:        domain,
			IsJudol:       true,
			Confidence:    result.Score,
			Reason:        "Cocok dengan kluster infrastruktur situs judi online yang telah dikonfirmasi.",
			MatchedFields: result.MatchedFields,
			Source:        SourceL1,
			DetectedAt:    time.Now().UTC(),
		}
		if err := store.UpsertBlocked(ctx, v); err != nil {
			return err
		}
	}
	return nil
}
