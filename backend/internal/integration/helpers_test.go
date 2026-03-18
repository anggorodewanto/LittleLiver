package integration_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ablankz/LittleLiver/backend/internal/auth"
	"github.com/ablankz/LittleLiver/backend/internal/handler"
	"github.com/ablankz/LittleLiver/backend/internal/middleware"
	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

const testSessionSecret = "integration-test-secret"

// testClient wraps an HTTP client with session/CSRF info for a test user.
type testClient struct {
	t         *testing.T
	srv       *httptest.Server
	userID    string
	sessionID string
	csrfToken string
}

// setupIntegrationServer creates a test server and DB with the given handler options.
// Auth is handled by directly inserting sessions into the DB.
func setupIntegrationServer(t *testing.T, opts ...handler.Option) (*httptest.Server, *sql.DB, func()) {
	t.Helper()

	db := testutil.SetupTestDB(t)

	defaults := []handler.Option{
		handler.WithDB(db),
		handler.WithAuthConfig(auth.Config{
			ClientID:      "test-client-id",
			ClientSecret:  "test-client-secret",
			RedirectURL:   "http://localhost/auth/google/callback",
			TokenURL:      "http://localhost/fake-token",
			UserInfoURL:   "http://localhost/fake-userinfo",
			SessionSecret: testSessionSecret,
		}),
	}
	allOpts := append(defaults, opts...)
	mux := handler.NewMux(allOpts...)

	srv := httptest.NewServer(mux)
	cleanup := func() {
		srv.Close()
		db.Close()
	}
	return srv, db, cleanup
}

// newTestClient creates a user and authenticated client for integration tests.
func newTestClient(t *testing.T, srv *httptest.Server, db *sql.DB) *testClient {
	t.Helper()
	user := testutil.CreateTestUser(t, db)
	sess, err := store.CreateSession(db, user.ID)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	csrfToken := middleware.CSRFToken(sess.Token, testSessionSecret)
	return &testClient{
		t:         t,
		srv:       srv,
		userID:    user.ID,
		sessionID: sess.ID,
		csrfToken: csrfToken,
	}
}

// doJSON performs an HTTP request with auth headers and optional JSON body, returns status + decoded body.
func (tc *testClient) doJSON(method, path string, body any) (int, map[string]any) {
	tc.t.Helper()
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			tc.t.Fatalf("marshal body: %v", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, tc.srv.URL+path, bodyReader)
	if err != nil {
		tc.t.Fatalf("create request: %v", err)
	}
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: tc.sessionID})
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if method != http.MethodGet && method != http.MethodHead {
		req.Header.Set("X-CSRF-Token", tc.csrfToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		tc.t.Fatalf("do request %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		tc.t.Fatalf("read response: %v", err)
	}

	var result map[string]any
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &result); err != nil {
			result = map[string]any{"_raw": string(respBody)}
		}
	}
	return resp.StatusCode, result
}

// doJSONWithHeaders performs an HTTP request with auth headers, optional JSON body,
// and additional headers. Returns status and decoded body.
func (tc *testClient) doJSONWithHeaders(method, path string, body any, headers map[string]string) (int, map[string]any) {
	tc.t.Helper()
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			tc.t.Fatalf("marshal body: %v", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, tc.srv.URL+path, bodyReader)
	if err != nil {
		tc.t.Fatalf("create request: %v", err)
	}
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: tc.sessionID})
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if method != http.MethodGet && method != http.MethodHead {
		req.Header.Set("X-CSRF-Token", tc.csrfToken)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		tc.t.Fatalf("do request %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		tc.t.Fatalf("read response: %v", err)
	}

	var result map[string]any
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &result); err != nil {
			result = map[string]any{"_raw": string(respBody)}
		}
	}
	return resp.StatusCode, result
}

// doJSONList performs a GET and returns a list response (array at "data" key).
// Returns nil data for non-200 responses (e.g. 403).
func (tc *testClient) doJSONList(path string) (int, []any) {
	tc.t.Helper()
	req, err := http.NewRequest(http.MethodGet, tc.srv.URL+path, nil)
	if err != nil {
		tc.t.Fatalf("create request: %v", err)
	}
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: tc.sessionID})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		tc.t.Fatalf("do request GET %s: %v", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, nil
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		tc.t.Fatalf("decode list response: %v", err)
	}

	data, ok := result["data"].([]any)
	if !ok {
		data = []any{}
	}
	return resp.StatusCode, data
}

// doJSONArray performs a GET and returns the response as a raw JSON array.
func (tc *testClient) doJSONArray(path string) (int, []any) {
	tc.t.Helper()
	req, err := http.NewRequest(http.MethodGet, tc.srv.URL+path, nil)
	if err != nil {
		tc.t.Fatalf("create request: %v", err)
	}
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: tc.sessionID})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		tc.t.Fatalf("do request GET %s: %v", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, nil
	}

	var result []any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		tc.t.Fatalf("decode array response: %v", err)
	}
	return resp.StatusCode, result
}

// doRaw performs an HTTP request and returns the raw status code.
func (tc *testClient) doRaw(method, path string) int {
	tc.t.Helper()
	req, err := http.NewRequest(method, tc.srv.URL+path, nil)
	if err != nil {
		tc.t.Fatalf("create request: %v", err)
	}
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: tc.sessionID})
	if method != http.MethodGet && method != http.MethodHead {
		req.Header.Set("X-CSRF-Token", tc.csrfToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		tc.t.Fatalf("do request %s %s: %v", method, path, err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

// createBabyViaAPI creates a baby via the API and returns the baby ID.
func createBabyViaAPI(t *testing.T, client *testClient, name string) string {
	t.Helper()
	status, resp := client.doJSON(http.MethodPost, "/api/babies", map[string]any{
		"name":          name,
		"sex":           "female",
		"date_of_birth": "2025-01-01",
	})
	if status != http.StatusCreated {
		t.Fatalf("expected 201 creating baby, got %d: %v", status, resp)
	}
	return resp["id"].(string)
}

// verifyEntryCount checks that the given table has the expected number of rows for a baby.
func verifyEntryCount(t *testing.T, db *sql.DB, table, babyID string, expected int) {
	t.Helper()
	var count int
	err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE baby_id = ?", table), babyID).Scan(&count)
	if err != nil {
		t.Fatalf("query %s count: %v", table, err)
	}
	if count != expected {
		t.Errorf("expected %d entries in %s, got %d", expected, table, count)
	}
}

// verifyAnonymized checks that no rows in the table still have the original user ID
// in the specified column, and at least one row has the sentinel value.
func verifyAnonymized(t *testing.T, db *sql.DB, table, column, originalUserID, sentinel string) {
	t.Helper()

	var remaining int
	err := db.QueryRow(
		fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = ?", table, column),
		originalUserID,
	).Scan(&remaining)
	if err != nil {
		t.Fatalf("query %s.%s remaining: %v", table, column, err)
	}
	if remaining != 0 {
		t.Errorf("expected 0 rows with %s.%s=%q after anonymization, got %d", table, column, originalUserID, remaining)
	}

	var anonymized int
	err = db.QueryRow(
		fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = ?", table, column),
		sentinel,
	).Scan(&anonymized)
	if err != nil {
		t.Fatalf("query %s.%s anonymized: %v", table, column, err)
	}
	if anonymized == 0 {
		t.Errorf("expected at least 1 row with %s.%s=%q after anonymization, got 0", table, column, sentinel)
	}
}

// doRawBytes performs a GET request and returns raw response bytes and status code.
func (tc *testClient) doRawBytes(path string) (int, []byte) {
	tc.t.Helper()
	req, err := http.NewRequest(http.MethodGet, tc.srv.URL+path, nil)
	if err != nil {
		tc.t.Fatalf("create request: %v", err)
	}
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: tc.sessionID})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		tc.t.Fatalf("do request GET %s: %v", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		tc.t.Fatalf("read response body: %v", err)
	}
	return resp.StatusCode, body
}

// verifyNotAnonymized checks that the specified user's entries still have the
// original user ID in the specified column (not anonymized).
func verifyNotAnonymized(t *testing.T, db *sql.DB, table, column, userID string) {
	t.Helper()
	var count int
	err := db.QueryRow(
		fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = ?", table, column),
		userID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("query %s.%s for user %s: %v", table, column, userID, err)
	}
	if count == 0 {
		t.Errorf("expected at least 1 row with %s.%s=%q (should not be anonymized), got 0", table, column, userID)
	}
}
