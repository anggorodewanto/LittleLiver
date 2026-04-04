package handler_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/handler"
	"github.com/ablankz/LittleLiver/backend/internal/labextract"
	"github.com/ablankz/LittleLiver/backend/internal/middleware"
	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/storage"
	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

// mockClaudeClient implements labextract.ClaudeClient for testing.
type mockClaudeClient struct {
	response string
	err      error
}

func (m *mockClaudeClient) ExtractLabResults(ctx context.Context, images []labextract.ImageData, prompt string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

type extractTestFixture struct {
	db       *sql.DB
	user     *model.User
	baby     *model.Baby
	objStore *storage.MemoryStore
}

func setupExtractTest(t *testing.T) *extractTestFixture {
	t.Helper()
	db := testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()
	return &extractTestFixture{db: db, user: user, baby: baby, objStore: objStore}
}

func seedR2Photo(t *testing.T, db *sql.DB, objStore *storage.MemoryStore, babyID, key string) {
	t.Helper()
	err := objStore.Put(context.Background(), key, io.NopCloser(strings.NewReader("fake-image-bytes")), "image/jpeg")
	if err != nil {
		t.Fatalf("seedR2Photo: Put failed: %v", err)
	}
	id := model.NewULID()
	_, err = db.Exec(
		"INSERT INTO photo_uploads (id, baby_id, r2_key, uploaded_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP)",
		id, babyID, key,
	)
	if err != nil {
		t.Fatalf("seedR2Photo: insert failed: %v", err)
	}
}

func makeExtractMux(t *testing.T, f *extractTestFixture, client labextract.ClaudeClient) *http.ServeMux {
	t.Helper()
	svc := labextract.NewService(client)
	h := handler.LabExtractHandler(f.db, f.objStore, svc)

	authMw := middleware.Auth(f.db, testCookieName)
	csrfMw := middleware.CSRF(f.db, testCookieName, testSecret)

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/labs/extract", authMw(csrfMw(http.HandlerFunc(h))))
	return mux
}

func makeExtractReq(t *testing.T, f *extractTestFixture, photoKeys []string) *http.Request {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"photo_keys": photoKeys})
	req := testutil.AuthenticatedRequest(t, f.db, f.user.ID, testCookieName, testSecret, http.MethodPost,
		"/api/babies/"+f.baby.ID+"/labs/extract")
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

type extractResponse struct {
	Extracted []extractedResultResp `json:"extracted"`
	Notes     string                `json:"notes"`
}

type extractedResultResp struct {
	TestName      string         `json:"test_name"`
	Value         string         `json:"value"`
	Unit          string         `json:"unit"`
	NormalRange   string         `json:"normal_range"`
	Confidence    string         `json:"confidence"`
	ExistingMatch *existingMatch `json:"existing_match,omitempty"`
}

type existingMatch struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Value     string `json:"value"`
	Unit      string `json:"unit"`
}

func TestLabExtractHandler_TooManyPhotoKeys(t *testing.T) {
	t.Parallel()
	f := setupExtractTest(t)
	defer f.db.Close()

	client := &mockClaudeClient{response: "[]"}
	mux := makeExtractMux(t, f, client)

	keys := make([]string, 11)
	for i := range keys {
		keys[i] = fmt.Sprintf("key-%d", i)
	}

	req := makeExtractReq(t, f, keys)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestLabExtractHandler_EmptyPhotoKeys(t *testing.T) {
	t.Parallel()
	f := setupExtractTest(t)
	defer f.db.Close()

	client := &mockClaudeClient{response: "[]"}
	mux := makeExtractMux(t, f, client)

	req := makeExtractReq(t, f, []string{})
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestLabExtractHandler_InvalidR2Keys(t *testing.T) {
	t.Parallel()
	f := setupExtractTest(t)
	defer f.db.Close()

	client := &mockClaudeClient{response: "[]"}
	mux := makeExtractMux(t, f, client)

	// Key not in photo_uploads for this baby → 400
	req := makeExtractReq(t, f, []string{"nonexistent-key"})
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestLabExtractHandler_R2KeyWrongBaby(t *testing.T) {
	t.Parallel()
	f := setupExtractTest(t)
	defer f.db.Close()

	// Create another baby and seed a photo for it
	otherBaby := testutil.CreateTestBaby(t, f.db, f.user.ID)
	key := "photo-" + model.NewULID()
	seedR2Photo(t, f.db, f.objStore, otherBaby.ID, key)

	client := &mockClaudeClient{response: "[]"}
	mux := makeExtractMux(t, f, client)

	// Request with f.baby.ID but key belongs to otherBaby → 400
	req := makeExtractReq(t, f, []string{key})
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for key belonging to different baby, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestLabExtractHandler_ClaudeAPIFailure(t *testing.T) {
	t.Parallel()
	f := setupExtractTest(t)
	defer f.db.Close()

	client := &mockClaudeClient{err: fmt.Errorf("service unavailable")}
	mux := makeExtractMux(t, f, client)

	key := "photo-" + model.NewULID()
	seedR2Photo(t, f.db, f.objStore, f.baby.ID, key)

	req := makeExtractReq(t, f, []string{key})
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestLabExtractHandler_SuccessfulExtraction(t *testing.T) {
	t.Parallel()
	f := setupExtractTest(t)
	defer f.db.Close()

	cannedResp := `{
		"extracted": [
			{"test_name": "ALT", "value": "45", "unit": "U/L", "normal_range": "7-56", "confidence": "high"},
			{"test_name": "AST", "value": "32", "unit": "U/L", "normal_range": "10-40", "confidence": "medium"}
		],
		"notes": "Sample collected 2026-03-15"
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
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Extracted) != 2 {
		t.Fatalf("expected 2 results, got %d", len(resp.Extracted))
	}
	if resp.Extracted[0].TestName != "ALT" {
		t.Errorf("expected ALT, got %s", resp.Extracted[0].TestName)
	}
	if resp.Extracted[0].Confidence != "high" {
		t.Errorf("expected high confidence, got %s", resp.Extracted[0].Confidence)
	}
	if resp.Notes != "Sample collected 2026-03-15" {
		t.Errorf("expected notes, got %q", resp.Notes)
	}
}

func TestLabExtractHandler_DuplicateDetection(t *testing.T) {
	t.Parallel()
	f := setupExtractTest(t)
	defer f.db.Close()

	// Seed an existing lab result: ALT=45, 2 days ago
	twoDaysAgo := time.Now().UTC().AddDate(0, 0, -2).Format(model.DateTimeFormat)
	existingLab, err := store.CreateLabResult(f.db, f.baby.ID, f.user.ID, twoDaysAgo, "ALT", "45", strPtrExtract("U/L"), nil, nil)
	if err != nil {
		t.Fatalf("seed lab result: %v", err)
	}

	cannedResp := `[{"test_name": "ALT", "value": "45", "unit": "U/L", "normal_range": "7-56", "confidence": "high"}]`
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
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Extracted) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Extracted))
	}

	r := resp.Extracted[0]
	if r.ExistingMatch == nil {
		t.Fatal("expected existing_match to be set")
	}
	if r.ExistingMatch.ID != existingLab.ID {
		t.Errorf("expected existing_match.id=%s, got %s", existingLab.ID, r.ExistingMatch.ID)
	}
	if r.ExistingMatch.Value != "45" {
		t.Errorf("expected existing_match.value=45, got %s", r.ExistingMatch.Value)
	}
	if r.ExistingMatch.Unit != "U/L" {
		t.Errorf("expected existing_match.unit=U/L, got %s", r.ExistingMatch.Unit)
	}
}

func TestLabExtractHandler_NoFalseDuplicate(t *testing.T) {
	t.Parallel()
	f := setupExtractTest(t)
	defer f.db.Close()

	// Seed existing lab result: ALT=45, 10 days ago (outside +-3 day window)
	tenDaysAgo := time.Now().UTC().AddDate(0, 0, -10).Format(model.DateTimeFormat)
	_, err := store.CreateLabResult(f.db, f.baby.ID, f.user.ID, tenDaysAgo, "ALT", "45", strPtrExtract("U/L"), nil, nil)
	if err != nil {
		t.Fatalf("seed lab result: %v", err)
	}

	cannedResp := `[{"test_name": "ALT", "value": "45", "unit": "U/L", "normal_range": "7-56", "confidence": "high"}]`
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
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Extracted) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Extracted))
	}

	if resp.Extracted[0].ExistingMatch != nil {
		t.Error("expected existing_match to be nil for result outside +-3 day window")
	}
}

func TestLabExtractHandler_RateLimit(t *testing.T) {
	t.Parallel()
	f := setupExtractTest(t)
	defer f.db.Close()

	cannedResp := `[{"test_name": "ALT", "value": "45", "unit": "U/L", "normal_range": "7-56", "confidence": "high"}]`
	client := &mockClaudeClient{response: cannedResp}
	svc := labextract.NewService(client)
	rl := handler.NewExtractRateLimiter()
	h := handler.LabExtractHandlerWithRateLimit(f.db, f.objStore, svc, rl)

	authMw := middleware.Auth(f.db, testCookieName)
	csrfMw := middleware.CSRF(f.db, testCookieName, testSecret)

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/labs/extract", authMw(csrfMw(http.HandlerFunc(h))))

	key := "photo-" + model.NewULID()
	seedR2Photo(t, f.db, f.objStore, f.baby.ID, key)

	makeReq := func() int {
		req := makeExtractReq(t, f, []string{key})
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		return rr.Code
	}

	// First 10 should succeed
	for i := 0; i < 10; i++ {
		code := makeReq()
		if code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, code)
		}
	}

	// 11th should be rate limited
	code := makeReq()
	if code != http.StatusTooManyRequests {
		t.Errorf("11th request: expected 429, got %d", code)
	}
}

func TestLabExtractHandler_IntegrationMocked(t *testing.T) {
	t.Parallel()
	f := setupExtractTest(t)
	defer f.db.Close()

	// Seed existing lab for duplicate detection
	twoDaysAgo := time.Now().UTC().AddDate(0, 0, -2).Format(model.DateTimeFormat)
	existingLab, err := store.CreateLabResult(f.db, f.baby.ID, f.user.ID, twoDaysAgo, "GGT", "120", strPtrExtract("U/L"), nil, nil)
	if err != nil {
		t.Fatalf("seed lab: %v", err)
	}

	cannedResp := `{
		"extracted": [
			{"test_name": "ALT", "value": "45", "unit": "U/L", "normal_range": "7-56", "confidence": "high"},
			{"test_name": "GGT", "value": "120", "unit": "U/L", "normal_range": "9-48", "confidence": "high"},
			{"test_name": "ALT", "value": "48", "unit": "U/L", "normal_range": "7-56", "confidence": "high"}
		],
		"notes": "Lab report from Regional Hospital"
	}`
	client := &mockClaudeClient{response: cannedResp}
	mux := makeExtractMux(t, f, client)

	key1 := "photo-" + model.NewULID()
	key2 := "photo-" + model.NewULID()
	seedR2Photo(t, f.db, f.objStore, f.baby.ID, key1)
	seedR2Photo(t, f.db, f.objStore, f.baby.ID, key2)

	req := makeExtractReq(t, f, []string{key1, key2})
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp extractResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// Should have 2 deduplicated results (ALT deduped to value 48)
	if len(resp.Extracted) != 2 {
		t.Fatalf("expected 2 results, got %d", len(resp.Extracted))
	}

	if resp.Notes != "Lab report from Regional Hospital" {
		t.Errorf("expected notes, got %q", resp.Notes)
	}

	for _, r := range resp.Extracted {
		switch r.TestName {
		case "ALT":
			if r.Value != "48" {
				t.Errorf("ALT should be deduplicated to 48, got %s", r.Value)
			}
			if r.ExistingMatch != nil {
				t.Error("ALT should have no existing match")
			}
		case "GGT":
			if r.ExistingMatch == nil {
				t.Error("GGT should have existing_match")
			} else {
				if r.ExistingMatch.ID != existingLab.ID {
					t.Errorf("GGT existing_match.id should be %s, got %s", existingLab.ID, r.ExistingMatch.ID)
				}
				if r.ExistingMatch.Value != "120" {
					t.Errorf("GGT existing_match.value should be 120, got %s", r.ExistingMatch.Value)
				}
				if r.ExistingMatch.Unit != "U/L" {
					t.Errorf("GGT existing_match.unit should be U/L, got %s", r.ExistingMatch.Unit)
				}
			}
		default:
			t.Errorf("unexpected test_name: %s", r.TestName)
		}
	}
}

// strPtrExtract is a helper to create a string pointer (avoids redeclaration with other test files).
func strPtrExtract(s string) *string {
	return &s
}
