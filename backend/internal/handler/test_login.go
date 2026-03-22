package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"

	"github.com/ablankz/LittleLiver/backend/internal/auth"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

// TestLoginHandler creates a user and session without OAuth, for E2E testing only.
// Only works when TEST_MODE env var is set to "1".
func TestLoginHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("TEST_MODE") != "1" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		var req struct {
			GoogleID string `json:"google_id"`
			Email    string `json:"email"`
			Name     string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.GoogleID == "" || req.Email == "" || req.Name == "" {
			http.Error(w, "google_id, email, and name are required", http.StatusBadRequest)
			return
		}

		user, err := store.UpsertUser(db, req.GoogleID, req.Email, req.Name)
		if err != nil {
			http.Error(w, "failed to create user", http.StatusInternalServerError)
			return
		}

		session, err := store.CreateSession(db, user.ID)
		if err != nil {
			http.Error(w, "failed to create session", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     auth.CookieName,
			Value:    session.ID,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int(store.SessionDuration.Seconds()),
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"user_id":    user.ID,
			"session_id": session.ID,
		})
	}
}
