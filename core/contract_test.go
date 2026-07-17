package core

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

func TestNewVerdictDefaults(t *testing.T) {
	v := NewVerdict("gacor88x.xyz", true, 0.92, "slot UI, tombol deposit")
	if len(v.MatchedFields) != 0 {
		t.Fatalf("expected empty MatchedFields, got %v", v.MatchedFields)
	}
	if v.Source != SourceL2 {
		t.Fatalf("expected default source L2, got %v", v.Source)
	}
	if v.DetectedAt.IsZero() {
		t.Fatal("expected DetectedAt to be set")
	}
}

func TestEvidenceFields(t *testing.T) {
	e := Evidence{Domain: "x.xyz", EvidenceB64: "Zm9v", EvidenceType: EvidenceScreenshot}
	if e.EvidenceType != EvidenceScreenshot {
		t.Fatalf("expected screenshot, got %v", e.EvidenceType)
	}
	if e.Domain != "x.xyz" {
		t.Fatalf("expected domain x.xyz, got %v", e.Domain)
	}
}

func TestContractImportsStdlibOnly(t *testing.T) {
	src, err := os.ReadFile("contract.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "contract.go", src, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if strings.Contains(path, ".") {
			t.Fatalf("disallowed non-stdlib import: %s", path)
		}
	}
}
