package auth

import "net/http"

// RegisterRoutes registers auth-related routes on the given mux.
func RegisterRoutes(mux *http.ServeMux, h *Handlers) {
	mux.HandleFunc("GET /auth/google/login", h.Login)
	mux.HandleFunc("GET /auth/google/callback", h.Callback)
	mux.HandleFunc("POST /auth/logout", h.Logout)
}
