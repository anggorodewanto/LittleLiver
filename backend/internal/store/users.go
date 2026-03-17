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
