package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"prejudge/core/layer1"
)

type fakeExtractor struct {
	fp  layer1.Fingerprint
	err error
}

func (f fakeExtractor) Extract(ctx context.Context, domain string) (layer1.Fingerprint, error) {
	return f.fp, f.err
}

type fakeClusterLister struct {
	clusters []layer1.Cluster
	err      error
}

func (f fakeClusterLister) ListClusters(ctx context.Context) ([]layer1.Cluster, error) {
	return f.clusters, f.err
}

func testDeps() apiDeps {
	return apiDeps{extractor: fakeExtractor{}, clusters: fakeClusterLister{}}
}

func doJSON(t *testing.T, deps apiDeps, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	newMux(deps).ServeHTTP(rr, req)
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
	rr := doJSON(t, testDeps(), http.MethodGet, "/api/v1/blocklist", "")
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := decode(t, rr)
	if len(body["domains"].([]any)) != 0 {
		t.Fatalf("expected empty domains, got %v", body["domains"])
	}
}

func TestCheckReturnsContractShape(t *testing.T) {
	rr := doJSON(t, testDeps(), http.MethodPost, "/api/v1/check", `{"domain":"unknown.test"}`)
	body := decode(t, rr)
	for _, key := range []string{"status", "confidence", "source", "reason"} {
		if _, ok := body[key]; !ok {
			t.Fatalf("missing key %s in %v", key, body)
		}
	}
}

func TestAnalyzeReturnsDomainID(t *testing.T) {
	rr := doJSON(t, testDeps(), http.MethodPost, "/api/v1/analyze", `{"domain":"x.test","evidence_b64":"Zm9v"}`)
	if _, ok := decode(t, rr)["domain_id"]; !ok {
		t.Fatal("missing domain_id")
	}
}

func TestFingerprintNoMatchIsCleanNot500(t *testing.T) {
	deps := apiDeps{
		extractor: fakeExtractor{fp: layer1.Fingerprint{Domain: "x.test", TLD: "test"}},
		clusters:  fakeClusterLister{clusters: nil},
	}
	rr := doJSON(t, deps, http.MethodPost, "/api/v1/fingerprint", `{"domain":"x.test"}`)
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := decode(t, rr)
	if body["cluster_id"] != nil {
		t.Fatalf("expected nil cluster_id, got %v", body["cluster_id"])
	}
}

func TestFingerprintMatchReturnsMatchedFields(t *testing.T) {
	ip := "203.0.113.10"
	burst := 0.9
	deps := apiDeps{
		extractor: fakeExtractor{fp: layer1.Fingerprint{Domain: "sib.test", HostingIP: &ip, TLD: "xyz"}},
		clusters: fakeClusterLister{clusters: []layer1.Cluster{
			{ID: "cluster-1", HostingIP: ip, TLD: "xyz", RegistrationBurstScore: &burst},
		}},
	}
	rr := doJSON(t, deps, http.MethodPost, "/api/v1/fingerprint", `{"domain":"sib.test"}`)
	body := decode(t, rr)
	if body["cluster_id"] != "cluster-1" {
		t.Fatalf("expected cluster-1, got %v", body["cluster_id"])
	}
	fields, _ := body["matched_fields"].([]any)
	if len(fields) == 0 {
		t.Fatal("expected non-empty matched_fields")
	}
}

func TestDomainsListEmptyState(t *testing.T) {
	rr := doJSON(t, testDeps(), http.MethodGet, "/api/v1/domains", "")
	body := decode(t, rr)
	if len(body["items"].([]any)) != 0 || body["total"].(float64) != 0 {
		t.Fatalf("expected empty list, got %v", body)
	}
}

func TestDomainDetailHasSiblingsKey(t *testing.T) {
	rr := doJSON(t, testDeps(), http.MethodGet, "/api/v1/domains/some-id", "")
	if _, ok := decode(t, rr)["siblings"]; !ok {
		t.Fatal("missing siblings")
	}
}

func TestReportFalsePositiveOk(t *testing.T) {
	rr := doJSON(t, testDeps(), http.MethodPost, "/api/v1/report-false-positive", `{"domain_id":"some-id"}`)
	if decode(t, rr)["ok"] != true {
		t.Fatalf("expected ok:true, got %v", decode(t, rr))
	}
}

func TestBootstrapLatestZeroStateNotError(t *testing.T) {
	rr := doJSON(t, testDeps(), http.MethodGet, "/api/v1/bootstrap/latest", "")
	body := decode(t, rr)
	if body["l2_confirmations"].(float64) != 0 || body["ratio"].(float64) != 0 {
		t.Fatalf("expected zero state, got %v", body)
	}
}

func TestTrustPositifVerifyEchoesDomain(t *testing.T) {
	rr := doJSON(t, testDeps(), http.MethodPost, "/api/v1/trustpositif/verify", `{"domain":"x.test"}`)
	if decode(t, rr)["domain"] != "x.test" {
		t.Fatalf("expected domain echoed, got %v", decode(t, rr))
	}
}

func TestCorsAllowsAnyOrigin(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/blocklist", nil)
	req.Header.Set("Origin", "chrome-extension://abc")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rr := httptest.NewRecorder()
	newMux(testDeps()).ServeHTTP(rr, req)
	if rr.Code != 204 {
		t.Fatalf("expected 204 for preflight, got %d", rr.Code)
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("expected wildcard CORS header, got %q", rr.Header().Get("Access-Control-Allow-Origin"))
	}
}
