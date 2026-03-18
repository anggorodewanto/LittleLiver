package handler

import (
	"database/sql"
	"net/http"
	"os"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/auth"
	"github.com/ablankz/LittleLiver/backend/internal/middleware"
	"github.com/ablankz/LittleLiver/backend/internal/storage"
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

			// Metric CRUD endpoints
			registerMetricCRUD(mux, "/api/babies/{id}/feedings", rateMw, authMw, csrfMw,
				CreateFeedingHandler(cfg.db), ListFeedingsHandler(cfg.db),
				GetFeedingHandler(cfg.db), UpdateFeedingHandler(cfg.db), DeleteFeedingHandler(cfg.db))
			registerMetricCRUD(mux, "/api/babies/{id}/stools", rateMw, authMw, csrfMw,
				CreateStoolHandler(cfg.db, cfg.objStore), ListStoolsHandler(cfg.db, cfg.objStore),
				GetStoolHandler(cfg.db, cfg.objStore), UpdateStoolHandler(cfg.db, cfg.objStore), DeleteStoolHandler(cfg.db))
			registerMetricCRUD(mux, "/api/babies/{id}/urine", rateMw, authMw, csrfMw,
				CreateUrineHandler(cfg.db), ListUrineHandler(cfg.db),
				GetUrineHandler(cfg.db), UpdateUrineHandler(cfg.db), DeleteUrineHandler(cfg.db))
			registerMetricCRUD(mux, "/api/babies/{id}/weights", rateMw, authMw, csrfMw,
				CreateWeightHandler(cfg.db), ListWeightsHandler(cfg.db),
				GetWeightHandler(cfg.db), UpdateWeightHandler(cfg.db), DeleteWeightHandler(cfg.db))
			registerMetricCRUD(mux, "/api/babies/{id}/temperatures", rateMw, authMw, csrfMw,
				CreateTemperatureHandler(cfg.db), ListTemperaturesHandler(cfg.db),
				GetTemperatureHandler(cfg.db), UpdateTemperatureHandler(cfg.db), DeleteTemperatureHandler(cfg.db))
			registerMetricCRUD(mux, "/api/babies/{id}/abdomen", rateMw, authMw, csrfMw,
				CreateAbdomenHandler(cfg.db, cfg.objStore), ListAbdomenHandler(cfg.db, cfg.objStore),
				GetAbdomenHandler(cfg.db, cfg.objStore), UpdateAbdomenHandler(cfg.db, cfg.objStore), DeleteAbdomenHandler(cfg.db))
			registerMetricCRUD(mux, "/api/babies/{id}/skin", rateMw, authMw, csrfMw,
				CreateSkinObservationHandler(cfg.db, cfg.objStore), ListSkinObservationsHandler(cfg.db, cfg.objStore),
				GetSkinObservationHandler(cfg.db, cfg.objStore), UpdateSkinObservationHandler(cfg.db, cfg.objStore), DeleteSkinObservationHandler(cfg.db))
			registerMetricCRUD(mux, "/api/babies/{id}/bruising", rateMw, authMw, csrfMw,
				CreateBruisingHandler(cfg.db, cfg.objStore), ListBruisingHandler(cfg.db, cfg.objStore),
				GetBruisingHandler(cfg.db, cfg.objStore), UpdateBruisingHandler(cfg.db, cfg.objStore), DeleteBruisingHandler(cfg.db))
			registerMetricCRUD(mux, "/api/babies/{id}/labs", rateMw, authMw, csrfMw,
				CreateLabResultHandler(cfg.db), ListLabResultsHandler(cfg.db),
				GetLabResultHandler(cfg.db), UpdateLabResultHandler(cfg.db), DeleteLabResultHandler(cfg.db))
			registerMetricCRUD(mux, "/api/babies/{id}/notes", rateMw, authMw, csrfMw,
				CreateGeneralNoteHandler(cfg.db, cfg.objStore), ListGeneralNotesHandler(cfg.db, cfg.objStore),
				GetGeneralNoteHandler(cfg.db, cfg.objStore), UpdateGeneralNoteHandler(cfg.db, cfg.objStore), DeleteGeneralNoteHandler(cfg.db))

			// Medication CRUD endpoints (no DELETE — only deactivation via PUT)
			mux.Handle("POST /api/babies/{id}/medications", rateMw(authMw(csrfMw(http.HandlerFunc(CreateMedicationHandler(cfg.db))))))
			mux.Handle("GET /api/babies/{id}/medications", rateMw(authMw(http.HandlerFunc(ListMedicationsHandler(cfg.db)))))
			mux.Handle("GET /api/babies/{id}/medications/{medId}", rateMw(authMw(http.HandlerFunc(GetMedicationHandler(cfg.db)))))
			mux.Handle("PUT /api/babies/{id}/medications/{medId}", rateMw(authMw(csrfMw(http.HandlerFunc(UpdateMedicationHandler(cfg.db))))))
			mux.Handle("DELETE /api/babies/{id}/medications/{medId}", rateMw(authMw(csrfMw(http.HandlerFunc(DeleteMedicationHandler(cfg.db))))))

			// Med-log CRUD endpoints
			mux.Handle("POST /api/babies/{id}/med-logs", rateMw(authMw(csrfMw(http.HandlerFunc(CreateMedLogHandler(cfg.db))))))
			mux.Handle("GET /api/babies/{id}/med-logs", rateMw(authMw(http.HandlerFunc(ListMedLogsHandler(cfg.db)))))
			mux.Handle("GET /api/babies/{id}/med-logs/{entryId}", rateMw(authMw(http.HandlerFunc(GetMedLogHandler(cfg.db)))))
			mux.Handle("PUT /api/babies/{id}/med-logs/{entryId}", rateMw(authMw(csrfMw(http.HandlerFunc(UpdateMedLogHandler(cfg.db))))))
			mux.Handle("DELETE /api/babies/{id}/med-logs/{entryId}", rateMw(authMw(csrfMw(http.HandlerFunc(DeleteMedLogHandler(cfg.db))))))

			// Invite endpoints
			mux.Handle("POST /api/babies/{id}/invite", rateMw(authMw(csrfMw(http.HandlerFunc(CreateInviteHandler(cfg.db))))))
			mux.Handle("POST /api/babies/join", rateMw(authMw(csrfMw(http.HandlerFunc(JoinBabyHandler(cfg.db))))))

			// Dashboard endpoint
			mux.Handle("GET /api/babies/{id}/dashboard", rateMw(authMw(http.HandlerFunc(DashboardHandler(cfg.db)))))

			// Report endpoint
			mux.Handle("GET /api/babies/{id}/report", rateMw(authMw(http.HandlerFunc(ReportHandler(cfg.db, cfg.objStore)))))

			// WHO percentile curves endpoint
			mux.Handle("GET /api/who/percentiles", rateMw(authMw(http.HandlerFunc(WHOPercentilesHandler()))))

			// Push subscription endpoints
			mux.Handle("POST /api/push/subscribe", rateMw(authMw(csrfMw(http.HandlerFunc(SubscribePushHandler(cfg.db))))))
			mux.Handle("DELETE /api/push/subscribe", rateMw(authMw(csrfMw(http.HandlerFunc(UnsubscribePushHandler(cfg.db))))))

			// Photo upload endpoint
			if cfg.objStore != nil {
				mux.Handle("POST /api/babies/{id}/upload", rateMw(authMw(csrfMw(http.HandlerFunc(UploadPhotoHandler(cfg.db, cfg.objStore))))))
			}
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
	objStore   storage.ObjectStore
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

// WithObjectStore provides an ObjectStore for photo uploads.
func WithObjectStore(s storage.ObjectStore) Option {
	return func(o *options) {
		o.objStore = s
	}
}

// registerMetricCRUD registers the standard 5-route CRUD pattern for a metric resource.
func registerMetricCRUD(mux *http.ServeMux, path string,
	rateMw, authMw, csrfMw func(http.Handler) http.Handler,
	create, list, get, update, del http.HandlerFunc) {
	mux.Handle("POST "+path, rateMw(authMw(csrfMw(create))))
	mux.Handle("GET "+path, rateMw(authMw(list)))
	mux.Handle("GET "+path+"/{entryId}", rateMw(authMw(get)))
	mux.Handle("PUT "+path+"/{entryId}", rateMw(authMw(csrfMw(update))))
	mux.Handle("DELETE "+path+"/{entryId}", rateMw(authMw(csrfMw(del))))
}
