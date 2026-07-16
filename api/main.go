// Thin HTTP transport over core/. No detection logic lives here.
//
// PJ-106: all 9 routes return contract-shaped stub responses until the real
// Core functions (Epic 2 / Epic 4) are wired in behind them.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"prejudge/core"
	"prejudge/core/layer1"
	"prejudge/core/layer2"
	"prejudge/db"
)

// memoryDomainStore is a placeholder layer2.DomainStore until the real
// Postgres/Supabase-backed store lands (Epic 1, owner A). It exists so
// PJ-204's /analyze wiring can be exercised end-to-end without a DB
// dependency; swap this out, not the handler, once the real store exists.
type memoryDomainStore struct {
	mu       sync.Mutex
	blocked  map[string]core.Verdict
	domainID map[string]string
}

func newMemoryDomainStore() *memoryDomainStore {
	return &memoryDomainStore{
		blocked:  map[string]core.Verdict{},
		domainID: map[string]string{},
	}
}

func (s *memoryDomainStore) UpsertBlocked(v core.Verdict) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.blocked[v.Domain] = v
	s.idFor(v.Domain)
	return nil
}

func (s *memoryDomainStore) LogDetection(domain string, layer int, confidence float64, reason string, raw json.RawMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.idFor(domain)
	return nil
}

// idFor assigns a stable random id to a domain on first sight. Caller must
// hold s.mu.
func (s *memoryDomainStore) idFor(domain string) string {
	if id, ok := s.domainID[domain]; ok {
		return id
	}
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	id := hex.EncodeToString(buf)
	s.domainID[domain] = id
	return id
}

func (s *memoryDomainStore) IDFor(domain string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.idFor(domain)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type fingerprintExtractor interface {
	Extract(ctx context.Context, domain string) (layer1.Fingerprint, error)
}

type clusterLister interface {
	ListClusters(ctx context.Context) ([]layer1.Cluster, error)
}

// domainRepository backs the Blocker read surface (PJ-506), false-positive
// reports (PJ-507), the dashboard list/detail views (PJ-605), and the
// cold-start counter (PJ-704).
type domainRepository interface {
	Blocklist(ctx context.Context, since *time.Time) ([]db.BlocklistDomain, error)
	Check(ctx context.Context, domain string) (db.CheckResult, error)
	ReportFalsePositive(ctx context.Context, domainID, note string) error
	ListDomains(ctx context.Context, limit, offset int, source, status *string) ([]db.DomainListItem, int, error)
	DomainDetail(ctx context.Context, id string) (*db.DomainDetail, error)
	BootstrapLatest(ctx context.Context) (*db.BootstrapRun, error)
}

// noopClusterLister degrades /fingerprint to a clean no-match when
// DATABASE_URL isn't configured (e.g. local dev), instead of crashing the
// whole server — mirrors the vision-client "stub if no key" pattern below.
type noopClusterLister struct{}

func (noopClusterLister) ListClusters(ctx context.Context) ([]layer1.Cluster, error) {
	return nil, nil
}

// noopDomainRepository degrades every domains/blocklist/bootstrap endpoint
// to its documented empty state when DATABASE_URL isn't configured, instead
// of crashing the whole server.
type noopDomainRepository struct{}

func (noopDomainRepository) Blocklist(ctx context.Context, since *time.Time) ([]db.BlocklistDomain, error) {
	return nil, nil
}

func (noopDomainRepository) Check(ctx context.Context, domain string) (db.CheckResult, error) {
	return db.CheckResult{Status: "candidate"}, nil
}

func (noopDomainRepository) ReportFalsePositive(ctx context.Context, domainID, note string) error {
	return nil
}

func (noopDomainRepository) ListDomains(ctx context.Context, limit, offset int, source, status *string) ([]db.DomainListItem, int, error) {
	return nil, 0, nil
}

func (noopDomainRepository) DomainDetail(ctx context.Context, id string) (*db.DomainDetail, error) {
	return nil, nil
}

func (noopDomainRepository) BootstrapLatest(ctx context.Context) (*db.BootstrapRun, error) {
	return nil, nil
}

func newMux() http.Handler {
	vision := layer2.NewVisionClient(os.Getenv("GEMINI_API_KEY"))
	if model := os.Getenv("GEMINI_MODEL"); model != "" {
		vision.Model = model
	}
	store := newMemoryDomainStore()

	var clusters clusterLister = noopClusterLister{}
	var domains domainRepository = noopDomainRepository{}
	if pool, err := db.Connect(context.Background()); err != nil {
		log.Printf("db connect failed, /fingerprint and domain endpoints will report empty state: %v", err)
	} else {
		clusters = db.NewClusterRepository(pool)
		domains = db.NewDomainRepository(pool)
	}

	return newMuxWith(vision, store, layer1.NewExtractor(), clusters, domains)
}

// newMuxWith builds the router with explicit dependencies, so tests can
// inject a fake Gemini endpoint / store / fingerprint extractor / cluster
// source / domain repository without touching env vars or a live network/DB.
func newMuxWith(vision *layer2.VisionClient, store *memoryDomainStore, extractor fingerprintExtractor, clusters clusterLister, domains domainRepository) http.Handler {
	mux := http.NewServeMux()

	// PJ-506: real blocklist read. `since` filters to rows blocked after
	// that timestamp; an unparseable or absent value means "everything".
	mux.HandleFunc("GET /api/v1/blocklist", func(w http.ResponseWriter, r *http.Request) {
		var since *time.Time
		if raw := r.URL.Query().Get("since"); raw != "" {
			if t, err := time.Parse(time.RFC3339, raw); err == nil {
				since = &t
			}
		}

		rows, err := domains.Blocklist(r.Context(), since)
		if err != nil {
			log.Printf("blocklist query failed: %v", err)
			rows = nil
		}

		entries := make([]BlocklistEntry, 0, len(rows))
		for _, row := range rows {
			entries = append(entries, BlocklistEntry{
				Domain:        row.Domain,
				Confidence:    row.Confidence,
				Reason:        row.Reason,
				MatchedFields: row.MatchedFields,
			})
		}
		writeJSON(w, http.StatusOK, BlocklistResponse{
			Domains:   entries,
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		})
	})

	// PJ-506: an unknown domain is not an error — it reports "candidate",
	// the same shape as any domain not yet confirmed.
	mux.HandleFunc("POST /api/v1/check", func(w http.ResponseWriter, r *http.Request) {
		var body CheckRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
			return
		}

		result, err := domains.Check(r.Context(), body.Domain)
		if err != nil {
			writeJSON(w, http.StatusOK, CheckResponse{Status: "candidate"})
			return
		}
		writeJSON(w, http.StatusOK, CheckResponse{
			Status:     result.Status,
			Confidence: result.Confidence,
			Source:     result.Source,
			Reason:     result.Reason,
		})
	})

	mux.HandleFunc("POST /api/v1/analyze", func(w http.ResponseWriter, r *http.Request) {
		var body AnalyzeRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
			return
		}

		// No key configured (e.g. local dev without .env): stay on the
		// PJ-106 stub shape rather than fail every request.
		if vision.APIKey == "" {
			writeJSON(w, http.StatusOK, AnalyzeResponse{
				IsJudol:    false,
				Confidence: 0,
				Reason:     "stub",
				DomainID:   "00000000-0000-0000-0000-000000000000",
			})
			return
		}

		result, err := vision.Analyze(r.Context(), core.Evidence{
			Domain:       body.Domain,
			EvidenceB64:  body.EvidenceB64,
			EvidenceType: core.EvidenceScreenshot,
		})
		if err != nil {
			// PRD §14 risk #4: Gemini down must not take the endpoint down.
			log.Printf("layer2 analyze failed for %s: %v", body.Domain, err)
			writeJSON(w, http.StatusOK, AnalyzeResponse{
				IsJudol:    false,
				Confidence: 0,
				Reason:     "analisis gagal, coba lagi nanti",
				DomainID:   store.IDFor(body.Domain),
			})
			return
		}

		// PJ-204: feedback loop (PJ-301) fires in the background so it
		// never delays the response.
		go func() {
			if err := layer2.Decide(store, result); err != nil {
				log.Printf("layer2 decide failed for %s: %v", body.Domain, err)
			}
		}()

		writeJSON(w, http.StatusOK, AnalyzeResponse{
			IsJudol:    result.Verdict.IsJudol,
			Confidence: result.Verdict.Confidence,
			Reason:     result.Verdict.Reason,
			DomainID:   store.IDFor(body.Domain),
		})
	})

	// PJ-405: real extraction + matching behind the stub shape.
	mux.HandleFunc("POST /api/v1/fingerprint", func(w http.ResponseWriter, r *http.Request) {
		var body FingerprintRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
			return
		}

		noMatch := FingerprintResponse{MatchScore: 0, MatchedFields: []string{}}

		fp, err := extractor.Extract(r.Context(), body.Domain)
		if err != nil {
			writeJSON(w, http.StatusOK, noMatch)
			return
		}

		clusterList, err := clusters.ListClusters(r.Context())
		if err != nil {
			writeJSON(w, http.StatusOK, noMatch)
			return
		}

		result := layer1.Match(fp, clusterList)
		if result == nil {
			writeJSON(w, http.StatusOK, noMatch)
			return
		}

		clusterID := result.ClusterID
		tldCopy := fp.TLD
		writeJSON(w, http.StatusOK, FingerprintResponse{
			ClusterID:     &clusterID,
			Registrar:     fp.Registrar,
			IP:            fp.HostingIP,
			NS:            fp.Nameserver,
			TLD:           &tldCopy,
			MatchScore:    result.Score,
			MatchedFields: result.MatchedFields,
		})
	})

	// PJ-605: dashboard list, with optional source/status filters.
	mux.HandleFunc("GET /api/v1/domains", func(w http.ResponseWriter, r *http.Request) {
		limit := 20
		if raw := r.URL.Query().Get("limit"); raw != "" {
			if n, err := strconv.Atoi(raw); err == nil && n > 0 {
				limit = n
			}
		}
		offset := 0
		if raw := r.URL.Query().Get("offset"); raw != "" {
			if n, err := strconv.Atoi(raw); err == nil && n >= 0 {
				offset = n
			}
		}
		var source, status *string
		if v := r.URL.Query().Get("source"); v != "" {
			source = &v
		}
		if v := r.URL.Query().Get("status"); v != "" {
			status = &v
		}

		rows, total, err := domains.ListDomains(r.Context(), limit, offset, source, status)
		if err != nil {
			log.Printf("domains list query failed: %v", err)
			writeJSON(w, http.StatusOK, DomainListResponse{Items: []DomainListItem{}, Total: 0})
			return
		}

		items := make([]DomainListItem, 0, len(rows))
		for _, row := range rows {
			detectedAt := row.DetectedAt.UTC().Format(time.RFC3339)
			items = append(items, DomainListItem{
				ID:         row.ID,
				Domain:     row.Domain,
				Status:     row.Status,
				Source:     row.Source,
				Confidence: row.Confidence,
				DetectedAt: &detectedAt,
			})
		}
		writeJSON(w, http.StatusOK, DomainListResponse{Items: items, Total: total})
	})

	// PJ-605: cluster detail + siblings. Unknown id is a 404, unlike /check's
	// "candidate" fallback — the dashboard follows a link it expects to exist.
	mux.HandleFunc("GET /api/v1/domains/{id}", func(w http.ResponseWriter, r *http.Request) {
		detail, err := domains.DomainDetail(r.Context(), r.PathValue("id"))
		if err != nil {
			log.Printf("domain detail query failed: %v", err)
		}
		if detail == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "domain not found"})
			return
		}

		detections := make([]map[string]any, 0, len(detail.Detections))
		for _, d := range detail.Detections {
			detections = append(detections, map[string]any{
				"layer":        d.Layer,
				"confidence":   d.Confidence,
				"reason":       d.Reason,
				"evidence_url": d.EvidenceURL,
				"detected_at":  d.DetectedAt.UTC().Format(time.RFC3339),
			})
		}
		writeJSON(w, http.StatusOK, DomainDetailResponse{
			Domain:      detail.Domain,
			Detections:  detections,
			Whois:       detail.Whois,
			Cluster:     detail.Cluster,
			Siblings:    detail.Siblings,
			EvidenceURL: detail.EvidenceURL,
		})
	})

	// PJ-507: backs the block page's "Laporkan salah" button. No auth in the
	// MVP; always answers ok — a stuck-looking error is worse than a silent
	// no-op on a bad id (PRD §14 risk #14).
	mux.HandleFunc("POST /api/v1/report-false-positive", func(w http.ResponseWriter, r *http.Request) {
		var body ReportFalsePositiveRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
			return
		}
		note := ""
		if body.Note != nil {
			note = *body.Note
		}
		if err := domains.ReportFalsePositive(r.Context(), body.DomainID, note); err != nil {
			log.Printf("report-false-positive failed for %s: %v", body.DomainID, err)
		}
		writeJSON(w, http.StatusOK, OkResponse{Ok: true})
	})

	// PJ-704: the demo's opening frame. No runs yet must be a valid
	// all-zeros response, not an error.
	mux.HandleFunc("GET /api/v1/bootstrap/latest", func(w http.ResponseWriter, r *http.Request) {
		run, err := domains.BootstrapLatest(r.Context())
		if err != nil {
			log.Printf("bootstrap latest query failed: %v", err)
		}
		if run == nil {
			writeJSON(w, http.StatusOK, BootstrapLatestResponse{})
			return
		}
		ratio := 0.0
		if run.L2Confirmations > 0 {
			ratio = float64(run.L1PreemptiveCatches) / float64(run.L2Confirmations)
		}
		writeJSON(w, http.StatusOK, BootstrapLatestResponse{
			L2Confirmations:     run.L2Confirmations,
			L1PreemptiveCatches: run.L1PreemptiveCatches,
			L1Misses:            run.L1Misses,
			Ratio:               ratio,
		})
	})

	mux.HandleFunc("POST /api/v1/trustpositif/verify", func(w http.ResponseWriter, r *http.Request) {
		var body TrustPositifVerifyRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		writeJSON(w, http.StatusOK, TrustPositifVerifyResponse{Domain: body.Domain, IsBlocked: false})
	})

	return withCORS(mux)
}

// withCORS mirrors FastAPI's CORSMiddleware(allow_origins=["*"]): wildcard
// origin/methods/headers, and it answers preflight OPTIONS requests itself
// since http.ServeMux route patterns are method-specific and would 404 them.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	log.Println("PREJUDGE API listening on :8000")
	log.Fatal(http.ListenAndServe(":8000", newMux()))
}
