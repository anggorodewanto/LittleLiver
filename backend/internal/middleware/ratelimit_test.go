package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/middleware"
)

func TestRateLimit_UnderLimit_Succeeds(t *testing.T) {
	t.Parallel()

	rl := middleware.NewRateLimiter(100, time.Minute)
	handler := rl.Middleware("session_id")(okHandler())

	// Send 100 requests — all should succeed
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess-1"})
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}
}

func TestRateLimit_ExceedingLimit_Returns429(t *testing.T) {
	t.Parallel()

	rl := middleware.NewRateLimiter(100, time.Minute)
	handler := rl.Middleware("session_id")(okHandler())

	// Send 100 requests to exhaust the limit
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess-1"})
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// The 101st request should be rate-limited
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess-1"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 for request exceeding limit, got %d", rec.Code)
	}
}

func TestRateLimit_ResetsAfterWindow(t *testing.T) {
	t.Parallel()

	// Use a very short window for testing
	rl := middleware.NewRateLimiter(5, 50*time.Millisecond)
	handler := rl.Middleware("session_id")(okHandler())

	// Exhaust the limit
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess-1"})
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// Verify limit is hit
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess-1"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}

	// Wait for the window to expire
	time.Sleep(60 * time.Millisecond)

	// Should succeed again
	req = httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess-1"})
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 after window reset, got %d", rec.Code)
	}
}

func TestRateLimit_DifferentSessions_IndependentLimits(t *testing.T) {
	t.Parallel()

	rl := middleware.NewRateLimiter(5, time.Minute)
	handler := rl.Middleware("session_id")(okHandler())

	// Exhaust session A's limit
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess-a"})
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// Session A should be rate-limited
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess-a"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 for session A, got %d", rec.Code)
	}

	// Session B should still work
	req = httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess-b"})
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for session B (independent limit), got %d", rec.Code)
	}
}

func TestRateLimit_NoCookie_Returns429(t *testing.T) {
	t.Parallel()

	rl := middleware.NewRateLimiter(100, time.Minute)
	handler := rl.Middleware("session_id")(okHandler())

	// Request without session cookie should be rejected
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing session cookie, got %d", rec.Code)
	}
}
