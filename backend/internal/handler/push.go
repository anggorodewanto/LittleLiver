package handler

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

// pushSubscribeRequest is the JSON request body for subscribing to push notifications.
type pushSubscribeRequest struct {
	Endpoint string `json:"endpoint"`
	P256dh   string `json:"p256dh"`
	Auth     string `json:"auth"`
}

// pushSubscribeResponse is the JSON response for a push subscription.
type pushSubscribeResponse struct {
	ID        string `json:"id"`
	Endpoint  string `json:"endpoint"`
	P256dh    string `json:"p256dh"`
	Auth      string `json:"auth"`
	CreatedAt string `json:"created_at"`
}

// pushUnsubscribeRequest is the JSON request body for unsubscribing from push notifications.
type pushUnsubscribeRequest struct {
	Endpoint string `json:"endpoint"`
}

// VAPIDKeyHandler handles GET /api/push/vapid-key.
// Returns the VAPID public key so the frontend can subscribe to push notifications.
func VAPIDKeyHandler(vapidPublicKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if vapidPublicKey == "" {
			http.Error(w, "push notifications not configured", http.StatusServiceUnavailable)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{
			"vapid_public_key": vapidPublicKey,
		})
	}
}

// SubscribePushHandler handles POST /api/push/subscribe.
func SubscribePushHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		var req pushSubscribeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.Endpoint == "" {
			http.Error(w, "endpoint is required", http.StatusBadRequest)
			return
		}
		if req.P256dh == "" {
			http.Error(w, "p256dh is required", http.StatusBadRequest)
			return
		}
		if req.Auth == "" {
			http.Error(w, "auth is required", http.StatusBadRequest)
			return
		}

		sub, err := store.UpsertPushSubscription(db, user.ID, req.Endpoint, req.P256dh, req.Auth)
		if err != nil {
			log.Printf("upsert push subscription: %v", err)
			http.Error(w, "failed to subscribe", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, pushSubscribeResponse{
			ID:        sub.ID,
			Endpoint:  sub.Endpoint,
			P256dh:    sub.P256dh,
			Auth:      sub.Auth,
			CreatedAt: sub.CreatedAt.Format(model.DateTimeFormat),
		})
	}
}

// UnsubscribePushHandler handles DELETE /api/push/subscribe.
func UnsubscribePushHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		var req pushUnsubscribeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.Endpoint == "" {
			http.Error(w, "endpoint is required", http.StatusBadRequest)
			return
		}

		err := store.DeletePushSubscription(db, user.ID, req.Endpoint)
		if err != nil {
			handleStoreError(w, err, "subscription not found")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
