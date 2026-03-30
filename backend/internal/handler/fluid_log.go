package handler

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

// fluidLogRequest is the JSON request body for creating/updating a standalone fluid_log entry.
type fluidLogRequest struct {
	Timestamp string   `json:"timestamp"`
	Direction string   `json:"direction"`
	Method    string   `json:"method"`
	VolumeMl  *float64 `json:"volume_ml,omitempty"`
	Notes     *string  `json:"notes,omitempty"`
}

// validate checks required fields for a fluid_log request.
func (req *fluidLogRequest) validate() (string, bool) {
	if msg, ok := validateTimestamp(req.Timestamp); !ok {
		return msg, false
	}
	if !model.ValidFluidDirection(req.Direction) {
		return "direction must be 'intake' or 'output'", false
	}
	if strings.TrimSpace(req.Method) == "" {
		return "method is required", false
	}
	return "", true
}

// fluidLogResponse is the JSON response for a fluid_log entry.
type fluidLogResponse struct {
	ID         string   `json:"id"`
	BabyID     string   `json:"baby_id"`
	LoggedBy   string   `json:"logged_by"`
	UpdatedBy  *string  `json:"updated_by,omitempty"`
	Timestamp  string   `json:"timestamp"`
	Direction  string   `json:"direction"`
	Method     string   `json:"method"`
	VolumeMl   *float64 `json:"volume_ml,omitempty"`
	SourceType *string  `json:"source_type,omitempty"`
	SourceID   *string  `json:"source_id,omitempty"`
	Notes      *string  `json:"notes,omitempty"`
	CreatedAt  string   `json:"created_at"`
	UpdatedAt  string   `json:"updated_at"`
}

func toFluidLogResponse(fl *model.FluidLog) fluidLogResponse {
	return fluidLogResponse{
		ID:         fl.ID,
		BabyID:     fl.BabyID,
		LoggedBy:   fl.LoggedBy,
		UpdatedBy:  fl.UpdatedBy,
		Timestamp:  fl.Timestamp.Format(model.DateTimeFormat),
		Direction:  fl.Direction,
		Method:     fl.Method,
		VolumeMl:   fl.VolumeMl,
		SourceType: fl.SourceType,
		SourceID:   fl.SourceID,
		Notes:      fl.Notes,
		CreatedAt:  fl.CreatedAt.Format(model.DateTimeFormat),
		UpdatedAt:  fl.UpdatedAt.Format(model.DateTimeFormat),
	}
}

// CreateFluidLogHandler handles POST /api/babies/{id}/fluid-log.
func CreateFluidLogHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		var req fluidLogRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		fl, err := store.CreateFluidLog(db, baby.ID, user.ID, req.Timestamp, req.Direction, req.Method, req.VolumeMl, req.Notes)
		if err != nil {
			log.Printf("create fluid_log: %v", err)
			http.Error(w, "failed to create fluid log entry", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, toFluidLogResponse(fl))
	}
}

// ListFluidLogHandler handles GET /api/babies/{id}/fluid-log.
func ListFluidLogHandler(db *sql.DB) http.HandlerFunc {
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

		page, err := store.ListFluidLogWithTZ(db, baby.ID, lp.From, lp.To, lp.Cursor, defaultPageSize, lp.Loc)
		if err != nil {
			log.Printf("list fluid_log: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, mapMetricPage(page, toFluidLogResponse))
	}
}

// GetFluidLogHandler handles GET /api/babies/{id}/fluid-log/{entryId}.
func GetFluidLogHandler(db *sql.DB) http.HandlerFunc {
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

		fl, err := store.GetFluidLogByID(db, baby.ID, entryID)
		if err != nil {
			handleStoreError(w, err, "fluid log entry not found")
			return
		}

		writeJSON(w, http.StatusOK, toFluidLogResponse(fl))
	}
}

// UpdateFluidLogHandler handles PUT /api/babies/{id}/fluid-log/{entryId}.
// Rejects updates to linked entries (source_type non-null).
func UpdateFluidLogHandler(db *sql.DB) http.HandlerFunc {
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

		var req fluidLogRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		fl, err := store.UpdateFluidLog(db, baby.ID, entryID, user.ID, req.Timestamp, req.Direction, req.Method, req.VolumeMl, req.Notes)
		if err != nil {
			if strings.Contains(err.Error(), "cannot update linked entry") {
				http.Error(w, "cannot update linked entry — edit via source metric", http.StatusBadRequest)
				return
			}
			handleStoreError(w, err, "fluid log entry not found")
			return
		}

		writeJSON(w, http.StatusOK, toFluidLogResponse(fl))
	}
}

// DeleteFluidLogHandler handles DELETE /api/babies/{id}/fluid-log/{entryId}.
// Rejects deletes of linked entries (source_type non-null).
func DeleteFluidLogHandler(db *sql.DB) http.HandlerFunc {
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

		err := store.DeleteFluidLog(db, baby.ID, entryID)
		if err != nil {
			if strings.Contains(err.Error(), "cannot delete linked entry") {
				http.Error(w, "cannot delete linked entry — delete via source metric", http.StatusBadRequest)
				return
			}
			handleStoreError(w, err, "fluid log entry not found")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
