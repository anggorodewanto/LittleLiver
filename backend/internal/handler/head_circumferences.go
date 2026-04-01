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

// headCircumferenceRequest is the JSON request body for creating/updating a head circumference.
type headCircumferenceRequest struct {
	Timestamp         string   `json:"timestamp"`
	CircumferenceCm   *float64 `json:"circumference_cm"`
	MeasurementSource *string  `json:"measurement_source,omitempty"`
	Notes             *string  `json:"notes,omitempty"`
}

// validate checks required fields for a head circumference request.
func (req *headCircumferenceRequest) validate() (string, bool) {
	if msg, ok := validateTimestamp(req.Timestamp); !ok {
		return msg, false
	}
	if req.CircumferenceCm == nil {
		return "circumference_cm is required", false
	}
	if *req.CircumferenceCm <= 0 {
		return "circumference_cm must be greater than 0", false
	}
	if req.MeasurementSource != nil && !model.ValidMeasurementSource(*req.MeasurementSource) {
		return "invalid measurement_source", false
	}
	return "", true
}

// headCircumferenceResponse is the JSON response for a head circumference.
type headCircumferenceResponse struct {
	ID                string  `json:"id"`
	BabyID            string  `json:"baby_id"`
	LoggedBy          string  `json:"logged_by"`
	UpdatedBy         *string `json:"updated_by,omitempty"`
	Timestamp         string  `json:"timestamp"`
	CircumferenceCm   float64 `json:"circumference_cm"`
	MeasurementSource *string `json:"measurement_source,omitempty"`
	Notes             *string `json:"notes,omitempty"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}

func toHeadCircumferenceResponse(hc *model.HeadCircumference) headCircumferenceResponse {
	return headCircumferenceResponse{
		ID:                hc.ID,
		BabyID:            hc.BabyID,
		LoggedBy:          hc.LoggedBy,
		UpdatedBy:         hc.UpdatedBy,
		Timestamp:         hc.Timestamp.Format(model.DateTimeFormat),
		CircumferenceCm:   math.Round(hc.CircumferenceCm*10) / 10,
		MeasurementSource: hc.MeasurementSource,
		Notes:             hc.Notes,
		CreatedAt:         hc.CreatedAt.Format(model.DateTimeFormat),
		UpdatedAt:         hc.UpdatedAt.Format(model.DateTimeFormat),
	}
}

// CreateHeadCircumferenceHandler handles POST /api/babies/{id}/head-circumferences.
func CreateHeadCircumferenceHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		var req headCircumferenceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		hc, err := store.CreateHeadCircumference(db, baby.ID, user.ID, req.Timestamp, *req.CircumferenceCm, req.MeasurementSource, req.Notes)
		if err != nil {
			log.Printf("create head circumference: %v", err)
			http.Error(w, "failed to create head circumference", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, toHeadCircumferenceResponse(hc))
	}
}

// ListHeadCircumferencesHandler handles GET /api/babies/{id}/head-circumferences.
func ListHeadCircumferencesHandler(db *sql.DB) http.HandlerFunc {
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

		page, err := store.ListHeadCircumferencesWithTZ(db, baby.ID, lp.From, lp.To, lp.Cursor, defaultPageSize, lp.Loc)
		if err != nil {
			log.Printf("list head circumferences: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, mapMetricPage(page, toHeadCircumferenceResponse))
	}
}

// GetHeadCircumferenceHandler handles GET /api/babies/{id}/head-circumferences/{entryId}.
func GetHeadCircumferenceHandler(db *sql.DB) http.HandlerFunc {
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

		hc, err := store.GetHeadCircumferenceByID(db, baby.ID, entryID)
		if err != nil {
			handleStoreError(w, err, "head circumference not found")
			return
		}

		writeJSON(w, http.StatusOK, toHeadCircumferenceResponse(hc))
	}
}

// UpdateHeadCircumferenceHandler handles PUT /api/babies/{id}/head-circumferences/{entryId}.
func UpdateHeadCircumferenceHandler(db *sql.DB) http.HandlerFunc {
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

		var req headCircumferenceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		hc, err := store.UpdateHeadCircumference(db, baby.ID, entryID, user.ID, req.Timestamp, *req.CircumferenceCm, req.MeasurementSource, req.Notes)
		if err != nil {
			handleStoreError(w, err, "head circumference not found")
			return
		}

		writeJSON(w, http.StatusOK, toHeadCircumferenceResponse(hc))
	}
}

// DeleteHeadCircumferenceHandler handles DELETE /api/babies/{id}/head-circumferences/{entryId}.
func DeleteHeadCircumferenceHandler(db *sql.DB) http.HandlerFunc {
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

		err := store.DeleteHeadCircumference(db, baby.ID, entryID)
		if err != nil {
			handleStoreError(w, err, "head circumference not found")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
