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
	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

func carePlansMux(db *sql.DB) *http.ServeMux {
	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/care-plans", authMw(csrfMw(http.HandlerFunc(handler.CreateCarePlanHandler(db)))))
	mux.Handle("GET /api/babies/{id}/care-plans", authMw(http.HandlerFunc(handler.ListCarePlansHandler(db))))
	mux.Handle("GET /api/babies/{id}/care-plans/{planId}", authMw(http.HandlerFunc(handler.GetCarePlanHandler(db))))
	mux.Handle("PUT /api/babies/{id}/care-plans/{planId}", authMw(csrfMw(http.HandlerFunc(handler.UpdateCarePlanHandler(db)))))
	mux.Handle("DELETE /api/babies/{id}/care-plans/{planId}", authMw(csrfMw(http.HandlerFunc(handler.DeleteCarePlanHandler(db)))))
	return mux
}

func TestCreateCarePlanHandler_201OnValidBody(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{
		"name": "Antibiotic Rotation",
		"timezone": "America/New_York",
		"phases": [
			{"seq": 1, "label": "Cefixime", "start_date": "2026-05-01"},
			{"seq": 2, "label": "Amoxicillin", "start_date": "2026-06-01"}
		]
	}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/care-plans")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	carePlansMux(db).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["name"] != "Antibiotic Rotation" {
		t.Errorf("name=%v", resp["name"])
	}
	if resp["timezone"] != "America/New_York" {
		t.Errorf("timezone=%v", resp["timezone"])
	}
	phases, ok := resp["phases"].([]any)
	if !ok {
		t.Fatalf("phases is not array: %T", resp["phases"])
	}
	if len(phases) != 2 {
		t.Errorf("expected 2 phases, got %d", len(phases))
	}
}

func TestCreateCarePlanHandler_403WhenBabyNotMine(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	owner := testutil.CreateTestUser(t, db)
	otherBaby := testutil.CreateTestBaby(t, db, owner.ID)

	intruder, err := store.UpsertUser(db, "intruder", "intruder@x.com", "Intruder")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}

	body := `{"name":"X","timezone":"UTC","phases":[{"seq":1,"label":"A","start_date":"2026-05-01"}]}`
	req := testutil.AuthenticatedRequest(t, db, intruder.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+otherBaby.ID+"/care-plans")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	carePlansMux(db).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden && rec.Code != http.StatusNotFound {
		t.Fatalf("expected 403/404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateCarePlanHandler_RejectsBadPhaseDates_400(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	plan, err := store.CreateCarePlan(db, baby.ID, user.ID, "Plan", nil, "UTC")
	if err != nil {
		t.Fatalf("CreateCarePlan: %v", err)
	}

	// phase 2 starts before phase 1 — must be rejected.
	body := `{
		"name": "Plan",
		"phases": [
			{"seq": 1, "label": "A", "start_date": "2026-06-01"},
			{"seq": 2, "label": "B", "start_date": "2026-05-01"}
		]
	}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/care-plans/"+plan.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	carePlansMux(db).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteCarePlanHandler_204AndRowGone(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	plan, err := store.CreateCarePlan(db, baby.ID, user.ID, "Plan", nil, "UTC")
	if err != nil {
		t.Fatalf("CreateCarePlan: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodDelete, "/api/babies/"+baby.ID+"/care-plans/"+plan.ID)

	rec := httptest.NewRecorder()
	carePlansMux(db).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	if _, err := store.GetCarePlanByID(db, baby.ID, plan.ID); err == nil {
		t.Error("expected plan to be deleted from DB")
	}
}

func TestDashboardHandler_IncludesCurrentCarePlanPhases(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	plan, err := store.CreateCarePlan(db, baby.ID, user.ID, "Antibiotic Rotation", nil, "UTC")
	if err != nil {
		t.Fatalf("CreateCarePlan: %v", err)
	}
	if _, err := store.ReplaceCarePlanPhases(db, plan.ID, []model.CarePlanPhase{
		// Phase 1 starts well in the past so it's the current one regardless
		// of when the test runs.
		{Seq: 1, Label: "Cefixime", StartDate: "2000-01-01"},
		{Seq: 2, Label: "Amoxicillin", StartDate: "2099-01-01"},
	}); err != nil {
		t.Fatalf("ReplaceCarePlanPhases: %v", err)
	}

	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.DashboardHandler(db)))
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/dashboard", h)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/dashboard")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	field, ok := raw["current_care_plan_phases"]
	if !ok {
		t.Fatalf("dashboard response missing current_care_plan_phases. body=%s", rec.Body.String())
	}
	if string(field) == "null" {
		t.Error("expected current_care_plan_phases to be array, got null")
	}

	var phases []map[string]any
	if err := json.Unmarshal(field, &phases); err != nil {
		t.Fatalf("unmarshal phases array: %v", err)
	}
	if len(phases) != 1 {
		t.Fatalf("expected 1 active phase, got %d", len(phases))
	}
	if phases[0]["plan_name"] != "Antibiotic Rotation" {
		t.Errorf("plan_name=%v", phases[0]["plan_name"])
	}
	if phases[0]["label"] != "Cefixime" {
		t.Errorf("label=%v", phases[0]["label"])
	}
	if phases[0]["ends_on"] != "2099-01-01" {
		t.Errorf("ends_on=%v (expected next phase start_date)", phases[0]["ends_on"])
	}
}

func TestGetCarePlanHandler_IncludesPhases(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{
		"name": "P",
		"timezone": "UTC",
		"phases": [{"seq":1,"label":"A","start_date":"2026-05-01"}]
	}`
	createReq := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/care-plans")
	createReq.Body = io.NopCloser(bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	carePlansMux(db).ServeHTTP(rec, createReq)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal create: %v", err)
	}
	planID := created["id"].(string)

	getReq := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/care-plans/"+planID)
	getRec := httptest.NewRecorder()
	carePlansMux(db).ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d. Body: %s", getRec.Code, getRec.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(getRec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal get: %v", err)
	}
	phases := got["phases"].([]any)
	if len(phases) != 1 {
		t.Errorf("expected 1 phase, got %d", len(phases))
	}
}
