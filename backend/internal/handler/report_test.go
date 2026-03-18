package handler_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/auth"
	"github.com/ablankz/LittleLiver/backend/internal/handler"
	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

// seedBabyWithKasaiForHandler creates a baby with kasai_date set.
func seedBabyWithKasaiForHandler(t *testing.T, db *sql.DB, userID string) *model.Baby {
	t.Helper()
	babyID := model.NewULID()
	_, err := db.Exec(
		`INSERT INTO babies (id, name, sex, date_of_birth, kasai_date)
		 VALUES (?, 'Report Baby', 'female', '2025-06-15', '2025-07-01')`,
		babyID,
	)
	if err != nil {
		t.Fatalf("insert baby: %v", err)
	}
	_, err = db.Exec(
		"INSERT INTO baby_parents (baby_id, user_id) VALUES (?, ?)",
		babyID, userID,
	)
	if err != nil {
		t.Fatalf("insert baby_parents: %v", err)
	}
	// Return minimal baby info
	return &model.Baby{ID: babyID, Name: "Report Baby"}
}

// seedReportDataForHandler seeds basic metrics data.
func seedReportDataForHandler(t *testing.T, db *sql.DB, babyID, userID string) {
	t.Helper()
	ts := time.Date(2025, 8, 1, 10, 0, 0, 0, time.UTC).Format(model.DateTimeFormat)
	_, err := db.Exec(
		`INSERT INTO feedings (id, baby_id, logged_by, timestamp, feed_type, volume_ml, calories)
		 VALUES (?, ?, ?, ?, 'formula', 120.0, 80.0)`,
		model.NewULID(), babyID, userID, ts,
	)
	if err != nil {
		t.Fatalf("insert feeding: %v", err)
	}
}

func TestReportHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := seedBabyWithKasaiForHandler(t, db, user.ID)
	seedReportDataForHandler(t, db, baby.ID, user.ID)

	mux := handler.NewMux(
		handler.WithDB(db),
		handler.WithAuthConfig(auth.Config{
			ClientID:     "test-id",
			ClientSecret: "test-secret",
			RedirectURL:  "http://localhost/callback",
		}),
	)

	req := testutil.AuthenticatedRequest(t, db, user.ID, "session_id", "", http.MethodGet,
		"/api/babies/"+baby.ID+"/report?from=2025-08-01&to=2025-08-01")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	ct := rr.Header().Get("Content-Type")
	if ct != "application/pdf" {
		t.Errorf("expected Content-Type application/pdf, got %q", ct)
	}

	body := rr.Body.Bytes()
	if len(body) < 5 || string(body[:5]) != "%PDF-" {
		t.Fatal("response body is not a valid PDF")
	}
}

func TestReportHandler_MissingParams(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := seedBabyWithKasaiForHandler(t, db, user.ID)

	mux := handler.NewMux(
		handler.WithDB(db),
		handler.WithAuthConfig(auth.Config{
			ClientID:     "test-id",
			ClientSecret: "test-secret",
			RedirectURL:  "http://localhost/callback",
		}),
	)

	req := testutil.AuthenticatedRequest(t, db, user.ID, "session_id", "", http.MethodGet,
		"/api/babies/"+baby.ID+"/report")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing params, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestReportHandler_InvalidDateFormat(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := seedBabyWithKasaiForHandler(t, db, user.ID)

	mux := handler.NewMux(
		handler.WithDB(db),
		handler.WithAuthConfig(auth.Config{
			ClientID:     "test-id",
			ClientSecret: "test-secret",
			RedirectURL:  "http://localhost/callback",
		}),
	)

	// Invalid "from" date
	req := testutil.AuthenticatedRequest(t, db, user.ID, "session_id", "", http.MethodGet,
		"/api/babies/"+baby.ID+"/report?from=not-a-date&to=2025-08-01")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid from date, got %d: %s", rr.Code, rr.Body.String())
	}

	// Invalid "to" date
	req = testutil.AuthenticatedRequest(t, db, user.ID, "session_id", "", http.MethodGet,
		"/api/babies/"+baby.ID+"/report?from=2025-08-01&to=13-2025-01")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid to date, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestReportHandler_Unauthorized(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := seedBabyWithKasaiForHandler(t, db, user.ID)

	mux := handler.NewMux(
		handler.WithDB(db),
		handler.WithAuthConfig(auth.Config{
			ClientID:     "test-id",
			ClientSecret: "test-secret",
			RedirectURL:  "http://localhost/callback",
		}),
	)

	// Request without auth cookie
	req := httptest.NewRequest(http.MethodGet,
		"/api/babies/"+baby.ID+"/report?from=2025-08-01&to=2025-08-01", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}
