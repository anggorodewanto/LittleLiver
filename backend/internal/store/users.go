package store

import (
	"database/sql"
	"fmt"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

// UpsertUser inserts a new user or updates email/name if the google_id already exists.
// Returns the user record (with its assigned or existing ID).
func UpsertUser(db *sql.DB, googleID, email, name string) (*model.User, error) {
	id := model.NewULID()
	_, err := db.Exec(
		`INSERT INTO users (id, google_id, email, name) VALUES (?, ?, ?, ?)
		 ON CONFLICT(google_id) DO UPDATE SET email=excluded.email, name=excluded.name`,
		id, googleID, email, name,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert user: %w", err)
	}

	var u model.User
	err = db.QueryRow(
		"SELECT id, google_id, email, name FROM users WHERE google_id = ?",
		googleID,
	).Scan(&u.ID, &u.GoogleID, &u.Email, &u.Name)
	if err != nil {
		return nil, fmt.Errorf("query user after upsert: %w", err)
	}

	return &u, nil
}

// GetUserByID retrieves a user by their ID.
// Returns sql.ErrNoRows if the user does not exist.
func GetUserByID(db *sql.DB, id string) (*model.User, error) {
	var u model.User
	var tz sql.NullString
	err := db.QueryRow(
		"SELECT id, google_id, email, name, timezone FROM users WHERE id = ?", id,
	).Scan(&u.ID, &u.GoogleID, &u.Email, &u.Name, &tz)
	if err != nil {
		return nil, err
	}
	if tz.Valid {
		u.Timezone = &tz.String
	}
	return &u, nil
}

// UpdateUserTimezone sets the user's timezone.
func UpdateUserTimezone(db *sql.DB, id, timezone string) error {
	_, err := db.Exec("UPDATE users SET timezone = ? WHERE id = ?", timezone, id)
	if err != nil {
		return fmt.Errorf("update timezone: %w", err)
	}
	return nil
}

