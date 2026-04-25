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

// carePlanRequest is the JSON body for create + update.
type carePlanRequest struct {
	Name     string                 `json:"name"`
	Notes    *string                `json:"notes,omitempty"`
	Timezone *string                `json:"timezone,omitempty"`
	Active   *bool                  `json:"active,omitempty"`
	Phases   []carePlanPhaseRequest `json:"phases,omitempty"`
}

type carePlanPhaseRequest struct {
	Seq       int     `json:"seq"`
	Label     string  `json:"label"`
	StartDate string  `json:"start_date"`
	EndsOn    *string `json:"ends_on,omitempty"`
	Notes     *string `json:"notes,omitempty"`
}

type carePlanResponse struct {
	ID        string                  `json:"id"`
	BabyID    string                  `json:"baby_id"`
	LoggedBy  string                  `json:"logged_by"`
	UpdatedBy *string                 `json:"updated_by,omitempty"`
	Name      string                  `json:"name"`
	Notes     *string                 `json:"notes,omitempty"`
	Timezone  string                  `json:"timezone"`
	Active    bool                    `json:"active"`
	CreatedAt string                  `json:"created_at"`
	UpdatedAt string                  `json:"updated_at"`
	Phases    []carePlanPhaseResponse `json:"phases"`
}

type carePlanPhaseResponse struct {
	ID         string  `json:"id"`
	CarePlanID string  `json:"care_plan_id"`
	Seq        int     `json:"seq"`
	Label      string  `json:"label"`
	StartDate  string  `json:"start_date"`
	EndsOn     *string `json:"ends_on,omitempty"`
	Notes      *string `json:"notes,omitempty"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

func toCarePlanPhaseResponse(p *model.CarePlanPhase) carePlanPhaseResponse {
	return carePlanPhaseResponse{
		ID:         p.ID,
		CarePlanID: p.CarePlanID,
		Seq:        p.Seq,
		Label:      p.Label,
		StartDate:  p.StartDate,
		EndsOn:     p.EndsOn,
		Notes:      p.Notes,
		CreatedAt:  p.CreatedAt.Format(model.DateTimeFormat),
		UpdatedAt:  p.UpdatedAt.Format(model.DateTimeFormat),
	}
}

func toCarePlanResponse(plan *model.CarePlan, phases []model.CarePlanPhase) carePlanResponse {
	ps := make([]carePlanPhaseResponse, 0, len(phases))
	for i := range phases {
		ps = append(ps, toCarePlanPhaseResponse(&phases[i]))
	}
	return carePlanResponse{
		ID:        plan.ID,
		BabyID:    plan.BabyID,
		LoggedBy:  plan.LoggedBy,
		UpdatedBy: plan.UpdatedBy,
		Name:      plan.Name,
		Notes:     plan.Notes,
		Timezone:  plan.Timezone,
		Active:    plan.Active,
		CreatedAt: plan.CreatedAt.Format(model.DateTimeFormat),
		UpdatedAt: plan.UpdatedAt.Format(model.DateTimeFormat),
		Phases:    ps,
	}
}

// validateCarePlanRequest checks fields common to create + update.
// requirePhases=true means at least one phase must be present (used on POST).
func (req *carePlanRequest) validate(requirePhases bool) (string, bool) {
	if req.Name == "" {
		return "name is required", false
	}
	if req.Timezone != nil && *req.Timezone != "" {
		if _, err := time.LoadLocation(*req.Timezone); err != nil {
			return "invalid timezone", false
		}
	}
	if requirePhases && len(req.Phases) == 0 {
		return "at least one phase is required", false
	}
	return validatePhasesRequest(req.Phases)
}

// validatePhasesRequest mirrors store-level validation so the handler returns
// 400 instead of bubbling a generic 500. Empty list is OK here (PUT plan
// without phases just leaves them alone).
func validatePhasesRequest(phases []carePlanPhaseRequest) (string, bool) {
	if len(phases) == 0 {
		return "", true
	}
	var prev time.Time
	for i, ph := range phases {
		if ph.Seq != i+1 {
			return "phase seq values must be contiguous starting at 1", false
		}
		if ph.Label == "" {
			return "phase label is required", false
		}
		d, err := time.Parse(model.DateFormat, ph.StartDate)
		if err != nil {
			return "phase start_date must be YYYY-MM-DD", false
		}
		if i > 0 && !d.After(prev) {
			return "phase start_date must be strictly increasing", false
		}
		if ph.EndsOn != nil && *ph.EndsOn != "" {
			if _, err := time.Parse(model.DateFormat, *ph.EndsOn); err != nil {
				return "phase ends_on must be YYYY-MM-DD", false
			}
		}
		prev = d
	}
	return "", true
}

func phasesFromRequest(in []carePlanPhaseRequest) []model.CarePlanPhase {
	out := make([]model.CarePlanPhase, 0, len(in))
	for _, p := range in {
		ph := model.CarePlanPhase{
			Seq:       p.Seq,
			Label:     p.Label,
			StartDate: p.StartDate,
			Notes:     p.Notes,
		}
		if p.EndsOn != nil && *p.EndsOn != "" {
			ph.EndsOn = p.EndsOn
		}
		out = append(out, ph)
	}
	return out
}

// CreateCarePlanHandler — POST /api/babies/{id}/care-plans
func CreateCarePlanHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}
		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		var req carePlanRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if msg, ok := req.validate(true); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		tz := ""
		if req.Timezone != nil && *req.Timezone != "" {
			tz = *req.Timezone
		} else if hdr := optionalTimezone(r); hdr != nil {
			tz = *hdr
		}
		if tz == "" {
			http.Error(w, "timezone is required (request body or X-Timezone header)", http.StatusBadRequest)
			return
		}

		plan, err := store.CreateCarePlan(db, baby.ID, user.ID, req.Name, req.Notes, tz)
		if err != nil {
			log.Printf("create care plan: %v", err)
			http.Error(w, "failed to create care plan", http.StatusInternalServerError)
			return
		}

		phases, err := store.ReplaceCarePlanPhases(db, plan.ID, phasesFromRequest(req.Phases))
		if err != nil {
			// Roll back the orphan plan so the user can retry cleanly.
			_ = store.DeleteCarePlan(db, baby.ID, plan.ID)
			log.Printf("replace phases on create: %v", err)
			http.Error(w, "failed to save phases", http.StatusBadRequest)
			return
		}

		writeJSON(w, http.StatusCreated, toCarePlanResponse(plan, phases))
	}
}

// ListCarePlansHandler — GET /api/babies/{id}/care-plans
func ListCarePlansHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}
		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		plans, err := store.ListCarePlans(db, baby.ID)
		if err != nil {
			log.Printf("list care plans: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		resp := make([]carePlanResponse, 0, len(plans))
		for i := range plans {
			phases, err := store.ListCarePlanPhases(db, plans[i].ID)
			if err != nil {
				log.Printf("list care plan phases: %v", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			resp = append(resp, toCarePlanResponse(&plans[i], phases))
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

// GetCarePlanHandler — GET /api/babies/{id}/care-plans/{planId}
func GetCarePlanHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}
		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}
		planID, ok := requirePathParam(w, r, "planId", "care plan ID")
		if !ok {
			return
		}

		plan, err := store.GetCarePlanByID(db, baby.ID, planID)
		if err != nil {
			handleStoreError(w, err, "care plan not found")
			return
		}
		phases, err := store.ListCarePlanPhases(db, plan.ID)
		if err != nil {
			log.Printf("list phases: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, toCarePlanResponse(plan, phases))
	}
}

// UpdateCarePlanHandler — PUT /api/babies/{id}/care-plans/{planId}
// If the body includes a non-empty `phases` list, phases are replaced atomically.
func UpdateCarePlanHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}
		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}
		planID, ok := requirePathParam(w, r, "planId", "care plan ID")
		if !ok {
			return
		}

		var req carePlanRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if msg, ok := req.validate(false); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		tz := ""
		if req.Timezone != nil {
			tz = *req.Timezone
		}

		plan, err := store.UpdateCarePlan(db, baby.ID, planID, user.ID, req.Name, req.Notes, tz, req.Active)
		if err != nil {
			handleStoreError(w, err, "care plan not found")
			return
		}

		if len(req.Phases) > 0 {
			if _, err := store.ReplaceCarePlanPhases(db, planID, phasesFromRequest(req.Phases)); err != nil {
				log.Printf("replace phases on update: %v", err)
				http.Error(w, "failed to save phases", http.StatusBadRequest)
				return
			}
		}

		phases, err := store.ListCarePlanPhases(db, plan.ID)
		if err != nil {
			log.Printf("list phases: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, toCarePlanResponse(plan, phases))
	}
}

// DeleteCarePlanHandler — DELETE /api/babies/{id}/care-plans/{planId}
func DeleteCarePlanHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}
		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}
		planID, ok := requirePathParam(w, r, "planId", "care plan ID")
		if !ok {
			return
		}

		if err := store.DeleteCarePlan(db, baby.ID, planID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "care plan not found", http.StatusNotFound)
				return
			}
			log.Printf("delete care plan: %v", err)
			http.Error(w, "failed to delete care plan", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
