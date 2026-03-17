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

			// Self-unlink endpoint
			mux.Handle("DELETE /api/babies/{id}/parents/me", rateMw(authMw(csrfMw(http.HandlerFunc(UnlinkSelfHandler(cfg.db))))))

			// Account deletion
			mux.Handle("DELETE /api/users/me", rateMw(authMw(csrfMw(http.HandlerFunc(DeleteAccountHandler(cfg.db))))))

			// Feeding CRUD endpoints
			mux.Handle("POST /api/babies/{id}/feedings", rateMw(authMw(csrfMw(http.HandlerFunc(CreateFeedingHandler(cfg.db))))))
			mux.Handle("GET /api/babies/{id}/feedings", rateMw(authMw(http.HandlerFunc(ListFeedingsHandler(cfg.db)))))
			mux.Handle("GET /api/babies/{id}/feedings/{entryId}", rateMw(authMw(http.HandlerFunc(GetFeedingHandler(cfg.db)))))
			mux.Handle("PUT /api/babies/{id}/feedings/{entryId}", rateMw(authMw(csrfMw(http.HandlerFunc(UpdateFeedingHandler(cfg.db))))))
			mux.Handle("DELETE /api/babies/{id}/feedings/{entryId}", rateMw(authMw(csrfMw(http.HandlerFunc(DeleteFeedingHandler(cfg.db))))))

			// Stool CRUD endpoints
			mux.Handle("POST /api/babies/{id}/stools", rateMw(authMw(csrfMw(http.HandlerFunc(CreateStoolHandler(cfg.db))))))
			mux.Handle("GET /api/babies/{id}/stools", rateMw(authMw(http.HandlerFunc(ListStoolsHandler(cfg.db)))))
			mux.Handle("GET /api/babies/{id}/stools/{entryId}", rateMw(authMw(http.HandlerFunc(GetStoolHandler(cfg.db)))))
			mux.Handle("PUT /api/babies/{id}/stools/{entryId}", rateMw(authMw(csrfMw(http.HandlerFunc(UpdateStoolHandler(cfg.db))))))
			mux.Handle("DELETE /api/babies/{id}/stools/{entryId}", rateMw(authMw(csrfMw(http.HandlerFunc(DeleteStoolHandler(cfg.db))))))

			// Urine CRUD endpoints
			mux.Handle("POST /api/babies/{id}/urine", rateMw(authMw(csrfMw(http.HandlerFunc(CreateUrineHandler(cfg.db))))))
			mux.Handle("GET /api/babies/{id}/urine", rateMw(authMw(http.HandlerFunc(ListUrineHandler(cfg.db)))))
			mux.Handle("GET /api/babies/{id}/urine/{entryId}", rateMw(authMw(http.HandlerFunc(GetUrineHandler(cfg.db)))))
			mux.Handle("PUT /api/babies/{id}/urine/{entryId}", rateMw(authMw(csrfMw(http.HandlerFunc(UpdateUrineHandler(cfg.db))))))
			mux.Handle("DELETE /api/babies/{id}/urine/{entryId}", rateMw(authMw(csrfMw(http.HandlerFunc(DeleteUrineHandler(cfg.db))))))

			// Weight CRUD endpoints
			mux.Handle("POST /api/babies/{id}/weights", rateMw(authMw(csrfMw(http.HandlerFunc(CreateWeightHandler(cfg.db))))))
			mux.Handle("GET /api/babies/{id}/weights", rateMw(authMw(http.HandlerFunc(ListWeightsHandler(cfg.db)))))
			mux.Handle("GET /api/babies/{id}/weights/{entryId}", rateMw(authMw(http.HandlerFunc(GetWeightHandler(cfg.db)))))
			mux.Handle("PUT /api/babies/{id}/weights/{entryId}", rateMw(authMw(csrfMw(http.HandlerFunc(UpdateWeightHandler(cfg.db))))))
			mux.Handle("DELETE /api/babies/{id}/weights/{entryId}", rateMw(authMw(csrfMw(http.HandlerFunc(DeleteWeightHandler(cfg.db))))))

			// Temperature CRUD endpoints
			mux.Handle("POST /api/babies/{id}/temperatures", rateMw(authMw(csrfMw(http.HandlerFunc(CreateTemperatureHandler(cfg.db))))))
			mux.Handle("GET /api/babies/{id}/temperatures", rateMw(authMw(http.HandlerFunc(ListTemperaturesHandler(cfg.db)))))
			mux.Handle("GET /api/babies/{id}/temperatures/{entryId}", rateMw(authMw(http.HandlerFunc(GetTemperatureHandler(cfg.db)))))
			mux.Handle("PUT /api/babies/{id}/temperatures/{entryId}", rateMw(authMw(csrfMw(http.HandlerFunc(UpdateTemperatureHandler(cfg.db))))))
			mux.Handle("DELETE /api/babies/{id}/temperatures/{entryId}", rateMw(authMw(csrfMw(http.HandlerFunc(DeleteTemperatureHandler(cfg.db))))))

			// Abdomen CRUD endpoints
			mux.Handle("POST /api/babies/{id}/abdomen", rateMw(authMw(csrfMw(http.HandlerFunc(CreateAbdomenHandler(cfg.db))))))
			mux.Handle("GET /api/babies/{id}/abdomen", rateMw(authMw(http.HandlerFunc(ListAbdomenHandler(cfg.db)))))
			mux.Handle("GET /api/babies/{id}/abdomen/{entryId}", rateMw(authMw(http.HandlerFunc(GetAbdomenHandler(cfg.db)))))
			mux.Handle("PUT /api/babies/{id}/abdomen/{entryId}", rateMw(authMw(csrfMw(http.HandlerFunc(UpdateAbdomenHandler(cfg.db))))))
			mux.Handle("DELETE /api/babies/{id}/abdomen/{entryId}", rateMw(authMw(csrfMw(http.HandlerFunc(DeleteAbdomenHandler(cfg.db))))))

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
