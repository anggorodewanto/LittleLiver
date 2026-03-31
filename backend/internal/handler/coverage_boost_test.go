package handler_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/ablankz/LittleLiver/backend/internal/handler"
	"github.com/ablankz/LittleLiver/backend/internal/middleware"
	"github.com/ablankz/LittleLiver/backend/internal/storage"
	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

// ============================================================
// List handlers with from/to query params (covers filter paths)
// ============================================================

func TestListWeightsHandler_WithFromTo(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	store.CreateWeight(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 4.25, nil, nil)
	store.CreateWeight(db, baby.ID, user.ID, "2025-07-02T10:30:00Z", 4.30, nil, nil)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/weights?from=2025-07-01&to=2025-07-01")
	req.Header.Set("X-Timezone", "America/New_York")
	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/weights", authMw(http.HandlerFunc(handler.ListWeightsHandler(db))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp map[string]any
	json.Unmarshal(rec.Body.Bytes(), &resp)
	data := resp["data"].([]any)
	if len(data) != 1 {
		t.Errorf("expected 1 weight for date range, got %d", len(data))
	}
}

func TestListTemperaturesHandler_WithFromTo(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	store.CreateTemperature(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 37.0, "rectal", nil)
	store.CreateTemperature(db, baby.ID, user.ID, "2025-07-02T10:30:00Z", 37.5, "rectal", nil)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/temperatures?from=2025-07-01&to=2025-07-01")
	req.Header.Set("X-Timezone", "UTC")
	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/temperatures", authMw(http.HandlerFunc(handler.ListTemperaturesHandler(db))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp map[string]any
	json.Unmarshal(rec.Body.Bytes(), &resp)
	data := resp["data"].([]any)
	if len(data) != 1 {
		t.Errorf("expected 1 temperature for date range, got %d", len(data))
	}
}

func TestListUrineHandler_WithFromTo(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	store.CreateUrine(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", nil, nil, nil)
	store.CreateUrine(db, baby.ID, user.ID, "2025-07-02T10:30:00Z", nil, nil, nil)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/urine?from=2025-07-01&to=2025-07-01")
	req.Header.Set("X-Timezone", "UTC")
	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/urine", authMw(http.HandlerFunc(handler.ListUrineHandler(db))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp map[string]any
	json.Unmarshal(rec.Body.Bytes(), &resp)
	data := resp["data"].([]any)
	if len(data) != 1 {
		t.Errorf("expected 1 urine for date range, got %d", len(data))
	}
}

func TestListLabResultsHandler_WithFromTo(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	store.CreateLabResult(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "bilirubin", "5.2", nil, nil, nil)
	store.CreateLabResult(db, baby.ID, user.ID, "2025-07-02T10:30:00Z", "bilirubin", "4.8", nil, nil, nil)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/labs?from=2025-07-02&to=2025-07-02")
	req.Header.Set("X-Timezone", "UTC")
	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/labs", authMw(http.HandlerFunc(handler.ListLabResultsHandler(db))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp map[string]any
	json.Unmarshal(rec.Body.Bytes(), &resp)
	data := resp["data"].([]any)
	if len(data) != 1 {
		t.Errorf("expected 1 lab result for date range, got %d", len(data))
	}
}

func TestListMedicationsHandler_WithData(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	store.CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, nil, nil)
	store.CreateMedication(db, baby.ID, user.ID, "VitD", "400IU", "once_daily", nil, nil, nil)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/medications")
	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/medications", authMw(http.HandlerFunc(handler.ListMedicationsHandler(db))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp []any
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if len(resp) != 2 {
		t.Errorf("expected 2 medications, got %d", len(resp))
	}
}

func TestListMedLogsHandler_WithFromTo(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	med, _ := store.CreateMedication(db, baby.ID, user.ID, "VitD", "400IU", "once_daily", nil, nil, nil)
	givenAt := "2025-07-01T08:00:00Z"
	store.CreateMedLog(db, baby.ID, med.ID, user.ID, nil, &givenAt, false, nil, nil)
	givenAt2 := "2025-07-02T08:00:00Z"
	store.CreateMedLog(db, baby.ID, med.ID, user.ID, nil, &givenAt2, false, nil, nil)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/med-logs?from=2025-07-01&to=2025-07-01")
	req.Header.Set("X-Timezone", "UTC")
	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/med-logs", authMw(http.HandlerFunc(handler.ListMedLogsHandler(db))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestListStoolsHandler_WithFromTo(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	store.CreateStool(db, baby.ID, user.ID, "2025-07-01T11:00:00Z", 5, nil, nil, nil, nil, nil)
	store.CreateStool(db, baby.ID, user.ID, "2025-07-02T11:00:00Z", 3, nil, nil, nil, nil, nil)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/stools?from=2025-07-01&to=2025-07-01")
	req.Header.Set("X-Timezone", "UTC")
	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/stools", authMw(http.HandlerFunc(handler.ListStoolsHandler(db, objStore))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp map[string]any
	json.Unmarshal(rec.Body.Bytes(), &resp)
	data := resp["data"].([]any)
	if len(data) != 1 {
		t.Errorf("expected 1 stool for date range, got %d", len(data))
	}
}

func TestListAbdomenHandler_WithFromTo(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	store.CreateAbdomen(db, baby.ID, user.ID, "2025-07-01T09:00:00Z", "soft", false, nil, nil)
	store.CreateAbdomen(db, baby.ID, user.ID, "2025-07-02T09:00:00Z", "firm", true, nil, nil)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/abdomen?from=2025-07-02&to=2025-07-02")
	req.Header.Set("X-Timezone", "UTC")
	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/abdomen", authMw(http.HandlerFunc(handler.ListAbdomenHandler(db, objStore))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp map[string]any
	json.Unmarshal(rec.Body.Bytes(), &resp)
	data := resp["data"].([]any)
	if len(data) != 1 {
		t.Errorf("expected 1 abdomen for date range, got %d", len(data))
	}
}

func TestListBruisingHandler_WithFromTo(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	store.CreateBruising(db, baby.ID, user.ID, "2025-07-01T09:00:00Z", "arm", "small_<1cm", nil, nil, nil)
	store.CreateBruising(db, baby.ID, user.ID, "2025-07-02T09:00:00Z", "leg", "medium_1-3cm", nil, nil, nil)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/bruising?from=2025-07-01&to=2025-07-01")
	req.Header.Set("X-Timezone", "UTC")
	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/bruising", authMw(http.HandlerFunc(handler.ListBruisingHandler(db, objStore))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp map[string]any
	json.Unmarshal(rec.Body.Bytes(), &resp)
	data := resp["data"].([]any)
	if len(data) != 1 {
		t.Errorf("expected 1 bruising for date range, got %d", len(data))
	}
}

func TestListSkinObservationsHandler_WithFromTo(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	store.CreateSkinObservation(db, baby.ID, user.ID, "2025-07-01T09:00:00Z", nil, false, nil, nil, nil)
	store.CreateSkinObservation(db, baby.ID, user.ID, "2025-07-02T09:00:00Z", nil, true, nil, nil, nil)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/skin?from=2025-07-02&to=2025-07-02")
	req.Header.Set("X-Timezone", "UTC")
	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/skin", authMw(http.HandlerFunc(handler.ListSkinObservationsHandler(db, objStore))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp map[string]any
	json.Unmarshal(rec.Body.Bytes(), &resp)
	data := resp["data"].([]any)
	if len(data) != 1 {
		t.Errorf("expected 1 skin observation for date range, got %d", len(data))
	}
}

func TestListGeneralNotesHandler_WithFromTo(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	store.CreateGeneralNote(db, baby.ID, user.ID, "2025-07-01T09:00:00Z", "note 1", nil, nil)
	store.CreateGeneralNote(db, baby.ID, user.ID, "2025-07-02T09:00:00Z", "note 2", nil, nil)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/notes?from=2025-07-01&to=2025-07-01")
	req.Header.Set("X-Timezone", "UTC")
	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/notes", authMw(http.HandlerFunc(handler.ListGeneralNotesHandler(db, objStore))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp map[string]any
	json.Unmarshal(rec.Body.Bytes(), &resp)
	data := resp["data"].([]any)
	if len(data) != 1 {
		t.Errorf("expected 1 general note for date range, got %d", len(data))
	}
}

// ============================================================
// Create handlers — validation error paths not covered elsewhere
// ============================================================

func TestCreateMedLogHandler_ValidationError_MissingMedicationID(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"given_at":"2025-07-01T08:00:00Z","skipped":false}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/med-logs")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/med-logs", authMw(csrfMw(http.HandlerFunc(handler.CreateMedLogHandler(db)))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

// ============================================================
// CSRFTokenHandler — direct call without session token in context
// ============================================================

func TestCSRFTokenHandler_NoSessionInContext(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	h := http.HandlerFunc(handler.CSRFTokenHandler(testSecret))
	req := httptest.NewRequest(http.MethodGet, "/api/csrf-token", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when no session in context, got %d", rec.Code)
	}
}

// ============================================================
// TestLoginHandler — invalid JSON body
// ============================================================

func TestTestLoginHandler_InvalidJSON(t *testing.T) {
	os.Setenv("TEST_MODE", "1")
	defer os.Unsetenv("TEST_MODE")

	db := testutil.SetupTestDB(t)
	defer db.Close()

	h := handler.TestLoginHandler(db)
	req := httptest.NewRequest("POST", "/api/test/login", strings.NewReader("not json"))
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d", rec.Code)
	}
}

// ============================================================
// Baby validation — invalid kasai_date format
// ============================================================

func TestCreateBabyHandler_InvalidKasaiDateFormat(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	body := `{"name":"Luna","sex":"female","date_of_birth":"2025-06-15","kasai_date":"bad-date"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateBabyHandler(db))))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid kasai_date, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// ============================================================
// Update med-log invalid JSON (not covered elsewhere)
// ============================================================

func TestUpdateMedLogHandler_InvalidJSON_Boost(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	med, _ := store.CreateMedication(db, baby.ID, user.ID, "Test", "10mg", "once_daily", nil, nil, nil)
	givenAt := "2025-07-01T08:00:00Z"
	ml, _ := store.CreateMedLog(db, baby.ID, med.ID, user.ID, nil, &givenAt, false, nil, nil)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/med-logs/"+ml.ID)
	req.Body = io.NopCloser(bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/med-logs/{entryId}", authMw(csrfMw(http.HandlerFunc(handler.UpdateMedLogHandler(db)))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

// ============================================================
// Store error paths — force DB errors by dropping tables
// These test the log.Printf + http.Error 500 paths
// ============================================================

// helperDropTable drops a table to force store errors.
func helperDropTable(t *testing.T, db *sql.DB, table string) {
	t.Helper()
	_, err := db.Exec("DROP TABLE IF EXISTS " + table)
	if err != nil {
		t.Fatalf("failed to drop table %s: %v", table, err)
	}
}

func TestListWeightsHandler_StoreError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/weights")
	helperDropTable(t, db, "weights")

	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/weights", authMw(http.HandlerFunc(handler.ListWeightsHandler(db))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on store error, got %d", rec.Code)
	}
}

func TestListTemperaturesHandler_StoreError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/temperatures")
	helperDropTable(t, db, "temperatures")

	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/temperatures", authMw(http.HandlerFunc(handler.ListTemperaturesHandler(db))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestListUrineHandler_StoreError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/urine")
	helperDropTable(t, db, "urine")

	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/urine", authMw(http.HandlerFunc(handler.ListUrineHandler(db))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestListLabResultsHandler_StoreError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/labs")
	helperDropTable(t, db, "lab_results")

	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/labs", authMw(http.HandlerFunc(handler.ListLabResultsHandler(db))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestListFeedingsHandler_StoreError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/feedings")
	helperDropTable(t, db, "feedings")

	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/feedings", authMw(http.HandlerFunc(handler.ListFeedingsHandler(db))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestListStoolsHandler_StoreError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/stools")
	helperDropTable(t, db, "stools")

	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/stools", authMw(http.HandlerFunc(handler.ListStoolsHandler(db, nil))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestListAbdomenHandler_StoreError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/abdomen")
	helperDropTable(t, db, "abdomen_observations")

	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/abdomen", authMw(http.HandlerFunc(handler.ListAbdomenHandler(db, nil))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestListBruisingHandler_StoreError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/bruising")
	helperDropTable(t, db, "bruising")

	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/bruising", authMw(http.HandlerFunc(handler.ListBruisingHandler(db, nil))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestListSkinObservationsHandler_StoreError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/skin")
	helperDropTable(t, db, "skin_observations")

	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/skin", authMw(http.HandlerFunc(handler.ListSkinObservationsHandler(db, nil))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestListGeneralNotesHandler_StoreError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/notes")
	helperDropTable(t, db, "general_notes")

	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/notes", authMw(http.HandlerFunc(handler.ListGeneralNotesHandler(db, nil))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestListMedicationsHandler_StoreError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/medications")
	helperDropTable(t, db, "medications")

	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/medications", authMw(http.HandlerFunc(handler.ListMedicationsHandler(db))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestListMedLogsHandler_StoreError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/med-logs")
	helperDropTable(t, db, "med_logs")

	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/med-logs", authMw(http.HandlerFunc(handler.ListMedLogsHandler(db))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestCreateWeightHandler_StoreError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T10:30:00Z","weight_kg":4.25}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/weights")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	helperDropTable(t, db, "weights")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/weights", authMw(csrfMw(http.HandlerFunc(handler.CreateWeightHandler(db)))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on store error, got %d", rec.Code)
	}
}

func TestCreateFeedingHandler_StoreError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T10:30:00Z","feed_type":"formula"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/feedings")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	helperDropTable(t, db, "feedings")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/feedings", authMw(csrfMw(http.HandlerFunc(handler.CreateFeedingHandler(db)))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestCreateTemperatureHandler_StoreError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T10:30:00Z","value":37.0,"method":"rectal"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/temperatures")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	helperDropTable(t, db, "temperatures")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/temperatures", authMw(csrfMw(http.HandlerFunc(handler.CreateTemperatureHandler(db)))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestCreateLabResultHandler_StoreError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T10:30:00Z","test_name":"bilirubin","value":"5.2"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/labs")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	helperDropTable(t, db, "lab_results")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/labs", authMw(csrfMw(http.HandlerFunc(handler.CreateLabResultHandler(db)))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestMeHandler_StoreError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/me")
	helperDropTable(t, db, "baby_parents")

	authMw := middleware.Auth(db, testCookieName)
	h := authMw(handler.MeHandler(db))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestListBabiesHandler_StoreError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies")
	helperDropTable(t, db, "baby_parents")

	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.ListBabiesHandler(db)))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestCreateStoolHandler_StoreError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T10:30:00Z","color_rating":3}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/stools")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	helperDropTable(t, db, "stools")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/stools", authMw(csrfMw(http.HandlerFunc(handler.CreateStoolHandler(db, nil)))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestCreateAbdomenHandler_StoreError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T10:30:00Z","firmness":"soft"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/abdomen")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	helperDropTable(t, db, "abdomen_observations")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/abdomen", authMw(csrfMw(http.HandlerFunc(handler.CreateAbdomenHandler(db, nil)))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestCreateBruisingHandler_StoreError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T10:30:00Z","location":"arm","size_estimate":"small_<1cm"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/bruising")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	helperDropTable(t, db, "bruising")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/bruising", authMw(csrfMw(http.HandlerFunc(handler.CreateBruisingHandler(db, nil)))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestCreateSkinObservationHandler_StoreError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T10:30:00Z"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/skin")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	helperDropTable(t, db, "skin_observations")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/skin", authMw(csrfMw(http.HandlerFunc(handler.CreateSkinObservationHandler(db, nil)))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestCreateGeneralNoteHandler_StoreError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T10:30:00Z","content":"note text"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/notes")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	helperDropTable(t, db, "general_notes")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/notes", authMw(csrfMw(http.HandlerFunc(handler.CreateGeneralNoteHandler(db, nil)))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestCreateMedicationHandler_StoreError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"name":"Ursodiol","dose":"50mg","frequency":"twice_daily"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/medications")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	helperDropTable(t, db, "medications")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/medications", authMw(csrfMw(http.HandlerFunc(handler.CreateMedicationHandler(db)))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestCreateUrineHandler_StoreError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T10:30:00Z"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/urine")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	helperDropTable(t, db, "urine")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/urine", authMw(csrfMw(http.HandlerFunc(handler.CreateUrineHandler(db)))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestDashboardHandler_StoreError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/dashboard")
	helperDropTable(t, db, "feedings")

	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/dashboard", authMw(http.HandlerFunc(handler.DashboardHandler(db))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestCreateInviteHandler_StoreError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/invite")
	req.SetPathValue("id", baby.ID)

	helperDropTable(t, db, "invites")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateInviteHandler(db))))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestSubscribePushHandler_StoreError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	body := `{"endpoint":"https://push.example.com/err","p256dh":"key","auth":"auth"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/push/subscribe")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	helperDropTable(t, db, "push_subscriptions")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.SubscribePushHandler(db))))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestCreateMedLogHandler_StoreError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	med, _ := store.CreateMedication(db, baby.ID, user.ID, "Test", "10mg", "once_daily", nil, nil, nil)

	body := `{"medication_id":"` + med.ID + `","given_at":"2025-07-01T08:00:00Z","skipped":false}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/med-logs")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	helperDropTable(t, db, "med_logs")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/med-logs", authMw(csrfMw(http.HandlerFunc(handler.CreateMedLogHandler(db)))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

// UnlinkSelfHandler store error is hard to trigger without mocking since
// the UnlinkParent function uses a transaction that cascades to baby deletion.

func TestCreateBabyHandler_StoreError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	body := `{"name":"Luna","sex":"female","date_of_birth":"2025-06-15"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	helperDropTable(t, db, "babies")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateBabyHandler(db))))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

// UpdateBabyHandler store error is hard to trigger: requireBabyAccess reads from
// the same table, so dropping it would cause a 403/404 before reaching the update call.

// Test dashboard with multiple sequential store errors (covers more error paths)
func TestDashboardHandler_StoolTrendError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/dashboard")
	helperDropTable(t, db, "stools")

	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/dashboard", authMw(http.HandlerFunc(handler.DashboardHandler(db))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestDashboardHandler_MedicationsError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/dashboard")
	helperDropTable(t, db, "medications")

	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/dashboard", authMw(http.HandlerFunc(handler.DashboardHandler(db))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestDashboardHandler_TemperatureSeriesError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/dashboard")
	helperDropTable(t, db, "temperatures")

	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/dashboard", authMw(http.HandlerFunc(handler.DashboardHandler(db))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestDashboardHandler_WeightSeriesError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/dashboard")
	helperDropTable(t, db, "weights")

	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/dashboard", authMw(http.HandlerFunc(handler.DashboardHandler(db))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestDashboardHandler_AbdomenSeriesError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/dashboard")
	helperDropTable(t, db, "abdomen_observations")

	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/dashboard", authMw(http.HandlerFunc(handler.DashboardHandler(db))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestDashboardHandler_LabTrendsError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/dashboard")
	helperDropTable(t, db, "lab_results")

	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/dashboard", authMw(http.HandlerFunc(handler.DashboardHandler(db))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestDashboardHandler_AlertsError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/dashboard")
	helperDropTable(t, db, "skin_observations")

	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/dashboard", authMw(http.HandlerFunc(handler.DashboardHandler(db))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

// ============================================================
// Dashboard — more error paths (diaper/stool color series)
// ============================================================

func TestDashboardHandler_DiaperDailyError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/dashboard")
	helperDropTable(t, db, "urine")

	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/dashboard", authMw(http.HandlerFunc(handler.DashboardHandler(db))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

// TestDashboardHandler_StoolColorSeriesError is skipped because it shares
// the same table (stools) as StoolTrendError and can't be isolated.

// ============================================================
// Get handlers with object store (exercises resolvePhotos)
// ============================================================

func TestGetAbdomenHandler_WithObjStore(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	abdomen, _ := store.CreateAbdomen(db, baby.ID, user.ID, "2025-07-01T09:00:00Z", "soft", false, nil, nil)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/abdomen/"+abdomen.ID)
	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/abdomen/{entryId}", authMw(http.HandlerFunc(handler.GetAbdomenHandler(db, objStore))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestGetBruisingHandler_WithObjStore(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	b, _ := store.CreateBruising(db, baby.ID, user.ID, "2025-07-01T09:00:00Z", "arm", "small_<1cm", nil, nil, nil)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/bruising/"+b.ID)
	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/bruising/{entryId}", authMw(http.HandlerFunc(handler.GetBruisingHandler(db, objStore))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestGetSkinObservationHandler_WithObjStore(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	s, _ := store.CreateSkinObservation(db, baby.ID, user.ID, "2025-07-01T09:00:00Z", nil, false, nil, nil, nil)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/skin/"+s.ID)
	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/skin/{entryId}", authMw(http.HandlerFunc(handler.GetSkinObservationHandler(db, objStore))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestGetGeneralNoteHandler_WithObjStore(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	n, _ := store.CreateGeneralNote(db, baby.ID, user.ID, "2025-07-01T09:00:00Z", "test note", nil, nil)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/notes/"+n.ID)
	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/notes/{entryId}", authMw(http.HandlerFunc(handler.GetGeneralNoteHandler(db, objStore))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

// ============================================================
// Delete handlers with obj store (exercises unlinkAllPhotos)
// ============================================================

func TestDeleteStoolHandler_WithObjStore(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	s, _ := store.CreateStool(db, baby.ID, user.ID, "2025-07-01T11:00:00Z", 5, nil, nil, nil, nil, nil)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodDelete, "/api/babies/"+baby.ID+"/stools/"+s.ID)
	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("DELETE /api/babies/{id}/stools/{entryId}", authMw(csrfMw(http.HandlerFunc(handler.DeleteStoolHandler(db, objStore)))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

func TestDeleteAbdomenHandler_WithObjStore(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	a, _ := store.CreateAbdomen(db, baby.ID, user.ID, "2025-07-01T09:00:00Z", "soft", false, nil, nil)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodDelete, "/api/babies/"+baby.ID+"/abdomen/"+a.ID)
	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("DELETE /api/babies/{id}/abdomen/{entryId}", authMw(csrfMw(http.HandlerFunc(handler.DeleteAbdomenHandler(db, objStore)))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

func TestDeleteBruisingHandler_WithObjStore(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	b, _ := store.CreateBruising(db, baby.ID, user.ID, "2025-07-01T09:00:00Z", "arm", "small_<1cm", nil, nil, nil)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodDelete, "/api/babies/"+baby.ID+"/bruising/"+b.ID)
	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("DELETE /api/babies/{id}/bruising/{entryId}", authMw(csrfMw(http.HandlerFunc(handler.DeleteBruisingHandler(db, objStore)))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

func TestDeleteSkinObservationHandler_WithObjStore(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	s, _ := store.CreateSkinObservation(db, baby.ID, user.ID, "2025-07-01T09:00:00Z", nil, false, nil, nil, nil)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodDelete, "/api/babies/"+baby.ID+"/skin/"+s.ID)
	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("DELETE /api/babies/{id}/skin/{entryId}", authMw(csrfMw(http.HandlerFunc(handler.DeleteSkinObservationHandler(db, objStore)))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

func TestDeleteGeneralNoteHandler_WithObjStore(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	n, _ := store.CreateGeneralNote(db, baby.ID, user.ID, "2025-07-01T09:00:00Z", "test", nil, nil)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodDelete, "/api/babies/"+baby.ID+"/notes/"+n.ID)
	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("DELETE /api/babies/{id}/notes/{entryId}", authMw(csrfMw(http.HandlerFunc(handler.DeleteGeneralNoteHandler(db, objStore)))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

// ============================================================
// Update handlers with obj store (exercises photo linking paths)
// ============================================================

func TestUpdateStoolHandler_WithObjStore(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	s, _ := store.CreateStool(db, baby.ID, user.ID, "2025-07-01T11:00:00Z", 5, nil, nil, nil, nil, nil)

	body := `{"timestamp":"2025-07-01T12:00:00Z","color_rating":4}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/stools/"+s.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/stools/{entryId}", authMw(csrfMw(http.HandlerFunc(handler.UpdateStoolHandler(db, objStore)))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateAbdomenHandler_WithObjStore(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	a, _ := store.CreateAbdomen(db, baby.ID, user.ID, "2025-07-01T09:00:00Z", "soft", false, nil, nil)

	body := `{"timestamp":"2025-07-01T10:00:00Z","firmness":"firm","tenderness":true}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/abdomen/"+a.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/abdomen/{entryId}", authMw(csrfMw(http.HandlerFunc(handler.UpdateAbdomenHandler(db, objStore)))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateBruisingHandler_WithObjStore(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	b, _ := store.CreateBruising(db, baby.ID, user.ID, "2025-07-01T09:00:00Z", "arm", "small_<1cm", nil, nil, nil)

	body := `{"timestamp":"2025-07-01T10:00:00Z","location":"leg","size_estimate":"medium_1-3cm"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/bruising/"+b.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/bruising/{entryId}", authMw(csrfMw(http.HandlerFunc(handler.UpdateBruisingHandler(db, objStore)))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateSkinObservationHandler_WithObjStore(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	s, _ := store.CreateSkinObservation(db, baby.ID, user.ID, "2025-07-01T09:00:00Z", nil, false, nil, nil, nil)

	body := `{"timestamp":"2025-07-01T10:00:00Z","dry_skin":true}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/skin/"+s.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/skin/{entryId}", authMw(csrfMw(http.HandlerFunc(handler.UpdateSkinObservationHandler(db, objStore)))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateGeneralNoteHandler_WithObjStore(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	n, _ := store.CreateGeneralNote(db, baby.ID, user.ID, "2025-07-01T09:00:00Z", "original", nil, nil)

	body := `{"timestamp":"2025-07-01T10:00:00Z","content":"updated note"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/notes/"+n.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/notes/{entryId}", authMw(csrfMw(http.HandlerFunc(handler.UpdateGeneralNoteHandler(db, objStore)))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// Suppress unused import warnings
var _ = storage.NewMemoryStore
