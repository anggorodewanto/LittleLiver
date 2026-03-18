package integration_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/auth"
	"github.com/ablankz/LittleLiver/backend/internal/handler"
	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

// setupOAuthIntegrationServer creates mock Google OAuth servers and an httptest.Server
// with the full router stack. Returns the app server, cleanup func, and DB.
func setupOAuthIntegrationServer(t *testing.T) (*httptest.Server, *sql.DB, func()) {
	t.Helper()

	db := testutil.SetupTestDB(t)

	// Mock Google token endpoint
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "mock-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))

	// Mock Google userinfo endpoint
	userinfoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer mock-access-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"id":    "google-integration-user",
			"email": "integration@example.com",
			"name":  "Integration User",
		})
	}))

	mux := handler.NewMux(
		handler.WithDB(db),
		handler.WithAuthConfig(auth.Config{
			ClientID:      "test-client-id",
			ClientSecret:  "test-client-secret",
			RedirectURL:   "http://localhost/auth/google/callback",
			TokenURL:      tokenServer.URL,
			UserInfoURL:   userinfoServer.URL,
			SessionSecret: testSessionSecret,
		}),
	)

	srv := httptest.NewServer(mux)

	cleanup := func() {
		srv.Close()
		tokenServer.Close()
		userinfoServer.Close()
		db.Close()
	}

	return srv, db, cleanup
}

// newClientWithCookies creates an http.Client with a cookie jar that does NOT
// follow redirects (returns the redirect response as-is for inspection).
func newClientWithCookies(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("failed to create cookie jar: %v", err)
	}
	return &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// TestAuthFlow_FullLifecycle tests the complete auth lifecycle:
// login redirect -> callback (session cookie set) -> fetch CSRF token ->
// /api/me (authorized GET) -> logout -> verify 401.
func TestAuthFlow_FullLifecycle(t *testing.T) {
	t.Parallel()
	srv, _, cleanup := setupOAuthIntegrationServer(t)
	defer cleanup()

	client := newClientWithCookies(t)

	// --- Step 1: GET /auth/google/login -> redirects to Google ---
	resp, err := client.Get(srv.URL + "/auth/google/login")
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected 302 from login, got %d", resp.StatusCode)
	}
	location := resp.Header.Get("Location")
	u, err := url.Parse(location)
	if err != nil {
		t.Fatalf("failed to parse login redirect URL: %v", err)
	}
	if u.Host != "accounts.google.com" {
		t.Fatalf("expected redirect to accounts.google.com, got %q", u.Host)
	}
	state := u.Query().Get("state")
	if state == "" {
		t.Fatal("expected non-empty state parameter in login redirect")
	}

	// --- Step 2: GET /auth/google/callback with code+state -> session cookie set ---
	callbackURL := srv.URL + "/auth/google/callback?code=test-auth-code&state=" + state
	resp, err = client.Get(callbackURL)
	if err != nil {
		t.Fatalf("callback request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected 302 from callback, got %d", resp.StatusCode)
	}

	// Verify session cookie was set in the jar
	srvURL, _ := url.Parse(srv.URL)
	cookies := client.Jar.Cookies(srvURL)
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == auth.CookieName {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected session cookie to be set after callback")
	}

	// --- Step 3: GET /api/csrf-token (authenticated) -> get CSRF token ---
	resp, err = client.Get(srv.URL + "/api/csrf-token")
	if err != nil {
		t.Fatalf("csrf-token request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from /api/csrf-token, got %d", resp.StatusCode)
	}
	var csrfResp map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&csrfResp); err != nil {
		t.Fatalf("failed to decode csrf-token response: %v", err)
	}
	csrfToken := csrfResp["csrf_token"]
	if csrfToken == "" {
		t.Fatal("expected non-empty CSRF token")
	}

	// --- Step 4: GET /api/me (authorized) -> verify user info ---
	resp, err = client.Get(srv.URL + "/api/me")
	if err != nil {
		t.Fatalf("/api/me request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from /api/me, got %d", resp.StatusCode)
	}
	var meResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&meResp); err != nil {
		t.Fatalf("failed to decode /api/me response: %v", err)
	}
	user, ok := meResp["user"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'user' key in /api/me response")
	}
	if user["email"] != "integration@example.com" {
		t.Errorf("expected email=integration@example.com, got %q", user["email"])
	}
	if user["name"] != "Integration User" {
		t.Errorf("expected name=Integration User, got %q", user["name"])
	}

	// --- Step 5: POST /auth/logout with CSRF token -> 204 ---
	logoutReq, err := http.NewRequest(http.MethodPost, srv.URL+"/auth/logout", nil)
	if err != nil {
		t.Fatalf("failed to create logout request: %v", err)
	}
	// Add cookies from jar
	for _, c := range client.Jar.Cookies(srvURL) {
		logoutReq.AddCookie(c)
	}
	resp, err = client.Do(logoutReq)
	if err != nil {
		t.Fatalf("logout request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 from logout, got %d", resp.StatusCode)
	}

	// --- Step 6: GET /api/me after logout -> 401 ---
	resp, err = client.Get(srv.URL + "/api/me")
	if err != nil {
		t.Fatalf("/api/me after logout failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 from /api/me after logout, got %d", resp.StatusCode)
	}

	// --- Step 7: GET /api/csrf-token after logout -> 401 ---
	resp, err = client.Get(srv.URL + "/api/csrf-token")
	if err != nil {
		t.Fatalf("/api/csrf-token after logout failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 from /api/csrf-token after logout, got %d", resp.StatusCode)
	}
}

// TestAuthFlow_SlidingWindowExtension verifies that accessing an authenticated
// endpoint extends the session's expiration (sliding window).
func TestAuthFlow_SlidingWindowExtension(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := setupOAuthIntegrationServer(t)
	defer cleanup()

	client := newClientWithCookies(t)

	// Login flow
	resp, err := client.Get(srv.URL + "/auth/google/login")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	resp.Body.Close()
	location := resp.Header.Get("Location")
	u, _ := url.Parse(location)
	state := u.Query().Get("state")

	resp, err = client.Get(srv.URL + "/auth/google/callback?code=test-code&state=" + state)
	if err != nil {
		t.Fatalf("callback failed: %v", err)
	}
	resp.Body.Close()

	// Get the session cookie value
	srvURL, _ := url.Parse(srv.URL)
	var sessionID string
	for _, c := range client.Jar.Cookies(srvURL) {
		if c.Name == auth.CookieName {
			sessionID = c.Value
			break
		}
	}
	if sessionID == "" {
		t.Fatal("no session cookie found")
	}

	// Record the initial expiry
	sess, err := store.GetSessionByID(db, sessionID)
	if err != nil {
		t.Fatalf("GetSessionByID failed: %v", err)
	}
	initialExpiry := sess.ExpiresAt

	// Manually set the session expiry to 1 day from now (shorter than default 30 days)
	// so we can detect extension clearly.
	shortExpiry := time.Now().Add(24 * time.Hour).UTC()
	_, err = db.Exec(
		"UPDATE sessions SET expires_at = ? WHERE id = ?",
		shortExpiry.Format(time.DateTime), sessionID,
	)
	if err != nil {
		t.Fatalf("failed to shorten session expiry: %v", err)
	}

	// Access an authenticated endpoint to trigger sliding window extension
	resp, err = client.Get(srv.URL + "/api/me")
	if err != nil {
		t.Fatalf("/api/me failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Verify the session expiry was extended (should be ~30 days from now, not 1 day)
	sess, err = store.GetSessionByID(db, sessionID)
	if err != nil {
		t.Fatalf("GetSessionByID after extension failed: %v", err)
	}

	// The extended expiry should be significantly longer than the short 1-day expiry
	// and approximately equal to the initial 30-day expiry.
	if sess.ExpiresAt.Before(shortExpiry.Add(24 * time.Hour)) {
		t.Errorf("session was not extended: expiry=%v, expected much later than %v",
			sess.ExpiresAt, shortExpiry)
	}

	// Verify it's approximately 30 days from now (within a few minutes tolerance)
	expectedExpiry := time.Now().Add(store.SessionDuration).UTC()
	diff := sess.ExpiresAt.Sub(expectedExpiry)
	if diff < 0 {
		diff = -diff
	}
	if diff > 5*time.Minute {
		t.Errorf("extended expiry %v is too far from expected %v (diff=%v)",
			sess.ExpiresAt, initialExpiry, diff)
	}
}

// TestAuthFlow_ExpiredSessionRejected verifies that a session that has expired
// results in a 401 response.
func TestAuthFlow_ExpiredSessionRejected(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := setupOAuthIntegrationServer(t)
	defer cleanup()

	client := newClientWithCookies(t)

	// Login flow
	resp, err := client.Get(srv.URL + "/auth/google/login")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	resp.Body.Close()
	location := resp.Header.Get("Location")
	u, _ := url.Parse(location)
	state := u.Query().Get("state")

	resp, err = client.Get(srv.URL + "/auth/google/callback?code=test-code&state=" + state)
	if err != nil {
		t.Fatalf("callback failed: %v", err)
	}
	resp.Body.Close()

	// Get the session cookie value
	srvURL, _ := url.Parse(srv.URL)
	var sessionID string
	for _, c := range client.Jar.Cookies(srvURL) {
		if c.Name == auth.CookieName {
			sessionID = c.Value
			break
		}
	}
	if sessionID == "" {
		t.Fatal("no session cookie found")
	}

	// Verify we can access /api/me while session is valid
	resp, err = client.Get(srv.URL + "/api/me")
	if err != nil {
		t.Fatalf("/api/me failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 while session valid, got %d", resp.StatusCode)
	}

	// Expire the session by setting expires_at to the past
	pastTime := time.Now().Add(-1 * time.Hour).UTC()
	_, err = db.Exec(
		"UPDATE sessions SET expires_at = ? WHERE id = ?",
		pastTime.Format(time.DateTime), sessionID,
	)
	if err != nil {
		t.Fatalf("failed to expire session: %v", err)
	}

	// Now /api/me should return 401
	resp, err = client.Get(srv.URL + "/api/me")
	if err != nil {
		t.Fatalf("/api/me after expiry failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for expired session, got %d", resp.StatusCode)
	}

	// /api/csrf-token should also return 401
	resp, err = client.Get(srv.URL + "/api/csrf-token")
	if err != nil {
		t.Fatalf("/api/csrf-token after expiry failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 from /api/csrf-token after session expiry, got %d", resp.StatusCode)
	}
}

// TestAuthFlow_UnauthenticatedAccess verifies that unauthenticated requests
// to protected endpoints return 401.
func TestAuthFlow_UnauthenticatedAccess(t *testing.T) {
	t.Parallel()
	srv, _, cleanup := setupOAuthIntegrationServer(t)
	defer cleanup()

	client := newClientWithCookies(t)

	// /api/me without session -> 401
	resp, err := client.Get(srv.URL + "/api/me")
	if err != nil {
		t.Fatalf("/api/me request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 from /api/me without session, got %d", resp.StatusCode)
	}

	// /api/csrf-token without session -> 401
	resp, err = client.Get(srv.URL + "/api/csrf-token")
	if err != nil {
		t.Fatalf("/api/csrf-token request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 from /api/csrf-token without session, got %d", resp.StatusCode)
	}
}

// TestAuthFlow_InvalidSessionCookie verifies that a request with an invalid
// session cookie value gets 401.
func TestAuthFlow_InvalidSessionCookie(t *testing.T) {
	t.Parallel()
	srv, _, cleanup := setupOAuthIntegrationServer(t)
	defer cleanup()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/me", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: "invalid-session-id"})

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid session cookie, got %d", resp.StatusCode)
	}
}

// TestAuthFlow_CallbackReplayPrevented verifies that a state token cannot be
// reused after the callback (replay attack prevention).
func TestAuthFlow_CallbackReplayPrevented(t *testing.T) {
	t.Parallel()
	srv, _, cleanup := setupOAuthIntegrationServer(t)
	defer cleanup()

	client := newClientWithCookies(t)

	// Login to get state
	resp, err := client.Get(srv.URL + "/auth/google/login")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	resp.Body.Close()
	location := resp.Header.Get("Location")
	u, _ := url.Parse(location)
	state := u.Query().Get("state")

	// First callback succeeds
	resp, err = client.Get(srv.URL + "/auth/google/callback?code=test-code&state=" + state)
	if err != nil {
		t.Fatalf("first callback failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected 302 from first callback, got %d", resp.StatusCode)
	}

	// Second callback with same state should fail (state consumed)
	resp, err = client.Get(srv.URL + "/auth/google/callback?code=test-code&state=" + state)
	if err != nil {
		t.Fatalf("second callback failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for replayed state, got %d", resp.StatusCode)
	}
}
