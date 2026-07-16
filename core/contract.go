// Package core defines the Verdict/Evidence contract — the Core Engine <-> Blocker adapter seam.
//
// FROZEN T+2. The seam. Changes require all 4 team members (PRD.md §11).
//
// This package imports nothing beyond the Go standard library. No HTTP
// framework, no Supabase client, no Chrome APIs. It must compile even if
// the extension didn't exist — that is what makes the Android port a port
// instead of a rewrite.
package core

import "time"

type Source string

const (
	SourceL1 Source = "L1"
	SourceL2 Source = "L2"
)

type EvidenceType string

const (
	EvidenceScreenshot EvidenceType = "screenshot"
	EvidenceDNSSNI     EvidenceType = "dns_sni"
)

type Evidence struct {
	Domain       string
	EvidenceB64  string
	EvidenceType EvidenceType
}

type Verdict struct {
	Domain        string
	IsJudol       bool
	Confidence    float64
	Reason        string
	MatchedFields []string
	Source        Source
	DetectedAt    time.Time
}

// NewVerdict applies the contract defaults: empty MatchedFields, source L2,
// DetectedAt set to now. Callers producing a Layer 1 verdict should set
// Source and MatchedFields explicitly on the returned value.
func NewVerdict(domain string, isJudol bool, confidence float64, reason string) Verdict {
	return Verdict{
		Domain:        domain,
		IsJudol:       isJudol,
		Confidence:    confidence,
		Reason:        reason,
		MatchedFields: []string{},
		Source:        SourceL2,
		DetectedAt:    time.Now().UTC(),
	}
}
