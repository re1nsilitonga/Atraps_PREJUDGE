package layer2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"prejudge/core"
)

func TestAnalyzeCachedWritesCacheOnSuccess(t *testing.T) {
	srv := geminiServer(t, `{"is_judol":true,"confidence":0.9,"reason":"slot UI"}`)
	c := NewVisionClient("test-key")
	c.Endpoint = srv.URL
	c.CacheDir = t.TempDir()

	result, err := c.AnalyzeCached(context.Background(), core.Evidence{Domain: "gacor88x.xyz", EvidenceB64: "Zm9v"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Verdict.IsJudol {
		t.Fatal("expected IsJudol true")
	}

	path := c.cacheFilePath("gacor88x.xyz")
	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("expected cache file to be written at %s: %v", path, statErr)
	}
}

func TestAnalyzeCachedServesCacheOnLiveFailure(t *testing.T) {
	cacheDir := t.TempDir()

	okSrv := geminiServer(t, `{"is_judol":true,"confidence":0.9,"reason":"slot UI"}`)
	c := NewVisionClient("test-key")
	c.Endpoint = okSrv.URL
	c.CacheDir = cacheDir

	if _, err := c.AnalyzeCached(context.Background(), core.Evidence{Domain: "gacor88x.xyz", EvidenceB64: "Zm9v"}); err != nil {
		t.Fatalf("unexpected error priming cache: %v", err)
	}

	failSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(failSrv.Close)
	c.Endpoint = failSrv.URL

	result, err := c.AnalyzeCached(context.Background(), core.Evidence{Domain: "gacor88x.xyz", EvidenceB64: "Zm9v"})
	if err != nil {
		t.Fatalf("expected cached fallback to suppress error, got %v", err)
	}
	if !result.Verdict.IsJudol {
		t.Fatal("expected cached IsJudol true to be served on live failure")
	}
}

func TestAnalyzeCachedNoCacheDirPropagatesError(t *testing.T) {
	failSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(failSrv.Close)

	c := NewVisionClient("test-key")
	c.Endpoint = failSrv.URL

	_, err := c.AnalyzeCached(context.Background(), core.Evidence{Domain: "x.test", EvidenceB64: "Zm9v"})
	if err == nil {
		t.Fatal("expected error when no cache is configured and the live call fails")
	}
}
