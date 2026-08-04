package handler_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ablankz/LittleLiver/backend/internal/handler"
	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

func TestNewMux_HealthRoute(t *testing.T) {
	t.Parallel()

	mux := handler.NewMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if body["status"] != "ok" {
		t.Fatalf("expected status=ok, got %q", body["status"])
	}
}

func TestNewMux_HealthRoute_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	mux := handler.NewMux()
	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestNewMux_UnknownRoute_Returns404(t *testing.T) {
	t.Parallel()

	mux := handler.NewMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/nonexistent")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestNewMux_StaticDir_ServesFiles(t *testing.T) {
	// Create a temp directory with a test file
	tmpDir := t.TempDir()
	testContent := "hello from static"
	if err := os.WriteFile(filepath.Join(tmpDir, "index.html"), []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	t.Setenv("STATIC_DIR", tmpDir)

	mux := handler.NewMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/index.html")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	if string(body) != testContent {
		t.Fatalf("expected body %q, got %q", testContent, string(body))
	}
}

func TestNewMux_StaticDir_NonexistentDir_HealthStillWorks(t *testing.T) {
	t.Setenv("STATIC_DIR", "/tmp/nonexistent-dir-littleliver-test")

	mux := handler.NewMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Health endpoint should still work
	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if body["status"] != "ok" {
		t.Fatalf("expected status=ok, got %q", body["status"])
	}
}

func TestNewMux_StaticDir_HealthTakesPriority(t *testing.T) {
	// Create a temp static dir with a file named "health" to test priority
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "health"), []byte("static health file"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	t.Setenv("STATIC_DIR", tmpDir)

	mux := handler.NewMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Should return JSON health response, not the static file
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("expected JSON from health handler, got non-JSON (static file may have been served): %v", err)
	}

	if body["status"] != "ok" {
		t.Fatalf("expected status=ok, got %q", body["status"])
	}
}

func TestNewMux_AuthRoutes_RegisteredWhenConfigured(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	t.Setenv("GOOGLE_CLIENT_ID", "test-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "test-secret")
	t.Setenv("BASE_URL", "http://localhost:8080")

	mux := handler.NewMux(handler.WithDB(db))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// GET /auth/google/login should redirect (302) to Google
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	resp, err := client.Get(srv.URL + "/auth/google/login")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected 302, got %d", resp.StatusCode)
	}

	loc := resp.Header.Get("Location")
	if loc == "" {
		t.Fatal("expected Location header")
	}
}

func TestNewMux_APIRoutes_CSRFTokenAndMe(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	t.Setenv("GOOGLE_CLIENT_ID", "test-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "test-secret")
	t.Setenv("SESSION_SECRET", "test-session-secret")
	t.Setenv("BASE_URL", "http://localhost:8080")

	// Create a user and session
	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	sess, err := store.CreateSession(db, "u1")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	mux := handler.NewMux(handler.WithDB(db))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}

	// GET /api/csrf-token with valid session should return token
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/csrf-token", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sess.ID})
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("csrf-token request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200 for csrf-token, got %d. Body: %s", resp.StatusCode, body)
	}

	var csrfBody map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&csrfBody); err != nil {
		t.Fatalf("decode csrf response failed: %v", err)
	}
	if csrfBody["csrf_token"] == "" {
		t.Fatal("expected non-empty csrf_token")
	}

	// GET /api/me with valid session should return user info
	req, _ = http.NewRequest(http.MethodGet, srv.URL+"/api/me", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sess.ID})
	resp2, err := client.Do(req)
	if err != nil {
		t.Fatalf("me request failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp2.Body)
		t.Fatalf("expected 200 for /api/me, got %d. Body: %s", resp2.StatusCode, body)
	}

	// GET /api/me without session should return 401
	req, _ = http.NewRequest(http.MethodGet, srv.URL+"/api/me", nil)
	resp3, err := client.Do(req)
	if err != nil {
		t.Fatalf("me request failed: %v", err)
	}
	defer resp3.Body.Close()

	if resp3.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for /api/me without session, got %d", resp3.StatusCode)
	}
}

func TestNewMux_Logout_ClearsSession_Integration(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	t.Setenv("GOOGLE_CLIENT_ID", "test-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "test-secret")
	t.Setenv("SESSION_SECRET", "test-session-secret")

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	sess, err := store.CreateSession(db, "u1")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	mux := handler.NewMux(handler.WithDB(db))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}

	// Fetch CSRF token first (required since logout is now behind CSRF middleware)
	csrfReq, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/csrf-token", nil)
	csrfReq.AddCookie(&http.Cookie{Name: "session_id", Value: sess.ID})
	csrfResp, err := client.Do(csrfReq)
	if err != nil {
		t.Fatalf("csrf-token request failed: %v", err)
	}
	var csrfBody map[string]string
	json.NewDecoder(csrfResp.Body).Decode(&csrfBody)
	csrfResp.Body.Close()
	csrfToken := csrfBody["csrf_token"]

	// POST /auth/logout with valid session and CSRF token
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sess.ID})
	req.Header.Set("X-CSRF-Token", csrfToken)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("logout request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 204 for logout, got %d. Body: %s", resp.StatusCode, body)
	}

	// Verify session is deleted
	_, err = store.GetSessionByID(db, sess.ID)
	if err == nil {
		t.Fatal("expected session to be deleted after logout")
	}
}

// staticTestDir creates a static dir containing an index.html, a hashed
// immutable asset, a plain asset, and an asset directory without an index.
func staticTestDir(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, "index.html"), []byte("<html>app shell</html>"), 0644); err != nil {
		t.Fatalf("failed to write index.html: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "manifest.json"), []byte(`{"name":"LittleLiver"}`), 0644); err != nil {
		t.Fatalf("failed to write manifest.json: %v", err)
	}

	immutableDir := filepath.Join(tmpDir, "_app", "immutable", "chunks")
	if err := os.MkdirAll(immutableDir, 0755); err != nil {
		t.Fatalf("failed to create immutable dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(immutableDir, "app.abc123.js"), []byte("export default 1"), 0644); err != nil {
		t.Fatalf("failed to write immutable asset: %v", err)
	}

	iconsDir := filepath.Join(tmpDir, "icons")
	if err := os.MkdirAll(iconsDir, 0755); err != nil {
		t.Fatalf("failed to create icons dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(iconsDir, "icon-192.png"), []byte("png"), 0644); err != nil {
		t.Fatalf("failed to write icon: %v", err)
	}

	return tmpDir
}

// The PWA start_url is "/". Without an explicit Cache-Control, browsers apply
// heuristic freshness from Last-Modified and serve a stale index.html — which
// pins the app to an old JS bundle until the user manually refreshes.
func TestNewMux_StaticDir_RootSendsNoCache(t *testing.T) {
	t.Setenv("STATIC_DIR", staticTestDir(t))

	mux := handler.NewMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	if got := resp.Header.Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("expected Cache-Control=no-cache on /, got %q", got)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	if string(body) != "<html>app shell</html>" {
		t.Fatalf("expected app shell body, got %q", string(body))
	}
}

func TestNewMux_StaticDir_NonImmutableFileSendsNoCache(t *testing.T) {
	t.Setenv("STATIC_DIR", staticTestDir(t))

	mux := handler.NewMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	for _, path := range []string{"/index.html", "/manifest.json"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("request %s failed: %v", path, err)
		}
		resp.Body.Close()

		if got := resp.Header.Get("Cache-Control"); got != "no-cache" {
			t.Fatalf("expected Cache-Control=no-cache on %s, got %q", path, got)
		}
	}
}

func TestNewMux_StaticDir_ImmutableAssetStaysCacheable(t *testing.T) {
	t.Setenv("STATIC_DIR", staticTestDir(t))

	mux := handler.NewMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/_app/immutable/chunks/app.abc123.js")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	const want = "public, max-age=31536000, immutable"
	if got := resp.Header.Get("Cache-Control"); got != want {
		t.Fatalf("expected Cache-Control=%q, got %q", want, got)
	}
}

// Directory paths must fall through to the SPA shell rather than reaching
// http.FileServer, which renders a browsable directory listing.
func TestNewMux_StaticDir_DirectoryPathServesAppShell(t *testing.T) {
	t.Setenv("STATIC_DIR", staticTestDir(t))

	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	mux := handler.NewMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := client.Get(srv.URL + "/icons")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	if string(body) != "<html>app shell</html>" {
		t.Fatalf("expected app shell body, got %q", string(body))
	}
}
