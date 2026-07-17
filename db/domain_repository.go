package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"prime/core"
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

func (r *DomainRepository) ListCandidates(ctx context.Context) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT domain FROM domains WHERE status = 'candidate'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var domain string
		if err := rows.Scan(&domain); err != nil {
			return nil, err
		}
		out = append(out, domain)
	}
	return out, rows.Err()
}

func (r *DomainRepository) ReportFalsePositive(ctx context.Context, domainID, note string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE domains SET status = 'false_pos' WHERE id = $1::uuid
	`, domainID)
	return err
}

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

func (r *DomainRepository) DomainDetail(ctx context.Context, id string) (*DomainDetail, error) {
	var domain string
	var clusterID *string
	err := r.pool.QueryRow(ctx, `SELECT domain, cluster_id::text FROM domains WHERE id = $1::uuid`, id).
		Scan(&domain, &clusterID)
	if err != nil {
		return nil, nil
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
				"registrar":                registrar,
				"nameserver":               nameserver,
				"tld":                      tld,
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

func (r *DomainRepository) RecordBootstrapRun(ctx context.Context, l2Confirmations, l1PreemptiveCatches, l1Misses int, notes string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO bootstrap_runs (l2_confirmations, l1_preemptive_catches, l1_misses, notes)
		VALUES ($1, $2, $3, $4)
	`, l2Confirmations, l1PreemptiveCatches, l1Misses, notes)
	return err
}

func (r *DomainRepository) EnsureDomain(ctx context.Context, domain string) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO domains (domain, status) VALUES ($1, 'candidate')
		ON CONFLICT (domain) DO UPDATE SET domain = EXCLUDED.domain
		RETURNING id
	`, domain).Scan(&id)
	return id, err
}

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
