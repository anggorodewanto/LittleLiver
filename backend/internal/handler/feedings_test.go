package handler_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ablankz/LittleLiver/backend/internal/handler"
	"github.com/ablankz/LittleLiver/backend/internal/middleware"
	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

// --- POST /api/babies/{id}/feedings ---

func TestCreateFeedingHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T10:30:00Z","feed_type":"breast_milk","volume_ml":120,"cal_density":20,"duration_min":15,"notes":"tolerated well"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/feedings")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateFeedingHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/feedings", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp["id"] == nil || resp["id"] == "" {
		t.Error("expected non-empty feeding ID")
	}
	if resp["feed_type"] != "breast_milk" {
		t.Errorf("expected feed_type=breast_milk, got %v", resp["feed_type"])
	}
	if resp["volume_ml"] != 120.0 {
		t.Errorf("expected volume_ml=120, got %v", resp["volume_ml"])
	}
}

func TestCreateFeedingHandler_MissingFields(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"feed_type":"breast_milk"}` // missing timestamp
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/feedings")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateFeedingHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/feedings", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateFeedingHandler_InvalidFeedType(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T10:30:00Z","feed_type":"invalid_type"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/feedings")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateFeedingHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/feedings", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateFeedingHandler_Unauthorized(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	otherUser := testutil.CreateTestUser(t, db) // not linked to baby

	body := `{"timestamp":"2025-07-01T10:30:00Z","feed_type":"formula"}`
	req := testutil.AuthenticatedRequest(t, db, otherUser.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/feedings")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateFeedingHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/feedings", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// --- GET /api/babies/{id}/feedings ---

func TestListFeedingsHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	// Create a few feedings
	for i := 0; i < 3; i++ {
		_, err := store.CreateFeeding(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "formula", nil, nil, nil, nil)
		if err != nil {
			t.Fatalf("CreateFeeding failed: %v", err)
		}
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/feedings")
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.ListFeedingsHandler(db)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/feedings", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data       []map[string]any `json:"data"`
		NextCursor *string          `json:"next_cursor"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(resp.Data) != 3 {
		t.Errorf("expected 3 feedings, got %d", len(resp.Data))
	}
}

func TestListFeedingsHandler_Unauthorized(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	otherUser := testutil.CreateTestUser(t, db)

	req := testutil.AuthenticatedRequest(t, db, otherUser.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/feedings")
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.ListFeedingsHandler(db)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/feedings", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestListFeedingsHandler_WithDateFilter(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	_, err := store.CreateFeeding(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "formula", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateFeeding failed: %v", err)
	}
	_, err = store.CreateFeeding(db, baby.ID, user.ID, "2025-07-03T10:30:00Z", "formula", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateFeeding failed: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/feedings?from=2025-07-01&to=2025-07-01")
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.ListFeedingsHandler(db)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/feedings", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Errorf("expected 1 feeding in date range, got %d", len(resp.Data))
	}
}

// --- GET /api/babies/{id}/feedings/{entryId} ---

func TestGetFeedingHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	vol := 120.0
	feeding, err := store.CreateFeeding(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "breast_milk", &vol, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateFeeding failed: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/feedings/"+feeding.ID)
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.GetFeedingHandler(db)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/feedings/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp["id"] != feeding.ID {
		t.Errorf("expected id=%q, got %v", feeding.ID, resp["id"])
	}
}

func TestGetFeedingHandler_NotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/feedings/nonexistent")
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.GetFeedingHandler(db)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/feedings/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

// --- PUT /api/babies/{id}/feedings/{entryId} ---

func TestUpdateFeedingHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	feeding, err := store.CreateFeeding(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "formula", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateFeeding failed: %v", err)
	}

	body := `{"timestamp":"2025-07-01T11:00:00Z","feed_type":"breast_milk","volume_ml":150,"notes":"updated"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/feedings/"+feeding.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateFeedingHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/feedings/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp["feed_type"] != "breast_milk" {
		t.Errorf("expected feed_type=breast_milk, got %v", resp["feed_type"])
	}
	if resp["updated_by"] == nil || resp["updated_by"] == "" {
		t.Error("expected non-empty updated_by after update")
	}
}

func TestUpdateFeedingHandler_Unauthorized(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	otherUser := testutil.CreateTestUser(t, db)

	feeding, err := store.CreateFeeding(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "formula", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateFeeding failed: %v", err)
	}

	body := `{"timestamp":"2025-07-01T11:00:00Z","feed_type":"breast_milk"}`
	req := testutil.AuthenticatedRequest(t, db, otherUser.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/feedings/"+feeding.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateFeedingHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/feedings/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

// --- DELETE /api/babies/{id}/feedings/{entryId} ---

func TestDeleteFeedingHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	feeding, err := store.CreateFeeding(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "formula", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateFeeding failed: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodDelete, "/api/babies/"+baby.ID+"/feedings/"+feeding.ID)

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.DeleteFeedingHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("DELETE /api/babies/{id}/feedings/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// Verify it's actually deleted
	_, err = store.GetFeedingByID(db, baby.ID, feeding.ID)
	if err == nil {
		t.Error("expected feeding to be deleted")
	}
}

func TestDeleteFeedingHandler_NotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodDelete, "/api/babies/"+baby.ID+"/feedings/nonexistent")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.DeleteFeedingHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("DELETE /api/babies/{id}/feedings/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestCreateFeedingHandler_InvalidJSON(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/feedings")
	req.Body = io.NopCloser(bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateFeedingHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/feedings", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateFeedingHandler_InvalidTimestamp(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"not-a-timestamp","feed_type":"formula"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/feedings")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateFeedingHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/feedings", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdateFeedingHandler_InvalidJSON(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	feeding, err := store.CreateFeeding(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "formula", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateFeeding failed: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/feedings/"+feeding.ID)
	req.Body = io.NopCloser(bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateFeedingHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/feedings/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdateFeedingHandler_NotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T11:00:00Z","feed_type":"formula"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/feedings/nonexistent")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateFeedingHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/feedings/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateFeedingHandler_InvalidFeedType(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	feeding, err := store.CreateFeeding(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "formula", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateFeeding failed: %v", err)
	}

	body := `{"timestamp":"2025-07-01T11:00:00Z","feed_type":"invalid"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/feedings/"+feeding.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateFeedingHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/feedings/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestListFeedingsHandler_WithTimezoneHeader(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	// A feeding at 2025-07-02 03:00 UTC = 2025-07-01 23:00 EDT
	_, err := store.CreateFeeding(db, baby.ID, user.ID, "2025-07-02T03:00:00Z", "formula", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateFeeding failed: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/feedings?from=2025-07-01&to=2025-07-01")
	req.Header.Set("X-Timezone", "America/New_York")
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.ListFeedingsHandler(db)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/feedings", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Errorf("expected 1 feeding in NY timezone, got %d", len(resp.Data))
	}
}

func TestListFeedingsHandler_WithCursor(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	// Create enough feedings to test cursor pagination
	for i := 0; i < 3; i++ {
		_, err := store.CreateFeeding(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "formula", nil, nil, nil, nil)
		if err != nil {
			t.Fatalf("CreateFeeding failed: %v", err)
		}
	}

	// First get all to find a cursor
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/feedings")
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.ListFeedingsHandler(db)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/feedings", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(rec.Body.Bytes(), &resp)

	// Use last ID as cursor
	cursor := resp.Data[len(resp.Data)-1].ID
	req2 := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/feedings?cursor="+cursor)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec2.Code)
	}
}

func TestDeleteFeedingHandler_Unauthorized(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	otherUser := testutil.CreateTestUser(t, db)

	feeding, err := store.CreateFeeding(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "formula", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateFeeding failed: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, otherUser.ID, testCookieName, testSecret, http.MethodDelete, "/api/babies/"+baby.ID+"/feedings/"+feeding.ID)

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.DeleteFeedingHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("DELETE /api/babies/{id}/feedings/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestGetFeedingHandler_Unauthorized(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	otherUser := testutil.CreateTestUser(t, db)

	feeding, err := store.CreateFeeding(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "formula", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateFeeding failed: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, otherUser.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/feedings/"+feeding.ID)
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.GetFeedingHandler(db)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/feedings/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestCreateFeedingHandler_BabyNotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	body := `{"timestamp":"2025-07-01T10:30:00Z","feed_type":"formula"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/nonexistent/feedings")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateFeedingHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/feedings", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateFeedingHandler_MissingFeedType(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T10:30:00Z"}` // missing feed_type
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/feedings")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateFeedingHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/feedings", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
