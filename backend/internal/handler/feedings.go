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
	if req.Timestamp == "" {
		return "timestamp is required", false
	}
	if req.FeedType == "" {
		return "feed_type is required", false
	}
	if !model.ValidFeedType(req.FeedType) {
		return "invalid feed_type", false
	}
	if _, err := time.Parse(dateTimeFormat, req.Timestamp); err != nil {
		return "timestamp must be in ISO 8601 format (YYYY-MM-DDTHH:MM:SSZ)", false
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
		Timestamp:      f.Timestamp.Format(dateTimeFormat),
		FeedType:       f.FeedType,
		VolumeMl:       f.VolumeMl,
		CalDensity:     f.CalDensity,
		Calories:       f.Calories,
		UsedDefaultCal: f.UsedDefaultCal,
		DurationMin:    f.DurationMin,
		Notes:          f.Notes,
		CreatedAt:      f.CreatedAt.Format(dateTimeFormat),
		UpdatedAt:      f.UpdatedAt.Format(dateTimeFormat),
	}
}

// feedingListResponse is the paginated response envelope.
type feedingListResponse struct {
	Data       []feedingResponse `json:"data"`
	NextCursor *string           `json:"next_cursor"`
}

// CreateFeedingHandler handles POST /api/babies/{id}/feedings.
func CreateFeedingHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		_, ok = requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		babyID := extractBabyID(r)

		var req feedingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if msg, ok := req.validate(); !ok {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		feeding, err := store.CreateFeeding(db, babyID, user.ID, req.Timestamp, req.FeedType, req.VolumeMl, req.CalDensity, req.DurationMin, req.Notes)
		if err != nil {
			log.Printf("create feeding: %v", err)
			http.Error(w, "failed to create feeding", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(toFeedingResponse(feeding)); err != nil {
			log.Printf("create feeding: encode response: %v", err)
		}
	}
}

// ListFeedingsHandler handles GET /api/babies/{id}/feedings.
func ListFeedingsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		_, ok = requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		babyID := extractBabyID(r)

		var from, to, cursor *string
		if v := r.URL.Query().Get("from"); v != "" {
			from = &v
		}
		if v := r.URL.Query().Get("to"); v != "" {
			to = &v
		}
		if v := r.URL.Query().Get("cursor"); v != "" {
			cursor = &v
		}

		// Determine user timezone for date filtering
		loc := time.UTC
		if tz := r.Header.Get("X-Timezone"); tz != "" {
			if parsed, err := time.LoadLocation(tz); err == nil {
				loc = parsed
			}
		}

		page, err := store.ListFeedingsWithTZ(db, babyID, from, to, cursor, defaultPageSize, loc)
		if err != nil {
			log.Printf("list feedings: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		resp := feedingListResponse{
			Data:       make([]feedingResponse, 0, len(page.Data)),
			NextCursor: page.NextCursor,
		}
		for i := range page.Data {
			resp.Data = append(resp.Data, toFeedingResponse(&page.Data[i]))
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("list feedings: encode response: %v", err)
		}
	}
}

// GetFeedingHandler handles GET /api/babies/{id}/feedings/{entryId}.
func GetFeedingHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		_, ok = requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		babyID := extractBabyID(r)
		entryID := r.PathValue("entryId")
		if entryID == "" {
			http.Error(w, "missing entry ID", http.StatusBadRequest)
			return
		}

		feeding, err := store.GetFeedingByID(db, babyID, entryID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "feeding not found", http.StatusNotFound)
				return
			}
			log.Printf("get feeding: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(toFeedingResponse(feeding)); err != nil {
			log.Printf("get feeding: encode response: %v", err)
		}
	}
}

// UpdateFeedingHandler handles PUT /api/babies/{id}/feedings/{entryId}.
func UpdateFeedingHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		_, ok = requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		babyID := extractBabyID(r)
		entryID := r.PathValue("entryId")
		if entryID == "" {
			http.Error(w, "missing entry ID", http.StatusBadRequest)
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

		feeding, err := store.UpdateFeeding(db, babyID, entryID, user.ID, req.Timestamp, req.FeedType, req.VolumeMl, req.CalDensity, req.DurationMin, req.Notes)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "feeding not found", http.StatusNotFound)
				return
			}
			log.Printf("update feeding: %v", err)
			http.Error(w, "failed to update feeding", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(toFeedingResponse(feeding)); err != nil {
			log.Printf("update feeding: encode response: %v", err)
		}
	}
}

// DeleteFeedingHandler handles DELETE /api/babies/{id}/feedings/{entryId}.
func DeleteFeedingHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		_, ok = requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		babyID := extractBabyID(r)
		entryID := r.PathValue("entryId")
		if entryID == "" {
			http.Error(w, "missing entry ID", http.StatusBadRequest)
			return
		}

		err := store.DeleteFeeding(db, babyID, entryID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "feeding not found", http.StatusNotFound)
				return
			}
			log.Printf("delete feeding: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
