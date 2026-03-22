package auth

import "net/http"

// RegisterRoutes registers auth-related routes on the given mux.
// Optional middleware functions are chained around the handlers (e.g., IP rate limiting).
// Note: POST /auth/logout is registered separately in router.go with full middleware chain.
func RegisterRoutes(mux *http.ServeMux, h *Handlers, mws ...func(http.Handler) http.Handler) {
	var wrap func(http.HandlerFunc) http.Handler
	if len(mws) == 0 {
		wrap = func(hf http.HandlerFunc) http.Handler { return hf }
	} else {
		wrap = func(hf http.HandlerFunc) http.Handler {
			var handler http.Handler = hf
			// Apply middleware in reverse order so the first mw is outermost
			for i := len(mws) - 1; i >= 0; i-- {
				handler = mws[i](handler)
			}
			return handler
		}
	}
	mux.Handle("GET /auth/google/login", wrap(h.Login))
	mux.Handle("GET /auth/google/callback", wrap(h.Callback))
}
