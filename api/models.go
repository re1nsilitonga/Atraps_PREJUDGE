package main

type BlocklistEntry struct {
	ID            string   `json:"id"`
	Domain        string   `json:"domain"`
	Confidence    float64  `json:"confidence"`
	Reason        string   `json:"reason"`
	MatchedFields []string `json:"matched_fields"`
}

type BlocklistResponse struct {
	Domains   []BlocklistEntry `json:"domains"`
	UpdatedAt string           `json:"updated_at"`
}

type CheckRequest struct {
	Domain string `json:"domain"`
}

type CheckResponse struct {
	Status     string   `json:"status"`
	Confidence *float64 `json:"confidence"`
	Source     *string  `json:"source"`
	Reason     *string  `json:"reason"`
}

type AnalyzeRequest struct {
	Domain      string `json:"domain"`
	EvidenceB64 string `json:"evidence_b64"`
}

type AnalyzeResponse struct {
	IsJudol    bool    `json:"is_judol"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
	DomainID   string  `json:"domain_id"`
}

type FingerprintRequest struct {
	Domain string `json:"domain"`
}

type FingerprintResponse struct {
	ClusterID     *string  `json:"cluster_id"`
	Registrar     *string  `json:"registrar"`
	IP            *string  `json:"ip"`
	NS            *string  `json:"ns"`
	TLD           *string  `json:"tld"`
	MatchScore    float64  `json:"match_score"`
	MatchedFields []string `json:"matched_fields"`
}

type DomainListItem struct {
	ID         string   `json:"id"`
	Domain     string   `json:"domain"`
	Status     string   `json:"status"`
	Source     *string  `json:"source"`
	Confidence *float64 `json:"confidence"`
	DetectedAt *string  `json:"detected_at"`
}

type DomainListResponse struct {
	Items []DomainListItem `json:"items"`
	Total int              `json:"total"`
}

type DomainDetailResponse struct {
	Domain      string           `json:"domain"`
	Detections  []map[string]any `json:"detections"`
	Whois       map[string]any   `json:"whois,omitempty"`
	Cluster     map[string]any   `json:"cluster,omitempty"`
	Siblings    []string         `json:"siblings"`
	EvidenceURL *string          `json:"evidence_url"`
}

type ReportFalsePositiveRequest struct {
	DomainID string  `json:"domain_id"`
	Note     *string `json:"note"`
}

type OkResponse struct {
	Ok bool `json:"ok"`
}

type BootstrapLatestResponse struct {
	L2Confirmations     int     `json:"l2_confirmations"`
	L1PreemptiveCatches int     `json:"l1_preemptive_catches"`
	L1Misses            int     `json:"l1_misses"`
	Ratio               float64 `json:"ratio"`
}

type TrustPositifVerifyRequest struct {
	Domain string `json:"domain"`
}

type TrustPositifVerifyResponse struct {
	Domain    string `json:"domain"`
	IsBlocked bool   `json:"is_blocked"`
}
