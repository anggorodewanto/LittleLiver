package integration_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/auth"
	"github.com/ablankz/LittleLiver/backend/internal/cron"
	"github.com/ablankz/LittleLiver/backend/internal/handler"
	"github.com/ablankz/LittleLiver/backend/internal/middleware"
	"github.com/ablankz/LittleLiver/backend/internal/storage"
	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

// setupPhotoServer creates a test server with object store enabled.
func setupPhotoServer(t *testing.T) (*httptest.Server, *sql.DB, *storage.MemoryStore, func()) {
	t.Helper()

	db := testutil.SetupTestDB(t)
	objStore := storage.NewMemoryStore()

	mux := handler.NewMux(
		handler.WithDB(db),
		handler.WithAuthConfig(auth.Config{
			ClientID:      "test-client-id",
			ClientSecret:  "test-client-secret",
			RedirectURL:   "http://localhost/auth/google/callback",
			TokenURL:      "http://localhost/fake-token",
			UserInfoURL:   "http://localhost/fake-userinfo",
			SessionSecret: testSessionSecret,
		}),
		handler.WithObjectStore(objStore),
	)

	srv := httptest.NewServer(mux)
	cleanup := func() {
		srv.Close()
		db.Close()
	}
	return srv, db, objStore, cleanup
}

// photoTestClient extends testClient with multipart upload capability.
type photoTestClient struct {
	t         *testing.T
	srv       *httptest.Server
	userID    string
	sessionID string
	csrfToken string
}

func newPhotoTestClient(t *testing.T, srv *httptest.Server, db *sql.DB) *photoTestClient {
	t.Helper()
	user := testutil.CreateTestUser(t, db)
	sess, err := store.CreateSession(db, user.ID)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	csrfToken := middleware.CSRFToken(sess.Token, testSessionSecret)
	return &photoTestClient{
		t:         t,
		srv:       srv,
		userID:    user.ID,
		sessionID: sess.ID,
		csrfToken: csrfToken,
	}
}

// uploadPhoto sends a multipart file upload request and returns the status + decoded JSON body.
func (pc *photoTestClient) uploadPhoto(babyID, filename string, data []byte) (int, map[string]any) {
	pc.t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		pc.t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(data)); err != nil {
		pc.t.Fatalf("copy file data: %v", err)
	}
	writer.Close()

	req, err := http.NewRequest(http.MethodPost, pc.srv.URL+"/api/babies/"+babyID+"/upload", body)
	if err != nil {
		pc.t.Fatalf("create request: %v", err)
	}
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: pc.sessionID})
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-CSRF-Token", pc.csrfToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		pc.t.Fatalf("do upload: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		pc.t.Fatalf("read response: %v", err)
	}

	var result map[string]any
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &result); err != nil {
			result = map[string]any{"_raw": string(respBody)}
		}
	}
	return resp.StatusCode, result
}

// doJSON performs an HTTP request with auth headers and optional JSON body.
func (pc *photoTestClient) doJSON(method, path string, body any) (int, map[string]any) {
	pc.t.Helper()
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			pc.t.Fatalf("marshal body: %v", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, pc.srv.URL+path, bodyReader)
	if err != nil {
		pc.t.Fatalf("create request: %v", err)
	}
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: pc.sessionID})
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if method != http.MethodGet && method != http.MethodHead {
		req.Header.Set("X-CSRF-Token", pc.csrfToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		pc.t.Fatalf("do request %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		pc.t.Fatalf("read response: %v", err)
	}

	var result map[string]any
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &result); err != nil {
			result = map[string]any{"_raw": string(respBody)}
		}
	}
	return resp.StatusCode, result
}

// createTestJPEGData creates a minimal valid JPEG image.
func createTestJPEGData(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	buf := &bytes.Buffer{}
	if err := jpeg.Encode(buf, img, nil); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	return buf.Bytes()
}

// TestPhotoFlowLifecycle exercises the complete photo lifecycle:
//
//  1. Upload photo -> receive R2 key
//  2. Create stool entry with photo_key -> verify linked_at set
//  3. Read entry -> verify signed URL returned (url and thumbnail_url)
//  4. Update entry removing photo -> verify linked_at nulled
//  5. Wait for cleanup window -> run cleanup -> verify R2 delete called
//  6. Also test: 5MB limit rejection, invalid MIME rejection, 4-photo limit
func TestPhotoFlowLifecycle(t *testing.T) {
	t.Parallel()

	srv, db, objStore, cleanup := setupPhotoServer(t)
	defer cleanup()

	client := newPhotoTestClient(t, srv, db)

	// Create a baby
	status, babyResp := client.doJSON(http.MethodPost, "/api/babies", map[string]any{
		"name":          "Photo Flow Baby",
		"sex":           "female",
		"date_of_birth": "2025-01-01",
	})
	if status != http.StatusCreated {
		t.Fatalf("expected 201 creating baby, got %d: %v", status, babyResp)
	}
	babyID := babyResp["id"].(string)

	// === Step 1: Upload photo -> receive R2 key ===
	jpegData := createTestJPEGData(t, 800, 600)
	status, uploadResp := client.uploadPhoto(babyID, "test.jpg", jpegData)
	if status != http.StatusCreated {
		t.Fatalf("step 1: expected 201 uploading photo, got %d: %v", status, uploadResp)
	}

	r2Key, ok := uploadResp["r2_key"].(string)
	if !ok || r2Key == "" {
		t.Fatalf("step 1: expected non-empty r2_key, got %v", uploadResp["r2_key"])
	}
	thumbnailKey, ok := uploadResp["thumbnail_key"].(string)
	if !ok || thumbnailKey == "" {
		t.Fatalf("step 1: expected non-empty thumbnail_key, got %v", uploadResp["thumbnail_key"])
	}

	// Verify photo is in object store
	if _, _, found := objStore.Get(r2Key); !found {
		t.Fatal("step 1: original photo not found in object store")
	}
	if _, _, found := objStore.Get(thumbnailKey); !found {
		t.Fatal("step 1: thumbnail not found in object store")
	}

	// Verify photo_uploads row exists with linked_at = NULL
	photo, err := store.GetPhotoUploadByR2Key(db, r2Key)
	if err != nil {
		t.Fatalf("step 1: GetPhotoUploadByR2Key: %v", err)
	}
	if photo.LinkedAt != nil {
		t.Fatal("step 1: expected linked_at to be NULL before linking")
	}

	// === Step 2: Create stool entry with photo_key -> verify linked_at set ===
	stoolPath := fmt.Sprintf("/api/babies/%s/stools", babyID)
	status, stoolResp := client.doJSON(http.MethodPost, stoolPath, map[string]any{
		"timestamp":    "2025-06-15T10:00:00Z",
		"color_rating": 3,
		"consistency":  "soft",
		"photo_keys":   []string{r2Key},
	})
	if status != http.StatusCreated {
		t.Fatalf("step 2: expected 201 creating stool, got %d: %v", status, stoolResp)
	}
	stoolID := stoolResp["id"].(string)

	// Verify linked_at is now set
	photo, err = store.GetPhotoUploadByR2Key(db, r2Key)
	if err != nil {
		t.Fatalf("step 2: GetPhotoUploadByR2Key: %v", err)
	}
	if photo.LinkedAt == nil {
		t.Fatal("step 2: expected linked_at to be set after linking to stool")
	}

	// === Step 3: Read entry -> verify signed URL returned ===
	status, getResp := client.doJSON(http.MethodGet, fmt.Sprintf("%s/%s", stoolPath, stoolID), nil)
	if status != http.StatusOK {
		t.Fatalf("step 3: expected 200 reading stool, got %d: %v", status, getResp)
	}

	photos, ok := getResp["photos"].([]any)
	if !ok || len(photos) == 0 {
		t.Fatalf("step 3: expected non-empty photos array, got %v", getResp["photos"])
	}
	firstPhoto := photos[0].(map[string]any)
	photoURL, ok := firstPhoto["url"].(string)
	if !ok || photoURL == "" {
		t.Fatalf("step 3: expected non-empty url in photo, got %v", firstPhoto["url"])
	}
	if !strings.Contains(photoURL, "signed=true") {
		t.Errorf("step 3: expected signed URL, got %s", photoURL)
	}

	thumbURL, ok := firstPhoto["thumbnail_url"].(string)
	if !ok || thumbURL == "" {
		t.Fatalf("step 3: expected non-empty thumbnail_url in photo, got %v", firstPhoto["thumbnail_url"])
	}
	if !strings.Contains(thumbURL, "signed=true") {
		t.Errorf("step 3: expected signed thumbnail URL, got %s", thumbURL)
	}
	if !strings.Contains(thumbURL, "thumb_") {
		t.Errorf("step 3: expected thumbnail URL to contain 'thumb_', got %s", thumbURL)
	}

	// === Step 4: Update entry removing photo -> verify linked_at nulled ===
	status, updateResp := client.doJSON(http.MethodPut, fmt.Sprintf("%s/%s", stoolPath, stoolID), map[string]any{
		"timestamp":    "2025-06-15T10:00:00Z",
		"color_rating": 3,
		"consistency":  "soft",
		"photo_keys":   []string{}, // Remove all photos
	})
	if status != http.StatusOK {
		t.Fatalf("step 4: expected 200 updating stool, got %d: %v", status, updateResp)
	}

	// Verify linked_at is now NULL
	photo, err = store.GetPhotoUploadByR2Key(db, r2Key)
	if err != nil {
		t.Fatalf("step 4: GetPhotoUploadByR2Key: %v", err)
	}
	if photo.LinkedAt != nil {
		t.Fatal("step 4: expected linked_at to be NULL after unlinking from stool")
	}

	// Verify the stool response has empty photos
	updatedPhotos, ok := updateResp["photos"].([]any)
	if !ok {
		t.Fatalf("step 4: expected photos array in response, got %v", updateResp["photos"])
	}
	if len(updatedPhotos) != 0 {
		t.Fatalf("step 4: expected 0 photos after removal, got %d", len(updatedPhotos))
	}

	// === Step 5: Run cleanup -> verify R2 delete called ===
	// Set uploaded_at to 25 hours ago to make it eligible for cleanup.
	// Use a fixed reference time so cutoff math is deterministic.
	refTime := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	oldUploadedAt := refTime.Add(-25 * time.Hour).Format(time.DateTime)
	res, err := db.Exec(
		"UPDATE photo_uploads SET uploaded_at = ? WHERE r2_key = ?",
		oldUploadedAt, r2Key,
	)
	if err != nil {
		t.Fatalf("step 5: update uploaded_at: %v", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected != 1 {
		t.Fatalf("step 5: expected 1 row affected updating uploaded_at, got %d", rowsAffected)
	}

	// Run cleanup with the reference time
	deleted, err := cron.CleanupPhotos(db, objStore, refTime)
	if err != nil {
		t.Fatalf("step 5: CleanupPhotos: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("step 5: expected 1 deleted photo, got %d", deleted)
	}

	// Verify photo is removed from object store
	if _, _, found := objStore.Get(r2Key); found {
		t.Fatal("step 5: original photo should be deleted from object store after cleanup")
	}
	if _, _, found := objStore.Get(thumbnailKey); found {
		t.Fatal("step 5: thumbnail should be deleted from object store after cleanup")
	}

	// Verify DB row is also deleted
	_, err = store.GetPhotoUploadByR2Key(db, r2Key)
	if err == nil {
		t.Fatal("step 5: expected photo_uploads row to be deleted after cleanup")
	}
}

// TestPhotoFlow_5MBLimitRejection verifies that uploads over 5MB are rejected.
func TestPhotoFlow_5MBLimitRejection(t *testing.T) {
	t.Parallel()

	srv, db, _, cleanup := setupPhotoServer(t)
	defer cleanup()

	client := newPhotoTestClient(t, srv, db)
	babyID := createPhotoBaby(t, client)

	// Create data larger than 5MB
	bigData := make([]byte, 5*1024*1024+1)
	status, resp := client.uploadPhoto(babyID, "big.jpg", bigData)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 for >5MB file, got %d: %v", status, resp)
	}
	raw, _ := resp["_raw"].(string)
	if !strings.Contains(raw, "5MB") {
		t.Errorf("expected error mentioning 5MB, got: %v", resp)
	}
}

// TestPhotoFlow_InvalidMIMERejection verifies that non-image files are rejected.
func TestPhotoFlow_InvalidMIMERejection(t *testing.T) {
	t.Parallel()

	srv, db, _, cleanup := setupPhotoServer(t)
	defer cleanup()

	client := newPhotoTestClient(t, srv, db)
	babyID := createPhotoBaby(t, client)

	// Upload a text file disguised as a gif
	status, resp := client.uploadPhoto(babyID, "test.gif", []byte("GIF89a"))
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid MIME type, got %d: %v", status, resp)
	}
}

// TestPhotoFlow_4PhotoLimit verifies that no more than 4 photos can be linked to a single entry.
func TestPhotoFlow_4PhotoLimit(t *testing.T) {
	t.Parallel()

	srv, db, _, cleanup := setupPhotoServer(t)
	defer cleanup()

	client := newPhotoTestClient(t, srv, db)
	babyID := createPhotoBaby(t, client)
	stoolPath := fmt.Sprintf("/api/babies/%s/stools", babyID)

	// Upload 5 photos
	jpegData := createTestJPEGData(t, 100, 100)
	var keys []string
	for i := 0; i < 5; i++ {
		status, resp := client.uploadPhoto(babyID, fmt.Sprintf("photo%d.jpg", i), jpegData)
		if status != http.StatusCreated {
			t.Fatalf("expected 201 uploading photo %d, got %d: %v", i, status, resp)
		}
		keys = append(keys, resp["r2_key"].(string))
	}

	// Try to create stool with all 5 photos -> should fail
	status, resp := client.doJSON(http.MethodPost, stoolPath, map[string]any{
		"timestamp":    "2025-06-15T10:00:00Z",
		"color_rating": 3,
		"photo_keys":   keys, // 5 photos
	})
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 for >4 photos, got %d: %v", status, resp)
	}

	// Create stool with exactly 4 photos -> should succeed
	status, resp = client.doJSON(http.MethodPost, stoolPath, map[string]any{
		"timestamp":    "2025-06-15T10:00:00Z",
		"color_rating": 3,
		"photo_keys":   keys[:4], // 4 photos
	})
	if status != http.StatusCreated {
		t.Fatalf("expected 201 with 4 photos, got %d: %v", status, resp)
	}

	// Verify all 4 photos have signed URLs in the response
	photos, ok := resp["photos"].([]any)
	if !ok || len(photos) != 4 {
		t.Fatalf("expected 4 photos in response, got %v", resp["photos"])
	}
	for i, p := range photos {
		photo := p.(map[string]any)
		if photo["url"] == nil || photo["url"] == "" {
			t.Errorf("photo %d: expected non-empty url", i)
		}
		if photo["thumbnail_url"] == nil || photo["thumbnail_url"] == "" {
			t.Errorf("photo %d: expected non-empty thumbnail_url", i)
		}
	}
}

// createPhotoBaby creates a baby via the API and returns the baby ID.
func createPhotoBaby(t *testing.T, client *photoTestClient) string {
	t.Helper()
	status, resp := client.doJSON(http.MethodPost, "/api/babies", map[string]any{
		"name":          "Photo Test Baby",
		"sex":           "female",
		"date_of_birth": "2025-01-01",
	})
	if status != http.StatusCreated {
		t.Fatalf("expected 201 creating baby, got %d: %v", status, resp)
	}
	return resp["id"].(string)
}
