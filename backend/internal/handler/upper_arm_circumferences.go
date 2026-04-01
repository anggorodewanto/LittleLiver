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

// upperArmCircumferenceRequest is the JSON request body for creating/updating an upper arm circumference.
type upperArmCircumferenceRequest struct {
	Timestamp         string   `json:"timestamp"`
	CircumferenceCm   *float64 `json:"circumference_cm"`
	MeasurementSource *string  `json:"measurement_source,omitempty"`
	Notes             *string  `json:"notes,omitempty"`
}

// validate checks required fields for an upper arm circumference request.
func (req *upperArmCircumferenceRequest) validate() (string, bool) {
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

// upperArmCircumferenceResponse is the JSON response for an upper arm circumference.
type upperArmCircumferenceResponse struct {
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

func toUpperArmCircumferenceResponse(u *model.UpperArmCircumference) upperArmCircumferenceResponse {
	return upperArmCircumferenceResponse{
		ID:                u.ID,
		BabyID:            u.BabyID,
		LoggedBy:          u.LoggedBy,
		UpdatedBy:         u.UpdatedBy,
		Timestamp:         u.Timestamp.Format(model.DateTimeFormat),
		CircumferenceCm:   math.Round(u.CircumferenceCm*10) / 10,
		MeasurementSource: u.MeasurementSource,
		Notes:             u.Notes,
		CreatedAt:         u.CreatedAt.Format(model.DateTimeFormat),
		UpdatedAt:         u.UpdatedAt.Format(model.DateTimeFormat),
	}
}

// CreateUpperArmCircumferenceHandler handles POST /api/babies/{id}/upper-arm-circumferences.
func CreateUpperArmCircumferenceHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		var req upperArmCircumferenceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		uac, err := store.CreateUpperArmCircumference(db, baby.ID, user.ID, req.Timestamp, *req.CircumferenceCm, req.MeasurementSource, req.Notes)
		if err != nil {
			log.Printf("create upper arm circumference: %v", err)
			http.Error(w, "failed to create upper arm circumference", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, toUpperArmCircumferenceResponse(uac))
	}
}

// ListUpperArmCircumferencesHandler handles GET /api/babies/{id}/upper-arm-circumferences.
func ListUpperArmCircumferencesHandler(db *sql.DB) http.HandlerFunc {
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

		page, err := store.ListUpperArmCircumferencesWithTZ(db, baby.ID, lp.From, lp.To, lp.Cursor, defaultPageSize, lp.Loc)
		if err != nil {
			log.Printf("list upper arm circumferences: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, mapMetricPage(page, toUpperArmCircumferenceResponse))
	}
}

// GetUpperArmCircumferenceHandler handles GET /api/babies/{id}/upper-arm-circumferences/{entryId}.
func GetUpperArmCircumferenceHandler(db *sql.DB) http.HandlerFunc {
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

		uac, err := store.GetUpperArmCircumferenceByID(db, baby.ID, entryID)
		if err != nil {
			handleStoreError(w, err, "upper arm circumference not found")
			return
		}

		writeJSON(w, http.StatusOK, toUpperArmCircumferenceResponse(uac))
	}
}

// UpdateUpperArmCircumferenceHandler handles PUT /api/babies/{id}/upper-arm-circumferences/{entryId}.
func UpdateUpperArmCircumferenceHandler(db *sql.DB) http.HandlerFunc {
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

		var req upperArmCircumferenceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		uac, err := store.UpdateUpperArmCircumference(db, baby.ID, entryID, user.ID, req.Timestamp, *req.CircumferenceCm, req.MeasurementSource, req.Notes)
		if err != nil {
			handleStoreError(w, err, "upper arm circumference not found")
			return
		}

		writeJSON(w, http.StatusOK, toUpperArmCircumferenceResponse(uac))
	}
}

// DeleteUpperArmCircumferenceHandler handles DELETE /api/babies/{id}/upper-arm-circumferences/{entryId}.
func DeleteUpperArmCircumferenceHandler(db *sql.DB) http.HandlerFunc {
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

		err := store.DeleteUpperArmCircumference(db, baby.ID, entryID)
		if err != nil {
			handleStoreError(w, err, "upper arm circumference not found")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
