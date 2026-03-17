package handler

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/ablankz/LittleLiver/backend/internal/middleware"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

// CSRFTokenHandler handles GET /api/csrf-token.
// It reads the session cookie, looks up the session token, and derives
// a CSRF token using HMAC-SHA256.
func CSRFTokenHandler(db *sql.DB, cookieName, secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		csrfToken := middleware.CSRFToken(sess.Token, secret)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"csrf_token": csrfToken}); err != nil {
			log.Printf("csrf-token: failed to write response: %v", err)
		}
	}
}

// meResponse is the JSON response for GET /api/me.
type meResponse struct {
	User   meUser   `json:"user"`
	Babies []meBaby `json:"babies"`
}

type meUser struct {
	ID       string  `json:"id"`
	Email    string  `json:"email"`
	Name     string  `json:"name"`
	Timezone *string `json:"timezone,omitempty"`
}

type meBaby struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Sex         string  `json:"sex"`
	DateOfBirth string  `json:"date_of_birth"`
	Notes       *string `json:"notes,omitempty"`
}

// MeHandler handles GET /api/me.
// Requires the Auth middleware to set the user in context.
func MeHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := middleware.UserFromContext(r.Context())
		if user == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		babies, err := store.GetBabiesByUserID(db, user.ID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		resp := meResponse{
			User: meUser{
				ID:       user.ID,
				Email:    user.Email,
				Name:     user.Name,
				Timezone: user.Timezone,
			},
			Babies: make([]meBaby, 0, len(babies)),
		}

		for _, b := range babies {
			resp.Babies = append(resp.Babies, meBaby{
				ID:          b.ID,
				Name:        b.Name,
				Sex:         b.Sex,
				DateOfBirth: b.DateOfBirth.Format("2006-01-02"),
				Notes:       b.Notes,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("me: failed to write response: %v", err)
		}
	}
}
