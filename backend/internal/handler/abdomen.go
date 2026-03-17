package handler

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

// abdomenRequest is the JSON request body for creating/updating an abdomen observation.
type abdomenRequest struct {
	Timestamp  string   `json:"timestamp"`
	Firmness   *string  `json:"firmness"`
	Tenderness *bool    `json:"tenderness,omitempty"`
	GirthCm    *float64 `json:"girth_cm,omitempty"`
	Notes      *string  `json:"notes,omitempty"`
}

// validate checks required fields for an abdomen request.
func (req *abdomenRequest) validate() (string, bool) {
	if msg, ok := validateTimestamp(req.Timestamp); !ok {
		return msg, false
	}
	if req.Firmness == nil {
		return "firmness is required", false
	}
	if !model.ValidFirmness(*req.Firmness) {
		return "invalid firmness", false
	}
	if req.GirthCm != nil && *req.GirthCm <= 0 {
		return "girth_cm must be greater than 0", false
	}
	return "", true
}

// abdomenResponse is the JSON response for an abdomen observation.
type abdomenResponse struct {
	ID         string   `json:"id"`
	BabyID     string   `json:"baby_id"`
	LoggedBy   string   `json:"logged_by"`
	UpdatedBy  *string  `json:"updated_by,omitempty"`
	Timestamp  string   `json:"timestamp"`
	Firmness   string   `json:"firmness"`
	Tenderness bool     `json:"tenderness"`
	GirthCm    *float64 `json:"girth_cm,omitempty"`
	PhotoKeys  *string  `json:"photo_keys,omitempty"`
	Notes      *string  `json:"notes,omitempty"`
	CreatedAt  string   `json:"created_at"`
	UpdatedAt  string   `json:"updated_at"`
}

func toAbdomenResponse(a *model.AbdomenObservation) abdomenResponse {
	return abdomenResponse{
		ID:         a.ID,
		BabyID:     a.BabyID,
		LoggedBy:   a.LoggedBy,
		UpdatedBy:  a.UpdatedBy,
		Timestamp:  a.Timestamp.Format(model.DateTimeFormat),
		Firmness:   a.Firmness,
		Tenderness: a.Tenderness,
		GirthCm:    a.GirthCm,
		PhotoKeys:  a.PhotoKeys,
		Notes:      a.Notes,
		CreatedAt:  a.CreatedAt.Format(model.DateTimeFormat),
		UpdatedAt:  a.UpdatedAt.Format(model.DateTimeFormat),
	}
}

// CreateAbdomenHandler handles POST /api/babies/{id}/abdomen.
func CreateAbdomenHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		var req abdomenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		tenderness := false
		if req.Tenderness != nil {
			tenderness = *req.Tenderness
		}

		abdomen, err := store.CreateAbdomen(db, baby.ID, user.ID, req.Timestamp, *req.Firmness, tenderness, req.GirthCm, req.Notes)
		if err != nil {
			log.Printf("create abdomen: %v", err)
			http.Error(w, "failed to create abdomen observation", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, toAbdomenResponse(abdomen))
	}
}

// ListAbdomenHandler handles GET /api/babies/{id}/abdomen.
func ListAbdomenHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		lp := parseListParams(r)

		page, err := store.ListAbdomenWithTZ(db, baby.ID, lp.From, lp.To, lp.Cursor, defaultPageSize, lp.Loc)
		if err != nil {
			log.Printf("list abdomen: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, mapMetricPage(page, toAbdomenResponse))
	}
}

// GetAbdomenHandler handles GET /api/babies/{id}/abdomen/{entryId}.
func GetAbdomenHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		entryID, ok := requireEntryID(w, r)
		if !ok {
			return
		}

		abdomen, err := store.GetAbdomenByID(db, baby.ID, entryID)
		if err != nil {
			handleStoreError(w, err, "abdomen observation not found")
			return
		}

		writeJSON(w, http.StatusOK, toAbdomenResponse(abdomen))
	}
}

// UpdateAbdomenHandler handles PUT /api/babies/{id}/abdomen/{entryId}.
func UpdateAbdomenHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		entryID, ok := requireEntryID(w, r)
		if !ok {
			return
		}

		var req abdomenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		tenderness := false
		if req.Tenderness != nil {
			tenderness = *req.Tenderness
		}

		abdomen, err := store.UpdateAbdomen(db, baby.ID, entryID, user.ID, req.Timestamp, *req.Firmness, tenderness, req.GirthCm, req.Notes)
		if err != nil {
			handleStoreError(w, err, "abdomen observation not found")
			return
		}

		writeJSON(w, http.StatusOK, toAbdomenResponse(abdomen))
	}
}

// DeleteAbdomenHandler handles DELETE /api/babies/{id}/abdomen/{entryId}.
func DeleteAbdomenHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		entryID, ok := requireEntryID(w, r)
		if !ok {
			return
		}

		err := store.DeleteAbdomen(db, baby.ID, entryID)
		if err != nil {
			handleStoreError(w, err, "abdomen observation not found")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
