package handler

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

// labResultRequest is the JSON request body for creating/updating a lab result.
type labResultRequest struct {
	Timestamp   string  `json:"timestamp"`
	TestName    string  `json:"test_name"`
	Value       string  `json:"value"`
	Unit        *string `json:"unit,omitempty"`
	NormalRange *string `json:"normal_range,omitempty"`
	Notes       *string `json:"notes,omitempty"`
}

// validate checks required fields for a lab result request.
func (req *labResultRequest) validate() (string, bool) {
	if msg, ok := validateTimestamp(req.Timestamp); !ok {
		return msg, false
	}
	if req.TestName == "" {
		return "test_name is required", false
	}
	if req.Value == "" {
		return "value is required", false
	}
	return "", true
}

// labResultResponse is the JSON response for a lab result.
type labResultResponse struct {
	ID          string  `json:"id"`
	BabyID      string  `json:"baby_id"`
	LoggedBy    string  `json:"logged_by"`
	UpdatedBy   *string `json:"updated_by,omitempty"`
	Timestamp   string  `json:"timestamp"`
	TestName    string  `json:"test_name"`
	Value       string  `json:"value"`
	Unit        *string `json:"unit,omitempty"`
	NormalRange *string `json:"normal_range,omitempty"`
	Notes       *string `json:"notes,omitempty"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

func toLabResultResponse(l *model.LabResult) labResultResponse {
	return labResultResponse{
		ID:          l.ID,
		BabyID:      l.BabyID,
		LoggedBy:    l.LoggedBy,
		UpdatedBy:   l.UpdatedBy,
		Timestamp:   l.Timestamp.Format(model.DateTimeFormat),
		TestName:    l.TestName,
		Value:       l.Value,
		Unit:        l.Unit,
		NormalRange: l.NormalRange,
		Notes:       l.Notes,
		CreatedAt:   l.CreatedAt.Format(model.DateTimeFormat),
		UpdatedAt:   l.UpdatedAt.Format(model.DateTimeFormat),
	}
}

// CreateLabResultHandler handles POST /api/babies/{id}/labs.
func CreateLabResultHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		var req labResultRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		lab, err := store.CreateLabResult(db, baby.ID, user.ID, req.Timestamp, req.TestName, req.Value, req.Unit, req.NormalRange, req.Notes)
		if err != nil {
			log.Printf("create lab result: %v", err)
			http.Error(w, "failed to create lab result", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, toLabResultResponse(lab))
	}
}

// ListLabResultsHandler handles GET /api/babies/{id}/labs.
func ListLabResultsHandler(db *sql.DB) http.HandlerFunc {
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

		page, err := store.ListLabResultsWithTZ(db, baby.ID, lp.From, lp.To, lp.Cursor, defaultPageSize, lp.Loc)
		if err != nil {
			log.Printf("list lab results: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, mapMetricPage(page, toLabResultResponse))
	}
}

// GetLabResultHandler handles GET /api/babies/{id}/labs/{entryId}.
func GetLabResultHandler(db *sql.DB) http.HandlerFunc {
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

		lab, err := store.GetLabResultByID(db, baby.ID, entryID)
		if err != nil {
			handleStoreError(w, err, "lab result not found")
			return
		}

		writeJSON(w, http.StatusOK, toLabResultResponse(lab))
	}
}

// UpdateLabResultHandler handles PUT /api/babies/{id}/labs/{entryId}.
func UpdateLabResultHandler(db *sql.DB) http.HandlerFunc {
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

		var req labResultRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		lab, err := store.UpdateLabResult(db, baby.ID, entryID, user.ID, req.Timestamp, req.TestName, req.Value, req.Unit, req.NormalRange, req.Notes)
		if err != nil {
			handleStoreError(w, err, "lab result not found")
			return
		}

		writeJSON(w, http.StatusOK, toLabResultResponse(lab))
	}
}

// DeleteLabResultHandler handles DELETE /api/babies/{id}/labs/{entryId}.
func DeleteLabResultHandler(db *sql.DB) http.HandlerFunc {
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

		err := store.DeleteLabResult(db, baby.ID, entryID)
		if err != nil {
			handleStoreError(w, err, "lab result not found")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
