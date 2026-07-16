// Package layer2 implements Layer 2 (reactive) detection: page evidence in,
// a core.Verdict out via Gemini vision. This is Core's only AI call
// (PRD.md §4) — deliberately not the project's innovation claim, so no
// fine-tuning, no self-trained model. Imports nothing beyond the Go
// standard library (core/contract.go's seam rule applies to this package
// too: no Chrome APIs, no Supabase client).
package layer2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"prejudge/core"
)

const (
	defaultModel   = "gemini-2.0-flash"
	geminiEndpoint = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s"

	// PRD.md §4: "Apakah ini situs judi online? Ya/tidak, alasan singkat."
	// reason must render verbatim on the block page for Rina (PRD.md §3),
	// so it's requested in plain-language Indonesian, not developer jargon.
	visionPrompt = `Apakah ini situs judi online (judol)? Jawab HANYA dengan JSON, tanpa markdown, dengan format persis:
{"is_judol": true/false, "confidence": 0.0-1.0, "reason": "alasan singkat dalam Bahasa Indonesia"}`
)

// VisionClient calls Gemini's vision API to classify page evidence.
type VisionClient struct {
	APIKey     string
	Model      string
	HTTPClient *http.Client
	Endpoint   string // overrides the built endpoint; used by tests
}

func NewVisionClient(apiKey string) *VisionClient {
	return &VisionClient{
		APIKey:     apiKey,
		Model:      defaultModel,
		HTTPClient: &http.Client{Timeout: 8 * time.Second},
	}
}

// AnalyzeResult pairs the contract Verdict with the raw Gemini response body,
// which callers persist to detections.raw_response (PJ-201).
type AnalyzeResult struct {
	Verdict core.Verdict
	Raw     string
}

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text       string      `json:"text,omitempty"`
	InlineData *geminiBlob `json:"inline_data,omitempty"`
}

type geminiBlob struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

type visionVerdict struct {
	IsJudol    bool    `json:"is_judol"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
}

// Analyze sends page evidence to Gemini and returns a Verdict. A malformed
// or unparseable model response degrades to is_judol=false rather than
// erroring or crashing (PJ-201 acceptance) — the caller should still
// persist Raw to detections.raw_response for debugging.
func (c *VisionClient) Analyze(ctx context.Context, ev core.Evidence) (AnalyzeResult, error) {
	mime := "image/jpeg"
	if ev.EvidenceType != "" && ev.EvidenceType != core.EvidenceScreenshot {
		mime = "application/octet-stream"
	}

	reqBody := geminiRequest{Contents: []geminiContent{{Parts: []geminiPart{
		{Text: visionPrompt},
		{InlineData: &geminiBlob{MimeType: mime, Data: ev.EvidenceB64}},
	}}}}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return AnalyzeResult{}, fmt.Errorf("layer2: marshal request: %w", err)
	}

	endpoint := c.Endpoint
	if endpoint == "" {
		model := c.Model
		if model == "" {
			model = defaultModel
		}
		endpoint = fmt.Sprintf(geminiEndpoint, model, c.APIKey)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return AnalyzeResult{}, fmt.Errorf("layer2: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 8 * time.Second}
	}

	resp, err := client.Do(req)
	if err != nil {
		return AnalyzeResult{}, fmt.Errorf("layer2: gemini request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return AnalyzeResult{}, fmt.Errorf("layer2: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return AnalyzeResult{}, fmt.Errorf("layer2: gemini returned %d: %s", resp.StatusCode, string(body))
	}

	raw := string(body)

	var gr geminiResponse
	if err := json.Unmarshal(body, &gr); err != nil || len(gr.Candidates) == 0 || len(gr.Candidates[0].Content.Parts) == 0 {
		return AnalyzeResult{Verdict: malformedVerdict(ev.Domain), Raw: raw}, nil
	}

	text := stripMarkdownFences(gr.Candidates[0].Content.Parts[0].Text)

	var vv visionVerdict
	if err := json.Unmarshal([]byte(text), &vv); err != nil {
		return AnalyzeResult{Verdict: malformedVerdict(ev.Domain), Raw: raw}, nil
	}

	v := core.NewVerdict(ev.Domain, vv.IsJudol, vv.Confidence, vv.Reason)
	return AnalyzeResult{Verdict: v, Raw: raw}, nil
}

func malformedVerdict(domain string) core.Verdict {
	return core.NewVerdict(domain, false, 0, "gagal memproses hasil analisis")
}

func stripMarkdownFences(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
