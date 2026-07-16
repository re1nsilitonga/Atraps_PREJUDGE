package layer2

import (
	"encoding/json"
	"testing"

	"prejudge/core"
)

type fakeStore struct {
	blocked    []core.Verdict
	detections int
}

func (f *fakeStore) UpsertBlocked(v core.Verdict) error {
	f.blocked = append(f.blocked, v)
	return nil
}

func (f *fakeStore) LogDetection(domain string, layer int, confidence float64, reason string, raw json.RawMessage) error {
	f.detections++
	return nil
}

func TestDecideAboveThresholdBlocksDomain(t *testing.T) {
	store := &fakeStore{}
	result := AnalyzeResult{
		Verdict: core.NewVerdict("gacor88x.xyz", true, 0.92, "slot UI"),
		Raw:     `{"is_judol":true}`,
	}

	if err := Decide(store, result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.blocked) != 1 {
		t.Fatalf("expected 1 blocked domain, got %d", len(store.blocked))
	}
	if store.detections != 1 {
		t.Fatalf("expected detection logged, got %d", store.detections)
	}
}

func TestDecideBelowThresholdOnlyLogs(t *testing.T) {
	store := &fakeStore{}
	result := AnalyzeResult{
		Verdict: core.NewVerdict("maybe.test", true, 0.5, "kurang yakin"),
		Raw:     `{"is_judol":true}`,
	}

	if err := Decide(store, result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.blocked) != 0 {
		t.Fatalf("expected no blocked domains below threshold, got %d", len(store.blocked))
	}
	if store.detections != 1 {
		t.Fatalf("expected detection logged even below threshold, got %d", store.detections)
	}
}

func TestDecideNotJudolNeverBlocks(t *testing.T) {
	store := &fakeStore{}
	result := AnalyzeResult{
		Verdict: core.NewVerdict("berita.test", false, 0.99, "situs berita"),
		Raw:     `{"is_judol":false}`,
	}

	if err := Decide(store, result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.blocked) != 0 {
		t.Fatalf("expected no blocked domains when is_judol=false, got %d", len(store.blocked))
	}
}
