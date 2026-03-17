package handler

import (
	"database/sql"
	"net/http"

	"github.com/ablankz/LittleLiver/backend/internal/middleware"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

// CSRFTokenHandler handles GET /api/csrf-token.
// Requires the Auth middleware to set the session token in context.
func CSRFTokenHandler(secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionToken := middleware.SessionTokenFromContext(r.Context())
		if sessionToken == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		csrfToken := middleware.CSRFToken(sessionToken, secret)

		writeJSON(w, http.StatusOK, map[string]string{"csrf_token": csrfToken})
	}
}

// meResponse is the JSON response for GET /api/me.
type meResponse struct {
	User   meUser         `json:"user"`
	Babies []babyResponse `json:"babies"`
}

type meUser struct {
	ID       string  `json:"id"`
	Email    string  `json:"email"`
	Name     string  `json:"name"`
	Timezone *string `json:"timezone,omitempty"`
}

// MeHandler handles GET /api/me.
// Requires the Auth middleware to set the user in context.
func MeHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
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
			Babies: make([]babyResponse, 0, len(babies)),
		}

		for i := range babies {
			resp.Babies = append(resp.Babies, toBabyResponse(&babies[i]))
		}

		writeJSON(w, http.StatusOK, resp)
	}
}
