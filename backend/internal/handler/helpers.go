package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/storage"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

// writeJSON encodes v as JSON and writes it with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode response: %v", err)
	}
}

// requireEntryID extracts the entry ID from the request path.
// Returns the ID and true, or writes a 400 response and returns false.
func requireEntryID(w http.ResponseWriter, r *http.Request) (string, bool) {
	entryID := r.PathValue("entryId")
	if entryID == "" {
		http.Error(w, "missing entry ID", http.StatusBadRequest)
		return "", false
	}
	return entryID, true
}

// optionalQuery returns a pointer to the query parameter value if present,
// or nil if the parameter is empty or absent.
func optionalQuery(r *http.Request, key string) *string {
	if v := r.URL.Query().Get(key); v != "" {
		return &v
	}
	return nil
}

// validateTimestamp checks that a timestamp string is non-empty and in the expected format.
func validateTimestamp(ts string) (string, bool) {
	if ts == "" {
		return "timestamp is required", false
	}
	if _, err := time.Parse(model.DateTimeFormat, ts); err != nil {
		return "timestamp must be in ISO 8601 format (YYYY-MM-DDTHH:MM:SSZ)", false
	}
	return "", true
}

// listParams holds common parameters for metric list requests.
type listParams struct {
	From   *string
	To     *string
	Cursor *string
	Loc    *time.Location
}

// parseListParams extracts common list parameters from the request.
func parseListParams(r *http.Request) listParams {
	loc := time.UTC
	if tz := r.Header.Get("X-Timezone"); tz != "" {
		if parsed, err := time.LoadLocation(tz); err == nil {
			loc = parsed
		}
	}
	return listParams{
		From:   optionalQuery(r, "from"),
		To:     optionalQuery(r, "to"),
		Cursor: optionalQuery(r, "cursor"),
		Loc:    loc,
	}
}

// handleStoreError writes an appropriate HTTP error response for store-layer errors.
func handleStoreError(w http.ResponseWriter, err error, notFoundMsg string) {
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, notFoundMsg, http.StatusNotFound)
		return
	}
	log.Printf("%s: %v", notFoundMsg, err)
	http.Error(w, "internal error", http.StatusInternalServerError)
}

// mapMetricPage converts a MetricPage of one type to another using the given convert function.
func mapMetricPage[M any, R any](page *model.MetricPage[M], convert func(*M) R) model.MetricPage[R] {
	resp := model.MetricPage[R]{
		Data:       make([]R, 0, len(page.Data)),
		NextCursor: page.NextCursor,
	}
	for i := range page.Data {
		resp.Data = append(resp.Data, convert(&page.Data[i]))
	}
	return resp
}

// photoResponse represents a photo with signed URLs in API responses.
type photoResponse struct {
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnail_url"`
}

// resolvePhotos converts a JSON array photo_keys string into signed URL photo responses.
// Returns an empty slice (not nil) when there are no photos.
func resolvePhotos(ctx context.Context, db *sql.DB, objStore storage.ObjectStore, photoKeys *string) []photoResponse {
	if photoKeys == nil || *photoKeys == "" || objStore == nil {
		return []photoResponse{}
	}

	var keys []string
	if err := json.Unmarshal([]byte(*photoKeys), &keys); err != nil {
		log.Printf("unmarshal photo_keys: %v", err)
		return []photoResponse{}
	}
	photos, err := store.GetPhotoUploadsByR2Keys(db, keys)
	if err != nil {
		log.Printf("resolve photos: %v", err)
		return []photoResponse{}
	}

	result := make([]photoResponse, 0, len(photos))
	for _, p := range photos {
		url, err := objStore.SignedURL(ctx, p.R2Key)
		if err != nil {
			log.Printf("sign URL for %s: %v", p.R2Key, err)
			continue
		}
		thumbURL := ""
		if p.ThumbnailKey != nil {
			thumbURL, err = objStore.SignedURL(ctx, *p.ThumbnailKey)
			if err != nil {
				log.Printf("sign thumbnail URL for %s: %v", *p.ThumbnailKey, err)
			}
		}
		result = append(result, photoResponse{URL: url, ThumbnailURL: thumbURL})
	}
	return result
}

// handlePhotoLinking validates and links photos on create/update.
// oldPhotoKeys is the JSON array previous keys (nil for create).
// newPhotoKeys is the new list of keys from the request (nil means no change).
// Returns the JSON array string to store, or an error message and false.
func handlePhotoLinking(db *sql.DB, babyID string, oldPhotoKeys *string, newPhotoKeys []string) (*string, string, bool) {
	if newPhotoKeys == nil {
		return oldPhotoKeys, "", true
	}

	// Deduplicate newPhotoKeys
	newPhotoKeys = dedup(newPhotoKeys)

	if len(newPhotoKeys) == 0 {
		// Unlink all old photos
		if oldPhotoKeys != nil && *oldPhotoKeys != "" {
			oldKeys := parsePhotoKeysJSON(oldPhotoKeys)
			if err := store.UnlinkPhotos(db, oldKeys); err != nil {
				log.Printf("unlink photos: %v", err)
			}
		}
		return nil, "", true
	}

	// Check total photo count against limit
	if len(newPhotoKeys) > store.MaxPhotosPerMetric {
		return nil, fmt.Sprintf("exceeds maximum of %d photos per entry", store.MaxPhotosPerMetric), false
	}

	// Compute which keys are truly new vs carried over from old
	oldKeys := parsePhotoKeysJSON(oldPhotoKeys)
	oldSet := make(map[string]bool, len(oldKeys))
	for _, k := range oldKeys {
		oldSet[k] = true
	}

	var toLink []string
	newSet := make(map[string]bool, len(newPhotoKeys))
	for _, k := range newPhotoKeys {
		newSet[k] = true
		if !oldSet[k] {
			toLink = append(toLink, k)
		}
	}

	// Validate and link only truly new photos
	if len(toLink) > 0 {
		if err := store.ValidateAndLinkPhotos(db, babyID, toLink); err != nil {
			return nil, err.Error(), false
		}
	}

	// Unlink removed photos
	var toUnlink []string
	for _, k := range oldKeys {
		if !newSet[k] {
			toUnlink = append(toUnlink, k)
		}
	}
	if len(toUnlink) > 0 {
		if err := store.UnlinkPhotos(db, toUnlink); err != nil {
			log.Printf("unlink removed photos: %v", err)
		}
	}

	b, err := json.Marshal(newPhotoKeys)
	if err != nil {
		log.Printf("marshal photo_keys: %v", err)
		return nil, "internal error", false
	}
	joined := string(b)
	return &joined, "", true
}

// parsePhotoKeysJSON parses a JSON array string of photo keys.
// Returns nil on error or nil input.
func parsePhotoKeysJSON(s *string) []string {
	if s == nil || *s == "" {
		return nil
	}
	var keys []string
	if err := json.Unmarshal([]byte(*s), &keys); err != nil {
		log.Printf("unmarshal photo_keys: %v", err)
		return nil
	}
	return keys
}

// unlinkAllPhotos parses and unlinks all photos from a photo_keys JSON string.
func unlinkAllPhotos(db *sql.DB, photoKeys *string) {
	keys := parsePhotoKeysJSON(photoKeys)
	if len(keys) == 0 {
		return
	}
	if err := store.UnlinkPhotos(db, keys); err != nil {
		log.Printf("unlink photos on delete: %v", err)
	}
}

// firstObjStore returns the first ObjectStore from a variadic list, or nil.
func firstObjStore(stores []storage.ObjectStore) storage.ObjectStore {
	if len(stores) > 0 {
		return stores[0]
	}
	return nil
}

// linkPhotosForCreate validates and links photos for a create operation.
// Returns the photo keys JSON string and true, or writes a 400 response and returns false.
func linkPhotosForCreate(w http.ResponseWriter, db *sql.DB, babyID string, photoKeys []string) (*string, bool) {
	if len(photoKeys) == 0 {
		return nil, true
	}
	result, errMsg, ok := handlePhotoLinking(db, babyID, nil, photoKeys)
	if !ok {
		http.Error(w, "invalid photo_keys: "+errMsg, http.StatusBadRequest)
		return nil, false
	}
	return result, true
}

// dedup removes duplicate strings while preserving order.
func dedup(ss []string) []string {
	seen := make(map[string]bool, len(ss))
	result := make([]string, 0, len(ss))
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
