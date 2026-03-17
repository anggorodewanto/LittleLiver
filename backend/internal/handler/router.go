package handler

import (
	"database/sql"
	"net/http"
	"os"

	"github.com/ablankz/LittleLiver/backend/internal/auth"
	"github.com/ablankz/LittleLiver/backend/internal/middleware"
)

// NewMux returns an HTTP mux with all routes registered.
// If db is non-nil and Google OAuth env vars are set, auth routes are registered.
func NewMux(opts ...Option) *http.ServeMux {
	cfg := &options{}
	for _, o := range opts {
		o(cfg)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", HealthCheck)

	// Register auth routes if DB and OAuth config are provided
	if cfg.db != nil {
		clientID := os.Getenv("GOOGLE_CLIENT_ID")
		clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
		baseURL := os.Getenv("BASE_URL")
		if baseURL == "" {
			baseURL = "http://localhost:8080"
		}

		sessionSecret := os.Getenv("SESSION_SECRET")

		if clientID != "" && clientSecret != "" {
			h := auth.NewHandlers(cfg.db, auth.Config{
				ClientID:     clientID,
				ClientSecret: clientSecret,
				RedirectURL:  baseURL + "/auth/google/callback",
			})
			auth.RegisterRoutes(mux, h)

			// Register API routes with auth middleware
			cookieName := auth.CookieName
			authMw := middleware.Auth(cfg.db, cookieName)

			// CSRF middleware for state-changing /api/ routes. When chained after
			// authMw, it reads the session token from context (no extra DB query).
			// Apply csrfMw to POST/PUT/DELETE API routes as they are added.
			csrfMw := middleware.CSRF(cfg.db, cookieName, sessionSecret)
			_ = csrfMw // used by future state-changing API routes

			// CSRF token endpoint (needs session but not auth middleware context)
			mux.HandleFunc("GET /api/csrf-token", CSRFTokenHandler(cfg.db, cookieName, sessionSecret))

			// /api/me needs auth middleware (GET-only, no CSRF needed)
			mux.Handle("GET /api/me", authMw(http.HandlerFunc(MeHandler(cfg.db))))
		}
	}

	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "./static"
	}
	if info, err := os.Stat(staticDir); err == nil && info.IsDir() {
		fs := http.FileServer(http.Dir(staticDir))
		mux.Handle("/", fs)
	}

	return mux
}

// Option configures the mux.
type Option func(*options)

type options struct {
	db *sql.DB
}

// WithDB provides a database connection for routes that need it.
func WithDB(db *sql.DB) Option {
	return func(o *options) {
		o.db = db
	}
}
