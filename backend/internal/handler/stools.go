package handler

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

// stoolRequest is the JSON request body for creating/updating a stool.
type stoolRequest struct {
	Timestamp      string  `json:"timestamp"`
	ColorRating    *int    `json:"color_rating"`
	ColorLabel     *string `json:"color_label,omitempty"`
	Consistency    *string `json:"consistency,omitempty"`
	VolumeEstimate *string `json:"volume_estimate,omitempty"`
	Notes          *string `json:"notes,omitempty"`
}

// validate checks required fields for a stool request.
func (req *stoolRequest) validate() (string, bool) {
	if msg, ok := validateTimestamp(req.Timestamp); !ok {
		return msg, false
	}
	if req.ColorRating == nil {
		return "color_rating is required", false
	}
	if *req.ColorRating < 1 || *req.ColorRating > 7 {
		return "color_rating must be between 1 and 7", false
	}
	if req.ColorLabel != nil && !model.ValidStoolColorLabel(*req.ColorLabel) {
		return "invalid color_label", false
	}
	if req.Consistency != nil && !model.ValidStoolConsistency(*req.Consistency) {
		return "invalid consistency", false
	}
	if req.VolumeEstimate != nil && !model.ValidStoolVolume(*req.VolumeEstimate) {
		return "invalid volume_estimate", false
	}
	return "", true
}

// stoolResponse is the JSON response for a stool.
type stoolResponse struct {
	ID             string  `json:"id"`
	BabyID         string  `json:"baby_id"`
	LoggedBy       string  `json:"logged_by"`
	UpdatedBy      *string `json:"updated_by,omitempty"`
	Timestamp      string  `json:"timestamp"`
	ColorRating    int     `json:"color_rating"`
	ColorLabel     *string `json:"color_label,omitempty"`
	Consistency    *string `json:"consistency,omitempty"`
	VolumeEstimate *string `json:"volume_estimate,omitempty"`
	PhotoKeys      *string `json:"photo_keys,omitempty"`
	Notes          *string `json:"notes,omitempty"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

func toStoolResponse(s *model.Stool) stoolResponse {
	return stoolResponse{
		ID:             s.ID,
		BabyID:         s.BabyID,
		LoggedBy:       s.LoggedBy,
		UpdatedBy:      s.UpdatedBy,
		Timestamp:      s.Timestamp.Format(model.DateTimeFormat),
		ColorRating:    s.ColorRating,
		ColorLabel:     s.ColorLabel,
		Consistency:    s.Consistency,
		VolumeEstimate: s.VolumeEstimate,
		PhotoKeys:      s.PhotoKeys,
		Notes:          s.Notes,
		CreatedAt:      s.CreatedAt.Format(model.DateTimeFormat),
		UpdatedAt:      s.UpdatedAt.Format(model.DateTimeFormat),
	}
}

// CreateStoolHandler handles POST /api/babies/{id}/stools.
func CreateStoolHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		var req stoolRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		stool, err := store.CreateStool(db, baby.ID, user.ID, req.Timestamp, *req.ColorRating, req.ColorLabel, req.Consistency, req.VolumeEstimate, req.Notes)
		if err != nil {
			log.Printf("create stool: %v", err)
			http.Error(w, "failed to create stool", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, toStoolResponse(stool))
	}
}

// ListStoolsHandler handles GET /api/babies/{id}/stools.
func ListStoolsHandler(db *sql.DB) http.HandlerFunc {
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

		page, err := store.ListStoolsWithTZ(db, baby.ID, lp.From, lp.To, lp.Cursor, defaultPageSize, lp.Loc)
		if err != nil {
			log.Printf("list stools: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, mapMetricPage(page, toStoolResponse))
	}
}

// GetStoolHandler handles GET /api/babies/{id}/stools/{entryId}.
func GetStoolHandler(db *sql.DB) http.HandlerFunc {
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

		stool, err := store.GetStoolByID(db, baby.ID, entryID)
		if err != nil {
			handleStoreError(w, err, "stool not found")
			return
		}

		writeJSON(w, http.StatusOK, toStoolResponse(stool))
	}
}

// UpdateStoolHandler handles PUT /api/babies/{id}/stools/{entryId}.
func UpdateStoolHandler(db *sql.DB) http.HandlerFunc {
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

		var req stoolRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		stool, err := store.UpdateStool(db, baby.ID, entryID, user.ID, req.Timestamp, *req.ColorRating, req.ColorLabel, req.Consistency, req.VolumeEstimate, req.Notes)
		if err != nil {
			handleStoreError(w, err, "stool not found")
			return
		}

		writeJSON(w, http.StatusOK, toStoolResponse(stool))
	}
}

// DeleteStoolHandler handles DELETE /api/babies/{id}/stools/{entryId}.
func DeleteStoolHandler(db *sql.DB) http.HandlerFunc {
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

		err := store.DeleteStool(db, baby.ID, entryID)
		if err != nil {
			handleStoreError(w, err, "stool not found")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
