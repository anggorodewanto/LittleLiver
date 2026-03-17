package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"log"
	"net/http"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

type contextKey string

const userContextKey contextKey = "user"
const sessionTokenContextKey contextKey = "session_token"

// UserFromContext extracts the authenticated user from the request context.
// Returns nil if no user is present.
func UserFromContext(ctx context.Context) *model.User {
	u, _ := ctx.Value(userContextKey).(*model.User)
	return u
}

// SessionTokenFromContext extracts the session token placed by the Auth middleware.
// Returns an empty string if not present.
func SessionTokenFromContext(ctx context.Context) string {
	t, _ := ctx.Value(sessionTokenContextKey).(string)
	return t
}

// Auth returns middleware that validates the session cookie, extends the sliding
// window, sets the user and session token on the request context, and updates
// the user's timezone from the X-Timezone header.
func Auth(db *sql.DB, cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(cookieName)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			sess, err := store.GetSessionByID(db, cookie.Value)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// Extend sliding window
			if err := store.ExtendSession(db, sess.ID); err != nil {
				log.Printf("auth: failed to extend session %s: %v", sess.ID, err)
			}

			// Get user
			user, err := store.GetUserByID(db, sess.UserID)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// Update timezone from header if present and valid
			tz := r.Header.Get("X-Timezone")
			if tz != "" {
				if _, err := time.LoadLocation(tz); err == nil {
					if err := store.UpdateUserTimezone(db, user.ID, tz); err != nil {
						log.Printf("auth: failed to update timezone for user %s: %v", user.ID, err)
					} else {
						user.Timezone = &tz
					}
				}
			}

			ctx := context.WithValue(r.Context(), userContextKey, user)
			ctx = context.WithValue(ctx, sessionTokenContextKey, sess.Token)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CSRFToken derives a CSRF token from the session token using HMAC-SHA256.
func CSRFToken(sessionToken, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(sessionToken))
	return hex.EncodeToString(mac.Sum(nil))
}

// CSRF returns middleware that validates the X-CSRF-Token header on state-changing
// requests (POST, PUT, DELETE). It first tries to read the session token from
// context (set by Auth middleware) to avoid a duplicate DB lookup. If not found
// in context, it falls back to reading the session cookie and querying the DB.
func CSRF(db *sql.DB, cookieName, secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Safe methods don't need CSRF validation
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			// Try context first (set by Auth middleware), fall back to DB lookup
			sessionToken := SessionTokenFromContext(r.Context())
			if sessionToken == "" {
				cookie, err := r.Cookie(cookieName)
				if err != nil {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}

				sess, err := store.GetSessionByID(db, cookie.Value)
				if err != nil {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
				sessionToken = sess.Token
			}

			expected := CSRFToken(sessionToken, secret)
			provided := r.Header.Get("X-CSRF-Token")

			if subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) != 1 {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
