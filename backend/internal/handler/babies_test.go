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
	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

// --- POST /api/babies ---

func TestCreateBabyHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	body := `{"name":"Luna","sex":"female","date_of_birth":"2025-06-15"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateBabyHandler(db))))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		ID                string  `json:"id"`
		Name              string  `json:"name"`
		Sex               string  `json:"sex"`
		DateOfBirth       string  `json:"date_of_birth"`
		DefaultCalPerFeed float64 `json:"default_cal_per_feed"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if resp.ID == "" {
		t.Error("expected non-empty baby ID")
	}
	if resp.Name != "Luna" {
		t.Errorf("expected name=Luna, got %q", resp.Name)
	}
	if resp.Sex != "female" {
		t.Errorf("expected sex=female, got %q", resp.Sex)
	}
	if resp.DateOfBirth != "2025-06-15" {
		t.Errorf("expected dob=2025-06-15, got %q", resp.DateOfBirth)
	}
	if resp.DefaultCalPerFeed != model.DefaultCalPerFeed {
		t.Errorf("expected default_cal_per_feed=%v, got %f", model.DefaultCalPerFeed, resp.DefaultCalPerFeed)
	}

	// Verify the baby is linked to the creator
	linked, err := store.IsParentOfBaby(db, user.ID, resp.ID)
	if err != nil {
		t.Fatalf("IsParentOfBaby failed: %v", err)
	}
	if !linked {
		t.Error("expected creator to be linked to baby")
	}
}

func TestCreateBabyHandler_WithOptionalFields(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	body := `{"name":"Kai","sex":"male","date_of_birth":"2025-06-15","diagnosis_date":"2025-07-01","kasai_date":"2025-07-15","default_cal_per_feed":80,"notes":"Healthy baby"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateBabyHandler(db))))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		DiagnosisDate     *string `json:"diagnosis_date"`
		KasaiDate         *string `json:"kasai_date"`
		DefaultCalPerFeed float64 `json:"default_cal_per_feed"`
		Notes             *string `json:"notes"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if resp.DiagnosisDate == nil || *resp.DiagnosisDate != "2025-07-01" {
		t.Errorf("expected diagnosis_date=2025-07-01, got %v", resp.DiagnosisDate)
	}
	if resp.KasaiDate == nil || *resp.KasaiDate != "2025-07-15" {
		t.Errorf("expected kasai_date=2025-07-15, got %v", resp.KasaiDate)
	}
	if resp.DefaultCalPerFeed != 80 {
		t.Errorf("expected default_cal_per_feed=80, got %f", resp.DefaultCalPerFeed)
	}
	if resp.Notes == nil || *resp.Notes != "Healthy baby" {
		t.Errorf("expected notes='Healthy baby', got %v", resp.Notes)
	}
}

func TestCreateBabyHandler_InvalidJSON(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies")
	req.Body = io.NopCloser(bytes.NewBufferString("{bad json"))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateBabyHandler(db))))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateBabyHandler_MissingRequiredFields(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	// Missing name
	body := `{"sex":"female","date_of_birth":"2025-06-15"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateBabyHandler(db))))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateBabyHandler_Unauthorized(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	// No auth middleware, no user in context
	h := http.HandlerFunc(handler.CreateBabyHandler(db))
	req := httptest.NewRequest(http.MethodPost, "/api/babies", bytes.NewBufferString(`{"name":"Luna","sex":"female","date_of_birth":"2025-06-15"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

// --- GET /api/babies ---

func TestListBabiesHandler_ReturnsBabies(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	testutil.CreateTestBaby(t, db, user.ID)
	testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies")

	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.ListBabiesHandler(db)))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(resp) != 2 {
		t.Errorf("expected 2 babies, got %d", len(resp))
	}
}

func TestListBabiesHandler_EmptyList(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies")

	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.ListBabiesHandler(db)))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp []interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if resp == nil {
		t.Error("expected empty array, got nil")
	}
	if len(resp) != 0 {
		t.Errorf("expected 0 babies, got %d", len(resp))
	}
}

func TestListBabiesHandler_OnlyUsersBabies(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user1 := testutil.CreateTestUser(t, db)
	user2 := testutil.CreateTestUser(t, db)
	testutil.CreateTestBaby(t, db, user1.ID)
	testutil.CreateTestBaby(t, db, user2.ID)

	req := testutil.AuthenticatedRequest(t, db, user1.ID, testCookieName, testSecret, http.MethodGet, "/api/babies")

	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.ListBabiesHandler(db)))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(resp) != 1 {
		t.Errorf("expected 1 baby for user1, got %d", len(resp))
	}
}

func TestListBabiesHandler_Unauthorized(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	h := http.HandlerFunc(handler.ListBabiesHandler(db))
	req := httptest.NewRequest(http.MethodGet, "/api/babies", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

// --- GET /api/babies/:id ---

func TestGetBabyHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID)
	req.SetPathValue("id", baby.ID)

	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.GetBabyHandler(db)))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if resp.ID != baby.ID {
		t.Errorf("expected id=%q, got %q", baby.ID, resp.ID)
	}
}

func TestGetBabyHandler_NotLinked_Returns403(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user1 := testutil.CreateTestUser(t, db)
	user2 := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user1.ID)

	// user2 tries to access user1's baby
	req := testutil.AuthenticatedRequest(t, db, user2.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID)
	req.SetPathValue("id", baby.ID)

	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.GetBabyHandler(db)))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestGetBabyHandler_NotFound_Returns404(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/nonexistent")
	req.SetPathValue("id", "nonexistent")

	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.GetBabyHandler(db)))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestGetBabyHandler_Unauthorized(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	h := http.HandlerFunc(handler.GetBabyHandler(db))
	req := httptest.NewRequest(http.MethodGet, "/api/babies/someid", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

// --- PUT /api/babies/:id ---

func TestUpdateBabyHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"name":"Updated Name","sex":"male","date_of_birth":"2025-07-01","notes":"Updated notes"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID)
	req.SetPathValue("id", baby.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateBabyHandler(db))))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Name  string  `json:"name"`
		Sex   string  `json:"sex"`
		Notes *string `json:"notes"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if resp.Name != "Updated Name" {
		t.Errorf("expected name='Updated Name', got %q", resp.Name)
	}
	if resp.Sex != "male" {
		t.Errorf("expected sex=male, got %q", resp.Sex)
	}
	if resp.Notes == nil || *resp.Notes != "Updated notes" {
		t.Errorf("expected notes='Updated notes', got %v", resp.Notes)
	}
}

func TestUpdateBabyHandler_NotLinked_Returns403(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user1 := testutil.CreateTestUser(t, db)
	user2 := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user1.ID)

	body := `{"name":"Hacked","sex":"male","date_of_birth":"2025-07-01"}`
	req := testutil.AuthenticatedRequest(t, db, user2.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID)
	req.SetPathValue("id", baby.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateBabyHandler(db))))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateBabyHandler_NotFound_Returns404(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	body := `{"name":"Test","sex":"female","date_of_birth":"2025-06-15"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/nonexistent")
	req.SetPathValue("id", "nonexistent")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateBabyHandler(db))))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateBabyHandler_InvalidJSON(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID)
	req.SetPathValue("id", baby.ID)
	req.Body = io.NopCloser(bytes.NewBufferString("{bad"))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateBabyHandler(db))))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdateBabyHandler_Unauthorized(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	h := http.HandlerFunc(handler.UpdateBabyHandler(db))
	req := httptest.NewRequest(http.MethodPut, "/api/babies/someid", bytes.NewBufferString(`{"name":"Test","sex":"female","date_of_birth":"2025-06-15"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestUpdateBabyHandler_MissingRequiredFields(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	// Missing name
	body := `{"sex":"female","date_of_birth":"2025-06-15"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID)
	req.SetPathValue("id", baby.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateBabyHandler(db))))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateBabyHandler_InvalidSex(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	body := `{"name":"Luna","sex":"other","date_of_birth":"2025-06-15"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateBabyHandler(db))))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateBabyHandler_InvalidDateFormat(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	body := `{"name":"Luna","sex":"female","date_of_birth":"15-06-2025"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateBabyHandler(db))))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateBabyHandler_InvalidOptionalDateFormat(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	// Valid date_of_birth but invalid diagnosis_date
	body := `{"name":"Luna","sex":"female","date_of_birth":"2025-06-15","diagnosis_date":"not-a-date"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateBabyHandler(db))))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateBabyHandler_InvalidSex(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"name":"Luna","sex":"other","date_of_birth":"2025-06-15"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID)
	req.SetPathValue("id", baby.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateBabyHandler(db))))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateBabyHandler_InvalidDateFormat(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"name":"Luna","sex":"female","date_of_birth":"not-a-date"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID)
	req.SetPathValue("id", baby.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateBabyHandler(db))))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// --- DELETE /api/babies/:id/parents/me ---

func TestUnlinkSelfHandler_WithOtherParents_Returns204(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user1 := testutil.CreateTestUser(t, db)
	user2 := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user1.ID)

	// Link user2 as second parent
	_, err := db.Exec("INSERT INTO baby_parents (baby_id, user_id) VALUES (?, ?)", baby.ID, user2.ID)
	if err != nil {
		t.Fatalf("link user2 failed: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user1.ID, testCookieName, testSecret, http.MethodDelete, "/api/babies/"+baby.ID+"/parents/me")
	req.SetPathValue("id", baby.ID)

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UnlinkSelfHandler(db))))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// Baby should still exist
	linked, err := store.IsParentOfBaby(db, user2.ID, baby.ID)
	if err != nil {
		t.Fatalf("IsParentOfBaby failed: %v", err)
	}
	if !linked {
		t.Error("expected user2 to still be linked to baby")
	}

	// user1 should not be linked
	linked, err = store.IsParentOfBaby(db, user1.ID, baby.ID)
	if err != nil {
		t.Fatalf("IsParentOfBaby failed: %v", err)
	}
	if linked {
		t.Error("expected user1 to be unlinked from baby")
	}
}

func TestUnlinkSelfHandler_LastParent_Returns204_DeletesBaby(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodDelete, "/api/babies/"+baby.ID+"/parents/me")
	req.SetPathValue("id", baby.ID)

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UnlinkSelfHandler(db))))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// Baby should be deleted
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM babies WHERE id = ?", baby.ID).Scan(&count)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected baby to be deleted, got count=%d", count)
	}
}

func TestUnlinkSelfHandler_NotLinked_Returns403(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user1 := testutil.CreateTestUser(t, db)
	user2 := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user1.ID)

	req := testutil.AuthenticatedRequest(t, db, user2.ID, testCookieName, testSecret, http.MethodDelete, "/api/babies/"+baby.ID+"/parents/me")
	req.SetPathValue("id", baby.ID)

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UnlinkSelfHandler(db))))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUnlinkSelfHandler_NotFound_Returns404(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodDelete, "/api/babies/nonexistent/parents/me")
	req.SetPathValue("id", "nonexistent")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UnlinkSelfHandler(db))))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUnlinkSelfHandler_Unauthorized(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	h := http.HandlerFunc(handler.UnlinkSelfHandler(db))
	req := httptest.NewRequest(http.MethodDelete, "/api/babies/someid/parents/me", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

// --- End-to-end create + get flow ---

func TestBabyCRUD_EndToEnd(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	// 1. Create a baby
	body := `{"name":"Luna","sex":"female","date_of_birth":"2025-06-15","notes":"Born healthy"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	createH := authMw(csrfMw(http.HandlerFunc(handler.CreateBabyHandler(db))))
	rec := httptest.NewRecorder()
	createH.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var created struct {
		ID    string  `json:"id"`
		Name  string  `json:"name"`
		Notes *string `json:"notes"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	// 2. List babies - should include the new baby
	listReq := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies")
	listH := authMw(http.HandlerFunc(handler.ListBabiesHandler(db)))
	rec2 := httptest.NewRecorder()
	listH.ServeHTTP(rec2, listReq)

	if rec2.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", rec2.Code)
	}

	var listed []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec2.Body.Bytes(), &listed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 baby, got %d", len(listed))
	}
	if listed[0].ID != created.ID {
		t.Errorf("expected listed baby ID=%q, got %q", created.ID, listed[0].ID)
	}

	// 3. Get baby by ID
	getReq := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+created.ID)
	getReq.SetPathValue("id", created.ID)
	getH := authMw(http.HandlerFunc(handler.GetBabyHandler(db)))
	rec3 := httptest.NewRecorder()
	getH.ServeHTTP(rec3, getReq)

	if rec3.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d. Body: %s", rec3.Code, rec3.Body.String())
	}

	var fetched struct {
		ID    string  `json:"id"`
		Name  string  `json:"name"`
		Notes *string `json:"notes"`
	}
	if err := json.Unmarshal(rec3.Body.Bytes(), &fetched); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if fetched.Notes == nil || *fetched.Notes != "Born healthy" {
		t.Errorf("expected notes='Born healthy', got %v", fetched.Notes)
	}

	// 4. Update baby
	updateBody := `{"name":"Luna Updated","sex":"female","date_of_birth":"2025-06-15","notes":"Growing well"}`
	updateReq := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+created.ID)
	updateReq.SetPathValue("id", created.ID)
	updateReq.Body = io.NopCloser(bytes.NewBufferString(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateH := authMw(csrfMw(http.HandlerFunc(handler.UpdateBabyHandler(db))))
	rec4 := httptest.NewRecorder()
	updateH.ServeHTTP(rec4, updateReq)

	if rec4.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d. Body: %s", rec4.Code, rec4.Body.String())
	}

	var updated struct {
		Name  string  `json:"name"`
		Notes *string `json:"notes"`
	}
	if err := json.Unmarshal(rec4.Body.Bytes(), &updated); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if updated.Name != "Luna Updated" {
		t.Errorf("expected name='Luna Updated', got %q", updated.Name)
	}
	if updated.Notes == nil || *updated.Notes != "Growing well" {
		t.Errorf("expected notes='Growing well', got %v", updated.Notes)
	}
}
