package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

// SessionDuration is how long a session lasts (30-day sliding window).
const SessionDuration = 30 * 24 * time.Hour

// generateToken creates a cryptographically secure random token (32 bytes, hex-encoded).
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// CreateSession creates a new session for the given user with a 30-day expiry.
func CreateSession(db *sql.DB, userID string) (*model.Session, error) {
	id := model.NewULID()
	token, err := generateToken()
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().Add(SessionDuration).UTC()

	_, err = db.Exec(
		"INSERT INTO sessions (id, user_id, token, expires_at) VALUES (?, ?, ?, ?)",
		id, userID, token, expiresAt.Format(time.DateTime),
	)
	if err != nil {
		return nil, fmt.Errorf("insert session: %w", err)
	}

	return &model.Session{
		ID:        id,
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now().UTC(),
	}, nil
}

// GetSessionByID retrieves a non-expired session by its ID.
// Returns sql.ErrNoRows if the session does not exist or is expired.
func GetSessionByID(db *sql.DB, id string) (*model.Session, error) {
	var s model.Session
	var expiresStr, createdStr string
	err := db.QueryRow(
		"SELECT id, user_id, token, expires_at, created_at FROM sessions WHERE id = ? AND expires_at > ?",
		id, time.Now().UTC().Format(time.DateTime),
	).Scan(&s.ID, &s.UserID, &s.Token, &expiresStr, &createdStr)
	if err != nil {
		return nil, err
	}

	s.ExpiresAt, err = parseTime(expiresStr)
	if err != nil {
		return nil, fmt.Errorf("parse expires_at: %w", err)
	}
	s.CreatedAt, err = parseTime(createdStr)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}

	return &s, nil
}

// DeleteSession removes a session by ID.
func DeleteSession(db *sql.DB, id string) error {
	_, err := db.Exec("DELETE FROM sessions WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// parseTime tries multiple time formats used by SQLite.
func parseTime(s string) (time.Time, error) {
	for _, layout := range []string{
		time.DateTime,
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time %q", s)
}

// ExtendSession resets the session's expires_at to 30 days from now (sliding window).
func ExtendSession(db *sql.DB, id string) error {
	expiresAt := time.Now().Add(SessionDuration).UTC()
	_, err := db.Exec(
		"UPDATE sessions SET expires_at = ? WHERE id = ?",
		expiresAt.Format(time.DateTime), id,
	)
	if err != nil {
		return fmt.Errorf("extend session: %w", err)
	}
	return nil
}
