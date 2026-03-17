package handler_test

import (
	"bytes"
	"context"
	"database/sql"
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

// setupPhotoFixture creates a baby with uploaded photos and returns the R2 keys.
func setupPhotoFixture(t *testing.T, db *testutil.TestFixture, objStore *storage.MemoryStore, count int) []string {
	t.Helper()
	var keys []string
	for i := 0; i < count; i++ {
		r2Key := "photos/test" + string(rune('a'+i)) + ".jpg"
		thumbKey := "photos/thumb_test" + string(rune('a'+i)) + ".jpg"
		// Put objects into the memory store so SignedURL works
		_ = objStore.Put(context.Background(), r2Key, strings.NewReader("fake-image"), "image/jpeg")
		_ = objStore.Put(context.Background(), thumbKey, strings.NewReader("fake-thumb"), "image/jpeg")
		_, err := store.CreatePhotoUpload(db.DB, db.Baby.ID, r2Key, thumbKey)
		if err != nil {
			t.Fatalf("CreatePhotoUpload: %v", err)
		}
		keys = append(keys, r2Key)
	}
	return keys
}

// --- Test: Creating a stool with valid photo_keys sets linked_at ---

func TestCreateStoolHandler_WithPhotoKeys_LinksPhotos(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()
	fix := &testutil.TestFixture{DB: db, User: user, Baby: baby}

	keys := setupPhotoFixture(t, fix, objStore, 2)

	body, _ := json.Marshal(map[string]any{
		"timestamp":    "2025-07-01T10:30:00Z",
		"color_rating": 5,
		"photo_keys":   keys,
	})
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/stools")
	req.Body = io.NopCloser(bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateStoolHandler(db, objStore))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/stools", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// Verify linked_at is set
	photo, err := store.GetPhotoUploadByR2Key(db, keys[0])
	if err != nil {
		t.Fatalf("GetPhotoUploadByR2Key: %v", err)
	}
	if photo.LinkedAt == nil {
		t.Error("expected linked_at to be set after creating stool with photo_keys")
	}

	// Verify response contains photos with url and thumbnail_url
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	photos, ok := resp["photos"].([]any)
	if !ok {
		t.Fatalf("expected photos array in response, got %T", resp["photos"])
	}
	if len(photos) != 2 {
		t.Fatalf("expected 2 photos, got %d", len(photos))
	}
	firstPhoto := photos[0].(map[string]any)
	if _, ok := firstPhoto["url"]; !ok {
		t.Error("expected url in photo response")
	}
	if _, ok := firstPhoto["thumbnail_url"]; !ok {
		t.Error("expected thumbnail_url in photo response")
	}
}

// --- Test: Invalid/wrong-baby photo_keys are rejected ---

func TestCreateStoolHandler_WrongBabyPhotoKeys_Rejected(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby1 := testutil.CreateTestBaby(t, db, user.ID)
	baby2 := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	// Upload photo for baby1
	r2Key := "photos/wrong_baby.jpg"
	thumbKey := "photos/thumb_wrong_baby.jpg"
	_ = objStore.Put(context.Background(), r2Key, strings.NewReader("fake"), "image/jpeg")
	_ = objStore.Put(context.Background(), thumbKey, strings.NewReader("fake"), "image/jpeg")
	_, err := store.CreatePhotoUpload(db, baby1.ID, r2Key, thumbKey)
	if err != nil {
		t.Fatalf("CreatePhotoUpload: %v", err)
	}

	// Try to create stool for baby2 with baby1's photo
	body, _ := json.Marshal(map[string]any{
		"timestamp":    "2025-07-01T10:30:00Z",
		"color_rating": 5,
		"photo_keys":   []string{r2Key},
	})
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby2.ID+"/stools")
	req.Body = io.NopCloser(bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateStoolHandler(db, objStore))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/stools", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// --- Test: Exceeding 4 photos is rejected ---

func TestCreateStoolHandler_ExceedsPhotoLimit_Rejected(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()
	fix := &testutil.TestFixture{DB: db, User: user, Baby: baby}

	keys := setupPhotoFixture(t, fix, objStore, 5)

	body, _ := json.Marshal(map[string]any{
		"timestamp":    "2025-07-01T10:30:00Z",
		"color_rating": 5,
		"photo_keys":   keys,
	})
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/stools")
	req.Body = io.NopCloser(bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateStoolHandler(db, objStore))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/stools", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// --- Test: Removing a photo on update nulls linked_at ---

func TestUpdateStoolHandler_RemovePhoto_UnlinksPhoto(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()
	fix := &testutil.TestFixture{DB: db, User: user, Baby: baby}

	keys := setupPhotoFixture(t, fix, objStore, 2)

	// Create stool with 2 photos
	stool, err := store.CreateStool(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 5, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateStool: %v", err)
	}

	// Link photos manually (store as JSON array per spec)
	photoKeysJSON, _ := json.Marshal(keys)
	_, err = db.Exec(`UPDATE stools SET photo_keys = ? WHERE id = ?`, string(photoKeysJSON), stool.ID)
	if err != nil {
		t.Fatalf("update photo_keys: %v", err)
	}
	err = store.ValidateAndLinkPhotos(db, baby.ID, keys)
	if err != nil {
		t.Fatalf("ValidateAndLinkPhotos: %v", err)
	}

	// Update stool with only the first photo (remove second)
	body, _ := json.Marshal(map[string]any{
		"timestamp":    "2025-07-01T11:00:00Z",
		"color_rating": 5,
		"photo_keys":   []string{keys[0]},
	})
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/stools/"+stool.ID)
	req.Body = io.NopCloser(bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateStoolHandler(db, objStore))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/stools/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// Verify second photo is unlinked
	photo2, err := store.GetPhotoUploadByR2Key(db, keys[1])
	if err != nil {
		t.Fatalf("GetPhotoUploadByR2Key: %v", err)
	}
	if photo2.LinkedAt != nil {
		t.Error("expected removed photo to have linked_at = NULL")
	}

	// First photo should still be linked
	photo1, err := store.GetPhotoUploadByR2Key(db, keys[0])
	if err != nil {
		t.Fatalf("GetPhotoUploadByR2Key: %v", err)
	}
	if photo1.LinkedAt == nil {
		t.Error("expected kept photo to remain linked")
	}
}

// --- Test: List response contains signed URLs not raw keys ---

func TestListStoolsHandler_ContainsSignedURLs(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()
	fix := &testutil.TestFixture{DB: db, User: user, Baby: baby}

	keys := setupPhotoFixture(t, fix, objStore, 1)

	// Create stool with photo
	stool, err := store.CreateStool(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 5, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateStool: %v", err)
	}
	photoKeysJSON, _ := json.Marshal(keys)
	_, err = db.Exec(`UPDATE stools SET photo_keys = ? WHERE id = ?`, string(photoKeysJSON), stool.ID)
	if err != nil {
		t.Fatalf("update photo_keys: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/stools")
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.ListStoolsHandler(db, objStore)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/stools", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 stool, got %d", len(resp.Data))
	}

	photos, ok := resp.Data[0]["photos"].([]any)
	if !ok {
		t.Fatalf("expected photos array in list response, got %T: %v", resp.Data[0]["photos"], resp.Data[0])
	}
	if len(photos) != 1 {
		t.Fatalf("expected 1 photo, got %d", len(photos))
	}

	photo := photos[0].(map[string]any)
	urlStr, _ := photo["url"].(string)
	thumbStr, _ := photo["thumbnail_url"].(string)

	if !strings.Contains(urlStr, "signed=true") {
		t.Errorf("expected signed URL, got %q", urlStr)
	}
	if !strings.Contains(thumbStr, "signed=true") {
		t.Errorf("expected signed thumbnail URL, got %q", thumbStr)
	}

	// Ensure raw photo_keys is NOT in response
	if _, exists := resp.Data[0]["photo_keys"]; exists {
		t.Error("expected photo_keys to be absent from response")
	}
}

// --- Test: Detail response also contains signed URLs ---

func TestGetStoolHandler_ContainsSignedURLs(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()
	fix := &testutil.TestFixture{DB: db, User: user, Baby: baby}

	keys := setupPhotoFixture(t, fix, objStore, 1)

	stool, err := store.CreateStool(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 5, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateStool: %v", err)
	}
	detailKeysJSON, _ := json.Marshal(keys)
	_, err = db.Exec(`UPDATE stools SET photo_keys = ? WHERE id = ?`, string(detailKeysJSON), stool.ID)
	if err != nil {
		t.Fatalf("update photo_keys: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/stools/"+stool.ID)
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.GetStoolHandler(db, objStore)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/stools/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	photos, ok := resp["photos"].([]any)
	if !ok {
		t.Fatalf("expected photos array in detail response, got %T", resp["photos"])
	}
	if len(photos) != 1 {
		t.Fatalf("expected 1 photo, got %d", len(photos))
	}

	photo := photos[0].(map[string]any)
	if _, ok := photo["url"]; !ok {
		t.Error("expected url in photo response")
	}
	if _, ok := photo["thumbnail_url"]; !ok {
		t.Error("expected thumbnail_url in photo response")
	}
}

// --- Test: Duplicate photo_keys are deduplicated ---

func TestCreateStoolHandler_DuplicatePhotoKeys_Deduped(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()
	fix := &testutil.TestFixture{DB: db, User: user, Baby: baby}

	keys := setupPhotoFixture(t, fix, objStore, 1)

	// Send same key twice — should deduplicate and succeed
	body, _ := json.Marshal(map[string]any{
		"timestamp":    "2025-07-01T10:30:00Z",
		"color_rating": 5,
		"photo_keys":   []string{keys[0], keys[0]},
	})
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/stools")
	req.Body = io.NopCloser(bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateStoolHandler(db, objStore))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/stools", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// Verify response contains only 1 photo (deduplicated)
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	photos, ok := resp["photos"].([]any)
	if !ok {
		t.Fatalf("expected photos array, got %T", resp["photos"])
	}
	if len(photos) != 1 {
		t.Errorf("expected 1 photo (deduplicated), got %d", len(photos))
	}
}

// --- Test: Already-linked photo is rejected ---

func TestCreateStoolHandler_AlreadyLinkedPhoto_Rejected(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()
	fix := &testutil.TestFixture{DB: db, User: user, Baby: baby}

	keys := setupPhotoFixture(t, fix, objStore, 2)

	// Create first stool with keys[0]
	body1, _ := json.Marshal(map[string]any{
		"timestamp":    "2025-07-01T10:30:00Z",
		"color_rating": 5,
		"photo_keys":   []string{keys[0]},
	})
	req1 := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/stools")
	req1.Body = io.NopCloser(bytes.NewBuffer(body1))
	req1.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateStoolHandler(db, objStore))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/stools", h)
	rec1 := httptest.NewRecorder()
	mux.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusCreated {
		t.Fatalf("first stool: expected 201, got %d. Body: %s", rec1.Code, rec1.Body.String())
	}

	// Try to create second stool with keys[0] (already linked) — should fail
	body2, _ := json.Marshal(map[string]any{
		"timestamp":    "2025-07-01T11:30:00Z",
		"color_rating": 4,
		"photo_keys":   []string{keys[0]},
	})
	req2 := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/stools")
	req2.Body = io.NopCloser(bytes.NewBuffer(body2))
	req2.Header.Set("Content-Type", "application/json")

	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusBadRequest {
		t.Fatalf("second stool with already-linked photo: expected 400, got %d. Body: %s", rec2.Code, rec2.Body.String())
	}

	if !strings.Contains(rec2.Body.String(), "already linked") {
		t.Errorf("expected 'already linked' in error, got: %s", rec2.Body.String())
	}
}

// --- Test: photo_keys stored as JSON array ---

func TestCreateStoolHandler_PhotoKeysStoredAsJSON(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()
	fix := &testutil.TestFixture{DB: db, User: user, Baby: baby}

	keys := setupPhotoFixture(t, fix, objStore, 2)

	body, _ := json.Marshal(map[string]any{
		"timestamp":    "2025-07-01T10:30:00Z",
		"color_rating": 5,
		"photo_keys":   keys,
	})
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/stools")
	req.Body = io.NopCloser(bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateStoolHandler(db, objStore))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/stools", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// Read the raw photo_keys from DB and verify it's valid JSON
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	entryID, _ := resp["id"].(string)

	var rawPhotoKeys sql.NullString
	err := db.QueryRow(`SELECT photo_keys FROM stools WHERE id = ?`, entryID).Scan(&rawPhotoKeys)
	if err != nil {
		t.Fatalf("query photo_keys: %v", err)
	}
	if !rawPhotoKeys.Valid {
		t.Fatal("expected photo_keys to be set")
	}

	// Must be valid JSON array
	var storedKeys []string
	if err := json.Unmarshal([]byte(rawPhotoKeys.String), &storedKeys); err != nil {
		t.Fatalf("photo_keys is not valid JSON array: %v (raw: %s)", err, rawPhotoKeys.String)
	}
	if len(storedKeys) != 2 {
		t.Errorf("expected 2 keys in JSON array, got %d", len(storedKeys))
	}
}

// --- Test: Stool without photos has empty photos array ---

func TestGetStoolHandler_NoPhotos_EmptyArray(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	stool, err := store.CreateStool(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 5, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateStool: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/stools/"+stool.ID)
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.GetStoolHandler(db, objStore)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/stools/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	photos, ok := resp["photos"].([]any)
	if !ok {
		t.Fatalf("expected photos array, got %T", resp["photos"])
	}
	if len(photos) != 0 {
		t.Errorf("expected 0 photos, got %d", len(photos))
	}
}
