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

// temperatureRequest is the JSON request body for creating/updating a temperature.
type temperatureRequest struct {
	Timestamp string   `json:"timestamp"`
	Value     *float64 `json:"value"`
	Method    *string  `json:"method"`
	Notes     *string  `json:"notes,omitempty"`
}

// validate checks required fields for a temperature request.
func (req *temperatureRequest) validate() (string, bool) {
	if req.Timestamp == "" {
		return "timestamp is required", false
	}
	if _, err := time.Parse(model.DateTimeFormat, req.Timestamp); err != nil {
		return "timestamp must be in ISO 8601 format (YYYY-MM-DDTHH:MM:SSZ)", false
	}
	if req.Value == nil {
		return "value is required", false
	}
	if *req.Value < 30.0 || *req.Value > 45.0 {
		return "value must be between 30.0 and 45.0", false
	}
	if req.Method == nil {
		return "method is required", false
	}
	if !model.ValidTemperatureMethod(*req.Method) {
		return "invalid method", false
	}
	return "", true
}

// temperatureResponse is the JSON response for a temperature.
type temperatureResponse struct {
	ID        string  `json:"id"`
	BabyID    string  `json:"baby_id"`
	LoggedBy  string  `json:"logged_by"`
	UpdatedBy *string `json:"updated_by,omitempty"`
	Timestamp string  `json:"timestamp"`
	Value     float64 `json:"value"`
	Method    string  `json:"method"`
	Notes     *string `json:"notes,omitempty"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

func toTemperatureResponse(t *model.Temperature) temperatureResponse {
	return temperatureResponse{
		ID:        t.ID,
		BabyID:    t.BabyID,
		LoggedBy:  t.LoggedBy,
		UpdatedBy: t.UpdatedBy,
		Timestamp: t.Timestamp.Format(model.DateTimeFormat),
		Value:     math.Round(t.Value*10) / 10,
		Method:    t.Method,
		Notes:     t.Notes,
		CreatedAt: t.CreatedAt.Format(model.DateTimeFormat),
		UpdatedAt: t.UpdatedAt.Format(model.DateTimeFormat),
	}
}

// CreateTemperatureHandler handles POST /api/babies/{id}/temperatures.
func CreateTemperatureHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		var req temperatureRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		temp, err := store.CreateTemperature(db, baby.ID, user.ID, req.Timestamp, *req.Value, *req.Method, req.Notes)
		if err != nil {
			log.Printf("create temperature: %v", err)
			http.Error(w, "failed to create temperature", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, toTemperatureResponse(temp))
	}
}

// ListTemperaturesHandler handles GET /api/babies/{id}/temperatures.
func ListTemperaturesHandler(db *sql.DB) http.HandlerFunc {
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

		page, err := store.ListTemperaturesWithTZ(db, baby.ID, from, to, cursor, defaultPageSize, loc)
		if err != nil {
			log.Printf("list temperatures: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		resp := model.MetricPage[temperatureResponse]{
			Data:       make([]temperatureResponse, 0, len(page.Data)),
			NextCursor: page.NextCursor,
		}
		for i := range page.Data {
			resp.Data = append(resp.Data, toTemperatureResponse(&page.Data[i]))
		}

		writeJSON(w, http.StatusOK, resp)
	}
}

// GetTemperatureHandler handles GET /api/babies/{id}/temperatures/{entryId}.
func GetTemperatureHandler(db *sql.DB) http.HandlerFunc {
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

		temp, err := store.GetTemperatureByID(db, baby.ID, entryID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "temperature not found", http.StatusNotFound)
				return
			}
			log.Printf("get temperature: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, toTemperatureResponse(temp))
	}
}

// UpdateTemperatureHandler handles PUT /api/babies/{id}/temperatures/{entryId}.
func UpdateTemperatureHandler(db *sql.DB) http.HandlerFunc {
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

		var req temperatureRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		temp, err := store.UpdateTemperature(db, baby.ID, entryID, user.ID, req.Timestamp, *req.Value, *req.Method, req.Notes)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "temperature not found", http.StatusNotFound)
				return
			}
			log.Printf("update temperature: %v", err)
			http.Error(w, "failed to update temperature", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, toTemperatureResponse(temp))
	}
}

// DeleteTemperatureHandler handles DELETE /api/babies/{id}/temperatures/{entryId}.
func DeleteTemperatureHandler(db *sql.DB) http.HandlerFunc {
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

		err := store.DeleteTemperature(db, baby.ID, entryID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "temperature not found", http.StatusNotFound)
				return
			}
			log.Printf("delete temperature: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
