package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/storage"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

// imagingStudyRequest is the JSON request body for creating/updating an imaging study.
type imagingStudyRequest struct {
	StudyDate string   `json:"study_date"`
	StudyType string   `json:"study_type"`
	Notes     *string  `json:"notes,omitempty"`
	PhotoKeys []string `json:"photo_keys"`
}

// validate enforces required fields and date format.
func (req *imagingStudyRequest) validate() (string, bool) {
	if req.StudyDate == "" {
		return "study_date is required", false
	}
	if _, err := time.Parse(model.DateFormat, req.StudyDate); err != nil {
		return "study_date must be YYYY-MM-DD", false
	}
	if req.StudyType == "" {
		return "study_type is required", false
	}
	if len(req.PhotoKeys) == 0 {
		return "photo_keys must contain at least one key", false
	}
	if len(req.PhotoKeys) > store.MaxPhotosPerImagingStudy {
		return fmt.Sprintf("photo_keys exceeds maximum of %d", store.MaxPhotosPerImagingStudy), false
	}
	return "", true
}

// timestampForStudyDate computes the canonical timestamp for an imaging study:
// study_date at 12:00 in the X-Timezone header (UTC fallback).
func timestampForStudyDate(studyDate string, tzHeader string) (string, error) {
	loc := time.UTC
	if tzHeader != "" {
		if parsed, err := time.LoadLocation(tzHeader); err == nil {
			loc = parsed
		}
	}
	t, err := time.ParseInLocation(model.DateFormat, studyDate, loc)
	if err != nil {
		return "", err
	}
	t = time.Date(t.Year(), t.Month(), t.Day(), 12, 0, 0, 0, loc).UTC()
	return t.Format(model.DateTimeFormat), nil
}

// imagingStudyResponse is the JSON response for an imaging study.
type imagingStudyResponse struct {
	ID        string          `json:"id"`
	BabyID    string          `json:"baby_id"`
	LoggedBy  string          `json:"logged_by"`
	UpdatedBy *string         `json:"updated_by,omitempty"`
	Timestamp string          `json:"timestamp"`
	StudyDate string          `json:"study_date"`
	StudyType string          `json:"study_type"`
	Notes     *string         `json:"notes,omitempty"`
	Photos    []photoResponse `json:"photos"`
	CreatedAt string          `json:"created_at"`
	UpdatedAt string          `json:"updated_at"`
}

func toImagingStudyResponse(s *model.ImagingStudy, db *sql.DB, objStore storage.ObjectStore, r *http.Request) imagingStudyResponse {
	resp := imagingStudyResponse{
		ID:        s.ID,
		BabyID:    s.BabyID,
		LoggedBy:  s.LoggedBy,
		UpdatedBy: s.UpdatedBy,
		Timestamp: s.Timestamp.Format(model.DateTimeFormat),
		StudyDate: s.StudyDate,
		StudyType: s.StudyType,
		Notes:     s.Notes,
		Photos:    []photoResponse{},
		CreatedAt: s.CreatedAt.Format(model.DateTimeFormat),
		UpdatedAt: s.UpdatedAt.Format(model.DateTimeFormat),
	}
	if s.PhotoKeys != "" {
		pk := s.PhotoKeys
		resp.Photos = resolvePhotos(r.Context(), db, objStore, &pk)
	}
	return resp
}

// linkImagingPhotos validates + links photo keys with the 10-cap limit.
// Returns the JSON-serialized photo_keys string and ok=true on success.
func linkImagingPhotos(db *sql.DB, babyID string, keys []string) (string, string, bool) {
	keys = dedup(keys)
	if err := store.ValidateAndLinkPhotosWithMax(db, babyID, keys, store.MaxPhotosPerImagingStudy); err != nil {
		return "", err.Error(), false
	}
	b, err := json.Marshal(keys)
	if err != nil {
		log.Printf("marshal photo_keys: %v", err)
		return "", "internal error", false
	}
	return string(b), "", true
}

// updateImagingPhotos diffs old vs new keys, links the new ones (with 10-cap), unlinks removed ones.
// Returns the new photo_keys JSON string.
func updateImagingPhotos(db *sql.DB, babyID, oldPhotoKeys string, newKeys []string) (string, string, bool) {
	newKeys = dedup(newKeys)
	if len(newKeys) == 0 {
		return "", "photo_keys must contain at least one key", false
	}
	if len(newKeys) > store.MaxPhotosPerImagingStudy {
		return "", fmt.Sprintf("photo_keys exceeds maximum of %d", store.MaxPhotosPerImagingStudy), false
	}

	oldKeys := parsePhotoKeysJSON(&oldPhotoKeys)
	oldSet := make(map[string]bool, len(oldKeys))
	for _, k := range oldKeys {
		oldSet[k] = true
	}

	var toLink []string
	newSet := make(map[string]bool, len(newKeys))
	for _, k := range newKeys {
		newSet[k] = true
		if !oldSet[k] {
			toLink = append(toLink, k)
		}
	}
	if len(toLink) > 0 {
		if err := store.ValidateAndLinkPhotosWithMax(db, babyID, toLink, store.MaxPhotosPerImagingStudy); err != nil {
			return "", err.Error(), false
		}
	}

	var toUnlink []string
	for _, k := range oldKeys {
		if !newSet[k] {
			toUnlink = append(toUnlink, k)
		}
	}
	if len(toUnlink) > 0 {
		if err := store.UnlinkPhotos(db, toUnlink); err != nil {
			log.Printf("unlink removed imaging photos: %v", err)
		}
	}

	b, err := json.Marshal(newKeys)
	if err != nil {
		log.Printf("marshal photo_keys: %v", err)
		return "", "internal error", false
	}
	return string(b), "", true
}

// CreateImagingStudyHandler handles POST /api/babies/{id}/imaging-studies.
func CreateImagingStudyHandler(db *sql.DB, objStores ...storage.ObjectStore) http.HandlerFunc {
	objStore := firstObjStore(objStores)
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}
		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		var req imagingStudyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		ts, err := timestampForStudyDate(req.StudyDate, r.Header.Get("X-Timezone"))
		if err != nil {
			http.Error(w, "invalid study_date", http.StatusBadRequest)
			return
		}

		photoKeysJSON, errMsg, ok := linkImagingPhotos(db, baby.ID, req.PhotoKeys)
		if !ok {
			http.Error(w, "invalid photo_keys: "+errMsg, http.StatusBadRequest)
			return
		}

		study, err := store.CreateImagingStudy(db, baby.ID, user.ID, ts, req.StudyDate, req.StudyType, req.Notes, photoKeysJSON)
		if err != nil {
			log.Printf("create imaging study: %v", err)
			http.Error(w, "failed to create imaging study", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, toImagingStudyResponse(study, db, objStore, r))
	}
}

// ListImagingStudiesHandler handles GET /api/babies/{id}/imaging-studies.
func ListImagingStudiesHandler(db *sql.DB, objStores ...storage.ObjectStore) http.HandlerFunc {
	objStore := firstObjStore(objStores)
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
		page, err := store.ListImagingStudiesWithTZ(db, baby.ID, lp.From, lp.To, lp.Cursor, defaultPageSize, lp.Loc)
		if err != nil {
			log.Printf("list imaging studies: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		convert := func(s *model.ImagingStudy) imagingStudyResponse {
			return toImagingStudyResponse(s, db, objStore, r)
		}
		writeJSON(w, http.StatusOK, mapMetricPage(page, convert))
	}
}

// GetImagingStudyHandler handles GET /api/babies/{id}/imaging-studies/{entryId}.
func GetImagingStudyHandler(db *sql.DB, objStores ...storage.ObjectStore) http.HandlerFunc {
	objStore := firstObjStore(objStores)
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

		study, err := store.GetImagingStudyByID(db, baby.ID, entryID)
		if err != nil {
			handleStoreError(w, err, "imaging study not found")
			return
		}

		writeJSON(w, http.StatusOK, toImagingStudyResponse(study, db, objStore, r))
	}
}

// UpdateImagingStudyHandler handles PUT /api/babies/{id}/imaging-studies/{entryId}.
func UpdateImagingStudyHandler(db *sql.DB, objStores ...storage.ObjectStore) http.HandlerFunc {
	objStore := firstObjStore(objStores)
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

		var req imagingStudyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		existing, err := store.GetImagingStudyByID(db, baby.ID, entryID)
		if err != nil {
			handleStoreError(w, err, "imaging study not found")
			return
		}

		ts, err := timestampForStudyDate(req.StudyDate, r.Header.Get("X-Timezone"))
		if err != nil {
			http.Error(w, "invalid study_date", http.StatusBadRequest)
			return
		}

		photoKeysJSON, errMsg, ok := updateImagingPhotos(db, baby.ID, existing.PhotoKeys, req.PhotoKeys)
		if !ok {
			http.Error(w, "invalid photo_keys: "+errMsg, http.StatusBadRequest)
			return
		}

		updated, err := store.UpdateImagingStudy(db, baby.ID, entryID, user.ID, ts, req.StudyDate, req.StudyType, req.Notes, photoKeysJSON)
		if err != nil {
			handleStoreError(w, err, "imaging study not found")
			return
		}

		writeJSON(w, http.StatusOK, toImagingStudyResponse(updated, db, objStore, r))
	}
}

// DeleteImagingStudyHandler handles DELETE /api/babies/{id}/imaging-studies/{entryId}.
func DeleteImagingStudyHandler(db *sql.DB) http.HandlerFunc {
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

		existing, err := store.GetImagingStudyByID(db, baby.ID, entryID)
		if err != nil {
			handleStoreError(w, err, "imaging study not found")
			return
		}

		// Unlink photos so cleanup cron can reap them.
		if existing.PhotoKeys != "" {
			pk := existing.PhotoKeys
			unlinkAllPhotos(db, &pk)
		}

		if err := store.DeleteImagingStudy(db, baby.ID, entryID); err != nil {
			handleStoreError(w, err, "imaging study not found")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
