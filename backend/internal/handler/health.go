package handler

import (
	"net/http"
)

// HealthCheck handles GET /health requests.
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
