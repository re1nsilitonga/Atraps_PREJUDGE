package layer2

import (
	"context"
	"encoding/json"

	"prime/core"
)

const L2ConfidenceThreshold = 0.8

type DomainStore interface {
	UpsertBlocked(ctx context.Context, verdict core.Verdict) error
	LogDetection(ctx context.Context, domain string, layer int, confidence float64, reason string, raw json.RawMessage) error
}

func Decide(ctx context.Context, store DomainStore, result AnalyzeResult) error {
	raw := json.RawMessage(result.Raw)
	if len(raw) == 0 {
		raw = json.RawMessage("null")
	}

	if err := store.LogDetection(ctx, result.Verdict.Domain, 2, result.Verdict.Confidence, result.Verdict.Reason, raw); err != nil {
		return err
	}

	if !result.Verdict.IsJudol || result.Verdict.Confidence < L2ConfidenceThreshold {
		return nil
	}

	return store.UpsertBlocked(ctx, result.Verdict)
}
