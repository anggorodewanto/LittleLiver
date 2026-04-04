package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/handler"
	"github.com/ablankz/LittleLiver/backend/internal/labextract"
	"github.com/ablankz/LittleLiver/backend/internal/middleware"
	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

// TestLabExtractFullPipeline_ExtractReviewSave tests the full extraction pipeline:
// upload images -> extract -> review (save individual lab results) -> verify saved data.
// Uses a mocked Claude client but real DB and MemoryStore for R2.
func TestLabExtractFullPipeline_ExtractReviewSave(t *testing.T) {
	t.Parallel()
	f := setupExtractTest(t)
	defer f.db.Close()

	// Step 1: Seed R2 photos (simulating upload)
	key1 := "photos/" + model.NewULID() + ".jpg"
	key2 := "photos/" + model.NewULID() + ".jpg"
	seedR2Photo(t, f.db, f.objStore, f.baby.ID, key1)
	seedR2Photo(t, f.db, f.objStore, f.baby.ID, key2)

	// Step 2: Call extract endpoint with mocked Claude response
	cannedResp := `{
		"extracted": [
			{"test_name": "total_bilirubin", "value": "1.8", "unit": "mg/dL", "normal_range": "0.1-1.2", "confidence": "high"},
			{"test_name": "ALT", "value": "52", "unit": "U/L", "normal_range": "7-56", "confidence": "high"},
			{"test_name": "AST", "value": "38", "unit": "U/L", "normal_range": "10-40", "confidence": "medium"}
		],
		"notes": "Sample collected at Regional Hospital"
	}`
	client := &mockClaudeClient{response: cannedResp}
	mux := makeExtractMux(t, f, client)

	// Also register the labs POST endpoint for saving
	addLabsSaveRoute(t, mux, f)

	extractReq := makeExtractReq(t, f, []string{key1, key2})
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, extractReq)

	if rr.Code != http.StatusOK {
		t.Fatalf("extract: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var extractResp extractResponse
	if err := json.NewDecoder(rr.Body).Decode(&extractResp); err != nil {
		t.Fatalf("extract: decode: %v", err)
	}

	if len(extractResp.Extracted) != 3 {
		t.Fatalf("extract: expected 3 results, got %d", len(extractResp.Extracted))
	}
	if extractResp.Notes != "Sample collected at Regional Hospital" {
		t.Errorf("extract: expected notes, got %q", extractResp.Notes)
	}

	// Step 3: "Review" phase — simulate editing one value (ALT 52 -> 55)
	// then save each result via POST /api/babies/{id}/labs
	reviewedItems := []struct {
		TestName    string
		Value       string
		Unit        string
		NormalRange string
	}{
		{"total_bilirubin", "1.8", "mg/dL", "0.1-1.2"},
		{"ALT", "55", "U/L", "7-56"}, // edited from 52 to 55
		{"AST", "38", "U/L", "10-40"},
	}

	timestamp := time.Now().UTC().Format(model.DateTimeFormat)
	for _, item := range reviewedItems {
		body, _ := json.Marshal(map[string]string{
			"timestamp":    timestamp,
			"test_name":    item.TestName,
			"value":        item.Value,
			"unit":         item.Unit,
			"normal_range": item.NormalRange,
		})
		req := testutil.AuthenticatedRequest(t, f.db, f.user.ID, testCookieName, testSecret,
			http.MethodPost, "/api/babies/"+f.baby.ID+"/labs")
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		saveRR := httptest.NewRecorder()
		mux.ServeHTTP(saveRR, req)
		if saveRR.Code != http.StatusCreated {
			t.Fatalf("save %s: expected 201, got %d: %s", item.TestName, saveRR.Code, saveRR.Body.String())
		}
	}

	// Step 4: Verify saved results via store (the "labs list" verification)
	saved, err := store.ListLabResults(f.db, f.baby.ID, nil, nil, nil, 100)
	if err != nil {
		t.Fatalf("list lab results: %v", err)
	}
	if len(saved.Data) != 3 {
		t.Fatalf("expected 3 saved lab results, got %d", len(saved.Data))
	}

	// Build a map of saved results by test_name for easier assertion
	savedMap := make(map[string]*model.LabResult)
	for i := range saved.Data {
		savedMap[saved.Data[i].TestName] = &saved.Data[i]
	}

	// Verify the edited value was saved correctly
	if alt, ok := savedMap["ALT"]; !ok {
		t.Error("ALT not found in saved results")
	} else if alt.Value != "55" {
		t.Errorf("ALT value: expected '55' (edited), got '%s'", alt.Value)
	}

	if bili, ok := savedMap["total_bilirubin"]; !ok {
		t.Error("total_bilirubin not found in saved results")
	} else if bili.Value != "1.8" {
		t.Errorf("total_bilirubin value: expected '1.8', got '%s'", bili.Value)
	}

	if ast, ok := savedMap["AST"]; !ok {
		t.Error("AST not found in saved results")
	} else if ast.Value != "38" {
		t.Errorf("AST value: expected '38', got '%s'", ast.Value)
	}
}

// TestLabExtractRateLimitPersistsAcrossFlow verifies that extraction requests
// count toward the per-user rate limit and the limit persists across multiple
// requests in a single flow.
func TestLabExtractRateLimitPersistsAcrossFlow(t *testing.T) {
	t.Parallel()
	f := setupExtractTest(t)
	defer f.db.Close()

	cannedResp := `{"extracted": [{"test_name": "ALT", "value": "45", "unit": "U/L", "normal_range": "7-56", "confidence": "high"}], "notes": ""}`
	client := &mockClaudeClient{response: cannedResp}
	svc := labextract.NewService(client)
	rl := handler.NewExtractRateLimiter()
	h := handler.LabExtractHandlerWithRateLimit(f.db, f.objStore, svc, rl)

	mux := makeRateLimitedExtractMux(t, f, h)

	// Seed a photo for each request
	keys := make([]string, 11)
	for i := range keys {
		keys[i] = "photo-" + model.NewULID()
		seedR2Photo(t, f.db, f.objStore, f.baby.ID, keys[i])
	}

	// Make 10 successful extraction requests
	for i := 0; i < 10; i++ {
		req := makeExtractReq(t, f, []string{keys[i]})
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d: %s", i+1, rr.Code, rr.Body.String())
		}
	}

	// 11th request should be rate limited (429)
	req := makeExtractReq(t, f, []string{keys[10]})
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("11th request: expected 429, got %d", rr.Code)
	}

	// Verify non-extract endpoints still work (saving labs is not rate limited by extract RL)
	addLabsSaveRoute(t, mux, f)
	labBody, _ := json.Marshal(map[string]string{
		"timestamp": time.Now().UTC().Format(model.DateTimeFormat),
		"test_name": "ALT",
		"value":     "45",
	})
	saveReq := testutil.AuthenticatedRequest(t, f.db, f.user.ID, testCookieName, testSecret,
		http.MethodPost, "/api/babies/"+f.baby.ID+"/labs")
	saveReq.Body = io.NopCloser(bytes.NewReader(labBody))
	saveReq.Header.Set("Content-Type", "application/json")
	saveRR := httptest.NewRecorder()
	mux.ServeHTTP(saveRR, saveReq)
	if saveRR.Code != http.StatusCreated {
		t.Errorf("save after extract rate limit: expected 201, got %d: %s", saveRR.Code, saveRR.Body.String())
	}
}

// TestLabExtractErrorRecovery_Timeout tests error recovery when Claude API times out.
// Verifies the user gets a 502 error and can retry successfully.
func TestLabExtractErrorRecovery_Timeout(t *testing.T) {
	t.Parallel()
	f := setupExtractTest(t)
	defer f.db.Close()

	key := "photo-" + model.NewULID()
	seedR2Photo(t, f.db, f.objStore, f.baby.ID, key)

	// First attempt: Claude client returns a timeout error
	timeoutClient := &mockClaudeClientSequential{
		responses: []mockResponse{
			{err: fmt.Errorf("context deadline exceeded: timeout waiting for response")},
			{response: `{"extracted": [{"test_name": "ALT", "value": "45", "unit": "U/L", "normal_range": "7-56", "confidence": "high"}], "notes": ""}`},
		},
	}

	svc := labextract.NewService(timeoutClient)
	h := handler.LabExtractHandler(f.db, f.objStore, svc)
	mux := makeExtractMuxFromHandler(t, f, h)

	// First request: should fail with 502
	req1 := makeExtractReq(t, f, []string{key})
	rr1 := httptest.NewRecorder()
	mux.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusBadGateway {
		t.Fatalf("timeout request: expected 502, got %d: %s", rr1.Code, rr1.Body.String())
	}
	if !strings.Contains(rr1.Body.String(), "extraction failed") {
		t.Errorf("expected error message about extraction failure, got: %s", rr1.Body.String())
	}

	// Second request (retry): should succeed
	req2 := makeExtractReq(t, f, []string{key})
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("retry request: expected 200, got %d: %s", rr2.Code, rr2.Body.String())
	}

	var resp extractResponse
	if err := json.NewDecoder(rr2.Body).Decode(&resp); err != nil {
		t.Fatalf("retry: decode: %v", err)
	}
	if len(resp.Extracted) != 1 {
		t.Fatalf("retry: expected 1 result, got %d", len(resp.Extracted))
	}
	if resp.Extracted[0].TestName != "ALT" {
		t.Errorf("retry: expected ALT, got %s", resp.Extracted[0].TestName)
	}
}

// TestLabExtractResponseSchema verifies the response schema of the extract endpoint.
func TestLabExtractResponseSchema(t *testing.T) {
	t.Parallel()
	f := setupExtractTest(t)
	defer f.db.Close()

	cannedResp := `{
		"extracted": [
			{"test_name": "total_bilirubin", "value": "1.2", "unit": "mg/dL", "normal_range": "0.1-1.2", "confidence": "high"},
			{"test_name": "GGT", "value": "85", "unit": "U/L", "normal_range": "9-48", "confidence": "low"}
		],
		"notes": "Lab date: 2026-04-01"
	}`
	client := &mockClaudeClient{response: cannedResp}
	mux := makeExtractMux(t, f, client)

	key := "photo-" + model.NewULID()
	seedR2Photo(t, f.db, f.objStore, f.baby.ID, key)

	req := makeExtractReq(t, f, []string{key})
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify JSON structure by unmarshaling into a generic map
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	// Must have "extracted" and "notes" fields
	if _, ok := raw["extracted"]; !ok {
		t.Error("response missing 'extracted' field")
	}
	if _, ok := raw["notes"]; !ok {
		t.Error("response missing 'notes' field")
	}

	// Verify each extracted item has required fields
	var items []map[string]interface{}
	if err := json.Unmarshal(raw["extracted"], &items); err != nil {
		t.Fatalf("'extracted' is not an array: %v", err)
	}

	requiredFields := []string{"test_name", "value", "unit", "normal_range", "confidence"}
	for i, item := range items {
		for _, field := range requiredFields {
			if _, ok := item[field]; !ok {
				t.Errorf("extracted[%d] missing required field '%s'", i, field)
			}
		}
	}

	// Verify confidence values are valid
	validConfidence := map[string]bool{"high": true, "medium": true, "low": true}
	for i, item := range items {
		conf, _ := item["confidence"].(string)
		if !validConfidence[conf] {
			t.Errorf("extracted[%d] has invalid confidence: %q", i, conf)
		}
	}

	// Verify notes is a string
	var notesStr string
	if err := json.Unmarshal(raw["notes"], &notesStr); err != nil {
		t.Errorf("'notes' is not a string: %v", err)
	}
}

// --- Helper types and functions for integration tests ---

// mockClaudeClientSequential returns responses in order, supporting error then success scenarios.
type mockClaudeClientSequential struct {
	mu        sync.Mutex
	responses []mockResponse
	callCount int
}

type mockResponse struct {
	response string
	err      error
}

func (m *mockClaudeClientSequential) ExtractLabResults(ctx context.Context, images []labextract.ImageData, prompt string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.callCount >= len(m.responses) {
		return "", fmt.Errorf("no more mock responses")
	}
	resp := m.responses[m.callCount]
	m.callCount++
	if resp.err != nil {
		return "", resp.err
	}
	return resp.response, nil
}

// addLabsSaveRoute adds the POST /api/babies/{id}/labs route to the mux for saving lab results.
func addLabsSaveRoute(t *testing.T, mux *http.ServeMux, f *extractTestFixture) {
	t.Helper()
	// We need auth and CSRF middleware to match the test fixture's authentication
	authMw := middleware.Auth(f.db, testCookieName)
	csrfMw := middleware.CSRF(f.db, testCookieName, testSecret)
	h := handler.CreateLabResultHandler(f.db)
	mux.Handle("POST /api/babies/{id}/labs", authMw(csrfMw(h)))
}

// makeRateLimitedExtractMux creates a mux with the extract handler and rate limiting.
func makeRateLimitedExtractMux(t *testing.T, f *extractTestFixture, h http.HandlerFunc) *http.ServeMux {
	t.Helper()
	authMw := middleware.Auth(f.db, testCookieName)
	csrfMw := middleware.CSRF(f.db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/labs/extract", authMw(csrfMw(http.HandlerFunc(h))))
	return mux
}

// makeExtractMuxFromHandler creates a mux with a pre-built handler (no Claude client needed).
func makeExtractMuxFromHandler(t *testing.T, f *extractTestFixture, h http.HandlerFunc) *http.ServeMux {
	t.Helper()
	authMw := middleware.Auth(f.db, testCookieName)
	csrfMw := middleware.CSRF(f.db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/labs/extract", authMw(csrfMw(http.HandlerFunc(h))))
	return mux
}

// TestBatchCreateLabResults tests the POST /api/babies/{id}/labs/batch endpoint.
func TestBatchCreateLabResults(t *testing.T) {
	t.Parallel()
	f := setupExtractTest(t)
	defer f.db.Close()

	authMw := middleware.Auth(f.db, testCookieName)
	csrfMw := middleware.CSRF(f.db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/labs/batch", authMw(csrfMw(http.HandlerFunc(handler.BatchCreateLabResultHandler(f.db)))))

	timestamp := time.Now().UTC().Format(model.DateTimeFormat)
	body, _ := json.Marshal(map[string]any{
		"items": []map[string]string{
			{"timestamp": timestamp, "test_name": "ALT", "value": "45", "unit": "U/L", "normal_range": "7-56"},
			{"timestamp": timestamp, "test_name": "AST", "value": "32", "unit": "U/L", "normal_range": "10-40"},
			{"timestamp": timestamp, "test_name": "GGT", "value": "120", "unit": "U/L"},
		},
	})

	req := testutil.AuthenticatedRequest(t, f.db, f.user.ID, testCookieName, testSecret,
		http.MethodPost, "/api/babies/"+f.baby.ID+"/labs/batch")
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var results []struct {
		ID       string `json:"id"`
		TestName string `json:"test_name"`
		Value    string `json:"value"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&results); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Verify all saved via store
	saved, err := store.ListLabResults(f.db, f.baby.ID, nil, nil, nil, 100)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(saved.Data) != 3 {
		t.Fatalf("expected 3 saved, got %d", len(saved.Data))
	}
}

// TestBatchCreateLabResults_EmptyItems returns 400 for empty items array.
func TestBatchCreateLabResults_EmptyItems(t *testing.T) {
	t.Parallel()
	f := setupExtractTest(t)
	defer f.db.Close()

	authMw := middleware.Auth(f.db, testCookieName)
	csrfMw := middleware.CSRF(f.db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/labs/batch", authMw(csrfMw(http.HandlerFunc(handler.BatchCreateLabResultHandler(f.db)))))

	body, _ := json.Marshal(map[string]any{"items": []map[string]string{}})
	req := testutil.AuthenticatedRequest(t, f.db, f.user.ID, testCookieName, testSecret,
		http.MethodPost, "/api/babies/"+f.baby.ID+"/labs/batch")
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestBatchCreateLabResults_ValidationError returns 400 when an item is invalid.
func TestBatchCreateLabResults_ValidationError(t *testing.T) {
	t.Parallel()
	f := setupExtractTest(t)
	defer f.db.Close()

	authMw := middleware.Auth(f.db, testCookieName)
	csrfMw := middleware.CSRF(f.db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/labs/batch", authMw(csrfMw(http.HandlerFunc(handler.BatchCreateLabResultHandler(f.db)))))

	timestamp := time.Now().UTC().Format(model.DateTimeFormat)
	body, _ := json.Marshal(map[string]any{
		"items": []map[string]string{
			{"timestamp": timestamp, "test_name": "ALT", "value": "45"},
			{"timestamp": timestamp, "test_name": "", "value": "32"}, // missing test_name
		},
	})

	req := testutil.AuthenticatedRequest(t, f.db, f.user.ID, testCookieName, testSecret,
		http.MethodPost, "/api/babies/"+f.baby.ID+"/labs/batch")
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify nothing was saved (transaction should have rolled back)
	saved, err := store.ListLabResults(f.db, f.baby.ID, nil, nil, nil, 100)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(saved.Data) != 0 {
		t.Errorf("expected 0 saved after validation error, got %d", len(saved.Data))
	}
}

// TestLabExtractHandler_ReportDateDuplicateDetection verifies that report_date is used
// for duplicate detection instead of current time.
func TestLabExtractHandler_ReportDateDuplicateDetection(t *testing.T) {
	t.Parallel()
	f := setupExtractTest(t)
	defer f.db.Close()

	// Seed existing lab result: ALT=45, timestamped at 2026-03-14 (1 day before report_date)
	_, err := store.CreateLabResult(f.db, f.baby.ID, f.user.ID, "2026-03-14T10:00:00Z", "ALT", "45", strPtrExtract("U/L"), nil, nil)
	if err != nil {
		t.Fatalf("seed lab result: %v", err)
	}

	// Claude returns report_date of 2026-03-15 — within ±3 days of the seeded result
	cannedResp := `{
		"extracted": [{"test_name": "ALT", "value": "45", "unit": "U/L", "normal_range": "7-56", "confidence": "high"}],
		"report_date": "2026-03-15",
		"notes": ""
	}`
	client := &mockClaudeClient{response: cannedResp}
	mux := makeExtractMux(t, f, client)

	key := "photo-" + model.NewULID()
	seedR2Photo(t, f.db, f.objStore, f.baby.ID, key)

	req := makeExtractReq(t, f, []string{key})
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp extractResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(resp.Extracted) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Extracted))
	}
	if resp.Extracted[0].ExistingMatch == nil {
		t.Fatal("expected existing_match to be set (report_date-based duplicate detection)")
	}
}
