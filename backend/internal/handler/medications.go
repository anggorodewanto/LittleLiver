package handler

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

// medicationRequest is the JSON request body for creating/updating a medication.
type medicationRequest struct {
	Name          string   `json:"name"`
	Dose          string   `json:"dose"`
	Frequency     string   `json:"frequency"`
	ScheduleTimes []string `json:"schedule_times,omitempty"`
	Timezone      *string  `json:"timezone,omitempty"` // IANA timezone; on create defaults to X-Timezone header; mutable via PUT
	Active        *bool    `json:"active,omitempty"`
	IntervalDays  *int     `json:"interval_days,omitempty"`
}

// validate checks required fields for a medication request.
func (req *medicationRequest) validate() (string, bool) {
	if req.Name == "" {
		return "name is required", false
	}
	if req.Dose == "" {
		return "dose is required", false
	}
	if req.Frequency == "" {
		return "frequency is required", false
	}
	if !model.ValidMedFrequency(req.Frequency) {
		return "invalid frequency", false
	}
	if req.Frequency == "every_x_days" {
		if req.IntervalDays == nil || *req.IntervalDays < 1 {
			return "interval_days is required and must be >= 1 for every_x_days frequency", false
		}
	} else if req.IntervalDays != nil {
		return "interval_days is only valid for every_x_days frequency", false
	}
	return "", true
}

// medicationResponse is the JSON response for a medication.
type medicationResponse struct {
	ID            string   `json:"id"`
	BabyID        string   `json:"baby_id"`
	LoggedBy      string   `json:"logged_by"`
	UpdatedBy     *string  `json:"updated_by,omitempty"`
	Name          string   `json:"name"`
	Dose          string   `json:"dose"`
	Frequency     string   `json:"frequency"`
	ScheduleTimes []string `json:"schedule_times"`
	Timezone      *string  `json:"timezone,omitempty"`
	IntervalDays  *int     `json:"interval_days,omitempty"`
	Active        bool     `json:"active"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

func toMedicationResponse(m *model.Medication) medicationResponse {
	return medicationResponse{
		ID:            m.ID,
		BabyID:        m.BabyID,
		LoggedBy:      m.LoggedBy,
		UpdatedBy:     m.UpdatedBy,
		Name:          m.Name,
		Dose:          m.Dose,
		Frequency:     m.Frequency,
		ScheduleTimes: parseScheduleTimes(m.Schedule),
		Timezone:      m.Timezone,
		IntervalDays:  m.IntervalDays,
		Active:        m.Active,
		CreatedAt:     m.CreatedAt.Format(model.DateTimeFormat),
		UpdatedAt:     m.UpdatedAt.Format(model.DateTimeFormat),
	}
}

// scheduleTimesToJSON converts a slice of time strings to a JSON array string.
func scheduleTimesToJSON(times []string) *string {
	if len(times) == 0 {
		return nil
	}
	b, err := json.Marshal(times)
	if err != nil {
		log.Printf("marshal schedule_times: %v", err)
		return nil
	}
	s := string(b)
	return &s
}

// CreateMedicationHandler handles POST /api/babies/{id}/medications.
func CreateMedicationHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		var req medicationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		schedule := scheduleTimesToJSON(req.ScheduleTimes)
		// Timezone: use request body field if provided, else fall back to X-Timezone header
		tz := req.Timezone
		if tz == nil {
			tz = optionalTimezone(r)
		}

		med, err := store.CreateMedication(db, baby.ID, user.ID, req.Name, req.Dose, req.Frequency, schedule, tz, req.IntervalDays)
		if err != nil {
			log.Printf("create medication: %v", err)
			http.Error(w, "failed to create medication", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, toMedicationResponse(med))
	}
}

// ListMedicationsHandler handles GET /api/babies/{id}/medications.
func ListMedicationsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		meds, err := store.ListMedications(db, baby.ID)
		if err != nil {
			log.Printf("list medications: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		resp := make([]medicationResponse, 0, len(meds))
		for i := range meds {
			resp = append(resp, toMedicationResponse(&meds[i]))
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

// GetMedicationHandler handles GET /api/babies/{id}/medications/{medId}.
func GetMedicationHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		medID, ok := requirePathParam(w, r, "medId", "medication ID")
		if !ok {
			return
		}

		med, err := store.GetMedicationByID(db, baby.ID, medID)
		if err != nil {
			handleStoreError(w, err, "medication not found")
			return
		}

		writeJSON(w, http.StatusOK, toMedicationResponse(med))
	}
}

// UpdateMedicationHandler handles PUT /api/babies/{id}/medications/{medId}.
func UpdateMedicationHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		medID, ok := requirePathParam(w, r, "medId", "medication ID")
		if !ok {
			return
		}

		var req medicationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		schedule := scheduleTimesToJSON(req.ScheduleTimes)

		med, err := store.UpdateMedication(db, baby.ID, medID, user.ID, req.Name, req.Dose, req.Frequency, schedule, req.Timezone, req.Active, req.IntervalDays)
		if err != nil {
			handleStoreError(w, err, "medication not found")
			return
		}

		writeJSON(w, http.StatusOK, toMedicationResponse(med))
	}
}

