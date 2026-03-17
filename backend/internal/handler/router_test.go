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
