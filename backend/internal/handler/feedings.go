package handler

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

const defaultPageSize = 50

// feedingRequest is the JSON request body for creating/updating a feeding.
type feedingRequest struct {
	Timestamp   string   `json:"timestamp"`
	FeedType    string   `json:"feed_type"`
	VolumeMl    *float64 `json:"volume_ml,omitempty"`
	CalDensity  *float64 `json:"cal_density,omitempty"`
	DurationMin *int     `json:"duration_min,omitempty"`
	Notes       *string  `json:"notes,omitempty"`
}

// validate checks required fields for a feeding request.
func (req *feedingRequest) validate() (string, bool) {
	if msg, ok := validateTimestamp(req.Timestamp); !ok {
		return msg, false
	}
	if req.FeedType == "" {
		return "feed_type is required", false
	}
	if !model.ValidFeedType(req.FeedType) {
		return "invalid feed_type", false
	}
	// cal_density without volume is invalid
	if req.CalDensity != nil && req.VolumeMl == nil {
		return "cal_density cannot be provided without volume_ml", false
	}
	return "", true
}


// feedingResponse is the JSON response for a feeding.
type feedingResponse struct {
	ID             string   `json:"id"`
	BabyID         string   `json:"baby_id"`
	LoggedBy       string   `json:"logged_by"`
	UpdatedBy      *string  `json:"updated_by,omitempty"`
	Timestamp      string   `json:"timestamp"`
	FeedType       string   `json:"feed_type"`
	VolumeMl       *float64 `json:"volume_ml,omitempty"`
	CalDensity     *float64 `json:"cal_density,omitempty"`
	Calories       *float64 `json:"calories,omitempty"`
	UsedDefaultCal bool     `json:"used_default_cal"`
	DurationMin    *int     `json:"duration_min,omitempty"`
	Notes          *string  `json:"notes,omitempty"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
}

func toFeedingResponse(f *model.Feeding) feedingResponse {
	return feedingResponse{
		ID:             f.ID,
		BabyID:         f.BabyID,
		LoggedBy:       f.LoggedBy,
		UpdatedBy:      f.UpdatedBy,
		Timestamp:      f.Timestamp.Format(model.DateTimeFormat),
		FeedType:       f.FeedType,
		VolumeMl:       f.VolumeMl,
		CalDensity:     f.CalDensity,
		Calories:       f.Calories,
		UsedDefaultCal: f.UsedDefaultCal,
		DurationMin:    f.DurationMin,
		Notes:          f.Notes,
		CreatedAt:      f.CreatedAt.Format(model.DateTimeFormat),
		UpdatedAt:      f.UpdatedAt.Format(model.DateTimeFormat),
	}
}

// CreateFeedingHandler handles POST /api/babies/{id}/feedings.
func CreateFeedingHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		var req feedingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		feeding, err := store.CreateFeeding(db, baby.ID, user.ID, req.Timestamp, req.FeedType, req.VolumeMl, req.CalDensity, req.DurationMin, req.Notes, baby.DefaultCalPerFeed)
		if err != nil {
			log.Printf("create feeding: %v", err)
			http.Error(w, "failed to create feeding", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, toFeedingResponse(feeding))
	}
}

// ListFeedingsHandler handles GET /api/babies/{id}/feedings.
func ListFeedingsHandler(db *sql.DB) http.HandlerFunc {
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

		page, err := store.ListFeedingsWithTZ(db, baby.ID, lp.From, lp.To, lp.Cursor, defaultPageSize, lp.Loc)
		if err != nil {
			log.Printf("list feedings: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, mapMetricPage(page, toFeedingResponse))
	}
}

// GetFeedingHandler handles GET /api/babies/{id}/feedings/{entryId}.
func GetFeedingHandler(db *sql.DB) http.HandlerFunc {
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

		feeding, err := store.GetFeedingByID(db, baby.ID, entryID)
		if err != nil {
			handleStoreError(w, err, "feeding not found")
			return
		}

		writeJSON(w, http.StatusOK, toFeedingResponse(feeding))
	}
}

// UpdateFeedingHandler handles PUT /api/babies/{id}/feedings/{entryId}.
func UpdateFeedingHandler(db *sql.DB) http.HandlerFunc {
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

		var req feedingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		feeding, err := store.UpdateFeeding(db, baby.ID, entryID, user.ID, req.Timestamp, req.FeedType, req.VolumeMl, req.CalDensity, req.DurationMin, req.Notes, baby.DefaultCalPerFeed)
		if err != nil {
			handleStoreError(w, err, "feeding not found")
			return
		}

		writeJSON(w, http.StatusOK, toFeedingResponse(feeding))
	}
}

// DeleteFeedingHandler handles DELETE /api/babies/{id}/feedings/{entryId}.
func DeleteFeedingHandler(db *sql.DB) http.HandlerFunc {
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

		err := store.DeleteFeeding(db, baby.ID, entryID)
		if err != nil {
			handleStoreError(w, err, "feeding not found")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
