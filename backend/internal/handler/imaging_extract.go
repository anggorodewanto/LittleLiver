package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/ablankz/LittleLiver/backend/internal/labextract"
	"github.com/ablankz/LittleLiver/backend/internal/storage"
)

// imagingExtractRequest is the JSON request body for imaging extraction.
type imagingExtractRequest struct {
	PhotoKeys []string `json:"photo_keys"`
}

// imagingExtractResponse is the JSON response shape for imaging extraction.
// suggested fields may be empty when the model couldn't determine them.
type imagingExtractResponse struct {
	Suggested labextract.ImagingSuggestion `json:"suggested"`
	Notes     string                       `json:"notes,omitempty"`
}

// ImagingExtractHandlerWithRateLimit handles POST /api/babies/{id}/imaging-studies/extract.
// Shares the rate limiter with /labs/extract.
func ImagingExtractHandlerWithRateLimit(db *sql.DB, objStore storage.ObjectStore, svc *labextract.Service, rl *ExtractRateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		if rl != nil {
			allowed, remaining := rl.Allow(user.ID)
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			if !allowed {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
		}

		var req imagingExtractRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if err := labextract.ValidatePhotoKeys(req.PhotoKeys); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		images := make([]labextract.ImageData, 0, len(req.PhotoKeys))
		for _, key := range req.PhotoKeys {
			var exists int
			err := db.QueryRow("SELECT COUNT(*) FROM photo_uploads WHERE r2_key = ? AND baby_id = ?", key, baby.ID).Scan(&exists)
			if err != nil || exists == 0 {
				http.Error(w, "invalid photo key: "+key, http.StatusBadRequest)
				return
			}

			// For PDF originals, send the JPEG thumbnail (already first-page rasterized at upload).
			// Vision API does not accept raw PDF bytes.
			data, contentType, err := loadVisionImage(r, db, objStore, key)
			if err != nil {
				http.Error(w, "invalid photo key: "+key, http.StatusBadRequest)
				return
			}

			images = append(images, labextract.ImageData{
				Data:        data,
				ContentType: contentType,
			})
		}

		suggestion, err := svc.ExtractImaging(r.Context(), images)
		if err != nil {
			log.Printf("imaging extraction: %v", err)
			http.Error(w, "extraction failed", http.StatusBadGateway)
			return
		}
		if suggestion == nil {
			suggestion = &labextract.ImagingSuggestion{}
		}

		writeJSON(w, http.StatusOK, imagingExtractResponse{
			Suggested: *suggestion,
			Notes:     suggestion.Notes,
		})
	}
}

// loadVisionImage fetches bytes suitable to send to Claude Vision for a photo_uploads key.
// For .pdf keys, returns the JPEG thumbnail (first-page rasterization done at upload).
// For image keys, returns the resized JPEG original.
func loadVisionImage(r *http.Request, db *sql.DB, objStore storage.ObjectStore, key string) ([]byte, string, error) {
	row := db.QueryRow("SELECT thumbnail_key FROM photo_uploads WHERE r2_key = ?", key)
	var thumbnailKey sql.NullString
	if err := row.Scan(&thumbnailKey); err != nil {
		return nil, "", err
	}

	// PDF: substitute the first-page JPEG thumbnail (no raw PDF to Vision).
	if isPDFKey(key) {
		if !thumbnailKey.Valid || thumbnailKey.String == "" {
			return nil, "", fmt.Errorf("no thumbnail for PDF key %s", key)
		}
		data, err := objStore.Get(r.Context(), thumbnailKey.String)
		if err != nil {
			return nil, "", err
		}
		return data, "image/jpeg", nil
	}

	data, err := objStore.Get(r.Context(), key)
	if err != nil {
		return nil, "", err
	}
	return data, http.DetectContentType(data), nil
}

func isPDFKey(key string) bool {
	if len(key) < 4 {
		return false
	}
	return key[len(key)-4:] == ".pdf"
}
