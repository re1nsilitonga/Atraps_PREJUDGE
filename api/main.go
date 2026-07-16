// Thin HTTP transport over core/. No detection logic lives here.
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"prejudge/core/layer1"
	"prejudge/db"
)

type fingerprintExtractor interface {
	Extract(ctx context.Context, domain string) (layer1.Fingerprint, error)
}

type clusterLister interface {
	ListClusters(ctx context.Context) ([]layer1.Cluster, error)
}

type apiDeps struct {
	extractor fingerprintExtractor
	clusters  clusterLister
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func newMux(deps apiDeps) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/blocklist", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, BlocklistResponse{
			Domains:   []BlocklistEntry{},
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		})
	})

	mux.HandleFunc("POST /api/v1/check", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, CheckResponse{Status: "candidate"})
	})

	mux.HandleFunc("POST /api/v1/analyze", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, AnalyzeResponse{
			IsJudol:    false,
			Confidence: 0,
			Reason:     "stub",
			DomainID:   "00000000-0000-0000-0000-000000000000",
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

		fp, err := deps.extractor.Extract(r.Context(), body.Domain)
		if err != nil {
			writeJSON(w, http.StatusOK, noMatch)
			return
		}

		clusters, err := deps.clusters.ListClusters(r.Context())
		if err != nil {
			writeJSON(w, http.StatusOK, noMatch)
			return
		}

		result := layer1.Match(fp, clusters)
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

	mux.HandleFunc("GET /api/v1/domains", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, DomainListResponse{Items: []DomainListItem{}, Total: 0})
	})

	mux.HandleFunc("GET /api/v1/domains/{id}", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, DomainDetailResponse{
			Domain:     "stub.test",
			Detections: []map[string]any{},
			Siblings:   []string{},
		})
	})

	mux.HandleFunc("POST /api/v1/report-false-positive", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, OkResponse{Ok: true})
	})

	mux.HandleFunc("GET /api/v1/bootstrap/latest", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, BootstrapLatestResponse{})
	})

	mux.HandleFunc("POST /api/v1/trustpositif/verify", func(w http.ResponseWriter, r *http.Request) {
		var body TrustPositifVerifyRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		writeJSON(w, http.StatusOK, TrustPositifVerifyResponse{Domain: body.Domain, IsBlocked: false})
	})

	return withCORS(mux)
}

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
	ctx := context.Background()
	pool, err := db.Connect(ctx)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	deps := apiDeps{
		extractor: layer1.NewExtractor(),
		clusters:  db.NewClusterRepository(pool),
	}

	log.Println("PREJUDGE API listening on :8000")
	log.Fatal(http.ListenAndServe(":8000", newMux(deps)))
}
