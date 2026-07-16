package docs

import (
	"os"
	"strings"
	"testing"
)

func readReadme(t *testing.T) string {
	t.Helper()
	src, err := os.ReadFile("../README.md")
	if err != nil {
		t.Fatal(err)
	}
	return string(src)
}

var endpoints = []string{
	"GET `/blocklist`",
	"POST `/check`",
	"POST `/analyze`",
	"POST `/fingerprint`",
	"GET `/domains`",
	"GET `/domains/{id}`",
	"POST `/report-false-positive`",
	"GET `/bootstrap/latest`",
	"POST `/trustpositif/verify`",
}

func TestAllNineEndpointsDocumented(t *testing.T) {
	readme := readReadme(t)
	for _, entry := range endpoints {
		parts := strings.SplitN(entry, " ", 2)
		method, path := parts[0], parts[1]
		if !strings.Contains(readme, method) || !strings.Contains(readme, path) {
			t.Fatalf("README missing %s", entry)
		}
	}
}

func TestModuleBoundaryDocumented(t *testing.T) {
	readme := readReadme(t)
	for _, word := range []string{"Core", "Blocker", "Presentation"} {
		if !strings.Contains(readme, word) {
			t.Fatalf("README missing module boundary term %s", word)
		}
	}
}

func TestEnvVarsDocumented(t *testing.T) {
	readme := readReadme(t)
	for _, v := range []string{"SUPABASE_URL", "SUPABASE_ANON_KEY", "SUPABASE_SERVICE_ROLE_KEY", "GEMINI_API_KEY"} {
		if !strings.Contains(readme, v) {
			t.Fatalf("README missing env var %s", v)
		}
	}
}
