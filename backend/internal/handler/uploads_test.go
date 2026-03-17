package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
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

// createMultipartFile creates a multipart form body with a file field.
func createMultipartFile(t *testing.T, fieldName, filename string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(fieldName, filename)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(content)); err != nil {
		t.Fatalf("copy file data: %v", err)
	}
	writer.Close()
	return body, writer.FormDataContentType()
}

// createTestJPEG creates a minimal valid JPEG image.
func createTestJPEG(t *testing.T, width, height int) []byte {
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

// createTestPNG creates a minimal valid PNG image.
func createTestPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 0, G: 255, B: 0, A: 255})
		}
	}
	buf := &bytes.Buffer{}
	if err := png.Encode(buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func setupUploadTest(t *testing.T) (*testutil.TestFixture, *storage.MemoryStore) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	return &testutil.TestFixture{DB: db, User: user, Baby: baby}, objStore
}

func doUploadRequest(t *testing.T, fix *testutil.TestFixture, objStore storage.ObjectStore,
	body *bytes.Buffer, contentType string) *httptest.ResponseRecorder {
	t.Helper()

	req := testutil.AuthenticatedRequest(t, fix.DB, fix.User.ID, testCookieName, testSecret,
		http.MethodPost, "/api/babies/"+fix.Baby.ID+"/upload")
	req.Body = io.NopCloser(body)
	req.Header.Set("Content-Type", contentType)

	authMw := middleware.Auth(fix.DB, testCookieName)
	csrfMw := middleware.CSRF(fix.DB, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UploadPhotoHandler(fix.DB, objStore))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/upload", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// --- Test: reject files over 5MB ---
func TestUploadPhoto_RejectsOver5MB(t *testing.T) {
	t.Parallel()
	fix, objStore := setupUploadTest(t)
	defer fix.DB.Close()

	// Create data larger than 5MB
	bigData := make([]byte, 5*1024*1024+1)
	body, ct := createMultipartFile(t, "file", "big.jpg", bigData)

	rec := doUploadRequest(t, fix, objStore, body, ct)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for >5MB file, got %d. Body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "5MB") {
		t.Errorf("expected error message to mention 5MB, got: %s", rec.Body.String())
	}
}

// --- Test: reject invalid MIME type ---
func TestUploadPhoto_RejectsInvalidMIME(t *testing.T) {
	t.Parallel()
	fix, objStore := setupUploadTest(t)
	defer fix.DB.Close()

	body, ct := createMultipartFile(t, "file", "test.gif", []byte("GIF89a"))

	rec := doUploadRequest(t, fix, objStore, body, ct)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid MIME, got %d. Body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "MIME") && !strings.Contains(rec.Body.String(), "mime") &&
		!strings.Contains(rec.Body.String(), "type") {
		t.Errorf("expected error about MIME type, got: %s", rec.Body.String())
	}
}

// --- Test: successful JPEG upload stores original + thumbnail ---
func TestUploadPhoto_JPEG_StoresOriginalAndThumbnail(t *testing.T) {
	t.Parallel()
	fix, objStore := setupUploadTest(t)
	defer fix.DB.Close()

	jpegData := createTestJPEG(t, 800, 600)
	body, ct := createMultipartFile(t, "file", "test.jpg", jpegData)

	rec := doUploadRequest(t, fix, objStore, body, ct)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	r2Key, ok := resp["r2_key"].(string)
	if !ok || r2Key == "" {
		t.Fatal("expected non-empty r2_key in response")
	}

	// Verify original is in store
	_, _, found := objStore.Get(r2Key)
	if !found {
		t.Error("original not found in object store")
	}

	// Verify thumbnail is in store
	thumbKey, ok := resp["thumbnail_key"].(string)
	if !ok || thumbKey == "" {
		t.Fatal("expected non-empty thumbnail_key in response")
	}
	_, _, found = objStore.Get(thumbKey)
	if !found {
		t.Error("thumbnail not found in object store")
	}

	// Verify keys have expected structure
	if !strings.HasPrefix(r2Key, "photos/") {
		t.Errorf("expected r2_key to start with photos/, got %s", r2Key)
	}
	if !strings.Contains(thumbKey, "thumb_") {
		t.Errorf("expected thumbnail_key to contain thumb_, got %s", thumbKey)
	}
}

// --- Test: successful PNG upload ---
func TestUploadPhoto_PNG_Success(t *testing.T) {
	t.Parallel()
	fix, objStore := setupUploadTest(t)
	defer fix.DB.Close()

	pngData := createTestPNG(t, 600, 400)
	body, ct := createMultipartFile(t, "file", "test.png", pngData)

	rec := doUploadRequest(t, fix, objStore, body, ct)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["r2_key"] == nil || resp["r2_key"] == "" {
		t.Error("expected non-empty r2_key")
	}
}

// --- Test: creates photo_uploads row with correct keys ---
func TestUploadPhoto_CreatesDBRow(t *testing.T) {
	t.Parallel()
	fix, objStore := setupUploadTest(t)
	defer fix.DB.Close()

	jpegData := createTestJPEG(t, 400, 300)
	body, ct := createMultipartFile(t, "file", "dbtest.jpg", jpegData)

	rec := doUploadRequest(t, fix, objStore, body, ct)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	r2Key := resp["r2_key"].(string)

	// Verify DB row
	photo, err := store.GetPhotoUploadByR2Key(fix.DB, r2Key)
	if err != nil {
		t.Fatalf("GetPhotoUploadByR2Key: %v", err)
	}
	if photo.R2Key != r2Key {
		t.Errorf("DB r2_key mismatch: got %s, want %s", photo.R2Key, r2Key)
	}
	if photo.BabyID == nil || *photo.BabyID != fix.Baby.ID {
		t.Error("DB baby_id mismatch")
	}
	if photo.ThumbnailKey == nil {
		t.Error("expected thumbnail_key in DB row")
	}
}

// --- Test: thumbnail is ~300px wide ---
func TestUploadPhoto_ThumbnailSize(t *testing.T) {
	t.Parallel()
	fix, objStore := setupUploadTest(t)
	defer fix.DB.Close()

	// Create a large image (1200x900)
	jpegData := createTestJPEG(t, 1200, 900)
	body, ct := createMultipartFile(t, "file", "large.jpg", jpegData)

	rec := doUploadRequest(t, fix, objStore, body, ct)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	thumbKey := resp["thumbnail_key"].(string)
	thumbData, _, found := objStore.Get(thumbKey)
	if !found {
		t.Fatal("thumbnail not found in store")
	}

	// Decode thumbnail and check width
	thumbImg, _, err := image.Decode(bytes.NewReader(thumbData))
	if err != nil {
		t.Fatalf("decode thumbnail: %v", err)
	}

	bounds := thumbImg.Bounds()
	thumbWidth := bounds.Dx()
	// Should be approximately 300px wide (allow some tolerance)
	if thumbWidth < 280 || thumbWidth > 320 {
		t.Errorf("expected thumbnail width ~300, got %d", thumbWidth)
	}

	// Aspect ratio should be preserved
	thumbHeight := bounds.Dy()
	expectedHeight := int(float64(900) * float64(thumbWidth) / float64(1200))
	if thumbHeight < expectedHeight-5 || thumbHeight > expectedHeight+5 {
		t.Errorf("expected thumbnail height ~%d, got %d", expectedHeight, thumbHeight)
	}
}

// --- Test: upload by unauthorized user is forbidden ---
func TestUploadPhoto_Unauthorized(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	otherUser := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	objStore := storage.NewMemoryStore()

	jpegData := createTestJPEG(t, 100, 100)
	body, ct := createMultipartFile(t, "file", "test.jpg", jpegData)

	// otherUser is NOT linked to this baby
	req := testutil.AuthenticatedRequest(t, db, otherUser.ID, testCookieName, testSecret,
		http.MethodPost, "/api/babies/"+baby.ID+"/upload")
	req.Body = io.NopCloser(body)
	req.Header.Set("Content-Type", ct)

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UploadPhotoHandler(db, objStore))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/upload", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// --- Test: missing file field ---
func TestUploadPhoto_MissingFile(t *testing.T) {
	t.Parallel()
	fix, objStore := setupUploadTest(t)
	defer fix.DB.Close()

	// Send an empty multipart form with no file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.Close()

	rec := doUploadRequest(t, fix, objStore, body, writer.FormDataContentType())

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing file, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// --- Test: image smaller than 300px wide should not be upscaled ---
func TestUploadPhoto_SmallImage_NoUpscale(t *testing.T) {
	t.Parallel()
	fix, objStore := setupUploadTest(t)
	defer fix.DB.Close()

	// Create a small image (100x80)
	jpegData := createTestJPEG(t, 100, 80)
	body, ct := createMultipartFile(t, "file", "small.jpg", jpegData)

	rec := doUploadRequest(t, fix, objStore, body, ct)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	thumbKey := resp["thumbnail_key"].(string)
	thumbData, _, found := objStore.Get(thumbKey)
	if !found {
		t.Fatal("thumbnail not found in store")
	}

	thumbImg, _, err := image.Decode(bytes.NewReader(thumbData))
	if err != nil {
		t.Fatalf("decode thumbnail: %v", err)
	}

	bounds := thumbImg.Bounds()
	// Small images should keep their original size
	if bounds.Dx() != 100 {
		t.Errorf("expected thumbnail width 100 (no upscale), got %d", bounds.Dx())
	}
}

// --- Test: returns r2_key in response ---
func TestUploadPhoto_ReturnsR2Key(t *testing.T) {
	t.Parallel()
	fix, objStore := setupUploadTest(t)
	defer fix.DB.Close()

	jpegData := createTestJPEG(t, 400, 300)
	body, ct := createMultipartFile(t, "file", "response.jpg", jpegData)

	rec := doUploadRequest(t, fix, objStore, body, ct)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Should have r2_key, thumbnail_key, and id
	for _, field := range []string{"r2_key", "thumbnail_key", "id"} {
		v, ok := resp[field].(string)
		if !ok || v == "" {
			t.Errorf("expected non-empty %s in response, got %v", field, resp[field])
		}
	}

	// Response contains all expected fields
}

// doUploadRequestWithHEIC sends an upload request with a mock HEIC converter injected.
func doUploadRequestWithHEIC(t *testing.T, fix *testutil.TestFixture, objStore storage.ObjectStore,
	body *bytes.Buffer, contentType string, heicConv handler.HEICConverter) *httptest.ResponseRecorder {
	t.Helper()

	req := testutil.AuthenticatedRequest(t, fix.DB, fix.User.ID, testCookieName, testSecret,
		http.MethodPost, "/api/babies/"+fix.Baby.ID+"/upload")
	req.Body = io.NopCloser(body)
	req.Header.Set("Content-Type", contentType)

	authMw := middleware.Auth(fix.DB, testCookieName)
	csrfMw := middleware.CSRF(fix.DB, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UploadPhotoHandler(fix.DB, objStore, heicConv))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/upload", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// --- Test: HEIC input is converted to JPEG before storage ---
func TestUploadPhoto_HEIC_ConvertedToJPEG(t *testing.T) {
	t.Parallel()
	fix, objStore := setupUploadTest(t)
	defer fix.DB.Close()

	// Create a fake "HEIC" file (just some bytes with .heic extension)
	heicData := []byte("fake-heic-data-not-real")

	// Mock HEIC converter that returns a valid JPEG
	mockJPEG := createTestJPEG(t, 800, 600)
	mockConverter := func(_ context.Context, data []byte) ([]byte, error) {
		// Verify the HEIC data was passed through
		if !bytes.Equal(data, heicData) {
			t.Error("converter received unexpected data")
		}
		return mockJPEG, nil
	}

	body, ct := createMultipartFile(t, "file", "photo.heic", heicData)
	rec := doUploadRequestWithHEIC(t, fix, objStore, body, ct, mockConverter)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for HEIC upload, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	r2Key := resp["r2_key"].(string)
	// After HEIC conversion, the stored file should be JPEG
	if !strings.HasSuffix(r2Key, ".jpg") {
		t.Errorf("expected .jpg extension after HEIC conversion, got %s", r2Key)
	}

	// Verify the stored data is the converted JPEG
	storedData, ct2, found := objStore.Get(r2Key)
	if !found {
		t.Fatal("original not found in store after HEIC conversion")
	}
	if ct2 != "image/jpeg" {
		t.Errorf("expected content type image/jpeg, got %s", ct2)
	}
	if !bytes.Equal(storedData, mockJPEG) {
		t.Error("stored data does not match converted JPEG")
	}
}

// --- Test: HEIC with .heif extension also works ---
func TestUploadPhoto_HEIF_Extension(t *testing.T) {
	t.Parallel()
	fix, objStore := setupUploadTest(t)
	defer fix.DB.Close()

	mockJPEG := createTestJPEG(t, 400, 300)
	mockConverter := func(_ context.Context, data []byte) ([]byte, error) {
		return mockJPEG, nil
	}

	body, ct := createMultipartFile(t, "file", "photo.heif", []byte("fake-heif"))
	rec := doUploadRequestWithHEIC(t, fix, objStore, body, ct, mockConverter)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for HEIF upload, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// --- Test: HEIC conversion failure returns 500 ---
func TestUploadPhoto_HEIC_ConversionFailure(t *testing.T) {
	t.Parallel()
	fix, objStore := setupUploadTest(t)
	defer fix.DB.Close()

	failConverter := func(_ context.Context, data []byte) ([]byte, error) {
		return nil, fmt.Errorf("imagemagick not found")
	}

	body, ct := createMultipartFile(t, "file", "photo.heic", []byte("fake-heic"))
	rec := doUploadRequestWithHEIC(t, fix, objStore, body, ct, failConverter)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for HEIC conversion failure, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}
