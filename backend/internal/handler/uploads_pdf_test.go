package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

func doPDFUploadRequest(t *testing.T, fix *testutil.TestFixture, objStore storage.ObjectStore,
	body *bytes.Buffer, contentType string, pdfRast handler.PDFRasterizer) *httptest.ResponseRecorder {
	t.Helper()

	req := testutil.AuthenticatedRequest(t, fix.DB, fix.User.ID, testCookieName, testSecret,
		http.MethodPost, "/api/babies/"+fix.Baby.ID+"/upload")
	req.Body = io.NopCloser(body)
	req.Header.Set("Content-Type", contentType)

	authMw := middleware.Auth(fix.DB, testCookieName)
	csrfMw := middleware.CSRF(fix.DB, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UploadPhotoHandlerWithPDF(fix.DB, objStore, nil, pdfRast))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/upload", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func minimalPDFBytes() []byte {
	// Minimal PDF header + EOF marker. Enough for content-type detection.
	return []byte("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n2 0 obj<</Type/Pages/Kids[]/Count 0>>endobj\nxref\n0 3\n0000000000 65535 f\n0000000009 00000 n\n0000000054 00000 n\ntrailer<</Size 3/Root 1 0 R>>\nstartxref\n95\n%%EOF\n")
}

func TestUploadPDF_Success_StoresOriginalAndThumbnail(t *testing.T) {
	t.Parallel()
	fix, objStore := setupUploadTest(t)
	defer fix.DB.Close()

	pdfData := minimalPDFBytes()
	body, ct := createMultipartFile(t, "file", "report.pdf", pdfData)

	// Inject a PDF rasterizer that returns a fake JPEG
	rast := handler.PDFRasterizer(func(ctx context.Context, data []byte) ([]byte, error) {
		return []byte{0xff, 0xd8, 0xff, 0xe0, 0, 0x10, 'J', 'F', 'I', 'F'}, nil
	})

	rec := doPDFUploadRequest(t, fix, objStore, body, ct, rast)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	r2Key, _ := resp["r2_key"].(string)
	if !strings.HasSuffix(r2Key, ".pdf") {
		t.Errorf("expected r2_key to end .pdf, got %s", r2Key)
	}
	if !strings.HasPrefix(r2Key, "photos/") {
		t.Errorf("expected r2_key to start photos/, got %s", r2Key)
	}

	thumbKey, _ := resp["thumbnail_key"].(string)
	if !strings.HasSuffix(thumbKey, ".jpg") {
		t.Errorf("expected thumbnail_key to end .jpg, got %s", thumbKey)
	}
	if !strings.Contains(thumbKey, "thumb_") {
		t.Errorf("expected thumbnail_key to contain thumb_, got %s", thumbKey)
	}

	// Verify both objects in store
	if _, _, found := objStore.GetWithMeta(r2Key); !found {
		t.Error("PDF original not in object store")
	}
	if _, _, found := objStore.GetWithMeta(thumbKey); !found {
		t.Error("PDF thumbnail not in object store")
	}

	// DB row
	photo, err := store.GetPhotoUploadByR2Key(fix.DB, r2Key)
	if err != nil {
		t.Fatalf("GetPhotoUploadByR2Key: %v", err)
	}
	if photo.ThumbnailKey == nil || *photo.ThumbnailKey != thumbKey {
		t.Errorf("DB thumbnail_key mismatch: got %v, want %s", photo.ThumbnailKey, thumbKey)
	}
}

func TestUploadPDF_RasterizeFailure_StoresWithNullThumbnail(t *testing.T) {
	t.Parallel()
	fix, objStore := setupUploadTest(t)
	defer fix.DB.Close()

	pdfData := minimalPDFBytes()
	body, ct := createMultipartFile(t, "file", "report.pdf", pdfData)

	// Rasterizer fails (simulating malformed PDF)
	rast := handler.PDFRasterizer(func(ctx context.Context, data []byte) ([]byte, error) {
		return nil, errors.New("simulated rasterize failure")
	})

	rec := doPDFUploadRequest(t, fix, objStore, body, ct, rast)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 (upload still succeeds on rasterize failure), got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)

	r2Key, _ := resp["r2_key"].(string)
	if !strings.HasSuffix(r2Key, ".pdf") {
		t.Errorf("expected r2_key to end .pdf, got %s", r2Key)
	}

	thumbKey, _ := resp["thumbnail_key"].(string)
	if thumbKey != "" {
		t.Errorf("expected empty thumbnail_key on rasterize failure, got %q", thumbKey)
	}

	// PDF original stored
	if _, _, found := objStore.GetWithMeta(r2Key); !found {
		t.Error("PDF original not in object store")
	}

	// DB row exists with NULL thumbnail
	photo, err := store.GetPhotoUploadByR2Key(fix.DB, r2Key)
	if err != nil {
		t.Fatalf("GetPhotoUploadByR2Key: %v", err)
	}
	if photo.ThumbnailKey != nil {
		t.Errorf("expected NULL thumbnail_key, got %v", *photo.ThumbnailKey)
	}
}

func TestUploadPDF_TooLarge_Rejected(t *testing.T) {
	t.Parallel()
	fix, objStore := setupUploadTest(t)
	defer fix.DB.Close()

	// 21 MB to exceed 20 MB PDF cap (still under 25 MB raw cap so we hit the PDF check)
	pdfHeader := []byte("%PDF-1.4\n")
	bigPDF := append(pdfHeader, make([]byte, 21*1024*1024)...)
	body, ct := createMultipartFile(t, "file", "big.pdf", bigPDF)

	rast := handler.PDFRasterizer(func(ctx context.Context, data []byte) ([]byte, error) {
		t.Error("rasterizer should not be called for over-size PDF")
		return nil, nil
	})

	rec := doPDFUploadRequest(t, fix, objStore, body, ct, rast)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "20MB") {
		t.Errorf("expected error message to mention 20MB, got: %s", rec.Body.String())
	}
}

func TestUploadPDF_DetectedByExtension(t *testing.T) {
	t.Parallel()
	fix, objStore := setupUploadTest(t)
	defer fix.DB.Close()

	// Send PDF with non-standard MIME — extension should still trigger PDF path
	pdfData := minimalPDFBytes()
	body, ct := createMultipartFile(t, "file", "report.pdf", pdfData)
	_ = ct

	rast := handler.PDFRasterizer(func(ctx context.Context, data []byte) ([]byte, error) {
		return []byte{0xff, 0xd8}, nil
	})

	rec := doPDFUploadRequest(t, fix, objStore, body, ct, rast)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}
