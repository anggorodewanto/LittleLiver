package handler

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/storage"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

// bruisingRequest is the JSON request body for creating/updating a bruising observation.
type bruisingRequest struct {
	Timestamp    string   `json:"timestamp"`
	Location     *string  `json:"location"`
	SizeEstimate *string  `json:"size_estimate"`
	SizeCm       *float64 `json:"size_cm,omitempty"`
	Color        *string  `json:"color,omitempty"`
	PhotoKeys    []string `json:"photo_keys,omitempty"`
	Notes        *string  `json:"notes,omitempty"`
}

// validate checks required fields for a bruising request.
func (req *bruisingRequest) validate() (string, bool) {
	if msg, ok := validateTimestamp(req.Timestamp); !ok {
		return msg, false
	}
	if req.Location == nil || *req.Location == "" {
		return "location is required", false
	}
	if req.SizeEstimate == nil {
		return "size_estimate is required", false
	}
	if !model.ValidBruisingSizeEstimate(*req.SizeEstimate) {
		return "invalid size_estimate", false
	}
	if req.SizeCm != nil && *req.SizeCm <= 0 {
		return "size_cm must be greater than 0", false
	}
	return "", true
}

// bruisingResponse is the JSON response for a bruising observation.
type bruisingResponse struct {
	ID           string          `json:"id"`
	BabyID       string          `json:"baby_id"`
	LoggedBy     string          `json:"logged_by"`
	UpdatedBy    *string         `json:"updated_by,omitempty"`
	Timestamp    string          `json:"timestamp"`
	Location     string          `json:"location"`
	SizeEstimate string          `json:"size_estimate"`
	SizeCm       *float64        `json:"size_cm,omitempty"`
	Color        *string         `json:"color,omitempty"`
	Photos       []photoResponse `json:"photos"`
	Notes        *string         `json:"notes,omitempty"`
	CreatedAt    string          `json:"created_at"`
	UpdatedAt    string          `json:"updated_at"`
}

func toBruisingResponse(b *model.BruisingObservation) bruisingResponse {
	return bruisingResponse{
		ID:           b.ID,
		BabyID:       b.BabyID,
		LoggedBy:     b.LoggedBy,
		UpdatedBy:    b.UpdatedBy,
		Timestamp:    b.Timestamp.Format(model.DateTimeFormat),
		Location:     b.Location,
		SizeEstimate: b.SizeEstimate,
		SizeCm:       b.SizeCm,
		Color:        b.Color,
		Photos:       []photoResponse{},
		Notes:        b.Notes,
		CreatedAt:    b.CreatedAt.Format(model.DateTimeFormat),
		UpdatedAt:    b.UpdatedAt.Format(model.DateTimeFormat),
	}
}

func toBruisingResponseWithPhotos(b *model.BruisingObservation, db *sql.DB, objStore storage.ObjectStore, r *http.Request) bruisingResponse {
	resp := toBruisingResponse(b)
	resp.Photos = resolvePhotos(r.Context(), db, objStore, b.PhotoKeys)
	return resp
}

// CreateBruisingHandler handles POST /api/babies/{id}/bruising.
func CreateBruisingHandler(db *sql.DB, objStores ...storage.ObjectStore) http.HandlerFunc {
	var objStore storage.ObjectStore
	if len(objStores) > 0 {
		objStore = objStores[0]
	}

	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		var req bruisingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		var photoKeysStr *string
		if len(req.PhotoKeys) > 0 {
			var errMsg string
			photoKeysStr, errMsg, ok = handlePhotoLinking(db, baby.ID, nil, req.PhotoKeys)
			if !ok {
				http.Error(w, "invalid photo_keys: "+errMsg, http.StatusBadRequest)
				return
			}
		}

		bruising, err := store.CreateBruisingWithPhotos(db, baby.ID, user.ID, req.Timestamp, *req.Location, *req.SizeEstimate, req.SizeCm, req.Color, photoKeysStr, req.Notes)
		if err != nil {
			log.Printf("create bruising: %v", err)
			http.Error(w, "failed to create bruising observation", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, toBruisingResponseWithPhotos(bruising, db, objStore, r))
	}
}

// ListBruisingHandler handles GET /api/babies/{id}/bruising.
func ListBruisingHandler(db *sql.DB, objStores ...storage.ObjectStore) http.HandlerFunc {
	var objStore storage.ObjectStore
	if len(objStores) > 0 {
		objStore = objStores[0]
	}

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

		page, err := store.ListBruisingWithTZ(db, baby.ID, lp.From, lp.To, lp.Cursor, defaultPageSize, lp.Loc)
		if err != nil {
			log.Printf("list bruising: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		convert := func(b *model.BruisingObservation) bruisingResponse {
			return toBruisingResponseWithPhotos(b, db, objStore, r)
		}
		writeJSON(w, http.StatusOK, mapMetricPage(page, convert))
	}
}

// GetBruisingHandler handles GET /api/babies/{id}/bruising/{entryId}.
func GetBruisingHandler(db *sql.DB, objStores ...storage.ObjectStore) http.HandlerFunc {
	var objStore storage.ObjectStore
	if len(objStores) > 0 {
		objStore = objStores[0]
	}

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

		bruising, err := store.GetBruisingByID(db, baby.ID, entryID)
		if err != nil {
			handleStoreError(w, err, "bruising observation not found")
			return
		}

		writeJSON(w, http.StatusOK, toBruisingResponseWithPhotos(bruising, db, objStore, r))
	}
}

// UpdateBruisingHandler handles PUT /api/babies/{id}/bruising/{entryId}.
func UpdateBruisingHandler(db *sql.DB, objStores ...storage.ObjectStore) http.HandlerFunc {
	var objStore storage.ObjectStore
	if len(objStores) > 0 {
		objStore = objStores[0]
	}

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

		var req bruisingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		existing, err := store.GetBruisingByID(db, baby.ID, entryID)
		if err != nil {
			handleStoreError(w, err, "bruising observation not found")
			return
		}

		photoKeysStr, errMsg, ok := handlePhotoLinking(db, baby.ID, existing.PhotoKeys, req.PhotoKeys)
		if !ok {
			http.Error(w, "invalid photo_keys: "+errMsg, http.StatusBadRequest)
			return
		}

		bruising, err := store.UpdateBruisingWithPhotos(db, baby.ID, entryID, user.ID, req.Timestamp, *req.Location, *req.SizeEstimate, req.SizeCm, req.Color, photoKeysStr, req.Notes)
		if err != nil {
			handleStoreError(w, err, "bruising observation not found")
			return
		}

		writeJSON(w, http.StatusOK, toBruisingResponseWithPhotos(bruising, db, objStore, r))
	}
}

// DeleteBruisingHandler handles DELETE /api/babies/{id}/bruising/{entryId}.
func DeleteBruisingHandler(db *sql.DB, objStores ...storage.ObjectStore) http.HandlerFunc {
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

		existing, err := store.GetBruisingByID(db, baby.ID, entryID)
		if err != nil {
			handleStoreError(w, err, "bruising observation not found")
			return
		}

		if existing.PhotoKeys != nil && *existing.PhotoKeys != "" {
			oldKeys := strings.Split(*existing.PhotoKeys, ",")
			if unlinkErr := store.UnlinkPhotos(db, oldKeys); unlinkErr != nil {
				log.Printf("unlink photos on delete: %v", unlinkErr)
			}
		}

		err = store.DeleteBruising(db, baby.ID, entryID)
		if err != nil {
			handleStoreError(w, err, "bruising observation not found")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
