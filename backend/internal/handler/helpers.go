package handler

import (
	"encoding/json"
	"log"
	"net/http"
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
