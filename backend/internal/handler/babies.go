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

const (
	dateFormat     = "2006-01-02"
	dateTimeFormat = "2006-01-02T15:04:05Z"
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

// validate checks required fields, sex validity, and date parsing.
// Returns an error message and false if validation fails.
func (req *babyRequest) validate() (string, bool) {
	if req.Name == "" || req.Sex == "" || req.DateOfBirth == "" {
		return "name, sex, and date_of_birth are required", false
	}
	if !model.ValidSex(req.Sex) {
		return "sex must be 'male' or 'female'", false
	}
	if _, err := time.Parse(dateFormat, req.DateOfBirth); err != nil {
		return "date_of_birth must be in YYYY-MM-DD format", false
	}
	if req.DiagnosisDate != nil {
		if _, err := time.Parse(dateFormat, *req.DiagnosisDate); err != nil {
			return "diagnosis_date must be in YYYY-MM-DD format", false
		}
	}
	if req.KasaiDate != nil {
		if _, err := time.Parse(dateFormat, *req.KasaiDate); err != nil {
			return "kasai_date must be in YYYY-MM-DD format", false
		}
	}
	return "", true
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
		DateOfBirth:       b.DateOfBirth.Format(dateFormat),
		DefaultCalPerFeed: b.DefaultCalPerFeed,
		Notes:             b.Notes,
		CreatedAt:         b.CreatedAt.Format(dateTimeFormat),
	}
	if b.DiagnosisDate != nil {
		s := b.DiagnosisDate.Format(dateFormat)
		resp.DiagnosisDate = &s
	}
	if b.KasaiDate != nil {
		s := b.KasaiDate.Format(dateFormat)
		resp.KasaiDate = &s
	}
	return resp
}

// requireUser extracts the authenticated user from context.
// Returns nil and writes a 401 response if not found.
func requireUser(w http.ResponseWriter, r *http.Request) (*model.User, bool) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil, false
	}
	return user, true
}

// requireBabyAccess extracts the baby ID from the path, loads the baby,
// and verifies the user is a parent. Writes appropriate error responses
// and returns nil if any check fails.
func requireBabyAccess(w http.ResponseWriter, r *http.Request, db *sql.DB, userID string) (*model.Baby, bool) {
	babyID := extractBabyID(r)
	if babyID == "" {
		http.Error(w, "missing baby ID", http.StatusBadRequest)
		return nil, false
	}

	baby, err := store.GetBabyByID(db, babyID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "baby not found", http.StatusNotFound)
			return nil, false
		}
		log.Printf("get baby: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return nil, false
	}

	linked, err := store.IsParentOfBaby(db, userID, baby.ID)
	if err != nil {
		log.Printf("check parent: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return nil, false
	}
	if !linked {
		http.Error(w, "forbidden", http.StatusForbidden)
		return nil, false
	}

	return baby, true
}

// extractBabyID extracts the baby ID from the request using Go 1.22+ PathValue.
func extractBabyID(r *http.Request) string {
	return r.PathValue("id")
}

// CreateBabyHandler handles POST /api/babies.
func CreateBabyHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		var req babyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
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
		user, ok := requireUser(w, r)
		if !ok {
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
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(toBabyResponse(baby)); err != nil {
			log.Printf("get baby: encode response: %v", err)
		}
	}
}

// UnlinkSelfHandler handles DELETE /api/babies/:id/parents/me.
// Unlinks the authenticated user from the baby. If the user was the last
// parent, the baby and all associated data are deleted via CASCADE.
// Returns 204 No Content on success.
func UnlinkSelfHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		_, ok = requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		babyID := extractBabyID(r)
		if err := store.UnlinkParent(db, babyID, user.ID); err != nil {
			log.Printf("unlink self: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// UpdateBabyHandler handles PUT /api/babies/:id.
func UpdateBabyHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		var req babyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		updated, err := store.UpdateBaby(db, baby.ID, req.Name, req.Sex, req.DateOfBirth,
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
