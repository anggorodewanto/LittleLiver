package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/labextract"
	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/storage"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

// labExtractRequest is the JSON request body for lab extraction.
type labExtractRequest struct {
	PhotoKeys []string `json:"photo_keys"`
}

// labExtractResponse is the JSON response for lab extraction.
type labExtractResponse struct {
	Extracted  []labextract.ExtractedResult `json:"extracted"`
	Notes      string                       `json:"notes"`
	ReportDate string                       `json:"report_date,omitempty"`
}

// ExtractRateLimiter tracks per-user extraction request counts (10/hour).
type ExtractRateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*extractBucket
}

type extractBucket struct {
	count       int
	windowStart time.Time
}

// NewExtractRateLimiter creates a rate limiter for lab extraction (10 req/hour/user).
func NewExtractRateLimiter() *ExtractRateLimiter {
	return &ExtractRateLimiter{
		buckets: make(map[string]*extractBucket),
	}
}

// allow checks whether the user is within the rate limit and returns (allowed, remaining).
func (rl *ExtractRateLimiter) allow(userID string) (bool, int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.buckets[userID]
	if !ok {
		rl.buckets[userID] = &extractBucket{count: 1, windowStart: now}
		return true, 9
	}

	if now.Sub(b.windowStart) >= time.Hour {
		b.count = 1
		b.windowStart = now
		return true, 9
	}

	if b.count >= 10 {
		return false, 0
	}

	b.count++
	return true, 10 - b.count
}

// LabExtractHandler handles POST /api/babies/{id}/labs/extract without rate limiting.
func LabExtractHandler(db *sql.DB, objStore storage.ObjectStore, svc *labextract.Service) http.HandlerFunc {
	return labExtractCore(db, objStore, svc, nil)
}

// LabExtractHandlerWithRateLimit handles POST /api/babies/{id}/labs/extract with rate limiting.
func LabExtractHandlerWithRateLimit(db *sql.DB, objStore storage.ObjectStore, svc *labextract.Service, rl *ExtractRateLimiter) http.HandlerFunc {
	return labExtractCore(db, objStore, svc, rl)
}

func labExtractCore(db *sql.DB, objStore storage.ObjectStore, svc *labextract.Service, rl *ExtractRateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		// Rate limit check
		if rl != nil {
			allowed, remaining := rl.allow(user.ID)
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			if !allowed {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
		}

		var req labExtractRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		// Validate photo keys count
		if err := labextract.ValidatePhotoKeys(req.PhotoKeys); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Validate R2 keys exist in photo_uploads for this baby, then fetch image bytes
		images := make([]labextract.ImageData, 0, len(req.PhotoKeys))
		for _, key := range req.PhotoKeys {
			var exists int
			err := db.QueryRow("SELECT COUNT(*) FROM photo_uploads WHERE r2_key = ? AND baby_id = ?", key, baby.ID).Scan(&exists)
			if err != nil || exists == 0 {
				http.Error(w, "invalid photo key: "+key, http.StatusBadRequest)
				return
			}

			data, err := objStore.Get(r.Context(), key)
			if err != nil {
				http.Error(w, "invalid photo key: "+key, http.StatusBadRequest)
				return
			}

			contentType := http.DetectContentType(data)
			images = append(images, labextract.ImageData{
				Data:        data,
				ContentType: contentType,
			})
		}

		// Call extraction service
		extraction, err := svc.Extract(r.Context(), images)
		if err != nil {
			log.Printf("lab extraction: %v", err)
			http.Error(w, "extraction failed", http.StatusBadGateway)
			return
		}

		// Use report_date for duplicate detection if available, otherwise fall back to now
		referenceDate := time.Now().UTC()
		if extraction.ReportDate != "" {
			if parsed, err := time.Parse("2006-01-02", extraction.ReportDate); err == nil {
				referenceDate = parsed
			}
		}

		// Check for duplicate lab results
		for i := range extraction.Results {
			match, err := store.FindDuplicateLabResult(db, baby.ID, extraction.Results[i].TestName, extraction.Results[i].Value, referenceDate)
			if err != nil {
				log.Printf("duplicate check: %v", err)
				continue
			}
			if match != nil {
				unit := ""
				if match.Unit != nil {
					unit = *match.Unit
				}
				extraction.Results[i].ExistingMatch = &labextract.ExistingMatch{
					ID:        match.ID,
					Timestamp: match.Timestamp.Format(model.DateTimeFormat),
					Value:     match.Value,
					Unit:      unit,
				}
			}
		}

		writeJSON(w, http.StatusOK, labExtractResponse{
			Extracted:  extraction.Results,
			Notes:      extraction.Notes,
			ReportDate: extraction.ReportDate,
		})
	}
}
