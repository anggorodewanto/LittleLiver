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

// containerTestSetup creates a baby with one medication that has structured
// dose info, ready for stock-container handler tests.
type containerTestSetup struct {
	db   *sql.DB
	user *model.User
	baby *model.Baby
	med  *model.Medication
}

func newContainerTestSetup(t *testing.T) *containerTestSetup {
	t.Helper()
	db := testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	med, err := store.CreateMedication(db, baby.ID, user.ID, "UDCA", "5mL", "twice_daily", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication: %v", err)
	}
	doseAmount := 5.0
	doseUnit := "ml"
	med, err = store.SetMedicationStockFields(db, baby.ID, med.ID, user.ID, store.MedicationStockFields{
		DoseAmount: &doseAmount,
		DoseUnit:   &doseUnit,
	})
	if err != nil {
		t.Fatalf("SetMedicationStockFields: %v", err)
	}
	return &containerTestSetup{db: db, user: user, baby: baby, med: med}
}

// containerMux registers all six container routes.
func containerMux(db *sql.DB) http.Handler {
	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/medications/{medId}/containers", authMw(csrfMw(http.HandlerFunc(handler.CreateMedicationContainerHandler(db)))))
	mux.Handle("GET /api/babies/{id}/medications/{medId}/containers", authMw(http.HandlerFunc(handler.ListMedicationContainersHandler(db))))
	mux.Handle("GET /api/babies/{id}/medications/{medId}/containers/{containerId}", authMw(http.HandlerFunc(handler.GetMedicationContainerHandler(db))))
	mux.Handle("PUT /api/babies/{id}/medications/{medId}/containers/{containerId}", authMw(csrfMw(http.HandlerFunc(handler.UpdateMedicationContainerHandler(db)))))
	mux.Handle("DELETE /api/babies/{id}/medications/{medId}/containers/{containerId}", authMw(csrfMw(http.HandlerFunc(handler.DeleteMedicationContainerHandler(db)))))
	mux.Handle("POST /api/babies/{id}/medications/{medId}/containers/{containerId}/adjust", authMw(csrfMw(http.HandlerFunc(handler.AdjustMedicationContainerHandler(db)))))
	return mux
}

func TestCreateContainerHandler_Success(t *testing.T) {
	t.Parallel()
	s := newContainerTestSetup(t)
	defer s.db.Close()

	body := `{"kind":"bottle","unit":"ml","quantity_initial":100}`
	req := testutil.AuthenticatedRequest(t, s.db, s.user.ID, testCookieName, testSecret,
		http.MethodPost, "/api/babies/"+s.baby.ID+"/medications/"+s.med.ID+"/containers")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	containerMux(s.db).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["id"] == nil || resp["id"] == "" {
		t.Error("expected non-empty container id")
	}
	if resp["quantity_remaining"].(float64) != 100 {
		t.Errorf("quantity_remaining = %v, want 100", resp["quantity_remaining"])
	}
}

func TestCreateContainerHandler_InvalidUnit(t *testing.T) {
	t.Parallel()
	s := newContainerTestSetup(t)
	defer s.db.Close()

	body := `{"kind":"bottle","unit":"gallons","quantity_initial":100}`
	req := testutil.AuthenticatedRequest(t, s.db, s.user.ID, testCookieName, testSecret,
		http.MethodPost, "/api/babies/"+s.baby.ID+"/medications/"+s.med.ID+"/containers")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	containerMux(s.db).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestListContainersHandler_ReturnsAll(t *testing.T) {
	t.Parallel()
	s := newContainerTestSetup(t)
	defer s.db.Close()

	for i := 0; i < 3; i++ {
		_, err := store.CreateMedicationContainer(s.db, store.CreateContainerParams{
			MedicationID:    s.med.ID,
			BabyID:          s.baby.ID,
			Kind:            "bottle",
			Unit:            "ml",
			QuantityInitial: 100,
			CreatedBy:       s.user.ID,
		})
		if err != nil {
			t.Fatalf("CreateMedicationContainer #%d: %v", i, err)
		}
	}

	req := testutil.AuthenticatedRequest(t, s.db, s.user.ID, testCookieName, testSecret,
		http.MethodGet, "/api/babies/"+s.baby.ID+"/medications/"+s.med.ID+"/containers")
	rec := httptest.NewRecorder()
	containerMux(s.db).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
	var list []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("len(list) = %d, want 3", len(list))
	}
}

func TestUpdateContainerHandler_PreservesUnchangedRemaining(t *testing.T) {
	t.Parallel()
	s := newContainerTestSetup(t)
	defer s.db.Close()
	c, _ := store.CreateMedicationContainer(s.db, store.CreateContainerParams{
		MedicationID:    s.med.ID,
		BabyID:          s.baby.ID,
		Kind:            "bottle",
		Unit:            "ml",
		QuantityInitial: 100,
		CreatedBy:       s.user.ID,
	})

	// PUT updates kind+unit only; quantity_remaining should stay at 100.
	body := `{"kind":"bottle","unit":"ml","quantity_initial":100}`
	req := testutil.AuthenticatedRequest(t, s.db, s.user.ID, testCookieName, testSecret,
		http.MethodPut, "/api/babies/"+s.baby.ID+"/medications/"+s.med.ID+"/containers/"+c.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	containerMux(s.db).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
	got, _ := store.GetMedicationContainerByID(s.db, s.baby.ID, c.ID)
	if got.QuantityRemaining != 100 {
		t.Errorf("QuantityRemaining = %v, want 100", got.QuantityRemaining)
	}
}

func TestDeleteContainerHandler_RemovesRow(t *testing.T) {
	t.Parallel()
	s := newContainerTestSetup(t)
	defer s.db.Close()
	c, _ := store.CreateMedicationContainer(s.db, store.CreateContainerParams{
		MedicationID:    s.med.ID,
		BabyID:          s.baby.ID,
		Kind:            "bottle",
		Unit:            "ml",
		QuantityInitial: 100,
		CreatedBy:       s.user.ID,
	})

	req := testutil.AuthenticatedRequest(t, s.db, s.user.ID, testCookieName, testSecret,
		http.MethodDelete, "/api/babies/"+s.baby.ID+"/medications/"+s.med.ID+"/containers/"+c.ID)

	rec := httptest.NewRecorder()
	containerMux(s.db).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d. Body: %s", rec.Code, rec.Body.String())
	}
	if _, err := store.GetMedicationContainerByID(s.db, s.baby.ID, c.ID); err == nil {
		t.Error("expected error after delete")
	}
}

func TestAdjustContainerHandler_AppliesDelta(t *testing.T) {
	t.Parallel()
	s := newContainerTestSetup(t)
	defer s.db.Close()
	c, _ := store.CreateMedicationContainer(s.db, store.CreateContainerParams{
		MedicationID:    s.med.ID,
		BabyID:          s.baby.ID,
		Kind:            "bottle",
		Unit:            "ml",
		QuantityInitial: 100,
		CreatedBy:       s.user.ID,
	})

	body := `{"delta":-50,"reason":"spilled"}`
	req := testutil.AuthenticatedRequest(t, s.db, s.user.ID, testCookieName, testSecret,
		http.MethodPost, "/api/babies/"+s.baby.ID+"/medications/"+s.med.ID+"/containers/"+c.ID+"/adjust")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	containerMux(s.db).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}
	got, _ := store.GetMedicationContainerByID(s.db, s.baby.ID, c.ID)
	if got.QuantityRemaining != 50 {
		t.Errorf("QuantityRemaining = %v, want 50", got.QuantityRemaining)
	}
}

func TestCreateMedLogHandler_PassesContainerOverride(t *testing.T) {
	t.Parallel()
	s := newContainerTestSetup(t)
	defer s.db.Close()

	c, _ := store.CreateMedicationContainer(s.db, store.CreateContainerParams{
		MedicationID:    s.med.ID,
		BabyID:          s.baby.ID,
		Kind:            "bottle",
		Unit:            "ml",
		QuantityInitial: 100,
		CreatedBy:       s.user.ID,
	})

	body := `{"medication_id":"` + s.med.ID + `","skipped":false,"container_id":"` + c.ID + `"}`
	req := testutil.AuthenticatedRequest(t, s.db, s.user.ID, testCookieName, testSecret,
		http.MethodPost, "/api/babies/"+s.baby.ID+"/med-logs")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(s.db, testCookieName)
	csrfMw := middleware.CSRF(s.db, testCookieName, testSecret)
	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/med-logs", authMw(csrfMw(http.HandlerFunc(handler.CreateMedLogHandler(s.db)))))

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["container_id"] != c.ID {
		t.Errorf("container_id = %v, want %s", resp["container_id"], c.ID)
	}
	if resp["stock_deducted"].(float64) != 5 {
		t.Errorf("stock_deducted = %v, want 5", resp["stock_deducted"])
	}
}
