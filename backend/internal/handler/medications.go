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
	Active        *bool    `json:"active,omitempty"`
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
	Active        bool     `json:"active"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

func toMedicationResponse(m *model.Medication) medicationResponse {
	var scheduleTimes []string
	if m.Schedule != nil && *m.Schedule != "" {
		if err := json.Unmarshal([]byte(*m.Schedule), &scheduleTimes); err != nil {
			log.Printf("unmarshal medication schedule: %v", err)
		}
	}
	if scheduleTimes == nil {
		scheduleTimes = []string{}
	}

	return medicationResponse{
		ID:            m.ID,
		BabyID:        m.BabyID,
		LoggedBy:      m.LoggedBy,
		UpdatedBy:     m.UpdatedBy,
		Name:          m.Name,
		Dose:          m.Dose,
		Frequency:     m.Frequency,
		ScheduleTimes: scheduleTimes,
		Timezone:      m.Timezone,
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

		// Get timezone from X-Timezone header
		var tz *string
		if tzHeader := r.Header.Get("X-Timezone"); tzHeader != "" {
			tz = &tzHeader
		}

		med, err := store.CreateMedication(db, baby.ID, user.ID, req.Name, req.Dose, req.Frequency, schedule, tz)
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

		medID := r.PathValue("medId")
		if medID == "" {
			http.Error(w, "missing medication ID", http.StatusBadRequest)
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

		medID := r.PathValue("medId")
		if medID == "" {
			http.Error(w, "missing medication ID", http.StatusBadRequest)
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

		// Get timezone from X-Timezone header
		var tz *string
		if tzHeader := r.Header.Get("X-Timezone"); tzHeader != "" {
			tz = &tzHeader
		}

		med, err := store.UpdateMedication(db, baby.ID, medID, user.ID, req.Name, req.Dose, req.Frequency, schedule, tz, req.Active)
		if err != nil {
			handleStoreError(w, err, "medication not found")
			return
		}

		writeJSON(w, http.StatusOK, toMedicationResponse(med))
	}
}

// DeleteMedicationHandler handles DELETE /api/babies/{id}/medications/{medId}.
// Medications cannot be deleted, only deactivated. Returns 405 Method Not Allowed.
func DeleteMedicationHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "medications cannot be deleted, use PUT to deactivate", http.StatusMethodNotAllowed)
	}
}
