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

// skinObservationRequest is the JSON request body for creating/updating a skin observation.
type skinObservationRequest struct {
	Timestamp      string   `json:"timestamp"`
	JaundiceLevel  *string  `json:"jaundice_level,omitempty"`
	ScleralIcterus *bool    `json:"scleral_icterus,omitempty"`
	Rashes         *string  `json:"rashes,omitempty"`
	Bruising       *string  `json:"bruising,omitempty"`
	PhotoKeys      []string `json:"photo_keys,omitempty"`
	Notes          *string  `json:"notes,omitempty"`
}

// validate checks required fields for a skin observation request.
func (req *skinObservationRequest) validate() (string, bool) {
	if msg, ok := validateTimestamp(req.Timestamp); !ok {
		return msg, false
	}
	if req.JaundiceLevel != nil && !model.ValidJaundiceLevel(*req.JaundiceLevel) {
		return "invalid jaundice_level", false
	}
	return "", true
}

// skinObservationResponse is the JSON response for a skin observation.
type skinObservationResponse struct {
	ID             string          `json:"id"`
	BabyID         string          `json:"baby_id"`
	LoggedBy       string          `json:"logged_by"`
	UpdatedBy      *string         `json:"updated_by,omitempty"`
	Timestamp      string          `json:"timestamp"`
	JaundiceLevel  *string         `json:"jaundice_level,omitempty"`
	ScleralIcterus bool            `json:"scleral_icterus"`
	Rashes         *string         `json:"rashes,omitempty"`
	Bruising       *string         `json:"bruising,omitempty"`
	Photos         []photoResponse `json:"photos"`
	Notes          *string         `json:"notes,omitempty"`
	CreatedAt      string          `json:"created_at"`
	UpdatedAt      string          `json:"updated_at"`
}

func toSkinObservationResponse(s *model.SkinObservation) skinObservationResponse {
	return skinObservationResponse{
		ID:             s.ID,
		BabyID:         s.BabyID,
		LoggedBy:       s.LoggedBy,
		UpdatedBy:      s.UpdatedBy,
		Timestamp:      s.Timestamp.Format(model.DateTimeFormat),
		JaundiceLevel:  s.JaundiceLevel,
		ScleralIcterus: s.ScleralIcterus,
		Rashes:         s.Rashes,
		Bruising:       s.Bruising,
		Photos:         []photoResponse{},
		Notes:          s.Notes,
		CreatedAt:      s.CreatedAt.Format(model.DateTimeFormat),
		UpdatedAt:      s.UpdatedAt.Format(model.DateTimeFormat),
	}
}

func toSkinObservationResponseWithPhotos(s *model.SkinObservation, db *sql.DB, objStore storage.ObjectStore, r *http.Request) skinObservationResponse {
	resp := toSkinObservationResponse(s)
	resp.Photos = resolvePhotos(r.Context(), db, objStore, s.PhotoKeys)
	return resp
}

// CreateSkinObservationHandler handles POST /api/babies/{id}/skin.
func CreateSkinObservationHandler(db *sql.DB, objStores ...storage.ObjectStore) http.HandlerFunc {
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

		var req skinObservationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		scleralIcterus := false
		if req.ScleralIcterus != nil {
			scleralIcterus = *req.ScleralIcterus
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

		skin, err := store.CreateSkinObservationWithPhotos(db, baby.ID, user.ID, req.Timestamp, req.JaundiceLevel, scleralIcterus, req.Rashes, req.Bruising, photoKeysStr, req.Notes)
		if err != nil {
			log.Printf("create skin observation: %v", err)
			http.Error(w, "failed to create skin observation", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, toSkinObservationResponseWithPhotos(skin, db, objStore, r))
	}
}

// ListSkinObservationsHandler handles GET /api/babies/{id}/skin.
func ListSkinObservationsHandler(db *sql.DB, objStores ...storage.ObjectStore) http.HandlerFunc {
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

		page, err := store.ListSkinObservationsWithTZ(db, baby.ID, lp.From, lp.To, lp.Cursor, defaultPageSize, lp.Loc)
		if err != nil {
			log.Printf("list skin observations: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		convert := func(s *model.SkinObservation) skinObservationResponse {
			return toSkinObservationResponseWithPhotos(s, db, objStore, r)
		}
		writeJSON(w, http.StatusOK, mapMetricPage(page, convert))
	}
}

// GetSkinObservationHandler handles GET /api/babies/{id}/skin/{entryId}.
func GetSkinObservationHandler(db *sql.DB, objStores ...storage.ObjectStore) http.HandlerFunc {
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

		skin, err := store.GetSkinObservationByID(db, baby.ID, entryID)
		if err != nil {
			handleStoreError(w, err, "skin observation not found")
			return
		}

		writeJSON(w, http.StatusOK, toSkinObservationResponseWithPhotos(skin, db, objStore, r))
	}
}

// UpdateSkinObservationHandler handles PUT /api/babies/{id}/skin/{entryId}.
func UpdateSkinObservationHandler(db *sql.DB, objStores ...storage.ObjectStore) http.HandlerFunc {
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

		var req skinObservationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		scleralIcterus := false
		if req.ScleralIcterus != nil {
			scleralIcterus = *req.ScleralIcterus
		}

		existing, err := store.GetSkinObservationByID(db, baby.ID, entryID)
		if err != nil {
			handleStoreError(w, err, "skin observation not found")
			return
		}

		photoKeysStr, errMsg, ok := handlePhotoLinking(db, baby.ID, existing.PhotoKeys, req.PhotoKeys)
		if !ok {
			http.Error(w, "invalid photo_keys: "+errMsg, http.StatusBadRequest)
			return
		}

		skin, err := store.UpdateSkinObservationWithPhotos(db, baby.ID, entryID, user.ID, req.Timestamp, req.JaundiceLevel, scleralIcterus, req.Rashes, req.Bruising, photoKeysStr, req.Notes)
		if err != nil {
			handleStoreError(w, err, "skin observation not found")
			return
		}

		writeJSON(w, http.StatusOK, toSkinObservationResponseWithPhotos(skin, db, objStore, r))
	}
}

// DeleteSkinObservationHandler handles DELETE /api/babies/{id}/skin/{entryId}.
func DeleteSkinObservationHandler(db *sql.DB, objStores ...storage.ObjectStore) http.HandlerFunc {
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

		existing, err := store.GetSkinObservationByID(db, baby.ID, entryID)
		if err != nil {
			handleStoreError(w, err, "skin observation not found")
			return
		}

		if existing.PhotoKeys != nil && *existing.PhotoKeys != "" {
			oldKeys := strings.Split(*existing.PhotoKeys, ",")
			if unlinkErr := store.UnlinkPhotos(db, oldKeys); unlinkErr != nil {
				log.Printf("unlink photos on delete: %v", unlinkErr)
			}
		}

		err = store.DeleteSkinObservation(db, baby.ID, entryID)
		if err != nil {
			handleStoreError(w, err, "skin observation not found")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
