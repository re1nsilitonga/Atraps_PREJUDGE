package layer2

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"prime/core"
)

func geminiServer(t *testing.T, text string) *httptest.Server {
	t.Helper()
	quoted, err := json.Marshal(text)
	if err != nil {
		t.Fatalf("marshal fixture text: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":` + string(quoted) + `}]}}]}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestAnalyzeParsesJudolVerdict(t *testing.T) {
	srv := geminiServer(t, `{"is_judol": true, "confidence": 0.92, "reason": "slot UI, tombol deposit"}`)
	c := NewVisionClient("test-key")
	c.Endpoint = srv.URL

	result, err := c.Analyze(context.Background(), core.Evidence{Domain: "gacor88x.xyz", EvidenceB64: "Zm9v", EvidenceType: core.EvidenceScreenshot})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Verdict.IsJudol {
		t.Fatal("expected IsJudol true")
	}
	if result.Verdict.Confidence != 0.92 {
		t.Fatalf("expected confidence 0.92, got %v", result.Verdict.Confidence)
	}
	if result.Verdict.Reason == "" {
		t.Fatal("expected non-empty reason")
	}
	if result.Verdict.Source != core.SourceL2 {
		t.Fatalf("expected source L2, got %v", result.Verdict.Source)
	}
	if result.Raw == "" {
		t.Fatal("expected raw response retained")
	}
}

func TestAnalyzeStripsMarkdownFences(t *testing.T) {
	srv := geminiServer(t, "```json\n{\"is_judol\": false, \"confidence\": 0.1, \"reason\": \"situs berita\"}\n```")
	c := NewVisionClient("test-key")
	c.Endpoint = srv.URL

	result, err := c.Analyze(context.Background(), core.Evidence{Domain: "berita.test", EvidenceB64: "Zm9v"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Verdict.IsJudol {
		t.Fatal("expected IsJudol false")
	}
}

func TestAnalyzeMalformedResponseDegradesGracefully(t *testing.T) {
	srv := geminiServer(t, "not json at all")
	c := NewVisionClient("test-key")
	c.Endpoint = srv.URL

	result, err := c.Analyze(context.Background(), core.Evidence{Domain: "x.test", EvidenceB64: "Zm9v"})
	if err != nil {
		t.Fatalf("expected no error on malformed response, got %v", err)
	}
	if result.Verdict.IsJudol {
		t.Fatal("expected IsJudol false on malformed response")
	}
	if result.Raw == "" {
		t.Fatal("expected raw response retained even when malformed")
	}
}

func TestAnalyzeHTTPErrorReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	c := NewVisionClient("test-key")
	c.Endpoint = srv.URL

	_, err := c.Analyze(context.Background(), core.Evidence{Domain: "x.test", EvidenceB64: "Zm9v"})
	if err == nil {
		t.Fatal("expected error on non-200 response")
	}
}
