package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/middleware"
	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

// optionalInt distinguishes "field omitted" from "field present, value null".
// Field absent → set=false, value=nil. JSON null → set=true, value=nil.
// JSON integer → set=true, value=&n.
type optionalInt struct {
	set   bool
	value *int
}

func (o *optionalInt) UnmarshalJSON(data []byte) error {
	o.set = true
	if string(data) == "null" {
		o.value = nil
		return nil
	}
	var v int
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	o.value = &v
	return nil
}

// babyRequest is the JSON request body for creating/updating a baby.
type babyRequest struct {
	Name                string      `json:"name"`
	Sex                 string      `json:"sex"`
	DateOfBirth         string      `json:"date_of_birth"`
	DiagnosisDate       *string     `json:"diagnosis_date,omitempty"`
	KasaiDate           *string     `json:"kasai_date,omitempty"`
	DefaultCalPerFeed   *float64    `json:"default_cal_per_feed,omitempty"`
	Notes               *string     `json:"notes,omitempty"`
	GestationalAgeWeeks optionalInt `json:"gestational_age_weeks"`
	GestationalAgeDays  optionalInt `json:"gestational_age_days"`
}

// validate checks required fields, sex validity, and date parsing.
// Returns an error message and false if validation fails.
func (req *babyRequest) validate() (string, bool) {
	if req.Name == "" || req.Sex == "" || req.DateOfBirth == "" {
		return "name, sex, and date_of_birth are required", false
	}
	if !model.ValidSex(req.Sex) {
		return "sex must be 'male' or 'female'", false
	}
	if _, err := time.Parse(model.DateFormat, req.DateOfBirth); err != nil {
		return "date_of_birth must be in YYYY-MM-DD format", false
	}
	if req.DiagnosisDate != nil {
		if _, err := time.Parse(model.DateFormat, *req.DiagnosisDate); err != nil {
			return "diagnosis_date must be in YYYY-MM-DD format", false
		}
	}
	if req.KasaiDate != nil {
		if _, err := time.Parse(model.DateFormat, *req.KasaiDate); err != nil {
			return "kasai_date must be in YYYY-MM-DD format", false
		}
	}
	weeksProvided := req.GestationalAgeWeeks.set && req.GestationalAgeWeeks.value != nil
	daysProvided := req.GestationalAgeDays.set && req.GestationalAgeDays.value != nil
	if daysProvided && !weeksProvided {
		return "gestational_age_days requires gestational_age_weeks", false
	}
	if weeksProvided {
		w := *req.GestationalAgeWeeks.value
		if w < 20 || w > 44 {
			return "gestational_age_weeks must be between 20 and 44", false
		}
	}
	if daysProvided {
		d := *req.GestationalAgeDays.value
		if d < 0 || d > 6 {
			return "gestational_age_days must be between 0 and 6", false
		}
	}
	return "", true
}

// resolveGestational returns (weeks, days, apply). apply is true when at least
// one of the fields was present in the request body, in which case the caller
// should persist both values via store.SetBabyGestationalAge. When weeks is
// supplied without days, days defaults to 0.
func (req *babyRequest) resolveGestational() (weeks, days *int, apply bool) {
	if !req.GestationalAgeWeeks.set && !req.GestationalAgeDays.set {
		return nil, nil, false
	}
	weeks = req.GestationalAgeWeeks.value
	days = req.GestationalAgeDays.value
	if weeks != nil && days == nil {
		zero := 0
		days = &zero
	}
	return weeks, days, true
}

// babyResponse is the JSON response for a baby.
type babyResponse struct {
	ID                  string  `json:"id"`
	Name                string  `json:"name"`
	Sex                 string  `json:"sex"`
	DateOfBirth         string  `json:"date_of_birth"`
	DiagnosisDate       *string `json:"diagnosis_date,omitempty"`
	KasaiDate           *string `json:"kasai_date,omitempty"`
	DefaultCalPerFeed   float64 `json:"default_cal_per_feed"`
	Notes               *string `json:"notes,omitempty"`
	GestationalAgeWeeks *int    `json:"gestational_age_weeks,omitempty"`
	GestationalAgeDays  *int    `json:"gestational_age_days,omitempty"`
	CreatedAt           string  `json:"created_at"`
}

func toBabyResponse(b *model.Baby) babyResponse {
	resp := babyResponse{
		ID:                  b.ID,
		Name:                b.Name,
		Sex:                 b.Sex,
		DateOfBirth:         b.DateOfBirth.Format(model.DateFormat),
		DefaultCalPerFeed:   b.DefaultCalPerFeed,
		Notes:               b.Notes,
		GestationalAgeWeeks: b.GestationalAgeWeeks,
		GestationalAgeDays:  b.GestationalAgeDays,
		CreatedAt:           b.CreatedAt.Format(model.DateTimeFormat),
	}
	if b.DiagnosisDate != nil {
		s := b.DiagnosisDate.Format(model.DateFormat)
		resp.DiagnosisDate = &s
	}
	if b.KasaiDate != nil {
		s := b.KasaiDate.Format(model.DateFormat)
		resp.KasaiDate = &s
	}
	return resp
}

// requireUser extracts the authenticated user from context.
// Returns nil and writes a 401 response if not found.
func requireUser(w http.ResponseWriter, r *http.Request) (*model.User, bool) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil, false
	}
	return user, true
}

// requireBabyAccess extracts the baby ID from the path, loads the baby,
// and verifies the user is a parent. Writes appropriate error responses
// and returns nil if any check fails.
func requireBabyAccess(w http.ResponseWriter, r *http.Request, db *sql.DB, userID string) (*model.Baby, bool) {
	babyID := extractBabyID(r)
	if babyID == "" {
		http.Error(w, "missing baby ID", http.StatusBadRequest)
		return nil, false
	}

	baby, err := store.GetBabyByID(db, babyID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "baby not found", http.StatusNotFound)
			return nil, false
		}
		log.Printf("get baby: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return nil, false
	}

	linked, err := store.IsParentOfBaby(db, userID, baby.ID)
	if err != nil {
		log.Printf("check parent: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return nil, false
	}
	if !linked {
		http.Error(w, "forbidden", http.StatusForbidden)
		return nil, false
	}

	return baby, true
}

// extractBabyID extracts the baby ID from the request using Go 1.22+ PathValue.
func extractBabyID(r *http.Request) string {
	return r.PathValue("id")
}

// CreateBabyHandler handles POST /api/babies.
func CreateBabyHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		var req babyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		baby, err := store.CreateBaby(db, user.ID, req.Name, req.Sex, req.DateOfBirth,
			req.DiagnosisDate, req.KasaiDate, req.DefaultCalPerFeed, req.Notes)
		if err != nil {
			log.Printf("create baby: %v", err)
			http.Error(w, "failed to create baby", http.StatusInternalServerError)
			return
		}

		if weeks, days, apply := req.resolveGestational(); apply {
			if err := store.SetBabyGestationalAge(db, baby.ID, weeks, days); err != nil {
				log.Printf("set gestational age on create: %v", err)
				http.Error(w, "failed to create baby", http.StatusInternalServerError)
				return
			}
			refreshed, err := store.GetBabyByID(db, baby.ID)
			if err != nil {
				log.Printf("reload baby after gestational set: %v", err)
				http.Error(w, "failed to create baby", http.StatusInternalServerError)
				return
			}
			baby = refreshed
		}

		writeJSON(w, http.StatusCreated, toBabyResponse(baby))
	}
}

// ListBabiesHandler handles GET /api/babies.
func ListBabiesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		babies, err := store.GetBabiesByUserID(db, user.ID)
		if err != nil {
			log.Printf("list babies: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		babyList := make([]babyResponse, 0, len(babies))
		for i := range babies {
			babyList = append(babyList, toBabyResponse(&babies[i]))
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"babies": babyList,
		})
	}
}

// GetBabyHandler handles GET /api/babies/:id.
func GetBabyHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		writeJSON(w, http.StatusOK, toBabyResponse(baby))
	}
}

// UnlinkSelfHandler handles DELETE /api/babies/:id/parents/me.
// Unlinks the authenticated user from the baby. If the user was the last
// parent, the baby and all associated data are deleted via CASCADE.
// Returns 204 No Content on success.
func UnlinkSelfHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		if err := store.UnlinkParent(db, baby.ID, user.ID); err != nil {
			log.Printf("unlink self: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// updateBabyEnvelope is the response envelope when recalculate_calories=true.
type updateBabyEnvelope struct {
	Baby              babyResponse `json:"baby"`
	RecalculatedCount int64        `json:"recalculated_count"`
}

// UpdateBabyHandler handles PUT /api/babies/:id.
// Supports ?recalculate_calories=true to recalculate all feedings using default_cal_per_feed.
func UpdateBabyHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		var req babyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		recalculate := r.URL.Query().Get("recalculate_calories") == "true"
		if !recalculate {
			updated, err := store.UpdateBaby(db, baby.ID, req.Name, req.Sex, req.DateOfBirth,
				req.DiagnosisDate, req.KasaiDate, req.DefaultCalPerFeed, req.Notes)
			if err != nil {
				log.Printf("update baby: %v", err)
				http.Error(w, "failed to update baby", http.StatusInternalServerError)
				return
			}
			updated, ok := applyGestationalUpdate(w, db, updated, &req)
			if !ok {
				return
			}
			writeJSON(w, http.StatusOK, toBabyResponse(updated))
			return
		}

		updated, count, err := store.UpdateBabyAndRecalculate(db, baby.ID, req.Name, req.Sex, req.DateOfBirth,
			req.DiagnosisDate, req.KasaiDate, req.DefaultCalPerFeed, req.Notes)
		if err != nil {
			log.Printf("update baby and recalculate: %v", err)
			http.Error(w, "failed to update baby", http.StatusInternalServerError)
			return
		}
		updated, ok = applyGestationalUpdate(w, db, updated, &req)
		if !ok {
			return
		}

		writeJSON(w, http.StatusOK, updateBabyEnvelope{
			Baby:              toBabyResponse(updated),
			RecalculatedCount: count,
		})
	}
}

// applyGestationalUpdate persists gestational age fields when present in the
// request and returns the (possibly reloaded) baby. Writes a 500 and returns
// ok=false if any DB call fails.
func applyGestationalUpdate(w http.ResponseWriter, db *sql.DB, baby *model.Baby, req *babyRequest) (*model.Baby, bool) {
	weeks, days, apply := req.resolveGestational()
	if !apply {
		return baby, true
	}
	if err := store.SetBabyGestationalAge(db, baby.ID, weeks, days); err != nil {
		log.Printf("set gestational age on update: %v", err)
		http.Error(w, "failed to update baby", http.StatusInternalServerError)
		return nil, false
	}
	refreshed, err := store.GetBabyByID(db, baby.ID)
	if err != nil {
		log.Printf("reload baby after gestational set: %v", err)
		http.Error(w, "failed to update baby", http.StatusInternalServerError)
		return nil, false
	}
	return refreshed, true
}
