package handler_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ablankz/LittleLiver/backend/internal/handler"
	"github.com/ablankz/LittleLiver/backend/internal/middleware"
	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

func TestSubscribePush_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	body := map[string]string{
		"endpoint": "https://push.example.com/sub1",
		"p256dh":   "test-p256dh-key",
		"auth":     "test-auth-key",
	}
	bodyBytes, _ := json.Marshal(body)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/push/subscribe")
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.SubscribePushHandler(db))))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["endpoint"] != "https://push.example.com/sub1" {
		t.Errorf("expected endpoint in response, got %v", resp["endpoint"])
	}
}

func TestSubscribePush_UpsertOnDuplicate(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	// Subscribe once
	body1 := map[string]string{
		"endpoint": "https://push.example.com/dup",
		"p256dh":   "old-key",
		"auth":     "old-auth",
	}
	bodyBytes1, _ := json.Marshal(body1)

	req1 := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/push/subscribe")
	req1.Body = io.NopCloser(bytes.NewReader(bodyBytes1))
	req1.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.SubscribePushHandler(db))))
	rec1 := httptest.NewRecorder()
	h.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusCreated {
		t.Fatalf("first subscribe: expected 201, got %d", rec1.Code)
	}

	// Subscribe again with same endpoint but new keys
	body2 := map[string]string{
		"endpoint": "https://push.example.com/dup",
		"p256dh":   "new-key",
		"auth":     "new-auth",
	}
	bodyBytes2, _ := json.Marshal(body2)

	req2 := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/push/subscribe")
	req2.Body = io.NopCloser(bytes.NewReader(bodyBytes2))
	req2.Header.Set("Content-Type", "application/json")

	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusCreated {
		t.Fatalf("second subscribe: expected 201, got %d. Body: %s", rec2.Code, rec2.Body.String())
	}

	var resp map[string]any
	_ = json.Unmarshal(rec2.Body.Bytes(), &resp)
	if resp["p256dh"] != "new-key" {
		t.Errorf("expected updated p256dh, got %v", resp["p256dh"])
	}

	// Verify only one subscription in DB
	subs, err := store.GetPushSubscriptionsByUserID(db, user.ID)
	if err != nil {
		t.Fatalf("get subs: %v", err)
	}
	if len(subs) != 1 {
		t.Errorf("expected 1 sub after upsert, got %d", len(subs))
	}
}

func TestSubscribePush_MissingFields(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	tests := []struct {
		name string
		body map[string]string
	}{
		{"missing endpoint", map[string]string{"p256dh": "k", "auth": "a"}},
		{"missing p256dh", map[string]string{"endpoint": "https://e.com", "auth": "a"}},
		{"missing auth", map[string]string{"endpoint": "https://e.com", "p256dh": "k"}},
	}

	for _, tc := range tests {
		bodyBytes, _ := json.Marshal(tc.body)
		req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/push/subscribe")
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		authMw := middleware.Auth(db, testCookieName)
		csrfMw := middleware.CSRF(db, testCookieName, testSecret)
		h := authMw(csrfMw(http.HandlerFunc(handler.SubscribePushHandler(db))))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d. Body: %s", tc.name, rec.Code, rec.Body.String())
		}
	}
}

func TestUnsubscribePush_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	// First subscribe
	_, err := store.UpsertPushSubscription(db, user.ID, "https://push.example.com/unsub", "k", "a")
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	body := map[string]string{"endpoint": "https://push.example.com/unsub"}
	bodyBytes, _ := json.Marshal(body)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodDelete, "/api/push/subscribe")
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UnsubscribePushHandler(db))))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// Verify deleted
	subs, err := store.GetPushSubscriptionsByUserID(db, user.ID)
	if err != nil {
		t.Fatalf("get subs: %v", err)
	}
	if len(subs) != 0 {
		t.Errorf("expected 0 subs after unsubscribe, got %d", len(subs))
	}
}

func TestUnsubscribePush_NotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	body := map[string]string{"endpoint": "https://push.example.com/nonexistent"}
	bodyBytes, _ := json.Marshal(body)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodDelete, "/api/push/subscribe")
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UnsubscribePushHandler(db))))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUnsubscribePush_MissingEndpoint(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	body := map[string]string{}
	bodyBytes, _ := json.Marshal(body)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodDelete, "/api/push/subscribe")
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UnsubscribePushHandler(db))))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestSubscribePush_InvalidJSON(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/push/subscribe")
	req.Body = io.NopCloser(bytes.NewReader([]byte("{invalid json")))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.SubscribePushHandler(db))))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUnsubscribePush_InvalidJSON(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodDelete, "/api/push/subscribe")
	req.Body = io.NopCloser(bytes.NewReader([]byte("{invalid")))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UnsubscribePushHandler(db))))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestSubscribePush_Unauthorized(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	body := map[string]string{
		"endpoint": "https://push.example.com/unauth",
		"p256dh":   "k",
		"auth":     "a",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/push/subscribe", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.SubscribePushHandler(db))))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}
