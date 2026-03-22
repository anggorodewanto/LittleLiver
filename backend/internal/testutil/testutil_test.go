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

func TestSetupTestDB_AllMetricTablesExist(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	metricTables := []string{
		"feedings", "stools", "urine", "weights", "temperatures",
		"abdomen_observations", "skin_observations", "bruising",
		"lab_results", "general_notes", "medications", "med_logs",
		"photo_uploads", "push_subscriptions",
	}
	for _, table := range metricTables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("expected table %q to exist in migrated DB, got error: %v", table, err)
		}
	}
}

func TestSetupTestDB_SentinelUserExists(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	var id string
	err := db.QueryRow("SELECT id FROM users WHERE id = 'deleted_user'").Scan(&id)
	if err != nil {
		t.Fatalf("sentinel user not found: %v", err)
	}
	if id != "deleted_user" {
		t.Errorf("expected 'deleted_user', got %q", id)
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

func TestCreateTestBaby_HasDefaultCalPerFeed(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	if baby.DefaultCalPerFeed != 67.0 {
		t.Errorf("expected default_cal_per_feed=67.0, got %f", baby.DefaultCalPerFeed)
	}
}

func TestSeedBabyWithKasai_HasKasaiDate(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.SeedBabyWithKasai(t, db, user.ID)

	if baby.ID == "" {
		t.Error("expected non-empty baby ID")
	}
	if baby.Name != "Report Baby" {
		t.Errorf("expected name 'Report Baby', got %q", baby.Name)
	}
	if baby.KasaiDate == nil {
		t.Fatal("expected kasai_date to be set")
	}
	kasaiStr := baby.KasaiDate.Format("2006-01-02")
	if kasaiStr != "2025-07-01" {
		t.Errorf("expected kasai_date='2025-07-01', got %q", kasaiStr)
	}
	dobStr := baby.DateOfBirth.Format("2006-01-02")
	if dobStr != "2025-06-15" {
		t.Errorf("expected date_of_birth='2025-06-15', got %q", dobStr)
	}

	// Verify linked to user
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM baby_parents WHERE baby_id = ? AND user_id = ?", baby.ID, user.ID).Scan(&count)
	if err != nil {
		t.Fatalf("query baby_parents failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 baby_parents row, got %d", count)
	}
}

func TestSeedPushSubscription_InsertsSubscription(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	testutil.SeedPushSubscription(t, db, user.ID, "https://push.example.com/sub1")

	var endpoint string
	err := db.QueryRow("SELECT endpoint FROM push_subscriptions WHERE user_id = ?", user.ID).Scan(&endpoint)
	if err != nil {
		t.Fatalf("query push_subscriptions failed: %v", err)
	}
	if endpoint != "https://push.example.com/sub1" {
		t.Errorf("expected endpoint 'https://push.example.com/sub1', got %q", endpoint)
	}
}

func TestSeedPushSubscription_MultipleSubscriptions(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	testutil.SeedPushSubscription(t, db, user.ID, "https://push.example.com/device1")
	testutil.SeedPushSubscription(t, db, user.ID, "https://push.example.com/device2")

	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM push_subscriptions WHERE user_id = ?", user.ID).Scan(&count)
	if err != nil {
		t.Fatalf("query push_subscriptions failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 subscriptions, got %d", count)
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

func TestAuthenticatedRequest_PUT_HasCSRFToken(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	req := testutil.AuthenticatedRequest(t, db, user.ID, "session_id", "test-secret", http.MethodPut, "/api/babies/123")

	csrfToken := req.Header.Get("X-CSRF-Token")
	if csrfToken == "" {
		t.Error("expected CSRF token for PUT request")
	}
}

func TestAuthenticatedRequest_DELETE_HasCSRFToken(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	req := testutil.AuthenticatedRequest(t, db, user.ID, "session_id", "test-secret", http.MethodDelete, "/api/babies/123")

	csrfToken := req.Header.Get("X-CSRF-Token")
	if csrfToken == "" {
		t.Error("expected CSRF token for DELETE request")
	}
}

func TestAuthenticatedRequest_HEAD_NoCSRFToken(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	req := testutil.AuthenticatedRequest(t, db, user.ID, "session_id", "test-secret", http.MethodHead, "/api/me")

	csrfToken := req.Header.Get("X-CSRF-Token")
	if csrfToken != "" {
		t.Errorf("expected no CSRF token for HEAD request, got %q", csrfToken)
	}
}

func TestAuthenticatedRequest_OPTIONS_NoCSRFToken(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	req := testutil.AuthenticatedRequest(t, db, user.ID, "session_id", "test-secret", http.MethodOptions, "/api/me")

	csrfToken := req.Header.Get("X-CSRF-Token")
	if csrfToken != "" {
		t.Errorf("expected no CSRF token for OPTIONS request, got %q", csrfToken)
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

func TestTestFixture_StructFields(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	fixture := testutil.TestFixture{
		DB:   db,
		User: user,
		Baby: baby,
	}

	if fixture.DB == nil {
		t.Error("expected non-nil DB")
	}
	if fixture.User == nil {
		t.Error("expected non-nil User")
	}
	if fixture.Baby == nil {
		t.Error("expected non-nil Baby")
	}
	if fixture.User.ID != user.ID {
		t.Errorf("expected user ID=%q, got %q", user.ID, fixture.User.ID)
	}
	if fixture.Baby.ID != baby.ID {
		t.Errorf("expected baby ID=%q, got %q", baby.ID, fixture.Baby.ID)
	}
}
