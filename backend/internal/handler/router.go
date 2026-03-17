package handler

import (
	"database/sql"
	"net/http"
	"os"
	"time"

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
		var authCfg auth.Config
		var sessionSecret string

		if cfg.authConfig != nil {
			authCfg = *cfg.authConfig
			sessionSecret = cfg.authConfig.SessionSecret
		} else {
			clientID := os.Getenv("GOOGLE_CLIENT_ID")
			clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
			baseURL := os.Getenv("BASE_URL")
			if baseURL == "" {
				baseURL = "http://localhost:8080"
			}
			sessionSecret = os.Getenv("SESSION_SECRET")
			authCfg = auth.Config{
				ClientID:     clientID,
				ClientSecret: clientSecret,
				RedirectURL:  baseURL + "/auth/google/callback",
			}
		}

		if authCfg.ClientID != "" && authCfg.ClientSecret != "" {
			h := auth.NewHandlers(cfg.db, authCfg)
			auth.RegisterRoutes(mux, h)

			// Register API routes with auth middleware
			cookieName := auth.CookieName
			authMw := middleware.Auth(cfg.db, cookieName)

			// Per-session rate limiter: 100 requests per minute
			rateLimiter := middleware.NewRateLimiter(100, time.Minute)
			rateMw := rateLimiter.Middleware(cookieName)

			// CSRF middleware for state-changing /api/ routes. When chained after
			// authMw, it reads the session token from context (no extra DB query).
			// Apply csrfMw to POST/PUT/DELETE API routes as they are added.
			csrfMw := middleware.CSRF(cfg.db, cookieName, sessionSecret)

			// Chain: rateMw -> authMw -> handler (rate limit checked first)
			// CSRF token endpoint — behind auth middleware so session token is in context
			mux.Handle("GET /api/csrf-token", rateMw(authMw(http.HandlerFunc(CSRFTokenHandler(sessionSecret)))))

			// /api/me needs auth middleware (GET-only, no CSRF needed)
			mux.Handle("GET /api/me", rateMw(authMw(http.HandlerFunc(MeHandler(cfg.db)))))

			// Baby CRUD endpoints
			mux.Handle("POST /api/babies", rateMw(authMw(csrfMw(http.HandlerFunc(CreateBabyHandler(cfg.db))))))
			mux.Handle("GET /api/babies", rateMw(authMw(http.HandlerFunc(ListBabiesHandler(cfg.db)))))
			mux.Handle("GET /api/babies/{id}", rateMw(authMw(http.HandlerFunc(GetBabyHandler(cfg.db)))))
			mux.Handle("PUT /api/babies/{id}", rateMw(authMw(csrfMw(http.HandlerFunc(UpdateBabyHandler(cfg.db))))))

			// Invite endpoints
			mux.Handle("POST /api/babies/{id}/invite", rateMw(authMw(csrfMw(http.HandlerFunc(CreateInviteHandler(cfg.db))))))
			mux.Handle("POST /api/babies/join", rateMw(authMw(csrfMw(http.HandlerFunc(JoinBabyHandler(cfg.db))))))
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
	db         *sql.DB
	authConfig *auth.Config
}

// WithDB provides a database connection for routes that need it.
func WithDB(db *sql.DB) Option {
	return func(o *options) {
		o.db = db
	}
}

// WithAuthConfig overrides the Google OAuth configuration (useful for testing).
func WithAuthConfig(cfg auth.Config) Option {
	return func(o *options) {
		o.authConfig = &cfg
	}
}
