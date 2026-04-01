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

// --- POST /api/babies/{id}/head-circumferences ---

func TestCreateHeadCircumferenceHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T10:30:00Z","circumference_cm":35.5,"measurement_source":"home_scale","notes":"after feeding"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/head-circumferences")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateHeadCircumferenceHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/head-circumferences", h)
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
		t.Error("expected non-empty head circumference ID")
	}
	if resp["circumference_cm"] != 35.5 {
		t.Errorf("expected circumference_cm=35.5, got %v", resp["circumference_cm"])
	}
	if resp["measurement_source"] != "home_scale" {
		t.Errorf("expected measurement_source=home_scale, got %v", resp["measurement_source"])
	}
}

func TestCreateHeadCircumferenceHandler_ClinicSource(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T10:30:00Z","circumference_cm":35.5,"measurement_source":"clinic"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/head-circumferences")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateHeadCircumferenceHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/head-circumferences", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp["measurement_source"] != "clinic" {
		t.Errorf("expected measurement_source=clinic, got %v", resp["measurement_source"])
	}
}

func TestCreateHeadCircumferenceHandler_InvalidMeasurementSource(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T10:30:00Z","circumference_cm":35.5,"measurement_source":"bathroom_scale"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/head-circumferences")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateHeadCircumferenceHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/head-circumferences", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateHeadCircumferenceHandler_MissingCircumferenceCm(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T10:30:00Z"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/head-circumferences")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateHeadCircumferenceHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/head-circumferences", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateHeadCircumferenceHandler_ZeroCircumference(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T10:30:00Z","circumference_cm":0}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/head-circumferences")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateHeadCircumferenceHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/head-circumferences", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateHeadCircumferenceHandler_NegativeCircumference(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T10:30:00Z","circumference_cm":-1.5}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/head-circumferences")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateHeadCircumferenceHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/head-circumferences", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateHeadCircumferenceHandler_MissingTimestamp(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"circumference_cm":35.5}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/head-circumferences")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateHeadCircumferenceHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/head-circumferences", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateHeadCircumferenceHandler_InvalidJSON(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/head-circumferences")
	req.Body = io.NopCloser(bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateHeadCircumferenceHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/head-circumferences", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateHeadCircumferenceHandler_Unauthorized(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	otherUser := testutil.CreateTestUser(t, db)

	body := `{"timestamp":"2025-07-01T10:30:00Z","circumference_cm":35.5}`
	req := testutil.AuthenticatedRequest(t, db, otherUser.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/head-circumferences")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateHeadCircumferenceHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/head-circumferences", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// --- GET /api/babies/{id}/head-circumferences ---

func TestListHeadCircumferencesHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	for i := 0; i < 3; i++ {
		_, err := store.CreateHeadCircumference(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 35.5, nil, nil)
		if err != nil {
			t.Fatalf("CreateHeadCircumference failed: %v", err)
		}
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/head-circumferences")
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.ListHeadCircumferencesHandler(db)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/head-circumferences", h)
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
		t.Errorf("expected 3 head circumferences, got %d", len(resp.Data))
	}
}

// --- GET /api/babies/{id}/head-circumferences/{entryId} ---

func TestGetHeadCircumferenceHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	hc, err := store.CreateHeadCircumference(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 35.5, nil, nil)
	if err != nil {
		t.Fatalf("CreateHeadCircumference failed: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/head-circumferences/"+hc.ID)
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.GetHeadCircumferenceHandler(db)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/head-circumferences/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp["id"] != hc.ID {
		t.Errorf("expected id=%q, got %v", hc.ID, resp["id"])
	}
}

func TestGetHeadCircumferenceHandler_NotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/head-circumferences/nonexistent")
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.GetHeadCircumferenceHandler(db)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/head-circumferences/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

// --- PUT /api/babies/{id}/head-circumferences/{entryId} ---

func TestUpdateHeadCircumferenceHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	hc, err := store.CreateHeadCircumference(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 35.5, nil, nil)
	if err != nil {
		t.Fatalf("CreateHeadCircumference failed: %v", err)
	}

	body := `{"timestamp":"2025-07-01T11:00:00Z","circumference_cm":36.0,"measurement_source":"clinic","notes":"updated"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/head-circumferences/"+hc.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateHeadCircumferenceHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/head-circumferences/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp["circumference_cm"] != 36.0 {
		t.Errorf("expected circumference_cm=36.0, got %v", resp["circumference_cm"])
	}
	if resp["updated_by"] == nil || resp["updated_by"] == "" {
		t.Error("expected non-empty updated_by after update")
	}
}

func TestUpdateHeadCircumferenceHandler_NotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T11:00:00Z","circumference_cm":35.5}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/head-circumferences/nonexistent")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateHeadCircumferenceHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/head-circumferences/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// --- DELETE /api/babies/{id}/head-circumferences/{entryId} ---

func TestDeleteHeadCircumferenceHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	hc, err := store.CreateHeadCircumference(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 35.5, nil, nil)
	if err != nil {
		t.Fatalf("CreateHeadCircumference failed: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodDelete, "/api/babies/"+baby.ID+"/head-circumferences/"+hc.ID)

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.DeleteHeadCircumferenceHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("DELETE /api/babies/{id}/head-circumferences/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	_, err = store.GetHeadCircumferenceByID(db, baby.ID, hc.ID)
	if err == nil {
		t.Error("expected head circumference to be deleted")
	}
}

func TestDeleteHeadCircumferenceHandler_NotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodDelete, "/api/babies/"+baby.ID+"/head-circumferences/nonexistent")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.DeleteHeadCircumferenceHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("DELETE /api/babies/{id}/head-circumferences/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

// Head circumference without measurement_source (optional field)
func TestCreateHeadCircumferenceHandler_NoMeasurementSource(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T10:30:00Z","circumference_cm":35.5}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/head-circumferences")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateHeadCircumferenceHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/head-circumferences", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}
