package handler

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
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

const maxUploadSize = 5 * 1024 * 1024 // 5MB

// allowedMIMETypes maps accepted MIME types to file extensions.
var allowedMIMETypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/heic": ".heic",
}

// HEICConverter is a function type for converting HEIC data to JPEG.
// It can be replaced in tests.
type HEICConverter func(ctx context.Context, data []byte) ([]byte, error)

// DefaultHEICConverter converts HEIC to JPEG using ImageMagick.
func DefaultHEICConverter(ctx context.Context, data []byte) ([]byte, error) {
	return convertHEICToJPEGWithCmd(ctx, data)
}

// UploadPhotoHandler handles POST /api/babies/:id/upload.
// Accepts multipart form data with a "file" field.
// The heicConv parameter allows injecting a mock HEIC converter for testing.
// If nil, DefaultHEICConverter (ImageMagick) is used.
func UploadPhotoHandler(db *sql.DB, objStore storage.ObjectStore, heicConv ...HEICConverter) http.HandlerFunc {
	converter := HEICConverter(DefaultHEICConverter)
	if len(heicConv) > 0 && heicConv[0] != nil {
		converter = heicConv[0]
	}

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
			http.Error(w, "file exceeds 5MB limit", http.StatusBadRequest)
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
			http.Error(w, "file exceeds 5MB limit", http.StatusBadRequest)
			return
		}

		// Read file data
		fileData, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "file exceeds 5MB limit", http.StatusBadRequest)
			return
		}

		if len(fileData) > maxUploadSize {
			http.Error(w, "file exceeds 5MB limit", http.StatusBadRequest)
			return
		}

		// Detect MIME type from content
		mimeType := http.DetectContentType(fileData)

		// HEIC files may not be detected correctly by DetectContentType;
		// also check file extension.
		ext := strings.ToLower(filepath.Ext(header.Filename))
		if ext == ".heic" || ext == ".heif" {
			mimeType = "image/heic"
		}

		if _, ok := allowedMIMETypes[mimeType]; !ok {
			http.Error(w, "unsupported file type; accepted: JPEG, PNG, HEIC", http.StatusBadRequest)
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

		// Generate unique key
		id := model.NewULID()
		ext = extensionForMIME(mimeType)
		r2Key := fmt.Sprintf("photos/%s%s", id, ext)
		thumbKey := fmt.Sprintf("photos/thumb_%s%s", id, ext)

		// Upload original
		ctx := r.Context()
		if err := objStore.Put(ctx, r2Key, bytes.NewReader(fileData), mimeType); err != nil {
			log.Printf("upload original: %v", err)
			http.Error(w, "failed to store photo", http.StatusInternalServerError)
			return
		}

		// Generate and upload thumbnail
		thumbData, thumbMIME, err := generateThumbnail(fileData, mimeType)
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

// generateThumbnail creates a ~300px wide thumbnail from image data.
// If the image is already <= 300px wide, it returns the original data unchanged.
func generateThumbnail(data []byte, mimeType string) ([]byte, string, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("decode image: %w", err)
	}

	bounds := img.Bounds()
	origWidth := bounds.Dx()
	origHeight := bounds.Dy()

	targetWidth := 300
	if origWidth <= targetWidth {
		// No resize needed — return original data
		return data, mimeType, nil
	}

	// Calculate new dimensions preserving aspect ratio
	ratio := float64(targetWidth) / float64(origWidth)
	newHeight := int(float64(origHeight) * ratio)

	// Simple nearest-neighbor resize using Go stdlib
	thumb := resizeNearestNeighbor(img, targetWidth, newHeight)

	var buf bytes.Buffer
	outMIME := mimeType

	switch mimeType {
	case "image/png":
		if err := png.Encode(&buf, thumb); err != nil {
			return nil, "", fmt.Errorf("encode png thumbnail: %w", err)
		}
	default:
		// Default to JPEG
		outMIME = "image/jpeg"
		if err := jpeg.Encode(&buf, thumb, &jpeg.Options{Quality: 85}); err != nil {
			return nil, "", fmt.Errorf("encode jpeg thumbnail: %w", err)
		}
	}

	return buf.Bytes(), outMIME, nil
}

// resizeNearestNeighbor performs nearest-neighbor image resizing using Go stdlib.
func resizeNearestNeighbor(src image.Image, width, height int) image.Image {
	bounds := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := bounds.Min.X + x*bounds.Dx()/width
			srcY := bounds.Min.Y + y*bounds.Dy()/height
			dst.Set(x, y, src.At(srcX, srcY))
		}
	}

	return dst
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
