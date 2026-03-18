package store

import (
	"database/sql"
	"fmt"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

// UpsertPushSubscription creates or updates a push subscription.
// On conflict (duplicate endpoint), it updates the p256dh and auth keys.
func UpsertPushSubscription(db *sql.DB, userID, endpoint, p256dh, auth string) (*model.PushSubscription, error) {
	id := model.NewULID()

	_, err := db.Exec(
		`INSERT INTO push_subscriptions (id, user_id, endpoint, p256dh, auth)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT (endpoint) DO UPDATE SET
		   p256dh = excluded.p256dh,
		   auth = excluded.auth`,
		id, userID, endpoint, p256dh, auth,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert push subscription: %w", err)
	}

	return getPushSubscriptionByEndpoint(db, endpoint)
}

// DeletePushSubscription removes a push subscription by endpoint, scoped to the user.
func DeletePushSubscription(db *sql.DB, userID, endpoint string) error {
	res, err := db.Exec(
		"DELETE FROM push_subscriptions WHERE user_id = ? AND endpoint = ?",
		userID, endpoint,
	)
	if err != nil {
		return fmt.Errorf("delete push subscription: %w", err)
	}
	return checkRowsAffected(res, "delete push subscription")
}

// GetPushSubscriptionsByUserID returns all push subscriptions for a user.
func GetPushSubscriptionsByUserID(db *sql.DB, userID string) ([]model.PushSubscription, error) {
	rows, err := db.Query(
		"SELECT id, user_id, endpoint, p256dh, auth, created_at FROM push_subscriptions WHERE user_id = ? ORDER BY created_at DESC",
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list push subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []model.PushSubscription
	for rows.Next() {
		sub, err := scanPushSubscription(rows)
		if err != nil {
			return nil, fmt.Errorf("scan push subscription: %w", err)
		}
		subs = append(subs, *sub)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	if subs == nil {
		subs = make([]model.PushSubscription, 0)
	}
	return subs, nil
}

// DeleteAllPushSubscriptionsByUserID removes all push subscriptions for a user.
func DeleteAllPushSubscriptionsByUserID(db *sql.DB, userID string) error {
	_, err := db.Exec("DELETE FROM push_subscriptions WHERE user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("delete all push subscriptions: %w", err)
	}
	return nil
}

// getPushSubscriptionByEndpoint retrieves a push subscription by its endpoint.
func getPushSubscriptionByEndpoint(db *sql.DB, endpoint string) (*model.PushSubscription, error) {
	row := db.QueryRow(
		"SELECT id, user_id, endpoint, p256dh, auth, created_at FROM push_subscriptions WHERE endpoint = ?",
		endpoint,
	)
	return scanPushSubscription(row)
}

// scanPushSubscription scans a single push subscription row from the given scanner.
func scanPushSubscription(s scanner) (*model.PushSubscription, error) {
	var sub model.PushSubscription
	var createdStr string

	err := s.Scan(&sub.ID, &sub.UserID, &sub.Endpoint, &sub.P256dh, &sub.Auth, &createdStr)
	if err != nil {
		return nil, err
	}

	sub.CreatedAt, err = parseTime(createdStr)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}

	return &sub, nil
}
