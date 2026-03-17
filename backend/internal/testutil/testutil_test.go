package testutil_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ablankz/LittleLiver/backend/internal/middleware"
	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

func TestSetupTestDB_ReturnsMigratedDB(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	// Verify the DB is migrated by checking core tables exist
	tables := []string{"users", "babies", "baby_parents", "sessions", "invites"}
	for _, table := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("expected table %q to exist in migrated DB, got error: %v", table, err)
		}
	}
}

func TestSetupTestDB_ForeignKeysEnabled(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	var fkEnabled int
	err := db.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled)
	if err != nil {
		t.Fatalf("PRAGMA foreign_keys query failed: %v", err)
	}
	if fkEnabled != 1 {
		t.Errorf("expected foreign_keys=1, got %d", fkEnabled)
	}
}

func TestCreateTestUser_InsertsUserAndReturnsIt(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	if user.ID == "" {
		t.Error("expected non-empty user ID")
	}
	if user.GoogleID == "" {
		t.Error("expected non-empty google_id")
	}
	if user.Email == "" {
		t.Error("expected non-empty email")
	}
	if user.Name == "" {
		t.Error("expected non-empty name")
	}

	// Verify user actually exists in the DB
	found, err := store.GetUserByID(db, user.ID)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if found.ID != user.ID {
		t.Errorf("expected user ID=%q in DB, got %q", user.ID, found.ID)
	}
}

func TestCreateTestUser_MultipleCallsCreateDistinctUsers(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	u1 := testutil.CreateTestUser(t, db)
	u2 := testutil.CreateTestUser(t, db)

	if u1.ID == u2.ID {
		t.Errorf("expected distinct user IDs, got both %q", u1.ID)
	}
	if u1.GoogleID == u2.GoogleID {
		t.Errorf("expected distinct google_ids, got both %q", u1.GoogleID)
	}
}

func TestCreateTestBaby_InsertsBabyLinkedToUser(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	if baby.ID == "" {
		t.Error("expected non-empty baby ID")
	}
	if baby.Name == "" {
		t.Error("expected non-empty baby name")
	}

	// Verify baby is linked to user via baby_parents
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM baby_parents WHERE baby_id = ? AND user_id = ?", baby.ID, user.ID).Scan(&count)
	if err != nil {
		t.Fatalf("query baby_parents failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 baby_parents row, got %d", count)
	}

	// Verify GetBabiesByUserID returns this baby
	babies, err := store.GetBabiesByUserID(db, user.ID)
	if err != nil {
		t.Fatalf("GetBabiesByUserID failed: %v", err)
	}
	if len(babies) != 1 {
		t.Fatalf("expected 1 baby, got %d", len(babies))
	}
	if babies[0].ID != baby.ID {
		t.Errorf("expected baby ID=%q, got %q", baby.ID, babies[0].ID)
	}
}

func TestCreateTestBaby_MultipleCallsCreateDistinctBabies(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	b1 := testutil.CreateTestBaby(t, db, user.ID)
	b2 := testutil.CreateTestBaby(t, db, user.ID)

	if b1.ID == b2.ID {
		t.Errorf("expected distinct baby IDs, got both %q", b1.ID)
	}
}

func TestAuthenticatedRequest_HasSessionCookieAndCSRFToken(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	const cookieName = "session_id"
	const csrfSecret = "test-secret"

	req := testutil.AuthenticatedRequest(t, db, user.ID, cookieName, csrfSecret, http.MethodPost, "/api/babies")

	// Verify session cookie is present
	cookie, err := req.Cookie(cookieName)
	if err != nil {
		t.Fatalf("expected session cookie, got error: %v", err)
	}
	if cookie.Value == "" {
		t.Error("expected non-empty session cookie value")
	}

	// Verify the session is valid in the DB
	sess, err := store.GetSessionByID(db, cookie.Value)
	if err != nil {
		t.Fatalf("session not found in DB: %v", err)
	}
	if sess.UserID != user.ID {
		t.Errorf("expected session user_id=%q, got %q", user.ID, sess.UserID)
	}

	// Verify CSRF token header is present and valid
	csrfToken := req.Header.Get("X-CSRF-Token")
	if csrfToken == "" {
		t.Error("expected non-empty X-CSRF-Token header")
	}
	expectedCSRF := middleware.CSRFToken(sess.Token, csrfSecret)
	if csrfToken != expectedCSRF {
		t.Errorf("expected CSRF token %q, got %q", expectedCSRF, csrfToken)
	}
}

func TestAuthenticatedRequest_GET_NoCSRFToken(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	req := testutil.AuthenticatedRequest(t, db, user.ID, "session_id", "test-secret", http.MethodGet, "/api/me")

	// GET requests should still have the session cookie
	cookie, err := req.Cookie("session_id")
	if err != nil {
		t.Fatalf("expected session cookie: %v", err)
	}
	if cookie.Value == "" {
		t.Error("expected non-empty session cookie")
	}

	// GET requests should NOT have CSRF token (it's not needed)
	csrfToken := req.Header.Get("X-CSRF-Token")
	if csrfToken != "" {
		t.Errorf("expected no CSRF token for GET request, got %q", csrfToken)
	}
}

func TestAuthenticatedRequest_PassesThroughAuthAndCSRFMiddleware(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	const cookieName = "session_id"
	const csrfSecret = "test-secret"

	// Build an authenticated POST request
	req := testutil.AuthenticatedRequest(t, db, user.ID, cookieName, csrfSecret, http.MethodPost, "/api/babies")

	// Verify it passes through Auth + CSRF middleware chain
	var capturedUserID string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := middleware.UserFromContext(r.Context())
		if u != nil {
			capturedUserID = u.ID
		}
		w.WriteHeader(http.StatusOK)
	})

	authMw := middleware.Auth(db, cookieName)
	csrfMw := middleware.CSRF(db, cookieName, csrfSecret)
	handler := authMw(csrfMw(inner))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 through middleware chain, got %d. Body: %s", rec.Code, rec.Body.String())
	}
	if capturedUserID != user.ID {
		t.Errorf("expected user ID=%q in context, got %q", user.ID, capturedUserID)
	}
}
