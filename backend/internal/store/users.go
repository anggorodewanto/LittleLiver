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

// GetBabiesByUserID returns all babies linked to the given user.
func GetBabiesByUserID(db *sql.DB, userID string) ([]model.Baby, error) {
	rows, err := db.Query(
		`SELECT b.id, b.name, b.sex, b.date_of_birth, b.diagnosis_date, b.kasai_date,
		        b.default_cal_per_feed, b.notes, b.created_at
		 FROM babies b
		 JOIN baby_parents bp ON b.id = bp.baby_id
		 WHERE bp.user_id = ?`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query babies: %w", err)
	}
	defer rows.Close()

	var babies []model.Baby
	for rows.Next() {
		var b model.Baby
		var dobStr string
		var diagStr, kasaiStr, notesStr sql.NullString
		var createdStr string
		err := rows.Scan(&b.ID, &b.Name, &b.Sex, &dobStr, &diagStr, &kasaiStr,
			&b.DefaultCalPerFeed, &notesStr, &createdStr)
		if err != nil {
			return nil, fmt.Errorf("scan baby: %w", err)
		}
		b.DateOfBirth, err = parseTime(dobStr)
		if err != nil {
			return nil, fmt.Errorf("parse date_of_birth: %w", err)
		}
		if diagStr.Valid {
			t, err := parseTime(diagStr.String)
			if err != nil {
				return nil, fmt.Errorf("parse diagnosis_date: %w", err)
			}
			b.DiagnosisDate = &t
		}
		if kasaiStr.Valid {
			t, err := parseTime(kasaiStr.String)
			if err != nil {
				return nil, fmt.Errorf("parse kasai_date: %w", err)
			}
			b.KasaiDate = &t
		}
		if notesStr.Valid {
			b.Notes = &notesStr.String
		}
		b.CreatedAt, err = parseTime(createdStr)
		if err != nil {
			return nil, fmt.Errorf("parse created_at: %w", err)
		}
		babies = append(babies, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return babies, nil
}
