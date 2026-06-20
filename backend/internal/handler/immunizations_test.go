package handler_test

import (
	"bytes"
	"database/sql"
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

func immDose(v int) *int { return &v }

// --- POST /api/babies/{id}/immunizations ---

func TestCreateImmunizationHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"vaccine_code":"DTP_HB_HIB","vaccine_name":"DTP-HB-Hib (Pentavalent)","dose_number":1,"administered_date":"2025-03-02","provider":"Clinic A","lot_number":"LOT9","notes":"ok"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/immunizations")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateImmunizationHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/immunizations", h)
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
		t.Error("expected non-empty id")
	}
	if resp["vaccine_code"] != "DTP_HB_HIB" {
		t.Errorf("expected vaccine_code=DTP_HB_HIB, got %v", resp["vaccine_code"])
	}
	if resp["administered_date"] != "2025-03-02" {
		t.Errorf("expected administered_date=2025-03-02, got %v", resp["administered_date"])
	}
	if resp["dose_number"] != float64(1) {
		t.Errorf("expected dose_number=1, got %v", resp["dose_number"])
	}
}

func TestCreateImmunizationHandler_MissingVaccineName(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"administered_date":"2025-03-02"}`
	assertImmCreateStatus(t, db, user.ID, baby.ID, body, http.StatusBadRequest)
}

func TestCreateImmunizationHandler_MissingDate(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"vaccine_name":"BCG"}`
	assertImmCreateStatus(t, db, user.ID, baby.ID, body, http.StatusBadRequest)
}

func TestCreateImmunizationHandler_InvalidDate(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"vaccine_name":"BCG","administered_date":"03/02/2025"}`
	assertImmCreateStatus(t, db, user.ID, baby.ID, body, http.StatusBadRequest)
}

func TestCreateImmunizationHandler_InvalidJSON(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	assertImmCreateStatus(t, db, user.ID, baby.ID, "not json", http.StatusBadRequest)
}

func TestCreateImmunizationHandler_Unauthorized(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	other := testutil.CreateTestUser(t, db)

	body := `{"vaccine_name":"BCG","administered_date":"2025-03-02"}`
	assertImmCreateStatus(t, db, other.ID, baby.ID, body, http.StatusForbidden)
}

// assertImmCreateStatus posts a create body and asserts the response status.
func assertImmCreateStatus(t *testing.T, db *sql.DB, userID, babyID, body string, want int) {
	t.Helper()
	req := testutil.AuthenticatedRequest(t, db, userID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+babyID+"/immunizations")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateImmunizationHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/immunizations", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != want {
		t.Fatalf("expected %d, got %d. Body: %s", want, rec.Code, rec.Body.String())
	}
}

// --- GET list ---

func TestListImmunizationsHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	for i := 0; i < 2; i++ {
		if _, err := store.CreateImmunization(db, baby.ID, user.ID, "BCG", "BCG", nil, "2025-01-02", nil, nil, nil); err != nil {
			t.Fatalf("CreateImmunization failed: %v", err)
		}
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/immunizations")
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.ListImmunizationsHandler(db)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/immunizations", h)
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
	if len(resp.Data) != 2 {
		t.Errorf("expected 2 records, got %d", len(resp.Data))
	}
}

// --- GET single ---

func TestGetImmunizationHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	rec0, _ := store.CreateImmunization(db, baby.ID, user.ID, "BCG", "BCG", immDose(1), "2025-01-02", nil, nil, nil)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/immunizations/"+rec0.ID)
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.GetImmunizationHandler(db)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/immunizations/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["id"] != rec0.ID {
		t.Errorf("expected id=%q, got %v", rec0.ID, resp["id"])
	}
}

func TestGetImmunizationHandler_NotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/immunizations/nope")
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.GetImmunizationHandler(db)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/immunizations/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

// --- PUT ---

func TestUpdateImmunizationHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	rec0, _ := store.CreateImmunization(db, baby.ID, user.ID, "BCG", "BCG", nil, "2025-01-02", nil, nil, nil)

	body := `{"vaccine_code":"BCG","vaccine_name":"BCG","dose_number":1,"administered_date":"2025-01-05","notes":"updated"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/immunizations/"+rec0.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateImmunizationHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/immunizations/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["administered_date"] != "2025-01-05" {
		t.Errorf("expected updated date, got %v", resp["administered_date"])
	}
	if resp["updated_by"] == nil || resp["updated_by"] == "" {
		t.Error("expected updated_by set")
	}
}

func TestUpdateImmunizationHandler_NotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"vaccine_name":"BCG","administered_date":"2025-01-05"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/immunizations/nope")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateImmunizationHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/immunizations/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// --- DELETE ---

func TestDeleteImmunizationHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	rec0, _ := store.CreateImmunization(db, baby.ID, user.ID, "BCG", "BCG", nil, "2025-01-02", nil, nil, nil)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodDelete, "/api/babies/"+baby.ID+"/immunizations/"+rec0.ID)
	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.DeleteImmunizationHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("DELETE /api/babies/{id}/immunizations/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d. Body: %s", rec.Code, rec.Body.String())
	}
	if _, err := store.GetImmunizationByID(db, baby.ID, rec0.ID); err == nil {
		t.Error("expected record deleted")
	}
}

func TestDeleteImmunizationHandler_NotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodDelete, "/api/babies/"+baby.ID+"/immunizations/nope")
	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.DeleteImmunizationHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("DELETE /api/babies/{id}/immunizations/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

// --- GET schedule ---

func TestImmunizationScheduleHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	if _, err := store.CreateImmunization(db, baby.ID, user.ID, "HB0", "Hepatitis B (birth dose)", immDose(1), "2025-01-01", nil, nil, nil); err != nil {
		t.Fatalf("seed immunization failed: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/immunizations/schedule")
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.ImmunizationScheduleHandler(db)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/immunizations/schedule", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Slots []map[string]any `json:"slots"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(resp.Slots) == 0 {
		t.Fatal("expected non-empty slots")
	}
	var foundDone bool
	for _, s := range resp.Slots {
		if s["code"] == "HB0" && s["status"] == "done" {
			foundDone = true
		}
	}
	if !foundDone {
		t.Error("expected HB0 slot to be done")
	}
}

// --- GET reference ---

func TestImmunizationReferenceHandler_Success(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/api/immunizations/reference", nil)
	rec := httptest.NewRecorder()
	handler.ImmunizationReferenceHandler()(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Schedule []map[string]any `json:"schedule"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(resp.Schedule) == 0 {
		t.Error("expected non-empty reference schedule")
	}
}
