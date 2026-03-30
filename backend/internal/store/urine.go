package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

const urineColumns = `id, baby_id, logged_by, updated_by, timestamp,
	color, volume_ml, notes, created_at, updated_at`

// scanUrine scans a single urine row from the given scanner.
func scanUrine(s scanner) (*model.Urine, error) {
	var u model.Urine
	var updatedBy, color, notes sql.NullString
	var volumeMl sql.NullFloat64
	var tsStr, createdStr, updatedStr string

	err := s.Scan(
		&u.ID, &u.BabyID, &u.LoggedBy, &updatedBy, &tsStr,
		&color, &volumeMl, &notes, &createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}

	u.Timestamp, u.CreatedAt, u.UpdatedAt, err = parseMetricTimes(tsStr, createdStr, updatedStr)
	if err != nil {
		return nil, err
	}

	u.UpdatedBy = nullStr(updatedBy)
	u.Color = nullStr(color)
	u.VolumeMl = nullFloat(volumeMl)
	u.Notes = nullStr(notes)

	return &u, nil
}

// CreateUrine inserts a new urine entry and returns it.
// If volumeMl is non-nil, also creates a linked fluid_log entry.
func CreateUrine(db *sql.DB, babyID, loggedBy, timestamp string, color *string, volumeMl *float64, notes *string) (*model.Urine, error) {
	id := model.NewULID()

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("create urine: begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO urine (id, baby_id, logged_by, timestamp, color, volume_ml, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, babyID, loggedBy, timestamp, color, volumeMl, notes,
	)
	if err != nil {
		return nil, fmt.Errorf("create urine: %w", err)
	}

	if volumeMl != nil {
		srcType := "urine"
		fluidID := model.NewULID()
		if err := createFluidLogTx(tx, fluidID, babyID, loggedBy, timestamp, "output", "urine", volumeMl, &srcType, &id, notes); err != nil {
			return nil, fmt.Errorf("create urine: fluid_log: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("create urine: commit: %w", err)
	}

	return GetUrineByID(db, babyID, id)
}

// GetUrineByID retrieves a urine entry by its ID, scoped to the given baby.
func GetUrineByID(db *sql.DB, babyID, urineID string) (*model.Urine, error) {
	row := db.QueryRow(
		`SELECT `+urineColumns+` FROM urine WHERE id = ? AND baby_id = ?`,
		urineID, babyID,
	)
	return scanUrine(row)
}

// ListUrine returns a paginated list of urine entries for a baby in ULID descending order.
func ListUrine(db *sql.DB, babyID string, from, to, cursor *string, limit int) (*model.MetricPage[model.Urine], error) {
	return ListUrineWithTZ(db, babyID, from, to, cursor, limit, time.UTC)
}

// ListUrineWithTZ returns a paginated list of urine entries with timezone-aware date filtering.
func ListUrineWithTZ(db *sql.DB, babyID string, from, to, cursor *string, limit int, loc *time.Location) (*model.MetricPage[model.Urine], error) {
	return listMetricWithTZ(db, "urine", urineColumns, babyID, from, to, cursor, limit, loc, scanUrine, func(u *model.Urine) string { return u.ID })
}

// UpdateUrine updates a urine entry. Also updates the linked fluid_log entry.
func UpdateUrine(db *sql.DB, babyID, urineID, updatedBy, timestamp string, color *string, volumeMl *float64, notes *string) (*model.Urine, error) {
	existing, err := GetUrineByID(db, babyID, urineID)
	if err != nil {
		return nil, fmt.Errorf("update urine: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("update urine: begin tx: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		`UPDATE urine SET
			updated_by = ?, timestamp = ?,
			color = ?, volume_ml = ?, notes = ?,
			updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND baby_id = ?`,
		updatedBy, timestamp,
		color, volumeMl, notes,
		urineID, babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("update urine: %w", err)
	}

	if err := checkRowsAffected(res, "update urine"); err != nil {
		return nil, err
	}

	// Delete old fluid_log, recreate if volume present
	if err := deleteFluidLogBySourceTx(tx, "urine", urineID); err != nil {
		return nil, fmt.Errorf("update urine: fluid_log delete: %w", err)
	}
	if volumeMl != nil {
		srcType := "urine"
		fluidID := model.NewULID()
		user := existing.LoggedBy
		if updatedBy != "" {
			user = updatedBy
		}
		if err := createFluidLogTx(tx, fluidID, babyID, user, timestamp, "output", "urine", volumeMl, &srcType, &urineID, notes); err != nil {
			return nil, fmt.Errorf("update urine: fluid_log create: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("update urine: commit: %w", err)
	}

	return GetUrineByID(db, babyID, urineID)
}

// DeleteUrine hard-deletes a urine entry and its linked fluid_log entry.
func DeleteUrine(db *sql.DB, babyID, urineID string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("delete urine: begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := deleteFluidLogBySourceTx(tx, "urine", urineID); err != nil {
		return fmt.Errorf("delete urine: fluid_log: %w", err)
	}

	res, err := tx.Exec("DELETE FROM urine WHERE id = ? AND baby_id = ?", urineID, babyID)
	if err != nil {
		return fmt.Errorf("delete urine: %w", err)
	}
	if err := checkRowsAffected(res, "delete urine"); err != nil {
		return err
	}

	return tx.Commit()
}
