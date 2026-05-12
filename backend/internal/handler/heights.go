package handler

import (
	"database/sql"
	"encoding/json"
	"log"
	"math"
	"net/http"

	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

type heightRequest struct {
	Timestamp         string   `json:"timestamp"`
	HeightCm          *float64 `json:"height_cm"`
	MeasurementSource *string  `json:"measurement_source,omitempty"`
	Notes             *string  `json:"notes,omitempty"`
}

func (req *heightRequest) validate() (string, bool) {
	if msg, ok := validateTimestamp(req.Timestamp); !ok {
		return msg, false
	}
	if req.HeightCm == nil {
		return "height_cm is required", false
	}
	if *req.HeightCm <= 0 {
		return "height_cm must be greater than 0", false
	}
	if req.MeasurementSource != nil && !model.ValidMeasurementSource(*req.MeasurementSource) {
		return "invalid measurement_source", false
	}
	return "", true
}

type heightResponse struct {
	ID                string  `json:"id"`
	BabyID            string  `json:"baby_id"`
	LoggedBy          string  `json:"logged_by"`
	UpdatedBy         *string `json:"updated_by,omitempty"`
	Timestamp         string  `json:"timestamp"`
	HeightCm          float64 `json:"height_cm"`
	MeasurementSource *string `json:"measurement_source,omitempty"`
	Notes             *string `json:"notes,omitempty"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}

func toHeightResponse(h *model.Height) heightResponse {
	return heightResponse{
		ID:                h.ID,
		BabyID:            h.BabyID,
		LoggedBy:          h.LoggedBy,
		UpdatedBy:         h.UpdatedBy,
		Timestamp:         h.Timestamp.Format(model.DateTimeFormat),
		HeightCm:          math.Round(h.HeightCm*10) / 10,
		MeasurementSource: h.MeasurementSource,
		Notes:             h.Notes,
		CreatedAt:         h.CreatedAt.Format(model.DateTimeFormat),
		UpdatedAt:         h.UpdatedAt.Format(model.DateTimeFormat),
	}
}

// CreateHeightHandler handles POST /api/babies/{id}/heights.
func CreateHeightHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		var req heightRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		height, err := store.CreateHeight(db, baby.ID, user.ID, req.Timestamp, *req.HeightCm, req.MeasurementSource, req.Notes)
		if err != nil {
			log.Printf("create height: %v", err)
			http.Error(w, "failed to create height", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, toHeightResponse(height))
	}
}

// ListHeightsHandler handles GET /api/babies/{id}/heights.
func ListHeightsHandler(db *sql.DB) http.HandlerFunc {
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

		page, err := store.ListHeightsWithTZ(db, baby.ID, lp.From, lp.To, lp.Cursor, defaultPageSize, lp.Loc)
		if err != nil {
			log.Printf("list heights: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, mapMetricPage(page, toHeightResponse))
	}
}

// GetHeightHandler handles GET /api/babies/{id}/heights/{entryId}.
func GetHeightHandler(db *sql.DB) http.HandlerFunc {
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

		height, err := store.GetHeightByID(db, baby.ID, entryID)
		if err != nil {
			handleStoreError(w, err, "height not found")
			return
		}

		writeJSON(w, http.StatusOK, toHeightResponse(height))
	}
}

// UpdateHeightHandler handles PUT /api/babies/{id}/heights/{entryId}.
func UpdateHeightHandler(db *sql.DB) http.HandlerFunc {
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

		var req heightRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		height, err := store.UpdateHeight(db, baby.ID, entryID, user.ID, req.Timestamp, *req.HeightCm, req.MeasurementSource, req.Notes)
		if err != nil {
			handleStoreError(w, err, "height not found")
			return
		}

		writeJSON(w, http.StatusOK, toHeightResponse(height))
	}
}

// DeleteHeightHandler handles DELETE /api/babies/{id}/heights/{entryId}.
func DeleteHeightHandler(db *sql.DB) http.HandlerFunc {
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

		err := store.DeleteHeight(db, baby.ID, entryID)
		if err != nil {
			handleStoreError(w, err, "height not found")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
