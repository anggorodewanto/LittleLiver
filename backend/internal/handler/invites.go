package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/ablankz/LittleLiver/backend/internal/middleware"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

// inviteResponse is the JSON response for a generated invite code.
type inviteResponse struct {
	Code      string `json:"code"`
	ExpiresAt string `json:"expires_at"`
}

// joinRequest is the JSON request body for joining a baby via invite code.
type joinRequest struct {
	Code string `json:"code"`
}

// joinResponse is the JSON response for joining a baby.
type joinResponse struct {
	BabyID  string `json:"baby_id"`
	Message string `json:"message"`
}

// CreateInviteHandler handles POST /api/babies/:id/invite.
// Generates a 6-digit invite code for the baby.
// Requires the authenticated user to be a parent of the baby.
func CreateInviteHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := middleware.UserFromContext(r.Context())
		if user == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		babyID := extractBabyID(r)
		if babyID == "" {
			http.Error(w, "missing baby ID", http.StatusBadRequest)
			return
		}

		// Check baby exists
		_, err := store.GetBabyByID(db, babyID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "baby not found", http.StatusNotFound)
				return
			}
			log.Printf("create invite: get baby: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		// Check authorization
		linked, err := store.IsParentOfBaby(db, user.ID, babyID)
		if err != nil {
			log.Printf("create invite: check parent: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !linked {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		invite, err := store.CreateInvite(db, babyID, user.ID)
		if err != nil {
			log.Printf("create invite: %v", err)
			http.Error(w, "failed to create invite", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(inviteResponse{
			Code:      invite.Code,
			ExpiresAt: invite.ExpiresAt.Format("2006-01-02T15:04:05Z"),
		}); err != nil {
			log.Printf("create invite: encode response: %v", err)
		}
	}
}

// JoinBabyHandler handles POST /api/babies/join.
// Redeems an invite code, linking the authenticated user to the baby.
func JoinBabyHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := middleware.UserFromContext(r.Context())
		if user == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var req joinRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.Code == "" {
			http.Error(w, "code is required", http.StatusBadRequest)
			return
		}

		babyID, err := store.RedeemInvite(db, req.Code, user.ID)
		if errors.Is(err, store.ErrAlreadyLinked) {
			w.Header().Set("Content-Type", "application/json")
			if encErr := json.NewEncoder(w).Encode(joinResponse{
				BabyID:  babyID,
				Message: "already linked to this baby",
			}); encErr != nil {
				log.Printf("join baby: encode response: %v", encErr)
			}
			return
		}
		if errors.Is(err, store.ErrInvalidInvite) {
			http.Error(w, "invalid or expired code", http.StatusBadRequest)
			return
		}
		if err != nil {
			log.Printf("join baby: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(joinResponse{
			BabyID:  babyID,
			Message: "successfully joined",
		}); err != nil {
			log.Printf("join baby: encode response: %v", err)
		}
	}
}
