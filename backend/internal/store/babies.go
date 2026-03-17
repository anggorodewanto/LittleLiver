package store

import (
	"database/sql"
	"fmt"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

// scanner is an interface satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

// scanBaby scans a single baby row from the given scanner and parses its
// time-related fields.
func scanBaby(s scanner) (*model.Baby, error) {
	var b model.Baby
	var dobStr string
	var diagStr, kasaiStr, notesStr sql.NullString
	var createdStr string

	err := s.Scan(&b.ID, &b.Name, &b.Sex, &dobStr, &diagStr, &kasaiStr,
		&b.DefaultCalPerFeed, &notesStr, &createdStr)
	if err != nil {
		return nil, err
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

	return &b, nil
}

// CreateBaby inserts a new baby and links the creator as a parent.
// Optional fields (diagnosisDate, kasaiDate, defaultCalPerFeed, notes) may be nil.
func CreateBaby(db *sql.DB, creatorID, name, sex, dob string, diagnosisDate, kasaiDate *string, defaultCalPerFeed *float64, notes *string) (*model.Baby, error) {
	id := model.NewULID()

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("create baby: begin tx: %w", err)
	}

	_, err = tx.Exec(
		`INSERT INTO babies (id, name, sex, date_of_birth, diagnosis_date, kasai_date, default_cal_per_feed, notes)
		 VALUES (?, ?, ?, ?, ?, ?, COALESCE(?, ?), ?)`,
		id, name, sex, dob, diagnosisDate, kasaiDate, defaultCalPerFeed, model.DefaultCalPerFeed, notes,
	)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("create baby: insert: %w", err)
	}

	_, err = tx.Exec(
		"INSERT INTO baby_parents (baby_id, user_id) VALUES (?, ?)",
		id, creatorID,
	)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("create baby: link parent: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("create baby: commit: %w", err)
	}

	return GetBabyByID(db, id)
}

// GetBabyByID retrieves a baby by its ID.
// Returns sql.ErrNoRows if the baby does not exist.
func GetBabyByID(db *sql.DB, id string) (*model.Baby, error) {
	row := db.QueryRow(
		`SELECT id, name, sex, date_of_birth, diagnosis_date, kasai_date,
		        default_cal_per_feed, notes, created_at
		 FROM babies WHERE id = ?`, id,
	)
	return scanBaby(row)
}

// IsParentOfBaby checks whether the given user is linked to the given baby.
func IsParentOfBaby(db *sql.DB, userID, babyID string) (bool, error) {
	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM baby_parents WHERE user_id = ? AND baby_id = ?",
		userID, babyID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("is parent of baby: %w", err)
	}
	return count > 0, nil
}

// UpdateBaby updates a baby's fields.
// Optional fields set to nil will be stored as NULL.
// Returns the updated baby or an error if the baby does not exist.
func UpdateBaby(db *sql.DB, id, name, sex, dob string, diagnosisDate, kasaiDate *string, defaultCalPerFeed *float64, notes *string) (*model.Baby, error) {
	res, err := db.Exec(
		`UPDATE babies SET name = ?, sex = ?, date_of_birth = ?,
		        diagnosis_date = ?, kasai_date = ?,
		        default_cal_per_feed = COALESCE(?, default_cal_per_feed),
		        notes = ?
		 WHERE id = ?`,
		name, sex, dob, diagnosisDate, kasaiDate, defaultCalPerFeed, notes, id,
	)
	if err != nil {
		return nil, fmt.Errorf("update baby: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("update baby: rows affected: %w", err)
	}
	if rows == 0 {
		return nil, fmt.Errorf("update baby: %w", sql.ErrNoRows)
	}

	return GetBabyByID(db, id)
}

// UnlinkParent removes the link between a user and a baby.
// If the user was the last parent, the baby is deleted (CASCADE removes associated data).
func UnlinkParent(db *sql.DB, babyID, userID string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("unlink parent: begin tx: %w", err)
	}

	_, err = tx.Exec(
		"DELETE FROM baby_parents WHERE baby_id = ? AND user_id = ?",
		babyID, userID,
	)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("unlink parent: delete link: %w", err)
	}

	var remaining int
	err = tx.QueryRow(
		"SELECT COUNT(*) FROM baby_parents WHERE baby_id = ?", babyID,
	).Scan(&remaining)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("unlink parent: count remaining: %w", err)
	}

	if remaining == 0 {
		_, err = tx.Exec("DELETE FROM babies WHERE id = ?", babyID)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("unlink parent: delete baby: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("unlink parent: commit: %w", err)
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
		b, err := scanBaby(rows)
		if err != nil {
			return nil, fmt.Errorf("scan baby: %w", err)
		}
		babies = append(babies, *b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return babies, nil
}
