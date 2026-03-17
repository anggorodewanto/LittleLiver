package middleware

import (
	"net/http"
	"sync"
	"time"
)

// sessionBucket tracks request count and window start for a single session.
type sessionBucket struct {
	count       int
	windowStart time.Time
}

// RateLimiter tracks per-session request counts within a fixed window.
type RateLimiter struct {
	limit    int
	window   time.Duration
	mu       sync.Mutex
	sessions map[string]*sessionBucket
}

// NewRateLimiter creates a rate limiter that allows limit requests per window.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		limit:    limit,
		window:   window,
		sessions: make(map[string]*sessionBucket),
	}
}

// allow checks whether the given session key is within its rate limit.
// Returns true if the request is allowed, false if rate-limited.
func (rl *RateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.sessions[key]
	if !ok {
		rl.sessions[key] = &sessionBucket{count: 1, windowStart: now}
		return true
	}

	// Reset window if expired
	if now.Sub(b.windowStart) >= rl.window {
		b.count = 1
		b.windowStart = now
		return true
	}

	if b.count >= rl.limit {
		return false
	}

	b.count++
	return true
}

// Middleware returns HTTP middleware that enforces per-session rate limiting.
// It reads the session ID from the cookie specified by cookieName.
func (rl *RateLimiter) Middleware(cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(cookieName)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			if !rl.allow(cookie.Value) {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
