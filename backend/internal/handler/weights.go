package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"math"
	"net/http"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

// weightRequest is the JSON request body for creating/updating a weight.
type weightRequest struct {
	Timestamp         string  `json:"timestamp"`
	WeightKg          *float64 `json:"weight_kg"`
	MeasurementSource *string `json:"measurement_source,omitempty"`
	Notes             *string `json:"notes,omitempty"`
}

// validate checks required fields for a weight request.
func (req *weightRequest) validate() (string, bool) {
	if req.Timestamp == "" {
		return "timestamp is required", false
	}
	if _, err := time.Parse(model.DateTimeFormat, req.Timestamp); err != nil {
		return "timestamp must be in ISO 8601 format (YYYY-MM-DDTHH:MM:SSZ)", false
	}
	if req.WeightKg == nil {
		return "weight_kg is required", false
	}
	if *req.WeightKg <= 0 {
		return "weight_kg must be greater than 0", false
	}
	if req.MeasurementSource != nil && !model.ValidMeasurementSource(*req.MeasurementSource) {
		return "invalid measurement_source", false
	}
	return "", true
}

// weightResponse is the JSON response for a weight.
type weightResponse struct {
	ID                string  `json:"id"`
	BabyID            string  `json:"baby_id"`
	LoggedBy          string  `json:"logged_by"`
	UpdatedBy         *string `json:"updated_by,omitempty"`
	Timestamp         string  `json:"timestamp"`
	WeightKg          float64 `json:"weight_kg"`
	MeasurementSource *string `json:"measurement_source,omitempty"`
	Notes             *string `json:"notes,omitempty"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}

func toWeightResponse(w *model.Weight) weightResponse {
	return weightResponse{
		ID:                w.ID,
		BabyID:            w.BabyID,
		LoggedBy:          w.LoggedBy,
		UpdatedBy:         w.UpdatedBy,
		Timestamp:         w.Timestamp.Format(model.DateTimeFormat),
		WeightKg:          math.Round(w.WeightKg*100) / 100,
		MeasurementSource: w.MeasurementSource,
		Notes:             w.Notes,
		CreatedAt:         w.CreatedAt.Format(model.DateTimeFormat),
		UpdatedAt:         w.UpdatedAt.Format(model.DateTimeFormat),
	}
}

// CreateWeightHandler handles POST /api/babies/{id}/weights.
func CreateWeightHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		var req weightRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		weight, err := store.CreateWeight(db, baby.ID, user.ID, req.Timestamp, *req.WeightKg, req.MeasurementSource, req.Notes)
		if err != nil {
			log.Printf("create weight: %v", err)
			http.Error(w, "failed to create weight", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, toWeightResponse(weight))
	}
}

// ListWeightsHandler handles GET /api/babies/{id}/weights.
func ListWeightsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		from := optionalQuery(r, "from")
		to := optionalQuery(r, "to")
		cursor := optionalQuery(r, "cursor")

		loc := time.UTC
		if tz := r.Header.Get("X-Timezone"); tz != "" {
			if parsed, err := time.LoadLocation(tz); err == nil {
				loc = parsed
			}
		}

		page, err := store.ListWeightsWithTZ(db, baby.ID, from, to, cursor, defaultPageSize, loc)
		if err != nil {
			log.Printf("list weights: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		resp := model.MetricPage[weightResponse]{
			Data:       make([]weightResponse, 0, len(page.Data)),
			NextCursor: page.NextCursor,
		}
		for i := range page.Data {
			resp.Data = append(resp.Data, toWeightResponse(&page.Data[i]))
		}

		writeJSON(w, http.StatusOK, resp)
	}
}

// GetWeightHandler handles GET /api/babies/{id}/weights/{entryId}.
func GetWeightHandler(db *sql.DB) http.HandlerFunc {
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

		weight, err := store.GetWeightByID(db, baby.ID, entryID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "weight not found", http.StatusNotFound)
				return
			}
			log.Printf("get weight: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, toWeightResponse(weight))
	}
}

// UpdateWeightHandler handles PUT /api/babies/{id}/weights/{entryId}.
func UpdateWeightHandler(db *sql.DB) http.HandlerFunc {
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

		var req weightRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		weight, err := store.UpdateWeight(db, baby.ID, entryID, user.ID, req.Timestamp, *req.WeightKg, req.MeasurementSource, req.Notes)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "weight not found", http.StatusNotFound)
				return
			}
			log.Printf("update weight: %v", err)
			http.Error(w, "failed to update weight", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, toWeightResponse(weight))
	}
}

// DeleteWeightHandler handles DELETE /api/babies/{id}/weights/{entryId}.
func DeleteWeightHandler(db *sql.DB) http.HandlerFunc {
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

		err := store.DeleteWeight(db, baby.ID, entryID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "weight not found", http.StatusNotFound)
				return
			}
			log.Printf("delete weight: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
