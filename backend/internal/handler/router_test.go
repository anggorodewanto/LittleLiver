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

	// POST /auth/logout with valid session
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sess.ID})
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
