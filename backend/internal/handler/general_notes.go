package handler

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

// generalNoteRequest is the JSON request body for creating/updating a general note.
type generalNoteRequest struct {
	Timestamp string  `json:"timestamp"`
	Content   string  `json:"content"`
	PhotoKeys *string `json:"photo_keys,omitempty"`
	Category  *string `json:"category,omitempty"`
}

// validate checks required fields for a general note request.
func (req *generalNoteRequest) validate() (string, bool) {
	if msg, ok := validateTimestamp(req.Timestamp); !ok {
		return msg, false
	}
	if req.Content == "" {
		return "content is required", false
	}
	if req.Category != nil && !model.ValidNoteCategory(*req.Category) {
		return "invalid category: must be one of behavior, sleep, vomiting, irritability, skin, other", false
	}
	return "", true
}

// generalNoteResponse is the JSON response for a general note.
type generalNoteResponse struct {
	ID        string  `json:"id"`
	BabyID    string  `json:"baby_id"`
	LoggedBy  string  `json:"logged_by"`
	UpdatedBy *string `json:"updated_by,omitempty"`
	Timestamp string  `json:"timestamp"`
	Content   string  `json:"content"`
	PhotoKeys *string `json:"photo_keys,omitempty"`
	Category  *string `json:"category,omitempty"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

func toGeneralNoteResponse(n *model.GeneralNote) generalNoteResponse {
	return generalNoteResponse{
		ID:        n.ID,
		BabyID:    n.BabyID,
		LoggedBy:  n.LoggedBy,
		UpdatedBy: n.UpdatedBy,
		Timestamp: n.Timestamp.Format(model.DateTimeFormat),
		Content:   n.Content,
		PhotoKeys: n.PhotoKeys,
		Category:  n.Category,
		CreatedAt: n.CreatedAt.Format(model.DateTimeFormat),
		UpdatedAt: n.UpdatedAt.Format(model.DateTimeFormat),
	}
}

// CreateGeneralNoteHandler handles POST /api/babies/{id}/notes.
func CreateGeneralNoteHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		var req generalNoteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		note, err := store.CreateGeneralNote(db, baby.ID, user.ID, req.Timestamp, req.Content, req.PhotoKeys, req.Category)
		if err != nil {
			log.Printf("create general note: %v", err)
			http.Error(w, "failed to create general note", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, toGeneralNoteResponse(note))
	}
}

// ListGeneralNotesHandler handles GET /api/babies/{id}/notes.
func ListGeneralNotesHandler(db *sql.DB) http.HandlerFunc {
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

		page, err := store.ListGeneralNotesWithTZ(db, baby.ID, lp.From, lp.To, lp.Cursor, defaultPageSize, lp.Loc)
		if err != nil {
			log.Printf("list general notes: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, mapMetricPage(page, toGeneralNoteResponse))
	}
}

// GetGeneralNoteHandler handles GET /api/babies/{id}/notes/{entryId}.
func GetGeneralNoteHandler(db *sql.DB) http.HandlerFunc {
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

		note, err := store.GetGeneralNoteByID(db, baby.ID, entryID)
		if err != nil {
			handleStoreError(w, err, "note not found")
			return
		}

		writeJSON(w, http.StatusOK, toGeneralNoteResponse(note))
	}
}

// UpdateGeneralNoteHandler handles PUT /api/babies/{id}/notes/{entryId}.
func UpdateGeneralNoteHandler(db *sql.DB) http.HandlerFunc {
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

		var req generalNoteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		note, err := store.UpdateGeneralNote(db, baby.ID, entryID, user.ID, req.Timestamp, req.Content, req.PhotoKeys, req.Category)
		if err != nil {
			handleStoreError(w, err, "note not found")
			return
		}

		writeJSON(w, http.StatusOK, toGeneralNoteResponse(note))
	}
}

// DeleteGeneralNoteHandler handles DELETE /api/babies/{id}/notes/{entryId}.
func DeleteGeneralNoteHandler(db *sql.DB) http.HandlerFunc {
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

		err := store.DeleteGeneralNote(db, baby.ID, entryID)
		if err != nil {
			handleStoreError(w, err, "note not found")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
