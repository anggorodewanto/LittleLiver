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

// medContainerRequest is the JSON request body for creating/updating a
// medication stock container.
type medContainerRequest struct {
	Kind                string   `json:"kind"`
	Unit                string   `json:"unit"`
	QuantityInitial     float64  `json:"quantity_initial"`
	QuantityRemaining   *float64 `json:"quantity_remaining,omitempty"`
	OpenedAt            *string  `json:"opened_at,omitempty"`
	MaxDaysAfterOpening *int     `json:"max_days_after_opening,omitempty"`
	ExpirationDate      *string  `json:"expiration_date,omitempty"`
	Depleted            *bool    `json:"depleted,omitempty"`
	Notes               *string  `json:"notes,omitempty"`
}

func (r *medContainerRequest) validateForCreate() (string, bool) {
	if !model.ValidContainerKind(r.Kind) {
		return "invalid kind", false
	}
	if !model.ValidDoseUnit(r.Unit) {
		return "invalid unit", false
	}
	if r.QuantityInitial < 0 {
		return "quantity_initial must be >= 0", false
	}
	if r.OpenedAt != nil {
		if msg, ok := validateTimestamp(*r.OpenedAt); !ok {
			return "invalid opened_at: " + msg, false
		}
	}
	if r.MaxDaysAfterOpening != nil && *r.MaxDaysAfterOpening < 0 {
		return "max_days_after_opening must be >= 0", false
	}
	if r.ExpirationDate != nil {
		if _, err := time.Parse(model.DateFormat, *r.ExpirationDate); err != nil {
			return "expiration_date must be YYYY-MM-DD", false
		}
	}
	return "", true
}

func (r *medContainerRequest) validateForUpdate() (string, bool) {
	if msg, ok := r.validateForCreate(); !ok {
		return msg, false
	}
	if r.QuantityRemaining != nil && *r.QuantityRemaining < 0 {
		return "quantity_remaining must be >= 0", false
	}
	return "", true
}

type medContainerResponse struct {
	ID                  string   `json:"id"`
	MedicationID        string   `json:"medication_id"`
	BabyID              string   `json:"baby_id"`
	Kind                string   `json:"kind"`
	Unit                string   `json:"unit"`
	QuantityInitial     float64  `json:"quantity_initial"`
	QuantityRemaining   float64  `json:"quantity_remaining"`
	OpenedAt            *string  `json:"opened_at,omitempty"`
	MaxDaysAfterOpening *int     `json:"max_days_after_opening,omitempty"`
	ExpirationDate      *string  `json:"expiration_date,omitempty"`
	EffectiveExpiry     *string  `json:"effective_expiry,omitempty"`
	Depleted            bool     `json:"depleted"`
	Notes               *string  `json:"notes,omitempty"`
	CreatedBy           string   `json:"created_by"`
	UpdatedBy           *string  `json:"updated_by,omitempty"`
	CreatedAt           string   `json:"created_at"`
	UpdatedAt           string   `json:"updated_at"`
}

func toContainerResponse(c *model.MedicationContainer) medContainerResponse {
	resp := medContainerResponse{
		ID:                  c.ID,
		MedicationID:        c.MedicationID,
		BabyID:              c.BabyID,
		Kind:                c.Kind,
		Unit:                c.Unit,
		QuantityInitial:     c.QuantityInitial,
		QuantityRemaining:   c.QuantityRemaining,
		MaxDaysAfterOpening: c.MaxDaysAfterOpening,
		ExpirationDate:      c.ExpirationDate,
		Depleted:            c.Depleted,
		Notes:               c.Notes,
		CreatedBy:           c.CreatedBy,
		UpdatedBy:           c.UpdatedBy,
		CreatedAt:           c.CreatedAt.Format(model.DateTimeFormat),
		UpdatedAt:           c.UpdatedAt.Format(model.DateTimeFormat),
	}
	if c.OpenedAt != nil {
		s := c.OpenedAt.Format(model.DateTimeFormat)
		resp.OpenedAt = &s
	}
	if eff, ok := containerEffectiveExpiry(c); ok {
		s := eff.Format(model.DateFormat)
		resp.EffectiveExpiry = &s
	}
	return resp
}

func containerEffectiveExpiry(c *model.MedicationContainer) (time.Time, bool) {
	var candidate time.Time
	have := false

	if c.ExpirationDate != nil {
		t, err := time.Parse(model.DateFormat, *c.ExpirationDate)
		if err == nil {
			candidate = t
			have = true
		}
	}
	if c.OpenedAt != nil && c.MaxDaysAfterOpening != nil {
		postOpen := c.OpenedAt.AddDate(0, 0, *c.MaxDaysAfterOpening)
		if !have || postOpen.Before(candidate) {
			candidate = postOpen
			have = true
		}
	}
	return candidate, have
}

// CreateMedicationContainerHandler handles
// POST /api/babies/{id}/medications/{medId}/containers.
func CreateMedicationContainerHandler(db *sql.DB) http.HandlerFunc {
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
		if _, err := store.GetMedicationByID(db, baby.ID, medID); err != nil {
			handleStoreError(w, err, "medication not found")
			return
		}

		var req medContainerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if msg, ok := req.validateForCreate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		c, err := store.CreateMedicationContainer(db, store.CreateContainerParams{
			MedicationID:        medID,
			BabyID:              baby.ID,
			Kind:                req.Kind,
			Unit:                req.Unit,
			QuantityInitial:     req.QuantityInitial,
			OpenedAt:            req.OpenedAt,
			MaxDaysAfterOpening: req.MaxDaysAfterOpening,
			ExpirationDate:      req.ExpirationDate,
			Notes:               req.Notes,
			CreatedBy:           user.ID,
		})
		if err != nil {
			log.Printf("create medication_container: %v", err)
			http.Error(w, "failed to create container", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, toContainerResponse(c))
	}
}

// ListMedicationContainersHandler handles
// GET /api/babies/{id}/medications/{medId}/containers.
func ListMedicationContainersHandler(db *sql.DB) http.HandlerFunc {
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
		list, err := store.ListMedicationContainers(db, baby.ID, medID)
		if err != nil {
			log.Printf("list medication_containers: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		resp := make([]medContainerResponse, 0, len(list))
		for i := range list {
			resp = append(resp, toContainerResponse(&list[i]))
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

// GetMedicationContainerHandler handles
// GET /api/babies/{id}/medications/{medId}/containers/{containerId}.
func GetMedicationContainerHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}
		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}
		containerID, ok := requirePathParam(w, r, "containerId", "container ID")
		if !ok {
			return
		}
		c, err := store.GetMedicationContainerByID(db, baby.ID, containerID)
		if err != nil {
			handleStoreError(w, err, "container not found")
			return
		}
		writeJSON(w, http.StatusOK, toContainerResponse(c))
	}
}

// UpdateMedicationContainerHandler handles
// PUT /api/babies/{id}/medications/{medId}/containers/{containerId}.
func UpdateMedicationContainerHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}
		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}
		containerID, ok := requirePathParam(w, r, "containerId", "container ID")
		if !ok {
			return
		}

		existing, err := store.GetMedicationContainerByID(db, baby.ID, containerID)
		if err != nil {
			handleStoreError(w, err, "container not found")
			return
		}

		var req medContainerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if msg, ok := req.validateForUpdate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		quantityRemaining := existing.QuantityRemaining
		if req.QuantityRemaining != nil {
			quantityRemaining = *req.QuantityRemaining
		}
		depleted := existing.Depleted
		if req.Depleted != nil {
			depleted = *req.Depleted
		}

		c, err := store.UpdateMedicationContainer(db, baby.ID, containerID, store.UpdateContainerParams{
			UpdatedBy:           user.ID,
			Kind:                req.Kind,
			Unit:                req.Unit,
			QuantityInitial:     req.QuantityInitial,
			QuantityRemaining:   quantityRemaining,
			OpenedAt:            req.OpenedAt,
			MaxDaysAfterOpening: req.MaxDaysAfterOpening,
			ExpirationDate:      req.ExpirationDate,
			Depleted:            depleted,
			Notes:               req.Notes,
		})
		if err != nil {
			handleStoreError(w, err, "container not found")
			return
		}
		writeJSON(w, http.StatusOK, toContainerResponse(c))
	}
}

// DeleteMedicationContainerHandler handles
// DELETE /api/babies/{id}/medications/{medId}/containers/{containerId}.
func DeleteMedicationContainerHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}
		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}
		containerID, ok := requirePathParam(w, r, "containerId", "container ID")
		if !ok {
			return
		}
		if err := store.DeleteMedicationContainer(db, baby.ID, containerID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "container not found", http.StatusNotFound)
				return
			}
			log.Printf("delete medication_container: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// adjustRequest is the JSON request body for container stock adjustments.
type adjustRequest struct {
	Delta  float64 `json:"delta"`
	Reason *string `json:"reason,omitempty"`
}

type stockAdjustmentResponse struct {
	ID          string  `json:"id"`
	ContainerID string  `json:"container_id"`
	Delta       float64 `json:"delta"`
	Reason      *string `json:"reason,omitempty"`
	AdjustedBy  string  `json:"adjusted_by"`
	AdjustedAt  string  `json:"adjusted_at"`
	CreatedAt   string  `json:"created_at"`
}

func toStockAdjustmentResponse(a *model.MedicationStockAdjustment) stockAdjustmentResponse {
	return stockAdjustmentResponse{
		ID:          a.ID,
		ContainerID: a.ContainerID,
		Delta:       a.Delta,
		Reason:      a.Reason,
		AdjustedBy:  a.AdjustedBy,
		AdjustedAt:  a.AdjustedAt.Format(model.DateTimeFormat),
		CreatedAt:   a.CreatedAt.Format(model.DateTimeFormat),
	}
}

// AdjustMedicationContainerHandler handles
// POST /api/babies/{id}/medications/{medId}/containers/{containerId}/adjust.
func AdjustMedicationContainerHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}
		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}
		containerID, ok := requirePathParam(w, r, "containerId", "container ID")
		if !ok {
			return
		}

		var req adjustRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Delta == 0 {
			http.Error(w, "delta cannot be zero", http.StatusBadRequest)
			return
		}

		adj, err := store.AdjustMedicationContainer(db, baby.ID, containerID, store.AdjustContainerParams{
			AdjustedBy: user.ID,
			Delta:      req.Delta,
			Reason:     req.Reason,
		})
		if err != nil {
			handleStoreError(w, err, "container not found")
			return
		}
		writeJSON(w, http.StatusCreated, toStockAdjustmentResponse(adj))
	}
}
