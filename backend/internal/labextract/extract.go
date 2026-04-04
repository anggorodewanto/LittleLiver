// Package labextract provides lab result extraction from photos using the Claude Vision API.
package labextract

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// MaxPhotoKeys is the maximum number of photo keys allowed per extraction request.
const MaxPhotoKeys = 10

// ImageData holds raw image bytes and their content type.
type ImageData struct {
	Data        []byte
	ContentType string
}

// ExtractedResult represents a single lab result extracted from a photo.
type ExtractedResult struct {
	TestName      string         `json:"test_name"`
	Value         string         `json:"value"`
	Unit          string         `json:"unit"`
	NormalRange   string         `json:"normal_range"`
	Confidence    string         `json:"confidence"`
	ExistingMatch *ExistingMatch `json:"existing_match,omitempty"`
}

// ExistingMatch represents a matching existing lab result in the database.
type ExistingMatch struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Value     string `json:"value"`
	Unit      string `json:"unit"`
}

// ExtractionResponse is the full response from the extraction endpoint.
type ExtractionResponse struct {
	Extracted []ExtractedResult `json:"extracted"`
	Notes     string            `json:"notes"`
}

// ClaudeClient is the interface for communicating with the Claude Vision API.
type ClaudeClient interface {
	ExtractLabResults(ctx context.Context, images []ImageData, prompt string) (string, error)
}

// Service orchestrates lab result extraction.
type Service struct {
	client ClaudeClient
}

// NewService creates a new extraction service.
func NewService(client ClaudeClient) *Service {
	return &Service{client: client}
}

// Extract sends images to the Claude API and returns deduplicated extracted results.
func (s *Service) Extract(ctx context.Context, images []ImageData) ([]ExtractedResult, string, error) {
	raw, err := s.client.ExtractLabResults(ctx, images, ExtractionPrompt())
	if err != nil {
		return nil, "", fmt.Errorf("claude API: %w", err)
	}

	results, notes, err := ParseExtractionResponse(raw)
	if err != nil {
		return nil, "", fmt.Errorf("parse response: %w", err)
	}

	return DeduplicateResults(results), notes, nil
}

// rawExtractionResponse is used for parsing the Claude API response which may be
// an object with "extracted" and "notes" fields.
type rawExtractionResponse struct {
	Extracted []ExtractedResult `json:"extracted"`
	Notes     string            `json:"notes"`
}

// ParseExtractionResponse parses the raw JSON response from Claude into ExtractedResults
// and optional notes. Supports both object format {"extracted": [...], "notes": "..."} and
// plain array format [...] for robustness.
func ParseExtractionResponse(raw string) ([]ExtractedResult, string, error) {
	// Strip markdown code fences if present
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "```") {
		lines := strings.SplitN(trimmed, "\n", 2)
		if len(lines) == 2 {
			trimmed = lines[1]
		}
		if idx := strings.LastIndex(trimmed, "```"); idx >= 0 {
			trimmed = trimmed[:idx]
		}
		trimmed = strings.TrimSpace(trimmed)
	}

	// Try object format first ({"extracted": [...], "notes": "..."})
	if strings.HasPrefix(trimmed, "{") {
		var resp rawExtractionResponse
		if err := json.Unmarshal([]byte(trimmed), &resp); err != nil {
			return nil, "", fmt.Errorf("invalid JSON: %w", err)
		}
		return resp.Extracted, resp.Notes, nil
	}

	// Fall back to plain array format
	var results []ExtractedResult
	if err := json.Unmarshal([]byte(trimmed), &results); err != nil {
		return nil, "", fmt.Errorf("invalid JSON: %w", err)
	}
	return results, "", nil
}

// DeduplicateResults removes duplicate test names (case-insensitive), keeping the last occurrence.
func DeduplicateResults(results []ExtractedResult) []ExtractedResult {
	seen := make(map[string]int) // lowercase test_name -> index in output
	var out []ExtractedResult

	for _, r := range results {
		key := strings.ToLower(r.TestName)
		if idx, ok := seen[key]; ok {
			out[idx] = r
		} else {
			seen[key] = len(out)
			out = append(out, r)
		}
	}
	return out
}

// ValidatePhotoKeys checks that the photo keys list is valid.
func ValidatePhotoKeys(keys []string) error {
	if len(keys) == 0 {
		return fmt.Errorf("at least one photo key is required")
	}
	if len(keys) > MaxPhotoKeys {
		return fmt.Errorf("maximum %d photo keys allowed, got %d", MaxPhotoKeys, len(keys))
	}
	return nil
}

// ExtractionPrompt returns the system prompt used for Claude Vision extraction.
func ExtractionPrompt() string {
	return `You are a medical lab result extraction assistant. Analyze the provided lab report image(s) and extract all lab test results.

Return ONLY a JSON object (no markdown, no explanation) with these fields:
- "extracted": An array where each element has:
  - "test_name": The standardized test name (use these when applicable: total_bilirubin, direct_bilirubin, ALT, AST, GGT, albumin, INR, platelets). For other tests, use the name as shown on the report.
  - "value": The numeric or text result value as a string.
  - "unit": The unit of measurement (e.g., "mg/dL", "U/L").
  - "normal_range": The reference/normal range if shown (e.g., "7-56"), or empty string if not available.
  - "confidence": Your confidence level in the extraction accuracy: "high", "medium", or "low".
- "notes": Optional free-text context from the report (e.g., sample collection date, lab name). Empty string if nothing notable.

If multiple pages show the same test, include all occurrences (deduplication happens downstream).

Example output:
{
  "extracted": [
    {"test_name": "ALT", "value": "45", "unit": "U/L", "normal_range": "7-56", "confidence": "high"},
    {"test_name": "total_bilirubin", "value": "1.2", "unit": "mg/dL", "normal_range": "0.1-1.2", "confidence": "high"}
  ],
  "notes": "Sample collected 2026-03-15"
}`
}
