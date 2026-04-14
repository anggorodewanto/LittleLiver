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

	results, notes, reportDate, err := ParseExtractionResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notes != "" {
		t.Errorf("expected empty notes for array format, got %q", notes)
	}
	if reportDate != "" {
		t.Errorf("expected empty report_date for array format, got %q", reportDate)
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
		"report_date": "2026-03-15",
		"notes": "Regional Hospital"
	}`

	results, notes, reportDate, err := ParseExtractionResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notes != "Regional Hospital" {
		t.Errorf("expected notes, got %q", notes)
	}
	if reportDate != "2026-03-15" {
		t.Errorf("expected report_date 2026-03-15, got %q", reportDate)
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

	_, _, _, err := ParseExtractionResponse("not json")
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

	result, err := svc.Extract(context.Background(), images)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].TestName != "ALT" {
		t.Errorf("expected ALT, got %s", result.Results[0].TestName)
	}
	if result.Notes != "test note" {
		t.Errorf("expected notes 'test note', got %q", result.Notes)
	}
}

func TestExtractService_ClaudeAPIFailure(t *testing.T) {
	t.Parallel()

	client := &mockClaudeClient{err: fmt.Errorf("API error: service unavailable")}
	svc := NewService(client)

	images := []ImageData{
		{Data: []byte("fake-image-data"), ContentType: "image/jpeg"},
	}

	_, err := svc.Extract(context.Background(), images)
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

	result, err := svc.Extract(context.Background(), images)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) != 2 {
		t.Fatalf("expected 2 deduplicated results, got %d", len(result.Results))
	}

	for _, r := range result.Results {
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
	if !strings.Contains(prompt, "report_date") {
		t.Error("prompt should mention report_date field")
	}
}

func TestExtractionPromptWithHints_NoHints(t *testing.T) {
	t.Parallel()

	if ExtractionPromptWithHints(nil) != ExtractionPrompt() {
		t.Error("nil hints should yield base prompt")
	}
	if ExtractionPromptWithHints([]LabTestHint{}) != ExtractionPrompt() {
		t.Error("empty hints should yield base prompt")
	}
}

func TestExtractionPromptWithHints_IncludesHints(t *testing.T) {
	t.Parallel()

	unit := "U/L"
	rng := "0-40"
	hints := []LabTestHint{
		{TestName: "SGOT/AST", Unit: &unit, NormalRange: &rng},
		{TestName: "total_bilirubin"},
	}
	prompt := ExtractionPromptWithHints(hints)
	if !strings.Contains(prompt, "SGOT/AST") {
		t.Error("prompt should contain canonical test name SGOT/AST")
	}
	if !strings.Contains(prompt, "total_bilirubin") {
		t.Error("prompt should contain test name total_bilirubin")
	}
	if !strings.Contains(prompt, "U/L") {
		t.Error("prompt should contain hint unit")
	}
	if !strings.Contains(prompt, "0-40") {
		t.Error("prompt should contain hint normal_range")
	}
	// Should still mention JSON / report_date
	if !strings.Contains(prompt, "report_date") {
		t.Error("prompt should still include base instructions")
	}
}

// captureClient records the prompt that was passed to ExtractLabResults.
type captureClient struct {
	response   string
	lastPrompt string
}

func (c *captureClient) ExtractLabResults(ctx context.Context, images []ImageData, prompt string) (string, error) {
	c.lastPrompt = prompt
	return c.response, nil
}

func TestExtractWithHints_PassesHintsIntoPrompt(t *testing.T) {
	t.Parallel()

	cannedResp := `{"extracted": [], "report_date": "", "notes": ""}`
	client := &captureClient{response: cannedResp}
	svc := NewService(client)

	unit := "U/L"
	hints := []LabTestHint{{TestName: "SGOT/AST", Unit: &unit}}

	_, err := svc.ExtractWithHints(context.Background(), []ImageData{{Data: []byte("img"), ContentType: "image/jpeg"}}, hints)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(client.lastPrompt, "SGOT/AST") {
		t.Errorf("expected hint name in prompt, got: %s", client.lastPrompt)
	}
}

func TestExtract_NoHintsKeepsBasePrompt(t *testing.T) {
	t.Parallel()

	cannedResp := `{"extracted": [], "report_date": "", "notes": ""}`
	client := &captureClient{response: cannedResp}
	svc := NewService(client)

	_, err := svc.Extract(context.Background(), []ImageData{{Data: []byte("img"), ContentType: "image/jpeg"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.lastPrompt != ExtractionPrompt() {
		t.Error("Extract should use base prompt without hints")
	}
}

func TestParseExtractionResponse_WithMarkdownFences(t *testing.T) {
	t.Parallel()

	raw := "```json\n[{\"test_name\": \"ALT\", \"value\": \"45\", \"unit\": \"U/L\", \"normal_range\": \"\", \"confidence\": \"high\"}]\n```"
	results, _, _, err := ParseExtractionResponse(raw)
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

func TestParseExtractionResponse_ReportDate(t *testing.T) {
	t.Parallel()

	raw := `{
		"extracted": [{"test_name": "ALT", "value": "45", "unit": "U/L", "normal_range": "7-56", "confidence": "high"}],
		"report_date": "2026-04-01",
		"notes": ""
	}`

	_, _, reportDate, err := ParseExtractionResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reportDate != "2026-04-01" {
		t.Errorf("expected report_date 2026-04-01, got %q", reportDate)
	}
}

func TestExtractService_ReturnsReportDate(t *testing.T) {
	t.Parallel()

	cannedResp := `{"extracted": [{"test_name": "ALT", "value": "45", "unit": "U/L", "normal_range": "7-56", "confidence": "high"}], "report_date": "2026-03-20", "notes": ""}`
	client := &mockClaudeClient{response: cannedResp}
	svc := NewService(client)

	result, err := svc.Extract(context.Background(), []ImageData{{Data: []byte("img"), ContentType: "image/jpeg"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ReportDate != "2026-03-20" {
		t.Errorf("expected report_date 2026-03-20, got %q", result.ReportDate)
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
