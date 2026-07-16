// Thin HTTP transport over core/. No detection logic lives here.
//
// PJ-106: all 9 routes return contract-shaped stub responses until the real
// Core functions (Epic 2 / Epic 4) are wired in behind them.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func newMux() http.Handler {
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

	mux.HandleFunc("POST /api/v1/fingerprint", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, FingerprintResponse{MatchScore: 0, MatchedFields: []string{}})
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
