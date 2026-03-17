package auth_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/ablankz/LittleLiver/backend/internal/auth"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

// setupTestDB creates an in-memory DB with migrations applied.
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
	// Navigate up from internal/auth to backend
	root := filepath.Join(dir, "..", "..")
	migDir := filepath.Join(root, "migrations")
	if _, err := os.Stat(migDir); os.IsNotExist(err) {
		t.Fatalf("migrations dir not found at %s", migDir)
	}
	return root
}

func TestLogin_RedirectsToGoogle(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	h := auth.NewHandlers(db, auth.Config{
		ClientID:    "test-client-id",
		RedirectURL: "http://localhost:8080/auth/google/callback",
	})

	req := httptest.NewRequest(http.MethodGet, "/auth/google/login", nil)
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}

	location := rec.Header().Get("Location")
	u, err := url.Parse(location)
	if err != nil {
		t.Fatalf("failed to parse redirect URL: %v", err)
	}

	if u.Host != "accounts.google.com" {
		t.Errorf("expected redirect to accounts.google.com, got %q", u.Host)
	}

	q := u.Query()
	if q.Get("client_id") != "test-client-id" {
		t.Errorf("expected client_id=test-client-id, got %q", q.Get("client_id"))
	}
	if q.Get("redirect_uri") != "http://localhost:8080/auth/google/callback" {
		t.Errorf("expected redirect_uri, got %q", q.Get("redirect_uri"))
	}
	if q.Get("response_type") != "code" {
		t.Errorf("expected response_type=code, got %q", q.Get("response_type"))
	}
	if q.Get("scope") != "openid email profile" {
		t.Errorf("expected scope=openid email profile, got %q", q.Get("scope"))
	}
	if q.Get("state") == "" {
		t.Error("expected non-empty state parameter")
	}
}

func TestCallback_ExchangesCodeAndCreatesSession(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	// Mock Google token endpoint
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.FormValue("code") != "test-auth-code" {
			t.Errorf("expected code=test-auth-code, got %q", r.FormValue("code"))
		}
		if r.FormValue("grant_type") != "authorization_code" {
			t.Errorf("expected grant_type=authorization_code, got %q", r.FormValue("grant_type"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "mock-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer tokenServer.Close()

	// Mock Google userinfo endpoint
	userinfoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer mock-access-token" {
			t.Errorf("expected Bearer token, got %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"id":    "google-user-123",
			"email": "parent@example.com",
			"name":  "Test Parent",
		})
	}))
	defer userinfoServer.Close()

	h := auth.NewHandlers(db, auth.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost:8080/auth/google/callback",
		TokenURL:     tokenServer.URL,
		UserInfoURL:  userinfoServer.URL,
	})

	// First, perform a login to get a valid state
	loginReq := httptest.NewRequest(http.MethodGet, "/auth/google/login", nil)
	loginRec := httptest.NewRecorder()
	h.Login(loginRec, loginReq)

	// Extract state from redirect URL
	loc := loginRec.Header().Get("Location")
	u, _ := url.Parse(loc)
	state := u.Query().Get("state")

	// Now simulate the callback
	callbackURL := "/auth/google/callback?code=test-auth-code&state=" + state
	callbackReq := httptest.NewRequest(http.MethodGet, callbackURL, nil)
	callbackRec := httptest.NewRecorder()

	h.Callback(callbackRec, callbackReq)

	if callbackRec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d. Body: %s", callbackRec.Code, callbackRec.Body.String())
	}

	// Check redirect goes to /
	if callbackRec.Header().Get("Location") != "/" {
		t.Errorf("expected redirect to /, got %q", callbackRec.Header().Get("Location"))
	}

	// Check session cookie is set
	cookies := callbackRec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == auth.CookieName {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected session cookie to be set")
	}
	if !sessionCookie.HttpOnly {
		t.Error("expected HttpOnly cookie")
	}
	if !sessionCookie.Secure {
		t.Error("expected Secure cookie")
	}
	if sessionCookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("expected SameSite=Lax, got %v", sessionCookie.SameSite)
	}
	if sessionCookie.Value == "" {
		t.Error("expected non-empty session cookie value")
	}

	// Verify session exists in DB
	sess, err := store.GetSessionByID(db, sessionCookie.Value)
	if err != nil {
		t.Fatalf("session not found in DB: %v", err)
	}
	if sess.UserID == "" {
		t.Error("expected session to have a user_id")
	}

	// Verify user was created
	var email string
	err = db.QueryRow("SELECT email FROM users WHERE google_id = ?", "google-user-123").Scan(&email)
	if err != nil {
		t.Fatalf("user not found: %v", err)
	}
	if email != "parent@example.com" {
		t.Errorf("expected email=parent@example.com, got %q", email)
	}
}

func TestCallback_InvalidState_Returns400(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	h := auth.NewHandlers(db, auth.Config{
		ClientID: "test-client-id",
	})

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?code=abc&state=invalid", nil)
	rec := httptest.NewRecorder()

	h.Callback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid state, got %d", rec.Code)
	}
}

func TestCallback_MissingCode_Returns400(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	h := auth.NewHandlers(db, auth.Config{
		ClientID: "test-client-id",
	})

	// Get a valid state first
	loginReq := httptest.NewRequest(http.MethodGet, "/auth/google/login", nil)
	loginRec := httptest.NewRecorder()
	h.Login(loginRec, loginReq)
	loc := loginRec.Header().Get("Location")
	u, _ := url.Parse(loc)
	state := u.Query().Get("state")

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state="+state, nil)
	rec := httptest.NewRecorder()

	h.Callback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing code, got %d", rec.Code)
	}
}

func TestCallback_TokenExchangeFailure_Returns500(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	// Mock a failing token endpoint
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid_grant"}`))
	}))
	defer tokenServer.Close()

	h := auth.NewHandlers(db, auth.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost:8080/auth/google/callback",
		TokenURL:     tokenServer.URL,
	})

	// Get a valid state
	loginReq := httptest.NewRequest(http.MethodGet, "/auth/google/login", nil)
	loginRec := httptest.NewRecorder()
	h.Login(loginRec, loginReq)
	loc := loginRec.Header().Get("Location")
	u, _ := url.Parse(loc)
	state := u.Query().Get("state")

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?code=bad-code&state="+state, nil)
	rec := httptest.NewRecorder()

	h.Callback(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for token exchange failure, got %d", rec.Code)
	}
}

func TestLogout_ClearsSession(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	// Create a user and session
	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	sess, err := store.CreateSession(db, "u1")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	h := auth.NewHandlers(db, auth.Config{})

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: sess.ID})
	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}

	// Check cookie is cleared
	cookies := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == auth.CookieName {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected session cookie to be set (for clearing)")
	}
	if sessionCookie.MaxAge != -1 {
		t.Errorf("expected MaxAge=-1 to clear cookie, got %d", sessionCookie.MaxAge)
	}

	// Session should be deleted from DB
	_, err = store.GetSessionByID(db, sess.ID)
	if err != sql.ErrNoRows {
		t.Fatalf("expected session to be deleted, got err=%v", err)
	}
}

func TestCallback_UserInfoFailure_Returns500(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	// Mock token endpoint succeeds
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "mock-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer tokenServer.Close()

	// Mock userinfo endpoint fails
	userinfoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid_token"}`))
	}))
	defer userinfoServer.Close()

	h := auth.NewHandlers(db, auth.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost:8080/auth/google/callback",
		TokenURL:     tokenServer.URL,
		UserInfoURL:  userinfoServer.URL,
	})

	// Get a valid state
	loginReq := httptest.NewRequest(http.MethodGet, "/auth/google/login", nil)
	loginRec := httptest.NewRecorder()
	h.Login(loginRec, loginReq)
	loc := loginRec.Header().Get("Location")
	u, _ := url.Parse(loc)
	state := u.Query().Get("state")

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?code=test-code&state="+state, nil)
	rec := httptest.NewRecorder()

	h.Callback(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for userinfo failure, got %d", rec.Code)
	}
}

func TestLogout_NoCookie_Returns204(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	h := auth.NewHandlers(db, auth.Config{})

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

func TestNewHandlers_DefaultURLs(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	h := auth.NewHandlers(db, auth.Config{
		ClientID: "test-id",
	})

	// Verify handlers were created (non-nil)
	if h == nil {
		t.Fatal("expected non-nil handlers")
	}
}

func TestLogin_SetsStateParam(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	h := auth.NewHandlers(db, auth.Config{
		ClientID:    "test-id",
		RedirectURL: "http://localhost/callback",
	})

	// Call Login twice - second call should still work (state cleanup)
	req1 := httptest.NewRequest(http.MethodGet, "/auth/google/login", nil)
	rec1 := httptest.NewRecorder()
	h.Login(rec1, req1)

	req2 := httptest.NewRequest(http.MethodGet, "/auth/google/login", nil)
	rec2 := httptest.NewRecorder()
	h.Login(rec2, req2)

	// Both should redirect
	if rec1.Code != http.StatusFound {
		t.Errorf("first login: expected 302, got %d", rec1.Code)
	}
	if rec2.Code != http.StatusFound {
		t.Errorf("second login: expected 302, got %d", rec2.Code)
	}

	// States should be different
	loc1 := rec1.Header().Get("Location")
	loc2 := rec2.Header().Get("Location")
	u1, _ := url.Parse(loc1)
	u2, _ := url.Parse(loc2)
	if u1.Query().Get("state") == u2.Query().Get("state") {
		t.Error("expected different state values for each login")
	}
}

func TestRegisterRoutes_RoutesExist(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	mux := http.NewServeMux()
	h := auth.NewHandlers(db, auth.Config{
		ClientID:    "test-id",
		RedirectURL: "http://localhost/callback",
	})
	auth.RegisterRoutes(mux, h)

	// Test login route exists
	req := httptest.NewRequest(http.MethodGet, "/auth/google/login", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Errorf("expected 302 for login, got %d", rec.Code)
	}

	// Test callback route exists (will fail validation but proves route is registered)
	req = httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=bad", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for callback with bad state, got %d", rec.Code)
	}

	// Test logout route exists
	req = httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204 for logout, got %d", rec.Code)
	}
}

func TestLogin_ConcurrentRequests_NoRace(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	h := auth.NewHandlers(db, auth.Config{
		ClientID:    "test-id",
		RedirectURL: "http://localhost/callback",
	})

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/auth/google/login", nil)
			rec := httptest.NewRecorder()
			h.Login(rec, req)
			if rec.Code != http.StatusFound {
				t.Errorf("expected 302, got %d", rec.Code)
			}
		}()
	}
	wg.Wait()
}

func TestCallback_MalformedTokenJSON_Returns500(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	// Token endpoint returns invalid JSON
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{not valid json`))
	}))
	defer tokenServer.Close()

	h := auth.NewHandlers(db, auth.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost:8080/auth/google/callback",
		TokenURL:     tokenServer.URL,
	})

	// Get a valid state
	loginReq := httptest.NewRequest(http.MethodGet, "/auth/google/login", nil)
	loginRec := httptest.NewRecorder()
	h.Login(loginRec, loginReq)
	loc := loginRec.Header().Get("Location")
	u, _ := url.Parse(loc)
	state := u.Query().Get("state")

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?code=test-code&state="+state, nil)
	rec := httptest.NewRecorder()
	h.Callback(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for malformed token JSON, got %d", rec.Code)
	}
}

func TestCallback_MalformedUserInfoJSON_Returns500(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	// Token endpoint succeeds
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "mock-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer tokenServer.Close()

	// Userinfo endpoint returns invalid JSON
	userinfoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{not valid json`))
	}))
	defer userinfoServer.Close()

	h := auth.NewHandlers(db, auth.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost:8080/auth/google/callback",
		TokenURL:     tokenServer.URL,
		UserInfoURL:  userinfoServer.URL,
	})

	// Get a valid state
	loginReq := httptest.NewRequest(http.MethodGet, "/auth/google/login", nil)
	loginRec := httptest.NewRecorder()
	h.Login(loginRec, loginReq)
	loc := loginRec.Header().Get("Location")
	u, _ := url.Parse(loc)
	state := u.Query().Get("state")

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?code=test-code&state="+state, nil)
	rec := httptest.NewRecorder()
	h.Callback(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for malformed userinfo JSON, got %d", rec.Code)
	}
}

func TestCallback_TokenExchangeNetworkError_Returns500(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	h := auth.NewHandlers(db, auth.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost:8080/auth/google/callback",
		TokenURL:     "http://127.0.0.1:1", // connection refused
	})

	loginReq := httptest.NewRequest(http.MethodGet, "/auth/google/login", nil)
	loginRec := httptest.NewRecorder()
	h.Login(loginRec, loginReq)
	loc := loginRec.Header().Get("Location")
	u, _ := url.Parse(loc)
	state := u.Query().Get("state")

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?code=test&state="+state, nil)
	rec := httptest.NewRecorder()
	h.Callback(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestCallback_UserInfoNetworkError_Returns500(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "mock-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer tokenServer.Close()

	h := auth.NewHandlers(db, auth.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost:8080/auth/google/callback",
		TokenURL:     tokenServer.URL,
		UserInfoURL:  "http://127.0.0.1:1", // connection refused
	})

	loginReq := httptest.NewRequest(http.MethodGet, "/auth/google/login", nil)
	loginRec := httptest.NewRecorder()
	h.Login(loginRec, loginReq)
	loc := loginRec.Header().Get("Location")
	u, _ := url.Parse(loc)
	state := u.Query().Get("state")

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?code=test&state="+state, nil)
	rec := httptest.NewRecorder()
	h.Callback(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestCallback_DBClosed_UpsertFails_Returns500(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "mock-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer tokenServer.Close()

	userinfoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"id":    "google-user-789",
			"email": "test@example.com",
			"name":  "Test",
		})
	}))
	defer userinfoServer.Close()

	h := auth.NewHandlers(db, auth.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost:8080/auth/google/callback",
		TokenURL:     tokenServer.URL,
		UserInfoURL:  userinfoServer.URL,
	})

	loginReq := httptest.NewRequest(http.MethodGet, "/auth/google/login", nil)
	loginRec := httptest.NewRecorder()
	h.Login(loginRec, loginReq)
	loc := loginRec.Header().Get("Location")
	u, _ := url.Parse(loc)
	state := u.Query().Get("state")

	// Close DB to trigger upsert failure
	db.Close()

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?code=test&state="+state, nil)
	rec := httptest.NewRecorder()
	h.Callback(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestCallback_UpsertExistingUser(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	// Pre-insert user
	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('existing-id', 'google-user-456', 'old@example.com', 'Old Name')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "mock-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer tokenServer.Close()

	userinfoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"id":    "google-user-456",
			"email": "new@example.com",
			"name":  "New Name",
		})
	}))
	defer userinfoServer.Close()

	h := auth.NewHandlers(db, auth.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost:8080/auth/google/callback",
		TokenURL:     tokenServer.URL,
		UserInfoURL:  userinfoServer.URL,
	})

	// Login to get state
	loginReq := httptest.NewRequest(http.MethodGet, "/auth/google/login", nil)
	loginRec := httptest.NewRecorder()
	h.Login(loginRec, loginReq)
	loc := loginRec.Header().Get("Location")
	u, _ := url.Parse(loc)
	state := u.Query().Get("state")

	// Callback
	callbackReq := httptest.NewRequest(http.MethodGet, "/auth/google/callback?code=test-code&state="+state, nil)
	callbackRec := httptest.NewRecorder()
	h.Callback(callbackRec, callbackReq)

	if callbackRec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d. Body: %s", callbackRec.Code, callbackRec.Body.String())
	}

	// Verify user was updated (not duplicated)
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE google_id = ?", "google-user-456").Scan(&count)
	if err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 user, got %d", count)
	}

	var email string
	err = db.QueryRow("SELECT email FROM users WHERE google_id = ?", "google-user-456").Scan(&email)
	if err != nil {
		t.Fatalf("query user failed: %v", err)
	}
	if email != "new@example.com" {
		t.Errorf("expected updated email=new@example.com, got %q", email)
	}
}
