package handler

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/immunization"
	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

type immunizationRequest struct {
	VaccineCode      string  `json:"vaccine_code,omitempty"`
	VaccineName      string  `json:"vaccine_name"`
	DoseNumber       *int    `json:"dose_number,omitempty"`
	AdministeredDate string  `json:"administered_date"`
	Provider         *string `json:"provider,omitempty"`
	LotNumber        *string `json:"lot_number,omitempty"`
	Notes            *string `json:"notes,omitempty"`
}

func (req *immunizationRequest) validate() (string, bool) {
	if req.VaccineName == "" {
		return "vaccine_name is required", false
	}
	if req.AdministeredDate == "" {
		return "administered_date is required", false
	}
	if _, err := time.Parse(model.DateFormat, req.AdministeredDate); err != nil {
		return "administered_date must be in YYYY-MM-DD format", false
	}
	if req.DoseNumber != nil && *req.DoseNumber < 1 {
		return "dose_number must be 1 or greater", false
	}
	return "", true
}

type immunizationResponse struct {
	ID               string  `json:"id"`
	BabyID           string  `json:"baby_id"`
	LoggedBy         string  `json:"logged_by"`
	UpdatedBy        *string `json:"updated_by,omitempty"`
	VaccineCode      string  `json:"vaccine_code"`
	VaccineName      string  `json:"vaccine_name"`
	DoseNumber       *int    `json:"dose_number,omitempty"`
	AdministeredDate string  `json:"administered_date"`
	Provider         *string `json:"provider,omitempty"`
	LotNumber        *string `json:"lot_number,omitempty"`
	Notes            *string `json:"notes,omitempty"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

func toImmunizationResponse(m *model.Immunization) immunizationResponse {
	return immunizationResponse{
		ID:               m.ID,
		BabyID:           m.BabyID,
		LoggedBy:         m.LoggedBy,
		UpdatedBy:        m.UpdatedBy,
		VaccineCode:      m.VaccineCode,
		VaccineName:      m.VaccineName,
		DoseNumber:       m.DoseNumber,
		AdministeredDate: m.AdministeredDate,
		Provider:         m.Provider,
		LotNumber:        m.LotNumber,
		Notes:            m.Notes,
		CreatedAt:        m.CreatedAt.Format(model.DateTimeFormat),
		UpdatedAt:        m.UpdatedAt.Format(model.DateTimeFormat),
	}
}

// CreateImmunizationHandler handles POST /api/babies/{id}/immunizations.
func CreateImmunizationHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}
		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		var req immunizationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		rec, err := store.CreateImmunization(db, baby.ID, user.ID, req.VaccineCode, req.VaccineName, req.DoseNumber, req.AdministeredDate, req.Provider, req.LotNumber, req.Notes)
		if err != nil {
			log.Printf("create immunization: %v", err)
			http.Error(w, "failed to create immunization", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, toImmunizationResponse(rec))
	}
}

// ListImmunizationsHandler handles GET /api/babies/{id}/immunizations.
// Records per baby are few, so all are returned (no pagination); the response
// mirrors the metric list envelope with a null cursor for client consistency.
func ListImmunizationsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}
		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		records, err := store.ListImmunizations(db, baby.ID)
		if err != nil {
			log.Printf("list immunizations: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		resps := make([]immunizationResponse, 0, len(records))
		for i := range records {
			resps = append(resps, toImmunizationResponse(&records[i]))
		}

		writeJSON(w, http.StatusOK, map[string]any{"data": resps, "next_cursor": nil})
	}
}

// GetImmunizationHandler handles GET /api/babies/{id}/immunizations/{entryId}.
func GetImmunizationHandler(db *sql.DB) http.HandlerFunc {
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

		rec, err := store.GetImmunizationByID(db, baby.ID, entryID)
		if err != nil {
			handleStoreError(w, err, "immunization not found")
			return
		}

		writeJSON(w, http.StatusOK, toImmunizationResponse(rec))
	}
}

// UpdateImmunizationHandler handles PUT /api/babies/{id}/immunizations/{entryId}.
func UpdateImmunizationHandler(db *sql.DB) http.HandlerFunc {
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

		var req immunizationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		rec, err := store.UpdateImmunization(db, baby.ID, entryID, user.ID, req.VaccineCode, req.VaccineName, req.DoseNumber, req.AdministeredDate, req.Provider, req.LotNumber, req.Notes)
		if err != nil {
			handleStoreError(w, err, "immunization not found")
			return
		}

		writeJSON(w, http.StatusOK, toImmunizationResponse(rec))
	}
}

// DeleteImmunizationHandler handles DELETE /api/babies/{id}/immunizations/{entryId}.
func DeleteImmunizationHandler(db *sql.DB) http.HandlerFunc {
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

		if err := store.DeleteImmunization(db, baby.ID, entryID); err != nil {
			handleStoreError(w, err, "immunization not found")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// ImmunizationScheduleHandler handles GET /api/babies/{id}/immunizations/schedule.
// It returns the IDAI reference schedule overlaid with the baby's completed
// doses, computing due/upcoming status from the baby's date of birth.
func ImmunizationScheduleHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}
		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		loc := parseListParams(r).Loc
		slots, err := store.GetImmunizationSchedule(db, baby, time.Now(), loc)
		if err != nil {
			log.Printf("immunization schedule: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"slots": slots})
	}
}

// ImmunizationReferenceHandler handles GET /api/immunizations/reference.
// It returns the static IDAI schedule for use in the log-form vaccine picker.
func ImmunizationReferenceHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"schedule": immunization.Schedule()})
	}
}
