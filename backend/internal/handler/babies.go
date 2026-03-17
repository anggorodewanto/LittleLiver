package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/middleware"
	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

// babyRequest is the JSON request body for creating/updating a baby.
type babyRequest struct {
	Name              string   `json:"name"`
	Sex               string   `json:"sex"`
	DateOfBirth       string   `json:"date_of_birth"`
	DiagnosisDate     *string  `json:"diagnosis_date,omitempty"`
	KasaiDate         *string  `json:"kasai_date,omitempty"`
	DefaultCalPerFeed *float64 `json:"default_cal_per_feed,omitempty"`
	Notes             *string  `json:"notes,omitempty"`
}

// babyResponse is the JSON response for a baby.
type babyResponse struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	Sex               string  `json:"sex"`
	DateOfBirth       string  `json:"date_of_birth"`
	DiagnosisDate     *string `json:"diagnosis_date,omitempty"`
	KasaiDate         *string `json:"kasai_date,omitempty"`
	DefaultCalPerFeed float64 `json:"default_cal_per_feed"`
	Notes             *string `json:"notes,omitempty"`
	CreatedAt         string  `json:"created_at"`
}

func toBabyResponse(b *model.Baby) babyResponse {
	resp := babyResponse{
		ID:                b.ID,
		Name:              b.Name,
		Sex:               b.Sex,
		DateOfBirth:       b.DateOfBirth.Format("2006-01-02"),
		DefaultCalPerFeed: b.DefaultCalPerFeed,
		Notes:             b.Notes,
		CreatedAt:         b.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if b.DiagnosisDate != nil {
		s := b.DiagnosisDate.Format("2006-01-02")
		resp.DiagnosisDate = &s
	}
	if b.KasaiDate != nil {
		s := b.KasaiDate.Format("2006-01-02")
		resp.KasaiDate = &s
	}
	return resp
}

// extractBabyID extracts the baby ID from the request using Go 1.22+ PathValue.
func extractBabyID(r *http.Request) string {
	return r.PathValue("id")
}

// CreateBabyHandler handles POST /api/babies.
func CreateBabyHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := middleware.UserFromContext(r.Context())
		if user == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var req babyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.Name == "" || req.Sex == "" || req.DateOfBirth == "" {
			http.Error(w, "name, sex, and date_of_birth are required", http.StatusBadRequest)
			return
		}

		if req.Sex != "male" && req.Sex != "female" {
			http.Error(w, "sex must be 'male' or 'female'", http.StatusBadRequest)
			return
		}

		if _, err := time.Parse("2006-01-02", req.DateOfBirth); err != nil {
			http.Error(w, "date_of_birth must be in YYYY-MM-DD format", http.StatusBadRequest)
			return
		}
		if req.DiagnosisDate != nil {
			if _, err := time.Parse("2006-01-02", *req.DiagnosisDate); err != nil {
				http.Error(w, "diagnosis_date must be in YYYY-MM-DD format", http.StatusBadRequest)
				return
			}
		}
		if req.KasaiDate != nil {
			if _, err := time.Parse("2006-01-02", *req.KasaiDate); err != nil {
				http.Error(w, "kasai_date must be in YYYY-MM-DD format", http.StatusBadRequest)
				return
			}
		}

		baby, err := store.CreateBaby(db, user.ID, req.Name, req.Sex, req.DateOfBirth,
			req.DiagnosisDate, req.KasaiDate, req.DefaultCalPerFeed, req.Notes)
		if err != nil {
			log.Printf("create baby: %v", err)
			http.Error(w, "failed to create baby", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(toBabyResponse(baby)); err != nil {
			log.Printf("create baby: encode response: %v", err)
		}
	}
}

// ListBabiesHandler handles GET /api/babies.
func ListBabiesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := middleware.UserFromContext(r.Context())
		if user == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		babies, err := store.GetBabiesByUserID(db, user.ID)
		if err != nil {
			log.Printf("list babies: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		resp := make([]babyResponse, 0, len(babies))
		for i := range babies {
			resp = append(resp, toBabyResponse(&babies[i]))
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("list babies: encode response: %v", err)
		}
	}
}

// GetBabyHandler handles GET /api/babies/:id.
func GetBabyHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := middleware.UserFromContext(r.Context())
		if user == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		babyID := extractBabyID(r)
		if babyID == "" {
			http.Error(w, "missing baby ID", http.StatusBadRequest)
			return
		}

		baby, err := store.GetBabyByID(db, babyID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "baby not found", http.StatusNotFound)
				return
			}
			log.Printf("get baby: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		linked, err := store.IsParentOfBaby(db, user.ID, baby.ID)
		if err != nil {
			log.Printf("get baby: check parent: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !linked {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(toBabyResponse(baby)); err != nil {
			log.Printf("get baby: encode response: %v", err)
		}
	}
}

// UpdateBabyHandler handles PUT /api/babies/:id.
func UpdateBabyHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := middleware.UserFromContext(r.Context())
		if user == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		babyID := extractBabyID(r)
		if babyID == "" {
			http.Error(w, "missing baby ID", http.StatusBadRequest)
			return
		}

		// Check baby exists
		_, err := store.GetBabyByID(db, babyID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "baby not found", http.StatusNotFound)
				return
			}
			log.Printf("update baby: get: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		// Check authorization
		linked, err := store.IsParentOfBaby(db, user.ID, babyID)
		if err != nil {
			log.Printf("update baby: check parent: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !linked {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		var req babyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.Name == "" || req.Sex == "" || req.DateOfBirth == "" {
			http.Error(w, "name, sex, and date_of_birth are required", http.StatusBadRequest)
			return
		}

		if req.Sex != "male" && req.Sex != "female" {
			http.Error(w, "sex must be 'male' or 'female'", http.StatusBadRequest)
			return
		}

		if _, err := time.Parse("2006-01-02", req.DateOfBirth); err != nil {
			http.Error(w, "date_of_birth must be in YYYY-MM-DD format", http.StatusBadRequest)
			return
		}
		if req.DiagnosisDate != nil {
			if _, err := time.Parse("2006-01-02", *req.DiagnosisDate); err != nil {
				http.Error(w, "diagnosis_date must be in YYYY-MM-DD format", http.StatusBadRequest)
				return
			}
		}
		if req.KasaiDate != nil {
			if _, err := time.Parse("2006-01-02", *req.KasaiDate); err != nil {
				http.Error(w, "kasai_date must be in YYYY-MM-DD format", http.StatusBadRequest)
				return
			}
		}

		updated, err := store.UpdateBaby(db, babyID, req.Name, req.Sex, req.DateOfBirth,
			req.DiagnosisDate, req.KasaiDate, req.DefaultCalPerFeed, req.Notes)
		if err != nil {
			log.Printf("update baby: %v", err)
			http.Error(w, "failed to update baby", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(toBabyResponse(updated)); err != nil {
			log.Printf("update baby: encode response: %v", err)
		}
	}
}
