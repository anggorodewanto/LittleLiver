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

// --- POST /api/babies/{id}/upper-arm-circumferences ---

func TestCreateUpperArmCircumferenceHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T10:30:00Z","circumference_cm":11.5,"measurement_source":"home_scale","notes":"after feeding"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/upper-arm-circumferences")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateUpperArmCircumferenceHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/upper-arm-circumferences", h)
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
		t.Error("expected non-empty ID")
	}
	if resp["circumference_cm"] != 11.5 {
		t.Errorf("expected circumference_cm=11.5, got %v", resp["circumference_cm"])
	}
	if resp["measurement_source"] != "home_scale" {
		t.Errorf("expected measurement_source=home_scale, got %v", resp["measurement_source"])
	}
}

func TestCreateUpperArmCircumferenceHandler_ClinicSource(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T10:30:00Z","circumference_cm":11.5,"measurement_source":"clinic"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/upper-arm-circumferences")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateUpperArmCircumferenceHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/upper-arm-circumferences", h)
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

func TestCreateUpperArmCircumferenceHandler_InvalidMeasurementSource(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T10:30:00Z","circumference_cm":11.5,"measurement_source":"bathroom_scale"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/upper-arm-circumferences")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateUpperArmCircumferenceHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/upper-arm-circumferences", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateUpperArmCircumferenceHandler_MissingCircumferenceCm(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T10:30:00Z"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/upper-arm-circumferences")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateUpperArmCircumferenceHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/upper-arm-circumferences", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateUpperArmCircumferenceHandler_ZeroCircumference(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T10:30:00Z","circumference_cm":0}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/upper-arm-circumferences")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateUpperArmCircumferenceHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/upper-arm-circumferences", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateUpperArmCircumferenceHandler_NegativeCircumference(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T10:30:00Z","circumference_cm":-1.5}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/upper-arm-circumferences")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateUpperArmCircumferenceHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/upper-arm-circumferences", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateUpperArmCircumferenceHandler_MissingTimestamp(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"circumference_cm":11.5}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/upper-arm-circumferences")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateUpperArmCircumferenceHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/upper-arm-circumferences", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateUpperArmCircumferenceHandler_InvalidJSON(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/upper-arm-circumferences")
	req.Body = io.NopCloser(bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateUpperArmCircumferenceHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/upper-arm-circumferences", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateUpperArmCircumferenceHandler_Unauthorized(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	otherUser := testutil.CreateTestUser(t, db)

	body := `{"timestamp":"2025-07-01T10:30:00Z","circumference_cm":11.5}`
	req := testutil.AuthenticatedRequest(t, db, otherUser.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/upper-arm-circumferences")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateUpperArmCircumferenceHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/upper-arm-circumferences", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// --- GET /api/babies/{id}/upper-arm-circumferences ---

func TestListUpperArmCircumferencesHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	for i := 0; i < 3; i++ {
		_, err := store.CreateUpperArmCircumference(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 11.5, nil, nil)
		if err != nil {
			t.Fatalf("CreateUpperArmCircumference failed: %v", err)
		}
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/upper-arm-circumferences")
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.ListUpperArmCircumferencesHandler(db)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/upper-arm-circumferences", h)
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
		t.Errorf("expected 3 entries, got %d", len(resp.Data))
	}
}

// --- GET /api/babies/{id}/upper-arm-circumferences/{entryId} ---

func TestGetUpperArmCircumferenceHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	uac, err := store.CreateUpperArmCircumference(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 11.5, nil, nil)
	if err != nil {
		t.Fatalf("CreateUpperArmCircumference failed: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/upper-arm-circumferences/"+uac.ID)
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.GetUpperArmCircumferenceHandler(db)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/upper-arm-circumferences/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp["id"] != uac.ID {
		t.Errorf("expected id=%q, got %v", uac.ID, resp["id"])
	}
}

func TestGetUpperArmCircumferenceHandler_NotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/upper-arm-circumferences/nonexistent")
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.GetUpperArmCircumferenceHandler(db)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/upper-arm-circumferences/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

// --- PUT /api/babies/{id}/upper-arm-circumferences/{entryId} ---

func TestUpdateUpperArmCircumferenceHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	uac, err := store.CreateUpperArmCircumference(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 11.5, nil, nil)
	if err != nil {
		t.Fatalf("CreateUpperArmCircumference failed: %v", err)
	}

	body := `{"timestamp":"2025-07-01T11:00:00Z","circumference_cm":12.0,"measurement_source":"clinic","notes":"updated"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/upper-arm-circumferences/"+uac.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateUpperArmCircumferenceHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/upper-arm-circumferences/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp["circumference_cm"] != 12.0 {
		t.Errorf("expected circumference_cm=12.0, got %v", resp["circumference_cm"])
	}
	if resp["updated_by"] == nil || resp["updated_by"] == "" {
		t.Error("expected non-empty updated_by after update")
	}
}

func TestUpdateUpperArmCircumferenceHandler_NotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T11:00:00Z","circumference_cm":11.5}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/upper-arm-circumferences/nonexistent")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateUpperArmCircumferenceHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/upper-arm-circumferences/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// --- DELETE /api/babies/{id}/upper-arm-circumferences/{entryId} ---

func TestDeleteUpperArmCircumferenceHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	uac, err := store.CreateUpperArmCircumference(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 11.5, nil, nil)
	if err != nil {
		t.Fatalf("CreateUpperArmCircumference failed: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodDelete, "/api/babies/"+baby.ID+"/upper-arm-circumferences/"+uac.ID)

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.DeleteUpperArmCircumferenceHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("DELETE /api/babies/{id}/upper-arm-circumferences/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	_, err = store.GetUpperArmCircumferenceByID(db, baby.ID, uac.ID)
	if err == nil {
		t.Error("expected upper arm circumference to be deleted")
	}
}

func TestDeleteUpperArmCircumferenceHandler_NotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodDelete, "/api/babies/"+baby.ID+"/upper-arm-circumferences/nonexistent")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.DeleteUpperArmCircumferenceHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("DELETE /api/babies/{id}/upper-arm-circumferences/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

// Upper arm circumference without measurement_source (optional field)
func TestCreateUpperArmCircumferenceHandler_NoMeasurementSource(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T10:30:00Z","circumference_cm":11.5}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/upper-arm-circumferences")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateUpperArmCircumferenceHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/upper-arm-circumferences", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}
