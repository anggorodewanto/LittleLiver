package auth

import "net/http"

// RegisterRoutes registers auth-related routes on the given mux.
// Note: POST /auth/logout is registered separately in router.go with full middleware chain.
func RegisterRoutes(mux *http.ServeMux, h *Handlers) {
	mux.HandleFunc("GET /auth/google/login", h.Login)
	mux.HandleFunc("GET /auth/google/callback", h.Callback)
}
