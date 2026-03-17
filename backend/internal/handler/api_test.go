package handler_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ablankz/LittleLiver/backend/internal/handler"
	"github.com/ablankz/LittleLiver/backend/internal/middleware"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

const testCookieName = "session_id"
const testSecret = "test-hmac-secret-key-for-csrf"

func setupTestDBForAPI(t *testing.T) *sql.DB {
	t.Helper()
	db, err := store.OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	migDir := filepath.Join(dir, "..", "..", "migrations")
	if err := store.RunMigrations(db, migDir); err != nil {
		db.Close()
		t.Fatalf("RunMigrations failed: %v", err)
	}
	return db
}

// --- CSRF Token Handler Tests ---

func TestCSRFTokenHandler_ReturnsToken(t *testing.T) {
	t.Parallel()
	db := setupTestDBForAPI(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	sess, err := store.CreateSession(db, "u1")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	h := handler.CSRFTokenHandler(db, testCookieName, testSecret)
	req := httptest.NewRequest(http.MethodGet, "/api/csrf-token", nil)
	req.AddCookie(&http.Cookie{Name: testCookieName, Value: sess.ID})
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	expectedToken := middleware.CSRFToken(sess.Token, testSecret)
	if body["csrf_token"] != expectedToken {
		t.Errorf("expected csrf_token=%q, got %q", expectedToken, body["csrf_token"])
	}
}

func TestCSRFTokenHandler_NoCookie_Returns401(t *testing.T) {
	t.Parallel()
	db := setupTestDBForAPI(t)
	defer db.Close()

	h := handler.CSRFTokenHandler(db, testCookieName, testSecret)
	req := httptest.NewRequest(http.MethodGet, "/api/csrf-token", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestCSRFTokenHandler_InvalidSession_Returns401(t *testing.T) {
	t.Parallel()
	db := setupTestDBForAPI(t)
	defer db.Close()

	h := handler.CSRFTokenHandler(db, testCookieName, testSecret)
	req := httptest.NewRequest(http.MethodGet, "/api/csrf-token", nil)
	req.AddCookie(&http.Cookie{Name: testCookieName, Value: "invalid-session"})
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

// --- Me Handler Tests ---

func TestMeHandler_ReturnsUserInfo(t *testing.T) {
	t.Parallel()
	db := setupTestDBForAPI(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name, timezone) VALUES ('u1', 'g1', 'a@b.com', 'Test Parent', 'America/New_York')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	sess, err := store.CreateSession(db, "u1")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Add a baby linked to this user
	_, err = db.Exec("INSERT INTO babies (id, name, sex, date_of_birth) VALUES ('b1', 'Baby1', 'male', '2025-01-01')")
	if err != nil {
		t.Fatalf("insert baby failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO baby_parents (baby_id, user_id) VALUES ('b1', 'u1')")
	if err != nil {
		t.Fatalf("insert baby_parents failed: %v", err)
	}

	// Use auth middleware to set user context
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(handler.MeHandler(db))

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: testCookieName, Value: sess.ID})
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var body struct {
		User struct {
			ID       string  `json:"id"`
			Email    string  `json:"email"`
			Name     string  `json:"name"`
			Timezone *string `json:"timezone"`
		} `json:"user"`
		Babies []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"babies"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if body.User.ID != "u1" {
		t.Errorf("expected user id=u1, got %q", body.User.ID)
	}
	if body.User.Email != "a@b.com" {
		t.Errorf("expected email=a@b.com, got %q", body.User.Email)
	}
	if body.User.Name != "Test Parent" {
		t.Errorf("expected name=Test Parent, got %q", body.User.Name)
	}
	if body.User.Timezone == nil || *body.User.Timezone != "America/New_York" {
		t.Errorf("expected timezone=America/New_York, got %v", body.User.Timezone)
	}
	if len(body.Babies) != 1 {
		t.Fatalf("expected 1 baby, got %d", len(body.Babies))
	}
	if body.Babies[0].ID != "b1" {
		t.Errorf("expected baby id=b1, got %q", body.Babies[0].ID)
	}
}

func TestMeHandler_NoBabies_ReturnsEmptyArray(t *testing.T) {
	t.Parallel()
	db := setupTestDBForAPI(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	sess, err := store.CreateSession(db, "u1")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	authMw := middleware.Auth(db, testCookieName)
	h := authMw(handler.MeHandler(db))

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: testCookieName, Value: sess.ID})
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var body struct {
		Babies []interface{} `json:"babies"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if body.Babies == nil {
		t.Error("expected babies to be an empty array, got nil")
	}
	if len(body.Babies) != 0 {
		t.Errorf("expected 0 babies, got %d", len(body.Babies))
	}
}

func TestMeHandler_NoUserInContext_Returns401(t *testing.T) {
	t.Parallel()
	db := setupTestDBForAPI(t)
	defer db.Close()

	h := handler.MeHandler(db)
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}
