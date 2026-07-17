package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"prime/core/layer1"
	"prime/core/layer2"
	"prime/db"
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

type fakeClusterStore struct{}

func (fakeClusterStore) Upsert(ctx context.Context, c layer1.Cluster) (string, error) {
	return "", nil
}

type fakeDomainRepository struct {
	blocklist    []db.BlocklistDomain
	blocklistErr error
	sinceSeen    *time.Time

	check    db.CheckResult
	checkErr error

	reportErr        error
	reportedDomainID string

	listItems []db.DomainListItem
	listTotal int
	listErr   error
	statusArg *string
	sourceArg *string

	detail    *db.DomainDetail
	detailErr error

	bootstrap    *db.BootstrapRun
	bootstrapErr error
}

func (f *fakeDomainRepository) Blocklist(ctx context.Context, since *time.Time) ([]db.BlocklistDomain, error) {
	f.sinceSeen = since
	return f.blocklist, f.blocklistErr
}

func (f *fakeDomainRepository) Check(ctx context.Context, domain string) (db.CheckResult, error) {
	return f.check, f.checkErr
}

func (f *fakeDomainRepository) ListCandidates(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (f *fakeDomainRepository) ReportFalsePositive(ctx context.Context, domainID, note string) error {
	f.reportedDomainID = domainID
	return f.reportErr
}

func (f *fakeDomainRepository) ListDomains(ctx context.Context, limit, offset int, source, status *string) ([]db.DomainListItem, int, error) {
	f.sourceArg = source
	f.statusArg = status
	return f.listItems, f.listTotal, f.listErr
}

func (f *fakeDomainRepository) DomainDetail(ctx context.Context, id string) (*db.DomainDetail, error) {
	return f.detail, f.detailErr
}

func (f *fakeDomainRepository) BootstrapLatest(ctx context.Context) (*db.BootstrapRun, error) {
	return f.bootstrap, f.bootstrapErr
}

func defaultMux() http.Handler {
	return newMuxWith(layer2.NewVisionClient(""), newMemoryDomainStore(), fakeExtractor{}, fakeClusterLister{}, fakeClusterStore{}, &fakeDomainRepository{}, newRealtimeHub())
}

func doJSON(t *testing.T, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	return doJSONWith(t, defaultMux(), method, path, body)
}

func doJSONWith(t *testing.T, mux http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
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

func TestBlocklistReturnsEntriesWithMatchedFields(t *testing.T) {
	domains := &fakeDomainRepository{blocklist: []db.BlocklistDomain{
		{ID: "d1", Domain: "gacor88x.xyz", Confidence: 0.92, Reason: "slot UI", MatchedFields: []string{"hosting_ip", "nameserver"}},
	}}
	mux := newMuxWith(layer2.NewVisionClient(""), newMemoryDomainStore(), fakeExtractor{}, fakeClusterLister{}, fakeClusterStore{}, domains, newRealtimeHub())

	rr := doJSONWith(t, mux, http.MethodGet, "/api/v1/blocklist", "")
	body := decode(t, rr)
	entries := body["domains"].([]any)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	entry := entries[0].(map[string]any)
	if entry["domain"] != "gacor88x.xyz" {
		t.Fatalf("expected gacor88x.xyz, got %v", entry["domain"])
	}
	if entry["id"] != "d1" {
		t.Fatalf("expected id d1 (needed by blocked.html's report-false-positive flow), got %v", entry["id"])
	}
	fields := entry["matched_fields"].([]any)
	if len(fields) != 2 {
		t.Fatalf("expected 2 matched_fields, got %v", fields)
	}
}

func TestBlocklistSinceQueryParamIsParsedAndForwarded(t *testing.T) {
	domains := &fakeDomainRepository{}
	mux := newMuxWith(layer2.NewVisionClient(""), newMemoryDomainStore(), fakeExtractor{}, fakeClusterLister{}, fakeClusterStore{}, domains, newRealtimeHub())

	doJSONWith(t, mux, http.MethodGet, "/api/v1/blocklist?since=2026-07-01T00:00:00Z", "")
	if domains.sinceSeen == nil {
		t.Fatal("expected since to be parsed and forwarded, got nil")
	}
	want := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	if !domains.sinceSeen.Equal(want) {
		t.Fatalf("expected %v, got %v", want, *domains.sinceSeen)
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

func TestCheckKnownDomainReturnsStoredStatus(t *testing.T) {
	confidence := 0.91
	source := "L1"
	reason := "IP hosting sama"
	domains := &fakeDomainRepository{check: db.CheckResult{Status: "blocked", Confidence: &confidence, Source: &source, Reason: &reason}}
	mux := newMuxWith(layer2.NewVisionClient(""), newMemoryDomainStore(), fakeExtractor{}, fakeClusterLister{}, fakeClusterStore{}, domains, newRealtimeHub())

	rr := doJSONWith(t, mux, http.MethodPost, "/api/v1/check", `{"domain":"gacor88x.xyz"}`)
	body := decode(t, rr)
	if body["status"] != "blocked" {
		t.Fatalf("expected blocked, got %v", body["status"])
	}
	if body["confidence"].(float64) != 0.91 {
		t.Fatalf("expected 0.91, got %v", body["confidence"])
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
	mux := newMuxWith(vision, store, fakeExtractor{}, fakeClusterLister{}, fakeClusterStore{}, &fakeDomainRepository{}, newRealtimeHub())

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

func TestFingerprintMatchReturnsMatchedFields(t *testing.T) {
	ip := "203.0.113.10"
	burst := 0.9
	extractor := fakeExtractor{fp: layer1.Fingerprint{Domain: "sib.test", HostingIP: &ip, TLD: "xyz"}}
	clusters := fakeClusterLister{clusters: []layer1.Cluster{
		{ID: "cluster-1", HostingIP: ip, TLD: "xyz", RegistrationBurstScore: &burst},
	}}
	mux := newMuxWith(layer2.NewVisionClient(""), newMemoryDomainStore(), extractor, clusters, fakeClusterStore{}, &fakeDomainRepository{}, newRealtimeHub())

	rr := doJSONWith(t, mux, http.MethodPost, "/api/v1/fingerprint", `{"domain":"sib.test"}`)
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
	rr := doJSON(t, http.MethodGet, "/api/v1/domains", "")
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := decode(t, rr)
	if len(body["items"].([]any)) != 0 || body["total"].(float64) != 0 {
		t.Fatalf("expected empty list, got %v", body)
	}
}

func TestDomainsListReturnsItemsAndTotal(t *testing.T) {
	confidence := 0.8
	domains := &fakeDomainRepository{
		listItems: []db.DomainListItem{{ID: "id-1", Domain: "gacor88x.xyz", Status: "blocked", Confidence: &confidence}},
		listTotal: 42,
	}
	mux := newMuxWith(layer2.NewVisionClient(""), newMemoryDomainStore(), fakeExtractor{}, fakeClusterLister{}, fakeClusterStore{}, domains, newRealtimeHub())

	rr := doJSONWith(t, mux, http.MethodGet, "/api/v1/domains", "")
	body := decode(t, rr)
	if body["total"].(float64) != 42 {
		t.Fatalf("expected total 42, got %v", body["total"])
	}
	items := body["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["domain"] != "gacor88x.xyz" {
		t.Fatalf("expected 1 item for gacor88x.xyz, got %v", items)
	}
}

func TestDomainsListFiltersForwardedToRepository(t *testing.T) {
	domains := &fakeDomainRepository{}
	mux := newMuxWith(layer2.NewVisionClient(""), newMemoryDomainStore(), fakeExtractor{}, fakeClusterLister{}, fakeClusterStore{}, domains, newRealtimeHub())

	doJSONWith(t, mux, http.MethodGet, "/api/v1/domains?source=L1&status=blocked", "")
	if domains.sourceArg == nil || *domains.sourceArg != "L1" {
		t.Fatalf("expected source filter L1, got %v", domains.sourceArg)
	}
	if domains.statusArg == nil || *domains.statusArg != "blocked" {
		t.Fatalf("expected status filter blocked, got %v", domains.statusArg)
	}
}

func TestDomainDetailUnknownIdIs404(t *testing.T) {
	rr := doJSON(t, http.MethodGet, "/api/v1/domains/does-not-exist", "")
	if rr.Code != 404 {
		t.Fatalf("expected 404 for unknown id, got %d", rr.Code)
	}
}

func TestDomainDetailFoundHasSiblingsKey(t *testing.T) {
	domains := &fakeDomainRepository{detail: &db.DomainDetail{
		Domain:   "gacor88x.xyz",
		Siblings: []string{"sib1.test", "sib2.test"},
	}}
	mux := newMuxWith(layer2.NewVisionClient(""), newMemoryDomainStore(), fakeExtractor{}, fakeClusterLister{}, fakeClusterStore{}, domains, newRealtimeHub())

	rr := doJSONWith(t, mux, http.MethodGet, "/api/v1/domains/some-id", "")
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := decode(t, rr)
	siblings := body["siblings"].([]any)
	if len(siblings) != 2 {
		t.Fatalf("expected 2 siblings, got %v", siblings)
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

func TestReportFalsePositiveForwardsDomainID(t *testing.T) {
	domains := &fakeDomainRepository{}
	mux := newMuxWith(layer2.NewVisionClient(""), newMemoryDomainStore(), fakeExtractor{}, fakeClusterLister{}, fakeClusterStore{}, domains, newRealtimeHub())

	doJSONWith(t, mux, http.MethodPost, "/api/v1/report-false-positive", `{"domain_id":"abc-123","note":"legit site"}`)
	if domains.reportedDomainID != "abc-123" {
		t.Fatalf("expected domain id abc-123 forwarded, got %q", domains.reportedDomainID)
	}
}

func TestReportFalsePositiveStillOkOnRepositoryError(t *testing.T) {
	domains := &fakeDomainRepository{reportErr: context.DeadlineExceeded}
	mux := newMuxWith(layer2.NewVisionClient(""), newMemoryDomainStore(), fakeExtractor{}, fakeClusterLister{}, fakeClusterStore{}, domains, newRealtimeHub())

	rr := doJSONWith(t, mux, http.MethodPost, "/api/v1/report-false-positive", `{"domain_id":"abc-123"}`)
	if rr.Code != 200 {
		t.Fatalf("expected 200 even on repository error (PRD §14 risk #14), got %d", rr.Code)
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

func TestBootstrapLatestComputesRatio(t *testing.T) {
	domains := &fakeDomainRepository{bootstrap: &db.BootstrapRun{L2Confirmations: 5, L1PreemptiveCatches: 2, L1Misses: 1}}
	mux := newMuxWith(layer2.NewVisionClient(""), newMemoryDomainStore(), fakeExtractor{}, fakeClusterLister{}, fakeClusterStore{}, domains, newRealtimeHub())

	rr := doJSONWith(t, mux, http.MethodGet, "/api/v1/bootstrap/latest", "")
	body := decode(t, rr)
	if body["l2_confirmations"].(float64) != 5 {
		t.Fatalf("expected 5 confirmations, got %v", body["l2_confirmations"])
	}
	if body["ratio"].(float64) != 0.4 {
		t.Fatalf("expected ratio 0.4 (2/5), got %v", body["ratio"])
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
	defaultMux().ServeHTTP(rr, req)
	if rr.Code != 204 {
		t.Fatalf("expected 204 for preflight, got %d", rr.Code)
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("expected wildcard CORS header, got %q", rr.Header().Get("Access-Control-Allow-Origin"))
	}
}
