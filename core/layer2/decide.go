package layer2

import (
	"encoding/json"

	"prejudge/core"
)

// L2ConfidenceThreshold is the bar a vision verdict must clear before a
// domain flips to status='blocked'. Named constant, not a magic number
// scattered through the code (TASKS.md PJ-203).
const L2ConfidenceThreshold = 0.8

// DomainStore is the persistence seam Decide writes through. Core defines
// the interface; a concrete Supabase/Postgres-backed implementation lives
// outside core/ (api/ or a dedicated store package) so this package stays
// framework-free — Core writes the verdict, Core does not know who's
// listening (PRD.md §9).
type DomainStore interface {
	// UpsertBlocked marks domain as status='blocked', source='L2' with the
	// verdict's reason/matched_fields. Must be idempotent on domain — no
	// dupes on repeat visits.
	UpsertBlocked(verdict core.Verdict) error
	// LogDetection appends a detections row regardless of threshold outcome.
	LogDetection(domain string, layer int, confidence float64, reason string, raw json.RawMessage) error
}

// Decide applies the L2 threshold to a vision AnalyzeResult and writes
// through store. Above threshold: domains flips to blocked. Below (or not
// judol at all): only the detection is logged, domains.status is untouched.
func Decide(store DomainStore, result AnalyzeResult) error {
	raw := json.RawMessage(result.Raw)
	if len(raw) == 0 {
		raw = json.RawMessage("null")
	}

	if err := store.LogDetection(result.Verdict.Domain, 2, result.Verdict.Confidence, result.Verdict.Reason, raw); err != nil {
		return err
	}

	if !result.Verdict.IsJudol || result.Verdict.Confidence < L2ConfidenceThreshold {
		return nil
	}

	return store.UpsertBlocked(result.Verdict)
}
