package handler

import "net/http"

// NewMux returns an HTTP mux with all routes registered.
func NewMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", HealthCheck)
	return mux
}
