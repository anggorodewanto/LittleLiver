// Package testutil provides reusable test helpers for setting up in-memory
// databases, creating test fixtures, and building authenticated HTTP requests.
package testutil

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/ablankz/LittleLiver/backend/internal/middleware"
	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

// counter provides unique IDs across test helper calls to avoid collisions.
var counter atomic.Int64

// projectRoot finds the backend/ directory by walking up from this source file.
func projectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	// filename is .../backend/internal/testutil/testutil.go
	return filepath.Join(filepath.Dir(filename), "..", "..")
}

// SetupTestDB creates an in-memory SQLite database with all migrations applied.
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := store.OpenDB(":memory:")
	if err != nil {
		t.Fatalf("testutil.SetupTestDB: OpenDB failed: %v", err)
	}

	migDir := filepath.Join(projectRoot(), "migrations")
	if _, err := os.Stat(migDir); os.IsNotExist(err) {
		db.Close()
		t.Fatalf("testutil.SetupTestDB: migrations dir not found at %s", migDir)
	}

	if err := store.RunMigrations(db, migDir); err != nil {
		db.Close()
		t.Fatalf("testutil.SetupTestDB: RunMigrations failed: %v", err)
	}
	return db
}

// CreateTestUser inserts a test user into the database and returns it.
// Each call creates a distinct user with unique google_id and email.
func CreateTestUser(t *testing.T, db *sql.DB) *model.User {
	t.Helper()
	n := counter.Add(1)
	googleID := fmt.Sprintf("google-test-%d", n)
	email := fmt.Sprintf("testuser%d@example.com", n)
	name := fmt.Sprintf("Test User %d", n)

	user, err := store.UpsertUser(db, googleID, email, name)
	if err != nil {
		t.Fatalf("testutil.CreateTestUser: UpsertUser failed: %v", err)
	}
	return user
}

// CreateTestBaby inserts a test baby linked to the given user and returns it.
// Each call creates a distinct baby.
func CreateTestBaby(t *testing.T, db *sql.DB, userID string) *model.Baby {
	t.Helper()
	n := counter.Add(1)
	babyID := model.NewULID()
	babyName := fmt.Sprintf("Test Baby %d", n)

	_, err := db.Exec(
		"INSERT INTO babies (id, name, sex, date_of_birth) VALUES (?, ?, 'female', '2025-01-01')",
		babyID, babyName,
	)
	if err != nil {
		t.Fatalf("testutil.CreateTestBaby: insert baby failed: %v", err)
	}

	_, err = db.Exec(
		"INSERT INTO baby_parents (baby_id, user_id) VALUES (?, ?)",
		babyID, userID,
	)
	if err != nil {
		t.Fatalf("testutil.CreateTestBaby: insert baby_parents failed: %v", err)
	}

	// Fetch the baby back so all fields (including defaults) are populated
	babies, err := store.GetBabiesByUserID(db, userID)
	if err != nil {
		t.Fatalf("testutil.CreateTestBaby: GetBabiesByUserID failed: %v", err)
	}
	for i := range babies {
		if babies[i].ID == babyID {
			return &babies[i]
		}
	}
	t.Fatalf("testutil.CreateTestBaby: baby %s not found after insert", babyID)
	return nil
}

// TestFixture bundles commonly used test objects for convenience.
type TestFixture struct {
	DB   *sql.DB
	User *model.User
	Baby *model.Baby
}

// AuthenticatedRequest builds an *http.Request with a valid session cookie
// and (for state-changing methods like POST, PUT, DELETE) a valid CSRF token header.
func AuthenticatedRequest(t *testing.T, db *sql.DB, userID, cookieName, csrfSecret, method, target string) *http.Request {
	t.Helper()

	sess, err := store.CreateSession(db, userID)
	if err != nil {
		t.Fatalf("testutil.AuthenticatedRequest: CreateSession failed: %v", err)
	}

	req := httptest.NewRequest(method, target, nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: sess.ID})

	// Only add CSRF token for state-changing methods
	if method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions {
		csrfToken := middleware.CSRFToken(sess.Token, csrfSecret)
		req.Header.Set("X-CSRF-Token", csrfToken)
	}

	return req
}

// SeedPushSubscription inserts a push subscription for a user.
func SeedPushSubscription(t *testing.T, db *sql.DB, userID, endpoint string) {
	t.Helper()
	_, err := db.Exec(
		"INSERT INTO push_subscriptions (id, user_id, endpoint, p256dh, auth) VALUES (?, ?, ?, 'key', 'auth')",
		fmt.Sprintf("ps-%s-%s", userID, endpoint), userID, endpoint,
	)
	if err != nil {
		t.Fatalf("testutil.SeedPushSubscription: %v", err)
	}
}
