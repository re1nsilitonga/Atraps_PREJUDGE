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
