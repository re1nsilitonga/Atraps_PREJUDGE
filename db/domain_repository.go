// DomainRepository backs the Blocker read surface (PJ-506), the false-positive
// report (PJ-507), the dashboard list/detail views (PJ-605), and the
// cold-start counter (PJ-704). It lives in db/, not core/ — same reasoning as
// ClusterRepository (PRD §9: Core emits shapes, adapters own persistence).
package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"prejudge/core"
)

type BlocklistDomain struct {
	ID            string
	Domain        string
	Confidence    float64
	Reason        string
	MatchedFields []string
}

type CheckResult struct {
	Status     string
	Confidence *float64
	Source     *string
	Reason     *string
}

type DomainListItem struct {
	ID         string
	Domain     string
	Status     string
	Source     *string
	Confidence *float64
	DetectedAt time.Time
}

type Detection struct {
	Layer       int
	Confidence  *float64
	Reason      *string
	EvidenceURL *string
	DetectedAt  time.Time
}

type DomainDetail struct {
	Domain      string
	Detections  []Detection
	Whois       map[string]any
	Cluster     map[string]any
	Siblings    []string
	EvidenceURL *string
}

type BootstrapRun struct {
	L2Confirmations     int
	L1PreemptiveCatches int
	L1Misses            int
}

type DomainRepository struct {
	pool *pgxpool.Pool
}

func NewDomainRepository(pool *pgxpool.Pool) *DomainRepository {
	return &DomainRepository{pool: pool}
}

// Blocklist reads status='blocked' domains, optionally filtered to rows
// blocked after `since` (PJ-506: "since actually filters").
func (r *DomainRepository) Blocklist(ctx context.Context, since *time.Time) ([]BlocklistDomain, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, domain, COALESCE(confidence, 0), COALESCE(reason, ''), matched_fields
		FROM domains
		WHERE status = 'blocked' AND ($1::timestamptz IS NULL OR blocked_at > $1)
		ORDER BY blocked_at DESC
	`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []BlocklistDomain
	for rows.Next() {
		var d BlocklistDomain
		var matchedFieldsRaw []byte
		if err := rows.Scan(&d.ID, &d.Domain, &d.Confidence, &d.Reason, &matchedFieldsRaw); err != nil {
			return nil, err
		}
		d.MatchedFields = decodeStringSlice(matchedFieldsRaw)
		out = append(out, d)
	}
	return out, rows.Err()
}

// Check looks up one domain's current status. A domain the system has never
// seen is not an error (PJ-506: "clean not-found state, not a 404 the
// Blocker must catch") — it reports status "candidate", same as any other
// domain not yet confirmed.
func (r *DomainRepository) Check(ctx context.Context, domain string) (CheckResult, error) {
	var res CheckResult
	err := r.pool.QueryRow(ctx, `
		SELECT status::text, confidence, source::text, reason FROM domains WHERE domain = $1
	`, domain).Scan(&res.Status, &res.Confidence, &res.Source, &res.Reason)
	if err != nil {
		return CheckResult{Status: "candidate"}, nil
	}
	return res, nil
}

// ReportFalsePositive backs the block page's "Laporkan salah" button
// (PJ-507). No auth in the MVP; an invalid or unknown domain_id is logged by
// the caller but still answers ok — a stuck-looking error here is worse than
// a silent no-op (PRD §14 risk #14).
func (r *DomainRepository) ReportFalsePositive(ctx context.Context, domainID, note string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE domains SET status = 'false_pos' WHERE id = $1::uuid
	`, domainID)
	return err
}

// ListDomains backs the dashboard list (PJ-605). nil source/status means
// "no filter" on that column.
func (r *DomainRepository) ListDomains(ctx context.Context, limit, offset int, source, status *string) ([]DomainListItem, int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, domain, status::text, source::text, confidence, first_seen
		FROM domains
		WHERE ($3::domain_status IS NULL OR status = $3::domain_status)
		  AND ($4::detection_source IS NULL OR source = $4::detection_source)
		ORDER BY first_seen DESC
		LIMIT $1 OFFSET $2
	`, limit, offset, status, source)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []DomainListItem
	for rows.Next() {
		var item DomainListItem
		if err := rows.Scan(&item.ID, &item.Domain, &item.Status, &item.Source, &item.Confidence, &item.DetectedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	var total int
	err = r.pool.QueryRow(ctx, `
		SELECT count(*) FROM domains
		WHERE ($1::domain_status IS NULL OR status = $1::domain_status)
		  AND ($2::detection_source IS NULL OR source = $2::detection_source)
	`, status, source).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

// DomainDetail backs the dashboard drill-down (PJ-605). A nil result (no
// error) means the id doesn't exist — the caller turns that into a 404.
func (r *DomainRepository) DomainDetail(ctx context.Context, id string) (*DomainDetail, error) {
	var domain string
	var clusterID *string
	err := r.pool.QueryRow(ctx, `SELECT domain, cluster_id::text FROM domains WHERE id = $1::uuid`, id).
		Scan(&domain, &clusterID)
	if err != nil {
		return nil, nil // not found, not a system error
	}

	detail := &DomainDetail{Domain: domain}

	detRows, err := r.pool.Query(ctx, `
		SELECT layer, confidence, reason, evidence_url, detected_at
		FROM detections WHERE domain_id = $1::uuid ORDER BY detected_at DESC
	`, id)
	if err != nil {
		return nil, err
	}
	for detRows.Next() {
		var d Detection
		if err := detRows.Scan(&d.Layer, &d.Confidence, &d.Reason, &d.EvidenceURL, &d.DetectedAt); err != nil {
			detRows.Close()
			return nil, err
		}
		if d.EvidenceURL != nil && detail.EvidenceURL == nil {
			detail.EvidenceURL = d.EvidenceURL
		}
		detail.Detections = append(detail.Detections, d)
	}
	detRows.Close()
	if err := detRows.Err(); err != nil {
		return nil, err
	}

	if clusterID != nil {
		var registrar, nameserver, tld *string
		var burstScore *float64
		err := r.pool.QueryRow(ctx, `
			SELECT registrar, nameserver, tld, registration_burst_score
			FROM fingerprint_clusters WHERE id = $1::uuid
		`, *clusterID).Scan(&registrar, &nameserver, &tld, &burstScore)
		if err == nil {
			detail.Cluster = map[string]any{
				"registrar":               registrar,
				"nameserver":              nameserver,
				"tld":                     tld,
				"registration_burst_score": burstScore,
			}
		}

		sibRows, err := r.pool.Query(ctx, `
			SELECT domain FROM domains WHERE cluster_id = $1::uuid AND id != $2::uuid
		`, *clusterID, id)
		if err != nil {
			return nil, err
		}
		for sibRows.Next() {
			var sibling string
			if err := sibRows.Scan(&sibling); err != nil {
				sibRows.Close()
				return nil, err
			}
			detail.Siblings = append(detail.Siblings, sibling)
		}
		sibRows.Close()
		if err := sibRows.Err(); err != nil {
			return nil, err
		}
	}

	var registrar *string
	var createdDate *time.Time
	err = r.pool.QueryRow(ctx, `
		SELECT registrar, created_date FROM whois_records WHERE domain_id = $1::uuid ORDER BY fetched_at DESC LIMIT 1
	`, id).Scan(&registrar, &createdDate)
	if err == nil {
		detail.Whois = map[string]any{"registrar": registrar, "created_date": createdDate}
	}

	return detail, nil
}

// BootstrapLatest backs GET /bootstrap/latest (PJ-704). No runs yet is not
// an error — the demo's opening frame is this exact all-zeros state.
func (r *DomainRepository) BootstrapLatest(ctx context.Context) (*BootstrapRun, error) {
	var run BootstrapRun
	err := r.pool.QueryRow(ctx, `
		SELECT l2_confirmations, l1_preemptive_catches, l1_misses
		FROM bootstrap_runs ORDER BY run_at DESC LIMIT 1
	`).Scan(&run.L2Confirmations, &run.L1PreemptiveCatches, &run.L1Misses)
	if err != nil {
		return nil, nil
	}
	return &run, nil
}

// RecordBootstrapRun writes one cold-start proof run (PJ-703). Each call is
// a new row — re-running the script from a clean state is safe by
// construction, not something the repository has to special-case.
func (r *DomainRepository) RecordBootstrapRun(ctx context.Context, l2Confirmations, l1PreemptiveCatches, l1Misses int, notes string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO bootstrap_runs (l2_confirmations, l1_preemptive_catches, l1_misses, notes)
		VALUES ($1, $2, $3, $4)
	`, l2Confirmations, l1PreemptiveCatches, l1Misses, notes)
	return err
}

// EnsureDomain returns the id of domain, inserting a status='candidate' row
// if none exists yet. Called synchronously from the /analyze handler (PJ-204)
// because the response contract requires domain_id in the same response —
// unlike UpsertBlocked/LogDetection/ListConfirmed below, this cannot run in
// the background goroutine.
func (r *DomainRepository) EnsureDomain(ctx context.Context, domain string) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO domains (domain, status) VALUES ($1, 'candidate')
		ON CONFLICT (domain) DO UPDATE SET domain = EXCLUDED.domain
		RETURNING id
	`, domain).Scan(&id)
	return id, err
}

// UpsertBlocked marks domain as status='blocked' with the verdict's
// source/confidence/reason/matched_fields. Idempotent on domain (PJ-203: "no
// dupes on repeat visits") via the same ON CONFLICT upsert as EnsureDomain.
func (r *DomainRepository) UpsertBlocked(ctx context.Context, verdict core.Verdict) error {
	matchedFields, err := json.Marshal(verdict.MatchedFields)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO domains (domain, status, source, confidence, reason, matched_fields, blocked_at)
		VALUES ($1, 'blocked', $2::detection_source, $3, $4, $5::jsonb, now())
		ON CONFLICT (domain) DO UPDATE SET
			status = 'blocked', source = EXCLUDED.source, confidence = EXCLUDED.confidence,
			reason = EXCLUDED.reason, matched_fields = EXCLUDED.matched_fields, blocked_at = now()
	`, verdict.Domain, string(verdict.Source), verdict.Confidence, verdict.Reason, string(matchedFields))
	return err
}

// LogDetection appends a detections row regardless of threshold outcome
// (core/layer2/decide.go's DomainStore contract). Self-ensures the parent
// domains row exists first — detections.domain_id is FK NOT NULL — so this
// is safe to call even if EnsureDomain wasn't (or couldn't have been)
// called first for this particular domain.
func (r *DomainRepository) LogDetection(ctx context.Context, domain string, layer int, confidence float64, reason string, raw json.RawMessage) error {
	domainID, err := r.EnsureDomain(ctx, domain)
	if err != nil {
		return err
	}
	if len(raw) == 0 {
		raw = json.RawMessage("null")
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO detections (domain_id, layer, confidence, reason, raw_response)
		VALUES ($1::uuid, $2, $3, $4, $5::jsonb)
	`, domainID, layer, confidence, reason, string(raw))
	return err
}

// ListConfirmed returns domains Layer 2 has confirmed (source='L2',
// status='blocked') — the feedback loop's (core/feedback.go, PJ-301) input
// set, and the leakage-assertion boundary for PJ-703's cold-start proof.
func (r *DomainRepository) ListConfirmed(ctx context.Context) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT domain FROM domains WHERE source = 'L2' AND status = 'blocked'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		domains = append(domains, d)
	}
	return domains, rows.Err()
}

func decodeStringSlice(raw []byte) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return []string{}
	}
	return out
}
