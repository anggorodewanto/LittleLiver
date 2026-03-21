package handler_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/handler"
	"github.com/ablankz/LittleLiver/backend/internal/middleware"
	"github.com/ablankz/LittleLiver/backend/internal/storage"
	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

// ============================================================
// Feeding handler — update validation error path
// ============================================================

func TestUpdateFeedingHandler_ValidationError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	feeding, err := store.CreateFeeding(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "formula", nil, nil, nil, nil, baby.DefaultCalPerFeed)
	if err != nil {
		t.Fatalf("CreateFeeding failed: %v", err)
	}

	body := `{"feed_type":""}`
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
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// ============================================================
// Temperature handler — update invalid JSON and validation error
// ============================================================

func TestUpdateTemperatureHandler_InvalidJSON(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	temp, err := store.CreateTemperature(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 37.0, "rectal", nil)
	if err != nil {
		t.Fatalf("CreateTemperature failed: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/temperatures/"+temp.ID)
	req.Body = io.NopCloser(bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateTemperatureHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/temperatures/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdateTemperatureHandler_ValidationError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	temp, err := store.CreateTemperature(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 37.0, "rectal", nil)
	if err != nil {
		t.Fatalf("CreateTemperature failed: %v", err)
	}

	body := `{"timestamp":"2025-07-01T11:00:00Z"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/temperatures/"+temp.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateTemperatureHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/temperatures/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// ============================================================
// Weight handler — update invalid JSON and validation error
// ============================================================

func TestUpdateWeightHandler_InvalidJSON(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	weight, err := store.CreateWeight(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 4.25, nil, nil)
	if err != nil {
		t.Fatalf("CreateWeight failed: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/weights/"+weight.ID)
	req.Body = io.NopCloser(bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateWeightHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/weights/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdateWeightHandler_ValidationError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	weight, err := store.CreateWeight(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 4.25, nil, nil)
	if err != nil {
		t.Fatalf("CreateWeight failed: %v", err)
	}

	body := `{"timestamp":"2025-07-01T11:00:00Z"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/weights/"+weight.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateWeightHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/weights/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// ============================================================
// Urine handler — update invalid JSON and validation error
// ============================================================

func TestUpdateUrineHandler_InvalidJSON(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	urine, err := store.CreateUrine(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", nil, nil)
	if err != nil {
		t.Fatalf("CreateUrine failed: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/urine/"+urine.ID)
	req.Body = io.NopCloser(bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateUrineHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/urine/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdateUrineHandler_ValidationError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	urine, err := store.CreateUrine(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", nil, nil)
	if err != nil {
		t.Fatalf("CreateUrine failed: %v", err)
	}

	body := `{"timestamp":"not-a-timestamp"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/urine/"+urine.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateUrineHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/urine/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// ============================================================
// Abdomen handler — update invalid JSON and validation error
// ============================================================

func TestUpdateAbdomenHandler_InvalidJSON(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	abdomen, err := store.CreateAbdomen(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "soft", false, nil, nil)
	if err != nil {
		t.Fatalf("CreateAbdomen failed: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/abdomen/"+abdomen.ID)
	req.Body = io.NopCloser(bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateAbdomenHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/abdomen/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdateAbdomenHandler_ValidationError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	abdomen, err := store.CreateAbdomen(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "soft", false, nil, nil)
	if err != nil {
		t.Fatalf("CreateAbdomen failed: %v", err)
	}

	body := `{"timestamp":"2025-07-01T11:00:00Z"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/abdomen/"+abdomen.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateAbdomenHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/abdomen/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// ============================================================
// Bruising handler — update invalid JSON and validation error
// ============================================================

func TestUpdateBruisingHandler_InvalidJSON(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	bruising, err := store.CreateBruising(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "arm", "small", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBruising failed: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/bruising/"+bruising.ID)
	req.Body = io.NopCloser(bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateBruisingHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/bruising/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdateBruisingHandler_ValidationError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	bruising, err := store.CreateBruising(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "arm", "small", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBruising failed: %v", err)
	}

	body := `{"timestamp":"2025-07-01T11:00:00Z"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/bruising/"+bruising.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateBruisingHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/bruising/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// ============================================================
// Skin Observation handler — update invalid JSON and validation error
// ============================================================

func TestUpdateSkinObservationHandler_InvalidJSON(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	skin, err := store.CreateSkinObservation(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", nil, false, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateSkinObservation failed: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/skin/"+skin.ID)
	req.Body = io.NopCloser(bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateSkinObservationHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/skin/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdateSkinObservationHandler_ValidationError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	skin, err := store.CreateSkinObservation(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", nil, false, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateSkinObservation failed: %v", err)
	}

	body := `{"timestamp":"2025-07-01T11:00:00Z","jaundice_level":"invalid_level"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/skin/"+skin.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateSkinObservationHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/skin/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// ============================================================
// General Notes handler — update invalid JSON and validation error
// ============================================================

func TestUpdateGeneralNoteHandler_InvalidJSON(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	note, err := store.CreateGeneralNote(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "test note", nil, nil)
	if err != nil {
		t.Fatalf("CreateGeneralNote failed: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/notes/"+note.ID)
	req.Body = io.NopCloser(bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateGeneralNoteHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/notes/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdateGeneralNoteHandler_ValidationError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	note, err := store.CreateGeneralNote(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "test note", nil, nil)
	if err != nil {
		t.Fatalf("CreateGeneralNote failed: %v", err)
	}

	body := `{"timestamp":"2025-07-01T11:00:00Z"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/notes/"+note.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateGeneralNoteHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/notes/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// ============================================================
// Lab Results handler — update invalid JSON and validation error
// ============================================================

func TestUpdateLabResultHandler_InvalidJSON(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	lab, err := store.CreateLabResult(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "bilirubin", "5.2", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateLabResult failed: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/labs/"+lab.ID)
	req.Body = io.NopCloser(bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateLabResultHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/labs/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdateLabResultHandler_ValidationError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	lab, err := store.CreateLabResult(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "bilirubin", "5.2", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateLabResult failed: %v", err)
	}

	body := `{"timestamp":"2025-07-01T11:00:00Z","value":"6.0"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/labs/"+lab.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateLabResultHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/labs/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// ============================================================
// Stool handler — update invalid JSON and validation error
// ============================================================

func TestUpdateStoolHandler_InvalidJSON(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	stool, err := store.CreateStool(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 3, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateStool failed: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/stools/"+stool.ID)
	req.Body = io.NopCloser(bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateStoolHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/stools/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdateStoolHandler_ValidationError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	stool, err := store.CreateStool(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 3, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateStool failed: %v", err)
	}

	body := `{"timestamp":"2025-07-01T11:00:00Z"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/stools/"+stool.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateStoolHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/stools/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// ============================================================
// Medication handler — update invalid JSON and validation error
// ============================================================

func TestUpdateMedicationHandler_InvalidJSON(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	med, err := store.CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/medications/"+med.ID)
	req.Body = io.NopCloser(bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateMedicationHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/medications/{medId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdateMedicationHandler_ValidationError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	med, err := store.CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	body := `{"dose":"50mg","frequency":"twice_daily"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/medications/"+med.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateMedicationHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/medications/{medId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// ============================================================
// MedLog handler — missing coverage paths
// ============================================================

func TestCreateMedLogHandler_InvalidJSON(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/med-logs")
	req.Body = io.NopCloser(bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateMedLogHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/med-logs", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateMedLogHandler_MedicationNotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"medication_id":"nonexistent-med-id","given_at":"2026-03-17T08:00:00Z","skipped":false}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/med-logs")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateMedLogHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/med-logs", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateMedLogHandler_GivenAtRequiredWhenNotSkipped(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	med, err := store.CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	body := `{"medication_id":"` + med.ID + `","skipped":false}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/med-logs")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateMedLogHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/med-logs", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateMedLogHandler_ValidationError_MutualExclusivity(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	med, err := store.CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	ml, err := store.CreateMedLog(db, baby.ID, med.ID, user.ID, nil, strPtr("2026-03-17T08:00:00Z"), false, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedLog failed: %v", err)
	}

	body := `{"given_at":"2026-03-17T08:00:00Z","skipped":true}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/med-logs/"+ml.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateMedLogHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/med-logs/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateMedLogHandler_InvalidScheduledTimeFormat(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	med, err := store.CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	ml, err := store.CreateMedLog(db, baby.ID, med.ID, user.ID, nil, strPtr("2026-03-17T08:00:00Z"), false, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedLog failed: %v", err)
	}

	body := `{"given_at":"2026-03-17T08:00:00Z","scheduled_time":"bad-time","skipped":false}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/med-logs/"+ml.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateMedLogHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/med-logs/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateMedLogHandler_InvalidGivenAtFormat(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	med, err := store.CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	ml, err := store.CreateMedLog(db, baby.ID, med.ID, user.ID, nil, strPtr("2026-03-17T08:00:00Z"), false, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedLog failed: %v", err)
	}

	body := `{"given_at":"bad-time","skipped":false}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/med-logs/"+ml.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateMedLogHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/med-logs/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// ============================================================
// Dashboard handler — computeNextDoseAt coverage via integration
// ============================================================

func TestDashboardHandler_NextDoseAt_ScheduledMedication(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	tz := "UTC"
	sched := `["00:01","23:59"]`
	store.CreateMedication(db, baby.ID, user.ID, "TestMed", "10mg", "twice_daily", &sched, &tz)

	rec, resp := doDashboardRequest(t, db, user.ID, baby.ID, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	if len(resp.UpcomingMeds) != 1 {
		t.Fatalf("expected 1 upcoming med, got %d", len(resp.UpcomingMeds))
	}

	if resp.UpcomingMeds[0].NextDoseAt == nil {
		t.Error("expected non-nil next_dose_at for scheduled medication with timezone")
	}
}

func TestDashboardHandler_NextDoseAt_NoTimezone(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	sched := `["08:00","20:00"]`
	store.CreateMedication(db, baby.ID, user.ID, "NoTZMed", "10mg", "twice_daily", &sched, nil)

	rec, resp := doDashboardRequest(t, db, user.ID, baby.ID, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	if len(resp.UpcomingMeds) != 1 {
		t.Fatalf("expected 1 upcoming med, got %d", len(resp.UpcomingMeds))
	}

	if resp.UpcomingMeds[0].NextDoseAt != nil {
		t.Errorf("expected nil next_dose_at when no timezone, got %v", *resp.UpcomingMeds[0].NextDoseAt)
	}
}

func TestDashboardHandler_NextDoseAt_NoSchedule(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	tz := "UTC"
	store.CreateMedication(db, baby.ID, user.ID, "AsNeeded", "5ml", "as_needed", nil, &tz)

	rec, resp := doDashboardRequest(t, db, user.ID, baby.ID, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	if len(resp.UpcomingMeds) != 1 {
		t.Fatalf("expected 1 upcoming med, got %d", len(resp.UpcomingMeds))
	}

	if resp.UpcomingMeds[0].NextDoseAt != nil {
		t.Errorf("expected nil next_dose_at for as_needed med, got %v", *resp.UpcomingMeds[0].NextDoseAt)
	}
}

func TestDashboardHandler_NextDoseAt_AllTimesPassedToday(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	tz := "UTC"
	sched := `["00:00","00:01"]`
	store.CreateMedication(db, baby.ID, user.ID, "EarlyMed", "10mg", "twice_daily", &sched, &tz)

	now := time.Now().UTC()
	if now.Hour() == 0 && now.Minute() <= 1 {
		t.Skip("Skipping: it is currently 00:00-00:01 UTC")
	}

	rec, resp := doDashboardRequest(t, db, user.ID, baby.ID, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	if len(resp.UpcomingMeds) != 1 {
		t.Fatalf("expected 1 upcoming med, got %d", len(resp.UpcomingMeds))
	}

	if resp.UpcomingMeds[0].NextDoseAt == nil {
		t.Error("expected non-nil next_dose_at when today's times have passed (should fall back to tomorrow)")
	}
}

func TestDashboardHandler_NextDoseAt_InvalidTimezone(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	invalidTZ := "Invalid/TZ"
	sched := `["08:00"]`
	store.CreateMedication(db, baby.ID, user.ID, "BadTZMed", "10mg", "once_daily", &sched, &invalidTZ)

	rec, resp := doDashboardRequest(t, db, user.ID, baby.ID, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	if len(resp.UpcomingMeds) != 1 {
		t.Fatalf("expected 1 upcoming med, got %d", len(resp.UpcomingMeds))
	}

	if resp.UpcomingMeds[0].NextDoseAt != nil {
		t.Errorf("expected nil next_dose_at for invalid timezone, got %v", *resp.UpcomingMeds[0].NextDoseAt)
	}
}

func TestDashboardHandler_BabyNotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	rec, _ := doDashboardRequest(t, db, user.ID, "nonexistent-baby", "")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// ============================================================
// Medication handler — missing dose/frequency validation
// ============================================================

func TestCreateMedicationHandler_MissingDose(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"name":"Ursodiol","frequency":"twice_daily"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/medications")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateMedicationHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/medications", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateMedicationHandler_MissingFrequency(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"name":"Ursodiol","dose":"50mg"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/medications")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateMedicationHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/medications", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// ============================================================
// Medication response — empty schedule_times returns empty array
// ============================================================

func TestMedicationResponse_EmptyScheduleTimes(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	med, err := store.CreateMedication(db, baby.ID, user.ID, "NoSched", "10mg", "as_needed", nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/medications/"+med.ID)
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.GetMedicationHandler(db)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/medications/{medId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	st, ok := resp["schedule_times"].([]any)
	if !ok {
		t.Fatalf("expected schedule_times to be array, got %T (%v)", resp["schedule_times"], resp["schedule_times"])
	}
	if len(st) != 0 {
		t.Errorf("expected empty schedule_times, got %d items", len(st))
	}
}

// ============================================================
// Not-found tests for med-logs and medications (not in other test files)
// ============================================================

func TestGetMedLogHandler_NotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/med-logs/nonexistent")
	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/med-logs/{entryId}", authMw(http.HandlerFunc(handler.GetMedLogHandler(db))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestUpdateMedLogHandler_NotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"skipped":true}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/med-logs/nonexistent")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/med-logs/{entryId}", authMw(csrfMw(http.HandlerFunc(handler.UpdateMedLogHandler(db)))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestDeleteMedLogHandler_NotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodDelete, "/api/babies/"+baby.ID+"/med-logs/nonexistent")
	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("DELETE /api/babies/{id}/med-logs/{entryId}", authMw(csrfMw(http.HandlerFunc(handler.DeleteMedLogHandler(db)))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

// ============================================================
// MedLog — create skipped dose, baby_id mismatch
// ============================================================

func TestCreateMedLogHandler_SkippedDose(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	med, err := store.CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication: %v", err)
	}

	body := `{"medication_id":"` + med.ID + `","skipped":true,"skip_reason":"vomited"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/med-logs")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/med-logs", authMw(csrfMw(http.HandlerFunc(handler.CreateMedLogHandler(db)))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["skipped"] != true {
		t.Errorf("expected skipped=true, got %v", resp["skipped"])
	}
}

func TestCreateMedLogHandler_BabyIDMismatch(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby1 := testutil.CreateTestBaby(t, db, user.ID)
	baby2 := testutil.CreateTestBaby(t, db, user.ID)

	med, err := store.CreateMedication(db, baby1.ID, user.ID, "VitD", "400IU", "once_daily", nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication: %v", err)
	}

	// Try to create med-log under baby2 but with med that belongs to baby1
	body := `{"medication_id":"` + med.ID + `","given_at":"2025-07-01T10:30:00Z"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby2.ID+"/med-logs")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/med-logs", authMw(csrfMw(http.HandlerFunc(handler.CreateMedLogHandler(db)))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// ============================================================
// MedLog — create with scheduled_time
// ============================================================

func TestCreateMedLogHandler_WithScheduledTime(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	med, err := store.CreateMedication(db, baby.ID, user.ID, "UDCA", "50mg", "twice_daily", nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication: %v", err)
	}

	body := `{"medication_id":"` + med.ID + `","given_at":"2025-07-01T08:05:00Z","scheduled_time":"2025-07-01T08:00:00Z"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/med-logs")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/med-logs", authMw(csrfMw(http.HandlerFunc(handler.CreateMedLogHandler(db)))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["scheduled_time"] == nil {
		t.Error("expected scheduled_time in response")
	}
}

// ============================================================
// List handlers with data (exercises the full success path)
// ============================================================

func TestListFeedingsHandler_WithData(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	store.CreateFeeding(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "formula", nil, nil, nil, nil, baby.DefaultCalPerFeed)
	store.CreateFeeding(db, baby.ID, user.ID, "2025-07-01T14:30:00Z", "breast_milk", nil, nil, nil, nil, baby.DefaultCalPerFeed)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/feedings?from=2025-07-01&to=2025-07-01")
	req.Header.Set("X-Timezone", "America/New_York")
	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/feedings", authMw(http.HandlerFunc(handler.ListFeedingsHandler(db))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp map[string]any
	json.Unmarshal(rec.Body.Bytes(), &resp)
	data := resp["data"].([]any)
	if len(data) != 2 {
		t.Errorf("expected 2 feedings, got %d", len(data))
	}
}

func TestListMedLogsHandler_WithFilter(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	med, _ := store.CreateMedication(db, baby.ID, user.ID, "VitD", "400IU", "once_daily", nil, nil)
	givenAt := "2025-07-01T08:00:00Z"
	store.CreateMedLog(db, baby.ID, med.ID, user.ID, nil, &givenAt, false, nil, nil)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/med-logs?medication_id="+med.ID)
	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/med-logs", authMw(http.HandlerFunc(handler.ListMedLogsHandler(db))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

// ============================================================
// Dashboard — with explicit from/to and seeded data
// ============================================================

func TestDashboardHandler_WithDateRange(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	// Seed data
	store.CreateFeeding(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "formula", nil, nil, nil, nil, baby.DefaultCalPerFeed)
	store.CreateStool(db, baby.ID, user.ID, "2025-07-01T11:00:00Z", 5, nil, nil, nil, nil)
	store.CreateUrine(db, baby.ID, user.ID, "2025-07-01T11:30:00Z", nil, nil)
	store.CreateTemperature(db, baby.ID, user.ID, "2025-07-01T12:00:00Z", 37.0, "rectal", nil)
	store.CreateWeight(db, baby.ID, user.ID, "2025-07-01T08:00:00Z", 4.5, nil, nil)
	girthVal := 35.5
	store.CreateAbdomen(db, baby.ID, user.ID, "2025-07-01T09:00:00Z", "soft", false, &girthVal, nil)
	store.CreateLabResult(db, baby.ID, user.ID, "2025-07-01T10:00:00Z", "total_bilirubin", "1.5", nil, nil, nil)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/dashboard?from=2025-07-01&to=2025-07-01")
	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/dashboard", authMw(http.HandlerFunc(handler.DashboardHandler(db))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(rec.Body.Bytes(), &resp)
	sc := resp["summary_cards"].(map[string]any)
	if sc["total_feeds"].(float64) != 1 {
		t.Errorf("expected 1 feed, got %v", sc["total_feeds"])
	}
	if sc["total_wet_diapers"].(float64) != 1 {
		t.Errorf("expected 1 wet diaper, got %v", sc["total_wet_diapers"])
	}
	if sc["total_stools"].(float64) != 1 {
		t.Errorf("expected 1 stool, got %v", sc["total_stools"])
	}

	cds := resp["chart_data_series"].(map[string]any)
	if len(cds["feeding_daily"].([]any)) == 0 {
		t.Error("expected feeding_daily data")
	}
	if len(cds["temperature"].([]any)) == 0 {
		t.Error("expected temperature data")
	}
	if len(cds["weight"].([]any)) == 0 {
		t.Error("expected weight data")
	}
	if len(cds["abdomen_girth"].([]any)) == 0 {
		t.Error("expected abdomen_girth data")
	}
}

// ============================================================
// List handlers for photo-capable metrics with object store
// ============================================================

func TestListStoolsHandler_WithObjStore(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	store.CreateStool(db, baby.ID, user.ID, "2025-07-01T11:00:00Z", 5, nil, nil, nil, nil)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/stools")
	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/stools", authMw(http.HandlerFunc(handler.ListStoolsHandler(db, objStore))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestGetStoolHandler_WithObjStore(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	s, _ := store.CreateStool(db, baby.ID, user.ID, "2025-07-01T11:00:00Z", 5, nil, nil, nil, nil)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/stools/"+s.ID)
	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/stools/{entryId}", authMw(http.HandlerFunc(handler.GetStoolHandler(db, objStore))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestListAbdomenHandler_WithObjStore(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	store.CreateAbdomen(db, baby.ID, user.ID, "2025-07-01T09:00:00Z", "soft", false, nil, nil)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/abdomen")
	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/abdomen", authMw(http.HandlerFunc(handler.ListAbdomenHandler(db, objStore))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestListSkinObservationsHandler_WithObjStore(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	store.CreateSkinObservation(db, baby.ID, user.ID, "2025-07-01T09:00:00Z", nil, false, nil, nil, nil)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/skin")
	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/skin", authMw(http.HandlerFunc(handler.ListSkinObservationsHandler(db, objStore))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestListBruisingHandler_WithObjStore(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	store.CreateBruising(db, baby.ID, user.ID, "2025-07-01T09:00:00Z", "left arm", "small_<1cm", nil, nil, nil)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/bruising")
	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/bruising", authMw(http.HandlerFunc(handler.ListBruisingHandler(db, objStore))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestListGeneralNotesHandler_WithObjStore(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	store.CreateGeneralNote(db, baby.ID, user.ID, "2025-07-01T09:00:00Z", "test note", nil, nil)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/notes")
	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/notes", authMw(http.HandlerFunc(handler.ListGeneralNotesHandler(db, objStore))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

// ============================================================
// Unauthorized baby access (403/404 on wrong baby)
// ============================================================

func TestListFeedingsHandler_UnauthorizedBaby(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user1 := testutil.CreateTestUser(t, db)
	user2 := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user1.ID)

	req := testutil.AuthenticatedRequest(t, db, user2.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/feedings")
	authMw := middleware.Auth(db, testCookieName)
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/feedings", authMw(http.HandlerFunc(handler.ListFeedingsHandler(db))))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden && rec.Code != http.StatusNotFound {
		t.Fatalf("expected 403 or 404, got %d", rec.Code)
	}
}

// Suppress unused import warnings
var _ = time.Now
var _ = storage.NewMemoryStore
