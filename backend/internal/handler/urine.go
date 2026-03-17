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

// urineRequest is the JSON request body for creating/updating a urine entry.
type urineRequest struct {
	Timestamp string  `json:"timestamp"`
	Color     *string `json:"color,omitempty"`
	Notes     *string `json:"notes,omitempty"`
}

// validate checks required fields for a urine request.
func (req *urineRequest) validate() (string, bool) {
	if req.Timestamp == "" {
		return "timestamp is required", false
	}
	if _, err := time.Parse(model.DateTimeFormat, req.Timestamp); err != nil {
		return "timestamp must be in ISO 8601 format (YYYY-MM-DDTHH:MM:SSZ)", false
	}
	if req.Color != nil && !model.ValidUrineColor(*req.Color) {
		return "invalid color", false
	}
	return "", true
}

// urineResponse is the JSON response for a urine entry.
type urineResponse struct {
	ID        string  `json:"id"`
	BabyID    string  `json:"baby_id"`
	LoggedBy  string  `json:"logged_by"`
	UpdatedBy *string `json:"updated_by,omitempty"`
	Timestamp string  `json:"timestamp"`
	Color     *string `json:"color,omitempty"`
	Notes     *string `json:"notes,omitempty"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

func toUrineResponse(u *model.Urine) urineResponse {
	return urineResponse{
		ID:        u.ID,
		BabyID:    u.BabyID,
		LoggedBy:  u.LoggedBy,
		UpdatedBy: u.UpdatedBy,
		Timestamp: u.Timestamp.Format(model.DateTimeFormat),
		Color:     u.Color,
		Notes:     u.Notes,
		CreatedAt: u.CreatedAt.Format(model.DateTimeFormat),
		UpdatedAt: u.UpdatedAt.Format(model.DateTimeFormat),
	}
}

// CreateUrineHandler handles POST /api/babies/{id}/urine.
func CreateUrineHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		var req urineRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		urine, err := store.CreateUrine(db, baby.ID, user.ID, req.Timestamp, req.Color, req.Notes)
		if err != nil {
			log.Printf("create urine: %v", err)
			http.Error(w, "failed to create urine entry", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, toUrineResponse(urine))
	}
}

// ListUrineHandler handles GET /api/babies/{id}/urine.
func ListUrineHandler(db *sql.DB) http.HandlerFunc {
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

		page, err := store.ListUrineWithTZ(db, baby.ID, from, to, cursor, defaultPageSize, loc)
		if err != nil {
			log.Printf("list urine: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		resp := model.MetricPage[urineResponse]{
			Data:       make([]urineResponse, 0, len(page.Data)),
			NextCursor: page.NextCursor,
		}
		for i := range page.Data {
			resp.Data = append(resp.Data, toUrineResponse(&page.Data[i]))
		}

		writeJSON(w, http.StatusOK, resp)
	}
}

// GetUrineHandler handles GET /api/babies/{id}/urine/{entryId}.
func GetUrineHandler(db *sql.DB) http.HandlerFunc {
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

		urine, err := store.GetUrineByID(db, baby.ID, entryID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "urine entry not found", http.StatusNotFound)
				return
			}
			log.Printf("get urine: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, toUrineResponse(urine))
	}
}

// UpdateUrineHandler handles PUT /api/babies/{id}/urine/{entryId}.
func UpdateUrineHandler(db *sql.DB) http.HandlerFunc {
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

		var req urineRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		urine, err := store.UpdateUrine(db, baby.ID, entryID, user.ID, req.Timestamp, req.Color, req.Notes)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "urine entry not found", http.StatusNotFound)
				return
			}
			log.Printf("update urine: %v", err)
			http.Error(w, "failed to update urine entry", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, toUrineResponse(urine))
	}
}

// DeleteUrineHandler handles DELETE /api/babies/{id}/urine/{entryId}.
func DeleteUrineHandler(db *sql.DB) http.HandlerFunc {
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

		err := store.DeleteUrine(db, baby.ID, entryID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "urine entry not found", http.StatusNotFound)
				return
			}
			log.Printf("delete urine: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
