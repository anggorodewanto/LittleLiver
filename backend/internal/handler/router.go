package handler

import (
	"database/sql"
	"net/http"
	"os"

	"github.com/ablankz/LittleLiver/backend/internal/auth"
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

		if clientID != "" && clientSecret != "" {
			h := auth.NewHandlers(cfg.db, auth.Config{
				ClientID:     clientID,
				ClientSecret: clientSecret,
				RedirectURL:  baseURL + "/auth/google/callback",
			})
			auth.RegisterRoutes(mux, h)
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
