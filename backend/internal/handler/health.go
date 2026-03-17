package handler

import (
	"encoding/json"
	"log"
	"net/http"
)

// HealthCheck handles GET /health requests.
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Printf("health: failed to write response: %v", err)
	}
}
