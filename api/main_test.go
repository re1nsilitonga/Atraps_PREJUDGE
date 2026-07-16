package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"prejudge/core/layer2"
)

func doJSON(t *testing.T, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reqBody *strings.Reader
	if body == "" {
		reqBody = strings.NewReader("")
	} else {
		reqBody = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	newMux().ServeHTTP(rr, req)
	return rr
}

func decode(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON body: %v (%s)", err, rr.Body.String())
	}
	return body
}

func TestBlocklistEmptyIsValidNotError(t *testing.T) {
	rr := doJSON(t, http.MethodGet, "/api/v1/blocklist", "")
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := decode(t, rr)
	if len(body["domains"].([]any)) != 0 {
		t.Fatalf("expected empty domains, got %v", body["domains"])
	}
	if _, ok := body["updated_at"]; !ok {
		t.Fatal("missing updated_at")
	}
}

func TestCheckReturnsContractShape(t *testing.T) {
	rr := doJSON(t, http.MethodPost, "/api/v1/check", `{"domain":"unknown.test"}`)
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := decode(t, rr)
	for _, key := range []string{"status", "confidence", "source", "reason"} {
		if _, ok := body[key]; !ok {
			t.Fatalf("missing key %s in %v", key, body)
		}
	}
}

func TestAnalyzeReturnsDomainID(t *testing.T) {
	rr := doJSON(t, http.MethodPost, "/api/v1/analyze", `{"domain":"x.test","evidence_b64":"Zm9v"}`)
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := decode(t, rr)
	if _, ok := body["domain_id"]; !ok {
		t.Fatal("missing domain_id")
	}
}

func TestAnalyzeWithVisionClientAppliesThreshold(t *testing.T) {
	geminiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"{\"is_judol\":true,\"confidence\":0.95,\"reason\":\"slot UI, tombol deposit\"}"}]}}]}`))
	}))
	t.Cleanup(geminiSrv.Close)

	vision := layer2.NewVisionClient("test-key")
	vision.Endpoint = geminiSrv.URL
	store := newMemoryDomainStore()
	mux := newMuxWith(vision, store)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/analyze", strings.NewReader(`{"domain":"gacor88x.xyz","evidence_b64":"Zm9v"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["is_judol"] != true {
		t.Fatalf("expected is_judol true, got %v", body)
	}
	if body["confidence"].(float64) != 0.95 {
		t.Fatalf("expected confidence 0.95, got %v", body["confidence"])
	}

	// Feedback loop runs in a goroutine (PJ-204) — poll briefly for the
	// domain to land in the store instead of sleeping a fixed duration.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		store.mu.Lock()
		_, blocked := store.blocked["gacor88x.xyz"]
		store.mu.Unlock()
		if blocked {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("expected domain to be blocked via async feedback loop")
}

func TestFingerprintNoMatchIsCleanNot500(t *testing.T) {
	rr := doJSON(t, http.MethodPost, "/api/v1/fingerprint", `{"domain":"x.test"}`)
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := decode(t, rr)
	if body["cluster_id"] != nil {
		t.Fatalf("expected nil cluster_id, got %v", body["cluster_id"])
	}
}

func TestDomainsListEmptyState(t *testing.T) {
	rr := doJSON(t, http.MethodGet, "/api/v1/domains", "")
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := decode(t, rr)
	if len(body["items"].([]any)) != 0 || body["total"].(float64) != 0 {
		t.Fatalf("expected empty list, got %v", body)
	}
}

func TestDomainDetailHasSiblingsKey(t *testing.T) {
	rr := doJSON(t, http.MethodGet, "/api/v1/domains/some-id", "")
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := decode(t, rr)
	if _, ok := body["siblings"]; !ok {
		t.Fatal("missing siblings")
	}
}

func TestReportFalsePositiveOk(t *testing.T) {
	rr := doJSON(t, http.MethodPost, "/api/v1/report-false-positive", `{"domain_id":"some-id"}`)
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := decode(t, rr)
	if body["ok"] != true {
		t.Fatalf("expected ok:true, got %v", body)
	}
}

func TestBootstrapLatestZeroStateNotError(t *testing.T) {
	rr := doJSON(t, http.MethodGet, "/api/v1/bootstrap/latest", "")
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := decode(t, rr)
	if body["l2_confirmations"].(float64) != 0 || body["ratio"].(float64) != 0 {
		t.Fatalf("expected zero state, got %v", body)
	}
}

func TestTrustPositifVerifyEchoesDomain(t *testing.T) {
	rr := doJSON(t, http.MethodPost, "/api/v1/trustpositif/verify", `{"domain":"x.test"}`)
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := decode(t, rr)
	if body["domain"] != "x.test" {
		t.Fatalf("expected domain echoed, got %v", body)
	}
}

func TestCorsAllowsAnyOrigin(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/blocklist", nil)
	req.Header.Set("Origin", "chrome-extension://abc")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rr := httptest.NewRecorder()
	newMux().ServeHTTP(rr, req)
	if rr.Code != 204 {
		t.Fatalf("expected 204 for preflight, got %d", rr.Code)
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("expected wildcard CORS header, got %q", rr.Header().Get("Access-Control-Allow-Origin"))
	}
}
