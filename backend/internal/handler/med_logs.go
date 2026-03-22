package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

// medLogRequest is the JSON request body for creating/updating a med-log.
type medLogRequest struct {
	MedicationID  string  `json:"medication_id"`
	ScheduledTime *string `json:"scheduled_time,omitempty"`
	GivenAt       *string `json:"given_at,omitempty"`
	Skipped       bool    `json:"skipped"`
	SkipReason    *string `json:"skip_reason,omitempty"`
	Notes         *string `json:"notes,omitempty"`
}

// validate checks that the request is valid for create.
// Note: given_at validation is lenient on create since the server overrides it with NOW().
func (req *medLogRequest) validate() (string, bool) {
	if req.MedicationID == "" {
		return "medication_id is required", false
	}
	// Mutual exclusivity: skipped and given_at
	if req.Skipped && req.GivenAt != nil {
		return "given_at must be null when skipped is true", false
	}
	// Validate scheduled_time format if provided
	if req.ScheduledTime != nil {
		if msg, ok := validateTimestamp(*req.ScheduledTime); !ok {
			return "invalid scheduled_time: " + msg, false
		}
	}
	return "", true
}

// medLogUpdateRequest is the JSON request body for updating a med-log.
// medication_id is not updatable. logged_by is immutable.
type medLogUpdateRequest struct {
	ScheduledTime *string `json:"scheduled_time,omitempty"`
	GivenAt       *string `json:"given_at,omitempty"`
	Skipped       bool    `json:"skipped"`
	SkipReason    *string `json:"skip_reason,omitempty"`
	Notes         *string `json:"notes,omitempty"`
}

// validate checks mutual exclusivity for update.
func (req *medLogUpdateRequest) validate() (string, bool) {
	if req.Skipped && req.GivenAt != nil {
		return "given_at must be null when skipped is true", false
	}
	if !req.Skipped && req.GivenAt == nil {
		return "given_at is required when skipped is false", false
	}
	if req.GivenAt != nil {
		if msg, ok := validateTimestamp(*req.GivenAt); !ok {
			return "invalid given_at: " + msg, false
		}
	}
	if req.ScheduledTime != nil {
		if msg, ok := validateTimestamp(*req.ScheduledTime); !ok {
			return "invalid scheduled_time: " + msg, false
		}
	}
	return "", true
}

// medLogResponse is the JSON response for a med-log.
type medLogResponse struct {
	ID            string  `json:"id"`
	MedicationID  string  `json:"medication_id"`
	BabyID        string  `json:"baby_id"`
	LoggedBy      string  `json:"logged_by"`
	UpdatedBy     *string `json:"updated_by,omitempty"`
	ScheduledTime *string `json:"scheduled_time,omitempty"`
	GivenAt       *string `json:"given_at,omitempty"`
	Skipped       bool    `json:"skipped"`
	SkipReason    *string `json:"skip_reason,omitempty"`
	Notes         *string `json:"notes,omitempty"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

func toMedLogResponse(m *model.MedLog) medLogResponse {
	resp := medLogResponse{
		ID:           m.ID,
		MedicationID: m.MedicationID,
		BabyID:       m.BabyID,
		LoggedBy:     m.LoggedBy,
		UpdatedBy:    m.UpdatedBy,
		Skipped:      m.Skipped,
		SkipReason:   m.SkipReason,
		Notes:        m.Notes,
		CreatedAt:    m.CreatedAt.Format(model.DateTimeFormat),
		UpdatedAt:    m.UpdatedAt.Format(model.DateTimeFormat),
	}
	if m.ScheduledTime != nil {
		s := m.ScheduledTime.Format(model.DateTimeFormat)
		resp.ScheduledTime = &s
	}
	if m.GivenAt != nil {
		s := m.GivenAt.Format(model.DateTimeFormat)
		resp.GivenAt = &s
	}
	return resp
}

// CreateMedLogHandler handles POST /api/babies/{id}/med-logs.
func CreateMedLogHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		var req medLogRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		// Per spec §3.9: when logging as "given", server sets given_at to NOW()
		if !req.Skipped {
			now := time.Now().UTC().Format(model.DateTimeFormat)
			req.GivenAt = &now
		}

		// Validate that the medication's baby_id matches
		medBabyID, err := store.GetMedicationBabyID(db, req.MedicationID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "medication not found", http.StatusBadRequest)
				return
			}
			log.Printf("get medication baby_id: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if medBabyID != baby.ID {
			http.Error(w, "medication does not belong to this baby", http.StatusBadRequest)
			return
		}

		ml, err := store.CreateMedLog(db, baby.ID, req.MedicationID, user.ID, req.ScheduledTime, req.GivenAt, req.Skipped, req.SkipReason, req.Notes)
		if err != nil {
			log.Printf("create med_log: %v", err)
			http.Error(w, "failed to create med-log", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, toMedLogResponse(ml))
	}
}

// ListMedLogsHandler handles GET /api/babies/{id}/med-logs.
func ListMedLogsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		medicationID := optionalQuery(r, "medication_id")
		lp := parseListParams(r)

		page, err := store.ListMedLogs(db, baby.ID, medicationID, lp.From, lp.To, lp.Cursor, defaultPageSize, lp.Loc)
		if err != nil {
			log.Printf("list med_logs: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, mapMetricPage(page, toMedLogResponse))
	}
}

// GetMedLogHandler handles GET /api/babies/{id}/med-logs/{entryId}.
func GetMedLogHandler(db *sql.DB) http.HandlerFunc {
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

		ml, err := store.GetMedLogByID(db, baby.ID, entryID)
		if err != nil {
			handleStoreError(w, err, "med-log not found")
			return
		}

		writeJSON(w, http.StatusOK, toMedLogResponse(ml))
	}
}

// UpdateMedLogHandler handles PUT /api/babies/{id}/med-logs/{entryId}.
func UpdateMedLogHandler(db *sql.DB) http.HandlerFunc {
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

		var req medLogUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		ml, err := store.UpdateMedLog(db, baby.ID, entryID, user.ID, req.ScheduledTime, req.GivenAt, req.Skipped, req.SkipReason, req.Notes)
		if err != nil {
			handleStoreError(w, err, "med-log not found")
			return
		}

		writeJSON(w, http.StatusOK, toMedLogResponse(ml))
	}
}

// DeleteMedLogHandler handles DELETE /api/babies/{id}/med-logs/{entryId}.
func DeleteMedLogHandler(db *sql.DB) http.HandlerFunc {
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

		err := store.DeleteMedLog(db, baby.ID, entryID)
		if err != nil {
			handleStoreError(w, err, "med-log not found")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
