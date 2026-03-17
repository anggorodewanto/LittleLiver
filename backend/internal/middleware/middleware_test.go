package middleware_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ablankz/LittleLiver/backend/internal/middleware"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

const testCookieName = "session_id"
const testSecret = "test-hmac-secret-key-for-csrf"

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := store.OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	migDir := filepath.Join(findProjectRoot(t), "migrations")
	if err := store.RunMigrations(db, migDir); err != nil {
		db.Close()
		t.Fatalf("RunMigrations failed: %v", err)
	}
	return db
}

func findProjectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	root := filepath.Join(dir, "..", "..")
	migDir := filepath.Join(root, "migrations")
	if _, err := os.Stat(migDir); os.IsNotExist(err) {
		t.Fatalf("migrations dir not found at %s", migDir)
	}
	return root
}

// okHandler is a simple handler that returns 200 OK.
func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := middleware.UserFromContext(r.Context())
		if user != nil {
			json.NewEncoder(w).Encode(map[string]string{"user_id": user.ID})
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

// --- Auth Middleware Tests ---

func TestAuth_NoCookie_Returns401(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	handler := middleware.Auth(db, testCookieName)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuth_InvalidSession_Returns401(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	handler := middleware.Auth(db, testCookieName)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: testCookieName, Value: "nonexistent-session"})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuth_ValidSession_SetsUserInContext(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	// Create user and session
	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	sess, err := store.CreateSession(db, "u1")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	handler := middleware.Auth(db, testCookieName)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: testCookieName, Value: sess.ID})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body failed: %v", err)
	}
	if body["user_id"] != "u1" {
		t.Errorf("expected user_id=u1, got %q", body["user_id"])
	}
}

func TestAuth_ValidSession_ExtendsExpiry(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	sess, err := store.CreateSession(db, "u1")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Get original expiry
	original, err := store.GetSessionByID(db, sess.ID)
	if err != nil {
		t.Fatalf("GetSessionByID failed: %v", err)
	}

	handler := middleware.Auth(db, testCookieName)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: testCookieName, Value: sess.ID})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Verify session was extended (new expiry >= original)
	extended, err := store.GetSessionByID(db, sess.ID)
	if err != nil {
		t.Fatalf("GetSessionByID after extend failed: %v", err)
	}
	if extended.ExpiresAt.Before(original.ExpiresAt) {
		t.Errorf("expected session to be extended, but new expiry %v is before original %v", extended.ExpiresAt, original.ExpiresAt)
	}
}

func TestAuth_UpdatesTimezoneFromHeader(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	sess, err := store.CreateSession(db, "u1")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	handler := middleware.Auth(db, testCookieName)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: testCookieName, Value: sess.ID})
	req.Header.Set("X-Timezone", "Asia/Tokyo")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Verify timezone was updated
	user, err := store.GetUserByID(db, "u1")
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if user.Timezone == nil || *user.Timezone != "Asia/Tokyo" {
		t.Errorf("expected timezone=Asia/Tokyo, got %v", user.Timezone)
	}
}

func TestAuth_NoTimezoneHeader_DoesNotClearExisting(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name, timezone) VALUES ('u1', 'g1', 'a@b.com', 'Test', 'America/New_York')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	sess, err := store.CreateSession(db, "u1")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	handler := middleware.Auth(db, testCookieName)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: testCookieName, Value: sess.ID})
	// No X-Timezone header
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	user, err := store.GetUserByID(db, "u1")
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if user.Timezone == nil || *user.Timezone != "America/New_York" {
		t.Errorf("expected timezone to remain America/New_York, got %v", user.Timezone)
	}
}

func TestAuth_InvalidTimezone_NotStored(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	sess, err := store.CreateSession(db, "u1")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	handler := middleware.Auth(db, testCookieName)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: testCookieName, Value: sess.ID})
	req.Header.Set("X-Timezone", "Not/A/Real/Timezone")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Verify timezone was NOT updated
	user, err := store.GetUserByID(db, "u1")
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if user.Timezone != nil {
		t.Errorf("expected nil timezone for invalid tz, got %v", *user.Timezone)
	}
}

func TestAuth_SetsSessionTokenInContext(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	sess, err := store.CreateSession(db, "u1")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	var capturedToken string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedToken = middleware.SessionTokenFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.Auth(db, testCookieName)(inner)
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: testCookieName, Value: sess.ID})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if capturedToken != sess.Token {
		t.Errorf("expected session token %q in context, got %q", sess.Token, capturedToken)
	}
}

func TestSessionTokenFromContext_NoToken(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	token := middleware.SessionTokenFromContext(req.Context())
	if token != "" {
		t.Fatalf("expected empty token, got %q", token)
	}
}

func TestCSRF_POST_UsesContextToken_WhenAuthMiddlewareRanFirst(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	sess, err := store.CreateSession(db, "u1")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	csrfToken := middleware.CSRFToken(sess.Token, testSecret)

	// Chain Auth -> CSRF -> handler
	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	handler := authMw(csrfMw(okHandler()))

	req := httptest.NewRequest(http.MethodPost, "/api/babies", nil)
	req.AddCookie(&http.Cookie{Name: testCookieName, Value: sess.ID})
	req.Header.Set("X-CSRF-Token", csrfToken)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for POST with Auth+CSRF chain, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// --- CSRF Token Tests ---

func TestCSRFToken_Deterministic(t *testing.T) {
	t.Parallel()

	token1 := middleware.CSRFToken("session-token-abc", testSecret)
	token2 := middleware.CSRFToken("session-token-abc", testSecret)

	if token1 == "" {
		t.Fatal("expected non-empty CSRF token")
	}
	if token1 != token2 {
		t.Errorf("expected deterministic CSRF token, got %q and %q", token1, token2)
	}
}

func TestCSRFToken_DifferentSessionsDifferentTokens(t *testing.T) {
	t.Parallel()

	token1 := middleware.CSRFToken("session-a", testSecret)
	token2 := middleware.CSRFToken("session-b", testSecret)

	if token1 == token2 {
		t.Error("expected different CSRF tokens for different sessions")
	}
}

func TestCSRFToken_DifferentSecretsDifferentTokens(t *testing.T) {
	t.Parallel()

	token1 := middleware.CSRFToken("same-session", "secret-1")
	token2 := middleware.CSRFToken("same-session", "secret-2")

	if token1 == token2 {
		t.Error("expected different CSRF tokens for different secrets")
	}
}

// --- CSRF Middleware Tests ---

func TestCSRF_GET_NoTokenRequired(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	handler := middleware.CSRF(db, testCookieName, testSecret)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for GET without CSRF token, got %d", rec.Code)
	}
}

func TestCSRF_HEAD_NoTokenRequired(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	handler := middleware.CSRF(db, testCookieName, testSecret)(okHandler())
	req := httptest.NewRequest(http.MethodHead, "/api/me", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for HEAD without CSRF token, got %d", rec.Code)
	}
}

func TestCSRF_OPTIONS_NoTokenRequired(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	handler := middleware.CSRF(db, testCookieName, testSecret)(okHandler())
	req := httptest.NewRequest(http.MethodOptions, "/api/me", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for OPTIONS without CSRF token, got %d", rec.Code)
	}
}

func TestCSRF_POST_NoToken_Returns403(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	sess, err := store.CreateSession(db, "u1")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	handler := middleware.CSRF(db, testCookieName, testSecret)(okHandler())
	req := httptest.NewRequest(http.MethodPost, "/api/babies", nil)
	req.AddCookie(&http.Cookie{Name: testCookieName, Value: sess.ID})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for POST without CSRF token, got %d", rec.Code)
	}
}

func TestCSRF_POST_InvalidToken_Returns403(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	sess, err := store.CreateSession(db, "u1")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	handler := middleware.CSRF(db, testCookieName, testSecret)(okHandler())
	req := httptest.NewRequest(http.MethodPost, "/api/babies", nil)
	req.AddCookie(&http.Cookie{Name: testCookieName, Value: sess.ID})
	req.Header.Set("X-CSRF-Token", "invalid-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for invalid CSRF token, got %d", rec.Code)
	}
}

func TestCSRF_POST_ValidToken_Passes(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	sess, err := store.CreateSession(db, "u1")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Derive the correct CSRF token
	csrfToken := middleware.CSRFToken(sess.Token, testSecret)

	handler := middleware.CSRF(db, testCookieName, testSecret)(okHandler())
	req := httptest.NewRequest(http.MethodPost, "/api/babies", nil)
	req.AddCookie(&http.Cookie{Name: testCookieName, Value: sess.ID})
	req.Header.Set("X-CSRF-Token", csrfToken)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid CSRF token, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCSRF_PUT_NoToken_Returns403(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	sess, err := store.CreateSession(db, "u1")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	handler := middleware.CSRF(db, testCookieName, testSecret)(okHandler())
	req := httptest.NewRequest(http.MethodPut, "/api/babies/b1", nil)
	req.AddCookie(&http.Cookie{Name: testCookieName, Value: sess.ID})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for PUT without CSRF token, got %d", rec.Code)
	}
}

func TestCSRF_DELETE_NoToken_Returns403(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	sess, err := store.CreateSession(db, "u1")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	handler := middleware.CSRF(db, testCookieName, testSecret)(okHandler())
	req := httptest.NewRequest(http.MethodDelete, "/api/babies/b1", nil)
	req.AddCookie(&http.Cookie{Name: testCookieName, Value: sess.ID})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for DELETE without CSRF token, got %d", rec.Code)
	}
}

func TestCSRF_POST_NoCookie_Returns403(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	handler := middleware.CSRF(db, testCookieName, testSecret)(okHandler())
	req := httptest.NewRequest(http.MethodPost, "/api/babies", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for POST without cookie, got %d", rec.Code)
	}
}

func TestAuth_UserDeletedAfterSession_Returns401(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	sess, err := store.CreateSession(db, "u1")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Delete the user after creating the session
	_, err = db.Exec("DELETE FROM users WHERE id = 'u1'")
	if err != nil {
		t.Fatalf("delete user failed: %v", err)
	}

	handler := middleware.Auth(db, testCookieName)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: testCookieName, Value: sess.ID})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when user deleted, got %d", rec.Code)
	}
}

// --- UserFromContext Tests ---

func TestUserFromContext_NoUser(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	user := middleware.UserFromContext(req.Context())
	if user != nil {
		t.Fatalf("expected nil user, got %+v", user)
	}
}
