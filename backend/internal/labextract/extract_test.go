package labextract

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// mockClaudeClient implements ClaudeClient for testing.
type mockClaudeClient struct {
	response string
	err      error
}

func (m *mockClaudeClient) ExtractLabResults(ctx context.Context, images []ImageData, prompt string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func TestParseExtractionResponse_ValidArrayJSON(t *testing.T) {
	t.Parallel()

	raw := `[
		{"test_name": "ALT", "value": "45", "unit": "U/L", "normal_range": "7-56", "confidence": "high"},
		{"test_name": "AST", "value": "32", "unit": "U/L", "normal_range": "10-40", "confidence": "medium"}
	]`

	results, notes, err := ParseExtractionResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notes != "" {
		t.Errorf("expected empty notes for array format, got %q", notes)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].TestName != "ALT" {
		t.Errorf("expected test_name ALT, got %s", results[0].TestName)
	}
	if results[0].Value != "45" {
		t.Errorf("expected value 45, got %s", results[0].Value)
	}
	if results[0].Unit != "U/L" {
		t.Errorf("expected unit U/L, got %s", results[0].Unit)
	}
	if results[0].NormalRange != "7-56" {
		t.Errorf("expected normal_range 7-56, got %s", results[0].NormalRange)
	}
	if results[0].Confidence != "high" {
		t.Errorf("expected confidence high, got %s", results[0].Confidence)
	}
}

func TestParseExtractionResponse_ObjectFormat(t *testing.T) {
	t.Parallel()

	raw := `{
		"extracted": [{"test_name": "ALT", "value": "45", "unit": "U/L", "normal_range": "7-56", "confidence": "high"}],
		"notes": "Sample collected 2026-03-15"
	}`

	results, notes, err := ParseExtractionResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notes != "Sample collected 2026-03-15" {
		t.Errorf("expected notes, got %q", notes)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].TestName != "ALT" {
		t.Errorf("expected ALT, got %s", results[0].TestName)
	}
}

func TestParseExtractionResponse_InvalidJSON(t *testing.T) {
	t.Parallel()

	_, _, err := ParseExtractionResponse("not json")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestDeduplicateResults(t *testing.T) {
	t.Parallel()

	results := []ExtractedResult{
		{TestName: "ALT", Value: "45", Unit: "U/L", Confidence: "medium"},
		{TestName: "AST", Value: "32", Unit: "U/L", Confidence: "high"},
		{TestName: "ALT", Value: "48", Unit: "U/L", Confidence: "high"},
	}

	deduped := DeduplicateResults(results)
	if len(deduped) != 2 {
		t.Fatalf("expected 2 results after dedup, got %d", len(deduped))
	}

	var alt *ExtractedResult
	for i := range deduped {
		if deduped[i].TestName == "ALT" {
			alt = &deduped[i]
			break
		}
	}
	if alt == nil {
		t.Fatal("ALT not found in deduplicated results")
	}
	if alt.Value != "48" {
		t.Errorf("expected ALT value 48 (last occurrence), got %s", alt.Value)
	}
	if alt.Confidence != "high" {
		t.Errorf("expected ALT confidence high, got %s", alt.Confidence)
	}
}

func TestDeduplicateResults_CaseInsensitive(t *testing.T) {
	t.Parallel()

	results := []ExtractedResult{
		{TestName: "alt", Value: "45", Unit: "U/L", Confidence: "medium"},
		{TestName: "ALT", Value: "48", Unit: "U/L", Confidence: "high"},
	}

	deduped := DeduplicateResults(results)
	if len(deduped) != 1 {
		t.Fatalf("expected 1 result after case-insensitive dedup, got %d", len(deduped))
	}
	if deduped[0].Value != "48" {
		t.Errorf("expected value 48, got %s", deduped[0].Value)
	}
}

func TestExtractService_Success(t *testing.T) {
	t.Parallel()

	cannedResp := `{"extracted": [{"test_name": "ALT", "value": "45", "unit": "U/L", "normal_range": "7-56", "confidence": "high"}], "notes": "test note"}`
	client := &mockClaudeClient{response: cannedResp}
	svc := NewService(client)

	images := []ImageData{
		{Data: []byte("fake-image-data"), ContentType: "image/jpeg"},
	}

	results, notes, err := svc.Extract(context.Background(), images)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].TestName != "ALT" {
		t.Errorf("expected ALT, got %s", results[0].TestName)
	}
	if notes != "test note" {
		t.Errorf("expected notes 'test note', got %q", notes)
	}
}

func TestExtractService_ClaudeAPIFailure(t *testing.T) {
	t.Parallel()

	client := &mockClaudeClient{err: fmt.Errorf("API error: service unavailable")}
	svc := NewService(client)

	images := []ImageData{
		{Data: []byte("fake-image-data"), ContentType: "image/jpeg"},
	}

	_, _, err := svc.Extract(context.Background(), images)
	if err == nil {
		t.Fatal("expected error on Claude API failure")
	}
}

func TestExtractService_DeduplicatesAcrossPages(t *testing.T) {
	t.Parallel()

	cannedResp := `[
		{"test_name": "ALT", "value": "45", "unit": "U/L", "normal_range": "7-56", "confidence": "medium"},
		{"test_name": "AST", "value": "32", "unit": "U/L", "normal_range": "10-40", "confidence": "high"},
		{"test_name": "ALT", "value": "48", "unit": "U/L", "normal_range": "7-56", "confidence": "high"}
	]`
	client := &mockClaudeClient{response: cannedResp}
	svc := NewService(client)

	images := []ImageData{
		{Data: []byte("page1"), ContentType: "image/jpeg"},
		{Data: []byte("page2"), ContentType: "image/jpeg"},
	}

	results, _, err := svc.Extract(context.Background(), images)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 deduplicated results, got %d", len(results))
	}

	for _, r := range results {
		if r.TestName == "ALT" && r.Value != "48" {
			t.Errorf("expected ALT=48, got ALT=%s", r.Value)
		}
	}
}

func TestValidatePhotoKeys_TooMany(t *testing.T) {
	t.Parallel()

	keys := make([]string, 11)
	for i := range keys {
		keys[i] = fmt.Sprintf("key-%d", i)
	}

	err := ValidatePhotoKeys(keys)
	if err == nil {
		t.Fatal("expected error for >10 photo keys")
	}
}

func TestValidatePhotoKeys_Empty(t *testing.T) {
	t.Parallel()

	err := ValidatePhotoKeys([]string{})
	if err == nil {
		t.Fatal("expected error for empty photo keys")
	}
}

func TestValidatePhotoKeys_Valid(t *testing.T) {
	t.Parallel()

	err := ValidatePhotoKeys([]string{"key1", "key2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExtractionPrompt(t *testing.T) {
	t.Parallel()

	prompt := ExtractionPrompt()
	if prompt == "" {
		t.Fatal("prompt should not be empty")
	}
	if !strings.Contains(prompt, "JSON") {
		t.Error("prompt should mention JSON format")
	}
}

func TestParseExtractionResponse_WithMarkdownFences(t *testing.T) {
	t.Parallel()

	raw := "```json\n[{\"test_name\": \"ALT\", \"value\": \"45\", \"unit\": \"U/L\", \"normal_range\": \"\", \"confidence\": \"high\"}]\n```"
	results, _, err := ParseExtractionResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].TestName != "ALT" {
		t.Errorf("expected ALT, got %s", results[0].TestName)
	}
}

func TestDeduplicateResults_Empty(t *testing.T) {
	t.Parallel()
	deduped := DeduplicateResults(nil)
	if deduped != nil && len(deduped) != 0 {
		t.Errorf("expected empty result, got %d", len(deduped))
	}
}

func TestExtractedResultJSON(t *testing.T) {
	t.Parallel()

	r := ExtractedResult{
		TestName:    "ALT",
		Value:       "45",
		Unit:        "U/L",
		NormalRange: "7-56",
		Confidence:  "high",
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded ExtractedResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.TestName != "ALT" {
		t.Errorf("expected ALT, got %s", decoded.TestName)
	}
}
