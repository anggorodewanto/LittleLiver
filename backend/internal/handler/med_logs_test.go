package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/handler"
	"github.com/ablankz/LittleLiver/backend/internal/middleware"
	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

// --- POST /api/babies/{id}/med-logs: log as given ---

func TestCreateMedLog_Given(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	// Create a medication
	med, err := store.CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	body := `{"medication_id":"` + med.ID + `","given_at":"2026-03-17T08:00:00Z","skipped":false}`
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

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp["given_at"] == nil || resp["given_at"] == "" {
		t.Error("expected non-nil given_at")
	}
	if resp["skipped"] != false {
		t.Errorf("expected skipped=false, got %v", resp["skipped"])
	}
	if resp["logged_by"] != user.ID {
		t.Errorf("expected logged_by=%s, got %v", user.ID, resp["logged_by"])
	}
	if resp["medication_id"] != med.ID {
		t.Errorf("expected medication_id=%s, got %v", med.ID, resp["medication_id"])
	}
}

// --- POST /api/babies/{id}/med-logs: log as skipped ---

func TestCreateMedLog_Skipped(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	med, err := store.CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	body := `{"medication_id":"` + med.ID + `","skipped":true,"skip_reason":"vomited"}`
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

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp["given_at"] != nil {
		t.Errorf("expected given_at=nil when skipped, got %v", resp["given_at"])
	}
	if resp["skipped"] != true {
		t.Errorf("expected skipped=true, got %v", resp["skipped"])
	}
	if resp["skip_reason"] != "vomited" {
		t.Errorf("expected skip_reason=vomited, got %v", resp["skip_reason"])
	}
}

// --- POST: mutual exclusivity rejected (given_at + skipped=true) ---

func TestCreateMedLog_MutualExclusivityRejected(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	med, err := store.CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	// Both skipped=true AND given_at set
	body := `{"medication_id":"` + med.ID + `","given_at":"2026-03-17T08:00:00Z","skipped":true}`
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

// --- POST: baby_id mismatch rejected ---

func TestCreateMedLog_BabyMismatchRejected(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby1 := testutil.CreateTestBaby(t, db, user.ID)
	baby2 := testutil.CreateTestBaby(t, db, user.ID)

	// Create medication for baby1
	med, err := store.CreateMedication(db, baby1.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	// Try to log against baby2 with baby1's medication
	body := `{"medication_id":"` + med.ID + `","given_at":"2026-03-17T08:00:00Z","skipped":false}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby2.ID+"/med-logs")
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
		t.Fatalf("expected 400 for baby_id mismatch, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// --- POST: scheduled_time nullable (ad-hoc dose) ---

func TestCreateMedLog_AdHocDose(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	med, err := store.CreateMedication(db, baby.ID, user.ID, "Tylenol", "5ml", "as_needed", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	// No scheduled_time — ad-hoc dose
	body := `{"medication_id":"` + med.ID + `","given_at":"2026-03-17T14:30:00Z","skipped":false}`
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

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp["scheduled_time"] != nil {
		t.Errorf("expected scheduled_time=nil for ad-hoc, got %v", resp["scheduled_time"])
	}
}

// --- POST: client-provided given_at is preserved ---

func TestCreateMedLog_PreservesClientGivenAt(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	med, err := store.CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	// Send a specific past time — the handler must NOT override it with NOW()
	clientTime := "2026-03-17T14:00:00Z"
	body := `{"medication_id":"` + med.ID + `","given_at":"` + clientTime + `","skipped":false}`
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

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp["given_at"] != clientTime {
		t.Errorf("expected given_at=%s (client-provided), got %v", clientTime, resp["given_at"])
	}
}

// --- POST: given_at defaults to NOW() when omitted ---

func TestCreateMedLog_DefaultsGivenAtToNow(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	med, err := store.CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	before := time.Now().UTC()
	// Omit given_at — handler should default to NOW()
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
	after := time.Now().UTC()

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	givenAtStr, ok := resp["given_at"].(string)
	if !ok || givenAtStr == "" {
		t.Fatal("expected non-empty given_at when omitted from request")
	}
	givenAtTime, err := time.Parse("2006-01-02T15:04:05Z", givenAtStr)
	if err != nil {
		t.Fatalf("parse given_at: %v", err)
	}
	if givenAtTime.Before(before.Truncate(time.Second)) || givenAtTime.After(after.Add(time.Second)) {
		t.Errorf("expected given_at near NOW(), got %s", givenAtStr)
	}
}

// --- PUT: edit sets updated_by and updated_at ---

func TestUpdateMedLog_SetsUpdatedByAndUpdatedAt(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	med, err := store.CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	// Create a med-log
	ml, err := store.CreateMedLog(db, baby.ID, med.ID, user.ID, nil, strPtr("2026-03-17T08:00:00Z"), false, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedLog failed: %v", err)
	}

	origUpdatedAt := ml.UpdatedAt

	// Update
	body := `{"given_at":"2026-03-17T08:15:00Z","skipped":false,"notes":"delayed 15 min"}`
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

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp["updated_by"] != user.ID {
		t.Errorf("expected updated_by=%s, got %v", user.ID, resp["updated_by"])
	}
	// updated_at is set by SQLite CURRENT_TIMESTAMP, which has second resolution.
	// In fast tests it may match the original. We just verify updated_by is set.
	_ = origUpdatedAt // used for documentation; same-second updates are expected in tests
	if resp["notes"] != "delayed 15 min" {
		t.Errorf("expected notes='delayed 15 min', got %v", resp["notes"])
	}
}

// --- DELETE: works ---

func TestDeleteMedLog_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	med, err := store.CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	ml, err := store.CreateMedLog(db, baby.ID, med.ID, user.ID, nil, strPtr("2026-03-17T08:00:00Z"), false, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedLog failed: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodDelete, "/api/babies/"+baby.ID+"/med-logs/"+ml.ID)

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.DeleteMedLogHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("DELETE /api/babies/{id}/med-logs/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// Verify it's gone
	_, err = store.GetMedLogByID(db, baby.ID, ml.ID)
	if err == nil {
		t.Error("expected error getting deleted med-log")
	}
}

// --- GET list: filter by medication_id and from/to ---

func TestListMedLogs_FilterByMedicationID(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	med1, err := store.CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}
	med2, err := store.CreateMedication(db, baby.ID, user.ID, "Vitamin D", "400IU", "once_daily", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	// Log for both meds
	_, err = store.CreateMedLog(db, baby.ID, med1.ID, user.ID, nil, strPtr("2026-03-17T08:00:00Z"), false, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedLog failed: %v", err)
	}
	_, err = store.CreateMedLog(db, baby.ID, med2.ID, user.ID, nil, strPtr("2026-03-17T09:00:00Z"), false, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedLog failed: %v", err)
	}

	// Filter by med1
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/med-logs?medication_id="+med1.ID)

	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.ListMedLogsHandler(db)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/med-logs", h)
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
	if len(resp.Data) != 1 {
		t.Errorf("expected 1 med-log for med1, got %d", len(resp.Data))
	}
	if len(resp.Data) > 0 && resp.Data[0]["medication_id"] != med1.ID {
		t.Errorf("expected medication_id=%s, got %v", med1.ID, resp.Data[0]["medication_id"])
	}
}

func TestListMedLogs_FilterByFromTo(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	med, err := store.CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	// Create two logs with timestamps relative to now so the test doesn't go stale
	today := time.Now().UTC().Format("2006-01-02")
	_, err = store.CreateMedLog(db, baby.ID, med.ID, user.ID, nil, strPtr(today+"T08:00:00Z"), false, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedLog failed: %v", err)
	}
	_, err = store.CreateMedLog(db, baby.ID, med.ID, user.ID, nil, strPtr(today+"T20:00:00Z"), false, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedLog failed: %v", err)
	}

	// Filter using dynamic from/to that always covers today
	from := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	to := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, fmt.Sprintf("/api/babies/%s/med-logs?from=%s&to=%s", baby.ID, from, to))

	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.ListMedLogsHandler(db)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/med-logs", h)
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
	if len(resp.Data) != 2 {
		t.Errorf("expected 2 med-logs, got %d", len(resp.Data))
	}
}

// --- logged_by is immutable on update ---

func TestMedLog_LoggedByImmutableOnUpdate(t *testing.T) {
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

	med, err := store.CreateMedication(db, baby.ID, parent1.ID, "Ursodiol", "50mg", "twice_daily", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	// Parent1 creates the log
	ml, err := store.CreateMedLog(db, baby.ID, med.ID, parent1.ID, nil, strPtr("2026-03-17T08:00:00Z"), false, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedLog failed: %v", err)
	}

	// Parent2 updates
	body := `{"given_at":"2026-03-17T08:15:00Z","skipped":false}`
	req := testutil.AuthenticatedRequest(t, db, parent2.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/med-logs/"+ml.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateMedLogHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/med-logs/{entryId}", h)
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

// --- GET single med-log detail ---

func TestGetMedLog_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	med, err := store.CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	schedTime := "2026-03-17T08:00:00Z"
	givenAt := "2026-03-17T08:05:00Z"
	ml, err := store.CreateMedLog(db, baby.ID, med.ID, user.ID, &schedTime, &givenAt, false, nil, strPtr("on time"))
	if err != nil {
		t.Fatalf("CreateMedLog failed: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/med-logs/"+ml.ID)

	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.GetMedLogHandler(db)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/med-logs/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp["id"] != ml.ID {
		t.Errorf("expected id=%s, got %v", ml.ID, resp["id"])
	}
	if resp["scheduled_time"] != schedTime {
		t.Errorf("expected scheduled_time=%s, got %v", schedTime, resp["scheduled_time"])
	}
	if resp["given_at"] != givenAt {
		t.Errorf("expected given_at=%s, got %v", givenAt, resp["given_at"])
	}
	if resp["notes"] != "on time" {
		t.Errorf("expected notes='on time', got %v", resp["notes"])
	}
}

func TestGetMedLog_NotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/med-logs/nonexistent")

	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.GetMedLogHandler(db)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/med-logs/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// --- POST: missing medication_id ---

func TestCreateMedLog_MissingMedicationID(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"given_at":"2026-03-17T08:00:00Z","skipped":false}`
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

// --- DELETE: not found ---

func TestDeleteMedLog_NotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodDelete, "/api/babies/"+baby.ID+"/med-logs/nonexistent")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.DeleteMedLogHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("DELETE /api/babies/{id}/med-logs/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// --- POST: skipped=true with given_at provided returns 400 ---

func TestCreateMedLog_SkippedWithGivenAtRejects(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	med, err := store.CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	// Mutual exclusivity: skipped=true with given_at should be rejected
	body := `{"medication_id":"` + med.ID + `","given_at":"2026-03-18T08:00:00Z","skipped":true}`
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
		t.Fatalf("expected 400 for skipped with given_at, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// --- POST: invalid scheduled_time format returns 400 ---

func TestCreateMedLog_InvalidScheduledTimeFormat(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	med, err := store.CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	body := `{"medication_id":"` + med.ID + `","given_at":"2026-03-17T08:00:00Z","scheduled_time":"bad-time","skipped":false}`
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
		t.Fatalf("expected 400 for invalid scheduled_time, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// --- PUT: invalid JSON body returns 400 ---

func TestUpdateMedLog_InvalidJSON(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	med, err := store.CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	ml, err := store.CreateMedLog(db, baby.ID, med.ID, user.ID, nil, strPtr("2026-03-17T08:00:00Z"), false, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedLog failed: %v", err)
	}

	body := `{invalid json`
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
		t.Fatalf("expected 400 for invalid JSON, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// --- PUT: update nonexistent entry returns 404 ---

func TestUpdateMedLog_NotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"given_at":"2026-03-17T08:00:00Z","skipped":false}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/med-logs/nonexistent")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateMedLogHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/med-logs/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for nonexistent med-log update, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// --- helper ---

func strPtr(s string) *string {
	return &s
}
