package handler

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/storage"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

const maxUploadSize = 25 * 1024 * 1024 // 25MB — server downscales images to ≤2000px before storage

const maxUploadSizeLabel = "25MB"

const maxPDFUploadSize = 20 * 1024 * 1024 // 20MB cap for PDF radiology reports

const maxPDFUploadSizeLabel = "20MB"

// allowedMIMETypes maps accepted MIME types to file extensions.
var allowedMIMETypes = map[string]string{
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/heic":      ".heic",
	"application/pdf": ".pdf",
}

// HEICConverter is a function type for converting HEIC data to JPEG.
// It can be replaced in tests.
type HEICConverter func(ctx context.Context, data []byte) ([]byte, error)

// DefaultHEICConverter converts HEIC to JPEG using ImageMagick.
func DefaultHEICConverter(ctx context.Context, data []byte) ([]byte, error) {
	return convertHEICToJPEGWithCmd(ctx, data)
}

// PDFRasterizer rasterizes the first page of a PDF to a JPEG thumbnail.
// Replaceable in tests; default uses ImageMagick + Ghostscript.
type PDFRasterizer func(ctx context.Context, pdfData []byte) ([]byte, error)

// DefaultPDFRasterizer rasterizes the first page of a PDF using ImageMagick + Ghostscript.
// Returns a JPEG thumbnail (300px wide). On rasterize failure (malformed PDF, missing
// Ghostscript, etc.), returns an error and the caller falls back to no thumbnail.
func DefaultPDFRasterizer(ctx context.Context, pdfData []byte) ([]byte, error) {
	return rasterizePDFFirstPage(ctx, pdfData)
}

// UploadPhotoHandler handles POST /api/babies/:id/upload.
// Accepts multipart form data with a "file" field.
// The heicConv parameter allows injecting a mock HEIC converter for testing.
// If nil, DefaultHEICConverter (ImageMagick) is used.
func UploadPhotoHandler(db *sql.DB, objStore storage.ObjectStore, heicConv ...HEICConverter) http.HandlerFunc {
	return uploadPhotoCore(db, objStore, firstHEIC(heicConv), DefaultPDFRasterizer)
}

// UploadPhotoHandlerWithPDF allows tests to inject a custom PDF rasterizer.
func UploadPhotoHandlerWithPDF(db *sql.DB, objStore storage.ObjectStore, heicConv HEICConverter, pdfRast PDFRasterizer) http.HandlerFunc {
	if heicConv == nil {
		heicConv = DefaultHEICConverter
	}
	if pdfRast == nil {
		pdfRast = DefaultPDFRasterizer
	}
	return uploadPhotoCore(db, objStore, heicConv, pdfRast)
}

func firstHEIC(in []HEICConverter) HEICConverter {
	if len(in) > 0 && in[0] != nil {
		return in[0]
	}
	return DefaultHEICConverter
}

func uploadPhotoCore(db *sql.DB, objStore storage.ObjectStore, converter HEICConverter, pdfRast PDFRasterizer) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		// Limit request body to maxUploadSize + some overhead for multipart headers
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize+1024)

		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			http.Error(w, "file exceeds " + maxUploadSizeLabel + " limit", http.StatusBadRequest)
			return
		}
		defer r.MultipartForm.RemoveAll()

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "missing file field", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Check file size
		if header.Size > maxUploadSize {
			http.Error(w, "file exceeds " + maxUploadSizeLabel + " limit", http.StatusBadRequest)
			return
		}

		// Read file data
		fileData, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "file exceeds " + maxUploadSizeLabel + " limit", http.StatusBadRequest)
			return
		}

		if len(fileData) > maxUploadSize {
			http.Error(w, "file exceeds " + maxUploadSizeLabel + " limit", http.StatusBadRequest)
			return
		}

		// Detect MIME type from content
		mimeType := http.DetectContentType(fileData)

		// HEIC and PDF files may not be detected correctly by DetectContentType;
		// also check file extension.
		ext := strings.ToLower(filepath.Ext(header.Filename))
		if ext == ".heic" || ext == ".heif" {
			mimeType = "image/heic"
		}
		if ext == ".pdf" {
			mimeType = "application/pdf"
		}

		if _, ok := allowedMIMETypes[mimeType]; !ok {
			http.Error(w, "unsupported file type; accepted: JPEG, PNG, HEIC, PDF", http.StatusBadRequest)
			return
		}

		ctx := r.Context()

		// PDF branch: store as-is, rasterize first page for thumbnail.
		if mimeType == "application/pdf" {
			if len(fileData) > maxPDFUploadSize {
				http.Error(w, "PDF exceeds "+maxPDFUploadSizeLabel+" limit", http.StatusBadRequest)
				return
			}
			handlePDFUpload(ctx, w, db, objStore, pdfRast, baby.ID, fileData)
			return
		}

		// Handle HEIC conversion
		if mimeType == "image/heic" {
			converted, err := converter(r.Context(), fileData)
			if err != nil {
				log.Printf("HEIC conversion failed: %v", err)
				http.Error(w, "failed to convert HEIC image", http.StatusInternalServerError)
				return
			}
			fileData = converted
			mimeType = "image/jpeg"
		}

		// Downscale the stored "original" to keep storage + memory bounded.
		// Output is JPEG regardless of input format.
		resized, resizedMIME, err := imagemagickResizeOriginal(ctx, fileData, mimeType)
		if err != nil {
			log.Printf("resize original: %v", err)
			http.Error(w, "failed to process image", http.StatusInternalServerError)
			return
		}
		fileData = resized
		mimeType = resizedMIME

		// Generate unique key
		id := model.NewULID()
		ext = extensionForMIME(mimeType)
		r2Key := fmt.Sprintf("photos/%s%s", id, ext)
		thumbKey := fmt.Sprintf("photos/thumb_%s%s", id, ext)

		// Upload original
		if err := objStore.Put(ctx, r2Key, bytes.NewReader(fileData), mimeType); err != nil {
			log.Printf("upload original: %v", err)
			http.Error(w, "failed to store photo", http.StatusInternalServerError)
			return
		}

		// Generate and upload thumbnail
		thumbData, thumbMIME, err := imagemagickThumbnail(ctx, fileData, mimeType)
		if err != nil {
			log.Printf("generate thumbnail: %v", err)
			http.Error(w, "failed to generate thumbnail", http.StatusInternalServerError)
			return
		}

		if err := objStore.Put(ctx, thumbKey, bytes.NewReader(thumbData), thumbMIME); err != nil {
			log.Printf("upload thumbnail: %v", err)
			http.Error(w, "failed to store thumbnail", http.StatusInternalServerError)
			return
		}

		// Create DB record
		photo, err := store.CreatePhotoUpload(db, baby.ID, r2Key, thumbKey)
		if err != nil {
			log.Printf("create photo_upload: %v", err)
			http.Error(w, "failed to create photo record", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, map[string]string{
			"id":            photo.ID,
			"r2_key":        photo.R2Key,
			"thumbnail_key": stringOrEmpty(photo.ThumbnailKey),
		})
	}
}

// handlePDFUpload stores a PDF original (no re-encode) and tries to generate a
// JPEG thumbnail of the first page. On rasterize failure (malformed PDF, etc.)
// the upload still succeeds with a null thumbnail_key — downstream consumers
// fall back to a generic PDF icon.
func handlePDFUpload(ctx context.Context, w http.ResponseWriter, db *sql.DB, objStore storage.ObjectStore, pdfRast PDFRasterizer, babyID string, pdfData []byte) {
	id := model.NewULID()
	r2Key := fmt.Sprintf("photos/%s.pdf", id)

	if err := objStore.Put(ctx, r2Key, bytes.NewReader(pdfData), "application/pdf"); err != nil {
		log.Printf("upload PDF original: %v", err)
		http.Error(w, "failed to store PDF", http.StatusInternalServerError)
		return
	}

	// Best-effort thumbnail generation; failure is non-fatal.
	thumbKey := ""
	if pdfRast != nil {
		thumbData, err := pdfRast(ctx, pdfData)
		if err != nil {
			log.Printf("PDF rasterize failed (non-fatal): %v", err)
		} else {
			tk := fmt.Sprintf("photos/thumb_%s.jpg", id)
			if putErr := objStore.Put(ctx, tk, bytes.NewReader(thumbData), "image/jpeg"); putErr != nil {
				log.Printf("upload PDF thumbnail (non-fatal): %v", putErr)
			} else {
				thumbKey = tk
			}
		}
	}

	photo, err := store.CreatePhotoUpload(db, babyID, r2Key, thumbKey)
	if err != nil {
		log.Printf("create photo_upload (PDF): %v", err)
		http.Error(w, "failed to create photo record", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"id":            photo.ID,
		"r2_key":        photo.R2Key,
		"thumbnail_key": stringOrEmpty(photo.ThumbnailKey),
	})
}

// extensionForMIME returns the file extension for a MIME type.
func extensionForMIME(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	default:
		return ".bin"
	}
}

// stringOrEmpty returns the string value of a *string or empty string if nil.
func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// imagemagickResizeOriginal downscales an image to fit within 2000x2000
// (longest side) and re-encodes it as JPEG to keep the stored "original"
// small. Memory peaks inside the ImageMagick subprocess, not the Go heap.
// `-resize 2000x2000>` only shrinks images larger than the bound.
func imagemagickResizeOriginal(ctx context.Context, data []byte, mimeType string) ([]byte, string, error) {
	cmd := exec.CommandContext(ctx, "convert", "-", "-resize", "2000x2000>", "-quality", "85", "jpg:-")
	cmd.Stdin = bytes.NewReader(data)

	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, "", fmt.Errorf("imagemagick resize: %w (stderr: %s)", err, stderr.String())
	}
	_ = mimeType
	return out.Bytes(), "image/jpeg", nil
}

// imagemagickThumbnail produces a ~300px-wide thumbnail by shelling out to
// ImageMagick. Memory peaks inside the subprocess instead of the Go heap, and
// `-resize 300x>` skips upscaling for images already smaller than the target.
func imagemagickThumbnail(ctx context.Context, data []byte, mimeType string) ([]byte, string, error) {
	outFormat := "jpg"
	outMIME := "image/jpeg"
	if mimeType == "image/png" {
		outFormat = "png"
		outMIME = "image/png"
	}

	cmd := exec.CommandContext(ctx, "convert", "-", "-resize", "300x>", outFormat+":-")
	cmd.Stdin = bytes.NewReader(data)

	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, "", fmt.Errorf("imagemagick thumbnail: %w (stderr: %s)", err, stderr.String())
	}
	return out.Bytes(), outMIME, nil
}

// rasterizePDFFirstPage rasterizes the first page of a PDF to a 300px-wide JPEG
// thumbnail using ImageMagick + Ghostscript. Memory peaks inside the subprocess
// chain rather than the Go heap.
func rasterizePDFFirstPage(ctx context.Context, pdfData []byte) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "convert", "-density", "150", "pdf:-[0]", "-resize", "300x>", "-quality", "85", "jpg:-")
	cmd.Stdin = bytes.NewReader(pdfData)

	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("imagemagick rasterize PDF: %w (stderr: %s)", err, stderr.String())
	}
	if out.Len() == 0 {
		return nil, fmt.Errorf("imagemagick rasterize PDF: empty output")
	}
	return out.Bytes(), nil
}

// convertHEICToJPEGWithCmd shells out to ImageMagick's convert command.
func convertHEICToJPEGWithCmd(ctx context.Context, data []byte) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "convert", "-", "jpg:-")
	cmd.Stdin = bytes.NewReader(data)

	var out bytes.Buffer
	cmd.Stdout = &out

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("imagemagick convert: %w (stderr: %s)", err, stderr.String())
	}

	return out.Bytes(), nil
}
