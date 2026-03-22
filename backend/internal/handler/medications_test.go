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

// --- POST /api/babies/{id}/medications ---

func TestCreateMedicationHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"name":"Ursodiol","dose":"50mg","frequency":"twice_daily","schedule_times":["08:00","20:00"]}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/medications")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Timezone", "America/New_York")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateMedicationHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/medications", h)
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
		t.Error("expected non-empty medication ID")
	}
	if resp["name"] != "Ursodiol" {
		t.Errorf("expected name=Ursodiol, got %v", resp["name"])
	}
	if resp["dose"] != "50mg" {
		t.Errorf("expected dose=50mg, got %v", resp["dose"])
	}
	if resp["frequency"] != "twice_daily" {
		t.Errorf("expected frequency=twice_daily, got %v", resp["frequency"])
	}
	if resp["logged_by"] != user.ID {
		t.Errorf("expected logged_by=%s, got %v", user.ID, resp["logged_by"])
	}
	if resp["timezone"] != "America/New_York" {
		t.Errorf("expected timezone=America/New_York, got %v", resp["timezone"])
	}
	if resp["active"] != true {
		t.Errorf("expected active=true, got %v", resp["active"])
	}

	// Verify schedule_times is a JSON array
	st, ok := resp["schedule_times"].([]any)
	if !ok {
		t.Fatalf("expected schedule_times to be array, got %T", resp["schedule_times"])
	}
	if len(st) != 2 {
		t.Errorf("expected 2 schedule times, got %d", len(st))
	}
}

func TestCreateMedicationHandler_StoresScheduleAsJSON(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"name":"Vitamin D","dose":"400IU","frequency":"once_daily","schedule_times":["09:00"]}`
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

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	// Verify the raw DB has schedule as JSON
	medID := resp["id"].(string)
	med, err := store.GetMedicationByID(db, baby.ID, medID)
	if err != nil {
		t.Fatalf("GetMedicationByID failed: %v", err)
	}
	if med.Schedule == nil {
		t.Fatal("expected non-nil schedule")
	}
	var times []string
	if err := json.Unmarshal([]byte(*med.Schedule), &times); err != nil {
		t.Fatalf("schedule is not valid JSON array: %v", err)
	}
	if len(times) != 1 || times[0] != "09:00" {
		t.Errorf("expected schedule=[09:00], got %v", times)
	}
}

func TestCreateMedicationHandler_MissingName(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"dose":"50mg","frequency":"twice_daily"}`
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

func TestCreateMedicationHandler_InvalidFrequency(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"name":"Ursodiol","dose":"50mg","frequency":"invalid_freq"}`
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
		t.Fatalf("expected 400 for invalid frequency, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateMedicationHandler_InvalidJSON(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/medications")
	req.Body = io.NopCloser(bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateMedicationHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/medications", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

// --- GET /api/babies/{id}/medications ---

func TestListMedicationsHandler_ReturnsBothActiveAndInactive(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	// Create two medications
	med1, err := store.CreateMedication(db, baby.ID, user.ID, "Active Med", "10mg", "once_daily", nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}
	_, err = store.CreateMedication(db, baby.ID, user.ID, "Another Med", "20mg", "twice_daily", nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	// Deactivate the first
	active := false
	_, err = store.UpdateMedication(db, baby.ID, med1.ID, user.ID, "Active Med", "10mg", "once_daily", nil, nil, &active)
	if err != nil {
		t.Fatalf("UpdateMedication failed: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/medications")
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.ListMedicationsHandler(db)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/medications", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("expected 2 medications, got %d", len(resp))
	}

	// Verify one is active, one is not
	activeCount := 0
	for _, m := range resp {
		if m["active"] == true {
			activeCount++
		}
	}
	if activeCount != 1 {
		t.Errorf("expected 1 active medication, got %d", activeCount)
	}
}

// --- GET /api/babies/{id}/medications/{medId} ---

func TestGetMedicationHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	med, err := store.CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, nil)
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
	if resp["id"] != med.ID {
		t.Errorf("expected id=%q, got %v", med.ID, resp["id"])
	}
	if resp["name"] != "Ursodiol" {
		t.Errorf("expected name=Ursodiol, got %v", resp["name"])
	}
}

func TestGetMedicationHandler_NotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/medications/nonexistent")
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.GetMedicationHandler(db)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/medications/{medId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

// --- PUT /api/babies/{id}/medications/{medId} ---

func TestUpdateMedicationHandler_Deactivate(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	med, err := store.CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	body := `{"name":"Ursodiol","dose":"50mg","frequency":"twice_daily","active":false}`
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

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp["active"] != false {
		t.Errorf("expected active=false, got %v", resp["active"])
	}
}

func TestUpdateMedicationHandler_SetsUpdatedBy(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	med, err := store.CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	body := `{"name":"Ursodiol","dose":"60mg","frequency":"twice_daily"}`
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

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp["updated_by"] == nil || resp["updated_by"] == "" {
		t.Error("expected non-empty updated_by after update")
	}
	if resp["updated_by"] != user.ID {
		t.Errorf("expected updated_by=%s, got %v", user.ID, resp["updated_by"])
	}
}

func TestUpdateMedicationHandler_TimezonePreserved(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	oldTZ := "America/New_York"
	med, err := store.CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, &oldTZ)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	// Update with a different X-Timezone header — timezone should NOT change
	body := `{"name":"Ursodiol","dose":"50mg","frequency":"twice_daily"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/medications/"+med.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Timezone", "America/Los_Angeles")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateMedicationHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/medications/{medId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	// Timezone should be preserved from creation, not overwritten by X-Timezone header
	if resp["timezone"] != "America/New_York" {
		t.Errorf("expected timezone=America/New_York (preserved), got %v", resp["timezone"])
	}
}

func TestUpdateMedicationHandler_NotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"name":"Ursodiol","dose":"50mg","frequency":"twice_daily"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/medications/nonexistent")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateMedicationHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/medications/{medId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// --- DELETE /api/babies/{id}/medications/{medId} returns 405 ---


// --- Unauthorized access ---

func TestMedicationHandler_UnauthorizedBabyAccess(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user1 := testutil.CreateTestUser(t, db)
	user2 := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user1.ID)

	// user2 is NOT linked to baby
	req := testutil.AuthenticatedRequest(t, db, user2.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/medications")
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.ListMedicationsHandler(db)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/medications", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// --- LoggedBy immutability + cross-parent ---

func TestMedication_LoggedByImmutableAfterEditByDifferentParent(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	parent1 := testutil.CreateTestUser(t, db)
	parent2 := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, parent1.ID)

	// Link parent2
	_, err := db.Exec("INSERT INTO baby_parents (baby_id, user_id) VALUES (?, ?)", baby.ID, parent2.ID)
	if err != nil {
		t.Fatalf("link parent2: %v", err)
	}

	// Parent1 creates the medication
	med, err := store.CreateMedication(db, baby.ID, parent1.ID, "Ursodiol", "50mg", "twice_daily", nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	// Parent2 updates
	body := `{"name":"Ursodiol","dose":"60mg","frequency":"twice_daily"}`
	req := testutil.AuthenticatedRequest(t, db, parent2.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/medications/"+med.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateMedicationHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/medications/{medId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if resp["logged_by"] != parent1.ID {
		t.Errorf("logged_by should be immutable: expected %s, got %v", parent1.ID, resp["logged_by"])
	}
	if resp["updated_by"] != parent2.ID {
		t.Errorf("updated_by should be parent2: expected %s, got %v", parent2.ID, resp["updated_by"])
	}
}
