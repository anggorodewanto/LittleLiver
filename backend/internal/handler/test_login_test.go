package handler_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/ablankz/LittleLiver/backend/internal/handler"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

func TestTestLoginHandler_NotAvailableWithoutTestMode(t *testing.T) {
	os.Unsetenv("TEST_MODE")
	db := testutil.SetupTestDB(t)
	defer db.Close()

	h := handler.TestLoginHandler(db)
	body := `{"google_id":"g1","email":"a@b.com","name":"Test"}`
	req := httptest.NewRequest("POST", "/api/test/login", strings.NewReader(body))
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 without TEST_MODE, got %d", w.Code)
	}
}

func TestTestLoginHandler_CreatesSessionWithTestMode(t *testing.T) {
	os.Setenv("TEST_MODE", "1")
	defer os.Unsetenv("TEST_MODE")

	db := testutil.SetupTestDB(t)
	defer db.Close()

	h := handler.TestLoginHandler(db)
	body := `{"google_id":"g1","email":"a@b.com","name":"Test User"}`
	req := httptest.NewRequest("POST", "/api/test/login", strings.NewReader(body))
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Check session cookie is set
	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "session_id" && c.Value != "" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected session_id cookie to be set")
	}
}

func TestTestLoginHandler_MissingFields(t *testing.T) {
	os.Setenv("TEST_MODE", "1")
	defer os.Unsetenv("TEST_MODE")

	db := testutil.SetupTestDB(t)
	defer db.Close()

	h := handler.TestLoginHandler(db)
	body := `{"google_id":"","email":"","name":""}`
	req := httptest.NewRequest("POST", "/api/test/login", strings.NewReader(body))
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty fields, got %d", w.Code)
	}
}
