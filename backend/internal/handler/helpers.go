package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
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
