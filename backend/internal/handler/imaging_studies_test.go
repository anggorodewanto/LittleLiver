package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ablankz/LittleLiver/backend/internal/handler"
	"github.com/ablankz/LittleLiver/backend/internal/middleware"
	"github.com/ablankz/LittleLiver/backend/internal/storage"
	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

func setupImagingPhotos(t *testing.T, fix *testutil.TestFixture, objStore *storage.MemoryStore, count int) []string {
	t.Helper()
	keys := make([]string, 0, count)
	for i := 0; i < count; i++ {
		r2 := "photos/img-" + string(rune('a'+i)) + ".jpg"
		thumb := "photos/thumb_img-" + string(rune('a'+i)) + ".jpg"
		_ = objStore.Put(context.Background(), r2, strings.NewReader("data"), "image/jpeg")
		_ = objStore.Put(context.Background(), thumb, strings.NewReader("thumb"), "image/jpeg")
		_, err := store.CreatePhotoUpload(fix.DB, fix.Baby.ID, r2, thumb)
		if err != nil {
			t.Fatalf("CreatePhotoUpload: %v", err)
		}
		keys = append(keys, r2)
	}
	return keys
}

func newImagingMux(db interface{ Close() error }, h http.Handler, method, path string) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle(method+" "+path, h)
	return mux
}

func TestCreateImagingStudyHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()
	fix := &testutil.TestFixture{DB: db, User: user, Baby: baby}
	keys := setupImagingPhotos(t, fix, objStore, 2)

	body, _ := json.Marshal(map[string]any{
		"study_date": "2026-04-15",
		"study_type": "CT",
		"notes":      "no acute findings",
		"photo_keys": keys,
	})
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/imaging-studies")
	req.Body = io.NopCloser(bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateImagingStudyHandler(db, objStore))))
	mux := newImagingMux(db, h, http.MethodPost, "/api/babies/{id}/imaging-studies")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["study_date"] != "2026-04-15" {
		t.Errorf("study_date: %v", resp["study_date"])
	}
	if resp["study_type"] != "CT" {
		t.Errorf("study_type: %v", resp["study_type"])
	}
	photos, _ := resp["photos"].([]any)
	if len(photos) != 2 {
		t.Errorf("expected 2 photos, got %d", len(photos))
	}

	// Verify photos got linked
	p, err := store.GetPhotoUploadByR2Key(db, keys[0])
	if err != nil {
		t.Fatalf("GetPhotoUploadByR2Key: %v", err)
	}
	if p.LinkedAt == nil {
		t.Error("expected linked_at to be set")
	}
}

func TestCreateImagingStudyHandler_MissingStudyDate(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"study_type":"CT","photo_keys":["photos/x.jpg"]}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/imaging-studies")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateImagingStudyHandler(db))))
	mux := newImagingMux(db, h, http.MethodPost, "/api/babies/{id}/imaging-studies")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateImagingStudyHandler_InvalidStudyDate(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"study_date":"04/15/2026","study_type":"CT","photo_keys":["x"]}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/imaging-studies")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateImagingStudyHandler(db))))
	mux := newImagingMux(db, h, http.MethodPost, "/api/babies/{id}/imaging-studies")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateImagingStudyHandler_MissingStudyType(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"study_date":"2026-04-15","photo_keys":["x"]}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/imaging-studies")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateImagingStudyHandler(db))))
	mux := newImagingMux(db, h, http.MethodPost, "/api/babies/{id}/imaging-studies")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateImagingStudyHandler_EmptyPhotoKeys(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"study_date":"2026-04-15","study_type":"CT","photo_keys":[]}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/imaging-studies")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateImagingStudyHandler(db))))
	mux := newImagingMux(db, h, http.MethodPost, "/api/babies/{id}/imaging-studies")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateImagingStudyHandler_TooManyPhotoKeys(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	keys := make([]string, 11)
	for i := range keys {
		keys[i] = "photos/k" + string(rune('a'+i)) + ".jpg"
	}
	body, _ := json.Marshal(map[string]any{
		"study_date": "2026-04-15",
		"study_type": "CT",
		"photo_keys": keys,
	})
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/imaging-studies")
	req.Body = io.NopCloser(bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateImagingStudyHandler(db))))
	mux := newImagingMux(db, h, http.MethodPost, "/api/babies/{id}/imaging-studies")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateImagingStudyHandler_TimezoneNoonTimestamp(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()
	fix := &testutil.TestFixture{DB: db, User: user, Baby: baby}
	keys := setupImagingPhotos(t, fix, objStore, 1)

	body, _ := json.Marshal(map[string]any{
		"study_date": "2026-04-15",
		"study_type": "MRI",
		"photo_keys": keys,
	})
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/imaging-studies")
	req.Body = io.NopCloser(bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Timezone", "America/New_York")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateImagingStudyHandler(db, objStore))))
	mux := newImagingMux(db, h, http.MethodPost, "/api/babies/{id}/imaging-studies")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	// 2026-04-15 12:00 in NY (EDT, UTC-4) = 2026-04-15 16:00 UTC
	if resp["timestamp"] != "2026-04-15T16:00:00Z" {
		t.Errorf("timestamp expected 2026-04-15T16:00:00Z, got %v", resp["timestamp"])
	}
}

func TestGetImagingStudyHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()
	fix := &testutil.TestFixture{DB: db, User: user, Baby: baby}
	keys := setupImagingPhotos(t, fix, objStore, 1)

	study, err := store.CreateImagingStudy(db, baby.ID, user.ID, "2026-04-15T12:00:00Z", "2026-04-15", "CT", nil, `["`+keys[0]+`"]`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/imaging-studies/"+study.ID)
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.GetImagingStudyHandler(db, objStore)))
	mux := newImagingMux(db, h, http.MethodGet, "/api/babies/{id}/imaging-studies/{entryId}")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["id"] != study.ID {
		t.Errorf("id mismatch")
	}
}

func TestGetImagingStudyHandler_NotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/imaging-studies/01XXXXXXXXXXXXXXXXXXXXXXXX")
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.GetImagingStudyHandler(db)))
	mux := newImagingMux(db, h, http.MethodGet, "/api/babies/{id}/imaging-studies/{entryId}")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestListImagingStudiesHandler_Empty(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/imaging-studies")
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.ListImagingStudiesHandler(db)))
	mux := newImagingMux(db, h, http.MethodGet, "/api/babies/{id}/imaging-studies")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	data, _ := resp["data"].([]any)
	if len(data) != 0 {
		t.Errorf("expected empty data, got %d items", len(data))
	}
}

func TestUpdateImagingStudyHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()
	fix := &testutil.TestFixture{DB: db, User: user, Baby: baby}
	keys := setupImagingPhotos(t, fix, objStore, 2)

	// Seed: study with first key linked
	if err := store.ValidateAndLinkPhotosWithMax(db, baby.ID, []string{keys[0]}, 10); err != nil {
		t.Fatalf("seed link: %v", err)
	}
	study, err := store.CreateImagingStudy(db, baby.ID, user.ID, "2026-04-15T12:00:00Z", "2026-04-15", "CT", nil, `["`+keys[0]+`"]`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Update: swap to keys[1]
	body, _ := json.Marshal(map[string]any{
		"study_date": "2026-04-16",
		"study_type": "Ultrasound",
		"notes":      "swapped",
		"photo_keys": []string{keys[1]},
	})
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/imaging-studies/"+study.ID)
	req.Body = io.NopCloser(bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateImagingStudyHandler(db, objStore))))
	mux := newImagingMux(db, h, http.MethodPut, "/api/babies/{id}/imaging-studies/{entryId}")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// keys[0] should be unlinked
	p0, _ := store.GetPhotoUploadByR2Key(db, keys[0])
	if p0.LinkedAt != nil {
		t.Error("expected keys[0] to be unlinked after photo swap")
	}
	// keys[1] should be linked
	p1, _ := store.GetPhotoUploadByR2Key(db, keys[1])
	if p1.LinkedAt == nil {
		t.Error("expected keys[1] to be linked")
	}
}

func TestDeleteImagingStudyHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()
	fix := &testutil.TestFixture{DB: db, User: user, Baby: baby}
	keys := setupImagingPhotos(t, fix, objStore, 1)
	if err := store.ValidateAndLinkPhotosWithMax(db, baby.ID, keys, 10); err != nil {
		t.Fatalf("seed link: %v", err)
	}
	study, err := store.CreateImagingStudy(db, baby.ID, user.ID, "2026-04-15T12:00:00Z", "2026-04-15", "CT", nil, `["`+keys[0]+`"]`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodDelete, "/api/babies/"+baby.ID+"/imaging-studies/"+study.ID)
	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.DeleteImagingStudyHandler(db))))
	mux := newImagingMux(db, h, http.MethodDelete, "/api/babies/{id}/imaging-studies/{entryId}")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}

	_, err = store.GetImagingStudyByID(db, baby.ID, study.ID)
	if err == nil {
		t.Error("expected study to be gone")
	}

	// photos unlinked
	p, _ := store.GetPhotoUploadByR2Key(db, keys[0])
	if p.LinkedAt != nil {
		t.Error("expected photo unlinked after delete")
	}
}

func TestImagingStudyHandler_BabyAccessForbidden(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	owner := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, owner.ID)

	// Intruder is a different user not linked to baby
	intruder, err := store.UpsertUser(db, "g-intruder", "x@y.z", "Intruder")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}

	body := `{"study_date":"2026-04-15","study_type":"CT","photo_keys":["x"]}`
	req := testutil.AuthenticatedRequest(t, db, intruder.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/imaging-studies")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateImagingStudyHandler(db))))
	mux := newImagingMux(db, h, http.MethodPost, "/api/babies/{id}/imaging-studies")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden && rec.Code != http.StatusNotFound {
		t.Fatalf("expected 403/404, got %d", rec.Code)
	}
}
