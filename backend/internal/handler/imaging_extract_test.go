package handler_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ablankz/LittleLiver/backend/internal/handler"
	"github.com/ablankz/LittleLiver/backend/internal/labextract"
	"github.com/ablankz/LittleLiver/backend/internal/middleware"
	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/storage"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

func seedImagingPhoto(t *testing.T, db *sql.DB, objStore *storage.MemoryStore, babyID, key, thumbKey string) {
	t.Helper()
	if err := objStore.Put(context.Background(), key, strings.NewReader("fake"), "image/jpeg"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if thumbKey != "" {
		if err := objStore.Put(context.Background(), thumbKey, strings.NewReader("fake-thumb"), "image/jpeg"); err != nil {
			t.Fatalf("Put thumb: %v", err)
		}
	}
	var tk any
	if thumbKey == "" {
		tk = nil
	} else {
		tk = thumbKey
	}
	_, err := db.Exec(
		"INSERT INTO photo_uploads (id, baby_id, r2_key, thumbnail_key, uploaded_at) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)",
		model.NewULID(), babyID, key, tk,
	)
	if err != nil {
		t.Fatalf("seed insert: %v", err)
	}
}

func makeImagingExtractMux(t *testing.T, db *sql.DB, objStore *storage.MemoryStore, client labextract.ClaudeClient, rl *handler.ExtractRateLimiter) *http.ServeMux {
	t.Helper()
	svc := labextract.NewService(client)
	h := handler.ImagingExtractHandlerWithRateLimit(db, objStore, svc, rl)

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/imaging-studies/extract", authMw(csrfMw(http.HandlerFunc(h))))
	return mux
}

func makeImagingExtractReq(t *testing.T, db *sql.DB, userID, babyID string, keys []string) *http.Request {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"photo_keys": keys})
	req := testutil.AuthenticatedRequest(t, db, userID, testCookieName, testSecret, http.MethodPost,
		"/api/babies/"+babyID+"/imaging-studies/extract")
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestImagingExtractHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	key := "photos/" + model.NewULID() + ".jpg"
	seedImagingPhoto(t, db, objStore, baby.ID, key, "")

	canned := `{"study_type":"Ultrasound","study_date":"2026-04-15","findings":"liver normal","notes":""}`
	mux := makeImagingExtractMux(t, db, objStore, &mockClaudeClient{response: canned}, nil)

	req := makeImagingExtractReq(t, db, user.ID, baby.ID, []string{key})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	suggested, _ := resp["suggested"].(map[string]any)
	if suggested["study_type"] != "Ultrasound" {
		t.Errorf("study_type: %v", suggested["study_type"])
	}
	if suggested["study_date"] != "2026-04-15" {
		t.Errorf("study_date: %v", suggested["study_date"])
	}
}

func TestImagingExtractHandler_PDFKeyUsesThumbnail(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	pdfKey := "photos/" + model.NewULID() + ".pdf"
	thumbKey := "photos/thumb_" + model.NewULID() + ".jpg"
	seedImagingPhoto(t, db, objStore, baby.ID, pdfKey, thumbKey)

	canned := `{"study_type":"CT","study_date":"2026-04-15","findings":"","notes":""}`
	mux := makeImagingExtractMux(t, db, objStore, &mockClaudeClient{response: canned}, nil)

	req := makeImagingExtractReq(t, db, user.ID, baby.ID, []string{pdfKey})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestImagingExtractHandler_PDFWithoutThumbnailFails(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	pdfKey := "photos/" + model.NewULID() + ".pdf"
	seedImagingPhoto(t, db, objStore, baby.ID, pdfKey, "")

	mux := makeImagingExtractMux(t, db, objStore, &mockClaudeClient{response: "{}"}, nil)

	req := makeImagingExtractReq(t, db, user.ID, baby.ID, []string{pdfKey})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for PDF without thumbnail, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestImagingExtractHandler_EmptyPhotoKeys(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	mux := makeImagingExtractMux(t, db, objStore, &mockClaudeClient{}, nil)

	req := makeImagingExtractReq(t, db, user.ID, baby.ID, []string{})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestImagingExtractHandler_TooManyPhotoKeys(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	keys := make([]string, 11)
	for i := range keys {
		keys[i] = "photos/k" + string(rune('a'+i)) + ".jpg"
	}
	mux := makeImagingExtractMux(t, db, objStore, &mockClaudeClient{}, nil)

	req := makeImagingExtractReq(t, db, user.ID, baby.ID, keys)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestImagingExtractHandler_ClaudeError_Returns502(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	key := "photos/" + model.NewULID() + ".jpg"
	seedImagingPhoto(t, db, objStore, baby.ID, key, "")

	mux := makeImagingExtractMux(t, db, objStore, &mockClaudeClient{err: errors.New("boom")}, nil)

	req := makeImagingExtractReq(t, db, user.ID, baby.ID, []string{key})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", rec.Code)
	}
}

func TestImagingExtractHandler_EmptyJSON_StillReturns200(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	key := "photos/" + model.NewULID() + ".jpg"
	seedImagingPhoto(t, db, objStore, baby.ID, key, "")

	canned := `{}`
	mux := makeImagingExtractMux(t, db, objStore, &mockClaudeClient{response: canned}, nil)

	req := makeImagingExtractReq(t, db, user.ID, baby.ID, []string{key})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 even with empty extraction, got %d", rec.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	suggested, _ := resp["suggested"].(map[string]any)
	if suggested["study_type"] != "" {
		t.Errorf("expected empty study_type, got %v", suggested["study_type"])
	}
}

func TestImagingExtractHandler_RateLimitShared(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	key := "photos/" + model.NewULID() + ".jpg"
	seedImagingPhoto(t, db, objStore, baby.ID, key, "")

	canned := `{"study_type":"CT"}`
	rl := handler.NewExtractRateLimiterWithCap(2)
	mux := makeImagingExtractMux(t, db, objStore, &mockClaudeClient{response: canned}, rl)

	for i := 0; i < 2; i++ {
		req := makeImagingExtractReq(t, db, user.ID, baby.ID, []string{key})
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("req %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	// 3rd should be 429
	req := makeImagingExtractReq(t, db, user.ID, baby.ID, []string{key})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 on 3rd, got %d", rec.Code)
	}
}
