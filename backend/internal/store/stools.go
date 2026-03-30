package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

const stoolColumns = `id, baby_id, logged_by, updated_by, timestamp,
	color_rating, color_label, consistency, volume_estimate, volume_ml, photo_keys,
	notes, created_at, updated_at`

// scanStool scans a single stool row from the given scanner.
func scanStool(s scanner) (*model.Stool, error) {
	var st model.Stool
	var updatedBy, colorLabel, consistency, volumeEstimate, photoKeys, notes sql.NullString
	var volumeMl sql.NullFloat64
	var tsStr, createdStr, updatedStr string

	err := s.Scan(
		&st.ID, &st.BabyID, &st.LoggedBy, &updatedBy, &tsStr,
		&st.ColorRating, &colorLabel, &consistency, &volumeEstimate, &volumeMl, &photoKeys,
		&notes, &createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}

	st.Timestamp, st.CreatedAt, st.UpdatedAt, err = parseMetricTimes(tsStr, createdStr, updatedStr)
	if err != nil {
		return nil, err
	}

	st.UpdatedBy = nullStr(updatedBy)
	st.ColorLabel = nullStr(colorLabel)
	st.Consistency = nullStr(consistency)
	st.VolumeEstimate = nullStr(volumeEstimate)
	st.VolumeMl = nullFloat(volumeMl)
	st.PhotoKeys = nullStr(photoKeys)
	st.Notes = nullStr(notes)

	return &st, nil
}

// CreateStool inserts a new stool entry and returns it.
func CreateStool(db *sql.DB, babyID, loggedBy, timestamp string, colorRating int, colorLabel, consistency, volumeEstimate *string, volumeMl *float64, notes *string) (*model.Stool, error) {
	return CreateStoolWithPhotos(db, babyID, loggedBy, timestamp, colorRating, colorLabel, consistency, volumeEstimate, volumeMl, nil, notes)
}

// CreateStoolWithPhotos inserts a new stool entry with optional photo keys and returns it.
// If volumeMl is non-nil, also creates a linked fluid_log entry.
func CreateStoolWithPhotos(db *sql.DB, babyID, loggedBy, timestamp string, colorRating int, colorLabel, consistency, volumeEstimate *string, volumeMl *float64, photoKeys, notes *string) (*model.Stool, error) {
	id := model.NewULID()

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("create stool: begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO stools (id, baby_id, logged_by, timestamp, color_rating, color_label, consistency, volume_estimate, volume_ml, photo_keys, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, babyID, loggedBy, timestamp, colorRating, colorLabel, consistency, volumeEstimate, volumeMl, photoKeys, notes,
	)
	if err != nil {
		return nil, fmt.Errorf("create stool: %w", err)
	}

	if volumeMl != nil {
		srcType := "stool"
		fluidID := model.NewULID()
		if err := createFluidLogTx(tx, fluidID, babyID, loggedBy, timestamp, "output", "stool", volumeMl, &srcType, &id, notes); err != nil {
			return nil, fmt.Errorf("create stool: fluid_log: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("create stool: commit: %w", err)
	}

	return GetStoolByID(db, babyID, id)
}

// GetStoolByID retrieves a stool by its ID, scoped to the given baby.
func GetStoolByID(db *sql.DB, babyID, stoolID string) (*model.Stool, error) {
	row := db.QueryRow(
		`SELECT `+stoolColumns+` FROM stools WHERE id = ? AND baby_id = ?`,
		stoolID, babyID,
	)
	return scanStool(row)
}

// ListStools returns a paginated list of stools for a baby in ULID descending order.
func ListStools(db *sql.DB, babyID string, from, to, cursor *string, limit int) (*model.MetricPage[model.Stool], error) {
	return ListStoolsWithTZ(db, babyID, from, to, cursor, limit, time.UTC)
}

// ListStoolsWithTZ returns a paginated list of stools with timezone-aware date filtering.
func ListStoolsWithTZ(db *sql.DB, babyID string, from, to, cursor *string, limit int, loc *time.Location) (*model.MetricPage[model.Stool], error) {
	return listMetricWithTZ(db, "stools", stoolColumns, babyID, from, to, cursor, limit, loc, scanStool, func(s *model.Stool) string { return s.ID })
}

// UpdateStool updates a stool entry.
func UpdateStool(db *sql.DB, babyID, stoolID, updatedBy, timestamp string, colorRating int, colorLabel, consistency, volumeEstimate *string, volumeMl *float64, notes *string) (*model.Stool, error) {
	return UpdateStoolWithPhotos(db, babyID, stoolID, updatedBy, timestamp, colorRating, colorLabel, consistency, volumeEstimate, volumeMl, nil, notes)
}

// UpdateStoolWithPhotos updates a stool entry with optional photo keys.
// Also updates the linked fluid_log entry.
func UpdateStoolWithPhotos(db *sql.DB, babyID, stoolID, updatedBy, timestamp string, colorRating int, colorLabel, consistency, volumeEstimate *string, volumeMl *float64, photoKeys, notes *string) (*model.Stool, error) {
	existing, err := GetStoolByID(db, babyID, stoolID)
	if err != nil {
		return nil, fmt.Errorf("update stool: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("update stool: begin tx: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		`UPDATE stools SET
			updated_by = ?, timestamp = ?, color_rating = ?,
			color_label = ?, consistency = ?, volume_estimate = ?,
			volume_ml = ?, photo_keys = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND baby_id = ?`,
		updatedBy, timestamp, colorRating,
		colorLabel, consistency, volumeEstimate,
		volumeMl, photoKeys, notes,
		stoolID, babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("update stool: %w", err)
	}

	if err := checkRowsAffected(res, "update stool"); err != nil {
		return nil, err
	}

	// Delete old fluid_log, recreate if volume present
	if err := deleteFluidLogBySourceTx(tx, "stool", stoolID); err != nil {
		return nil, fmt.Errorf("update stool: fluid_log delete: %w", err)
	}
	if volumeMl != nil {
		srcType := "stool"
		fluidID := model.NewULID()
		user := existing.LoggedBy
		if updatedBy != "" {
			user = updatedBy
		}
		if err := createFluidLogTx(tx, fluidID, babyID, user, timestamp, "output", "stool", volumeMl, &srcType, &stoolID, notes); err != nil {
			return nil, fmt.Errorf("update stool: fluid_log create: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("update stool: commit: %w", err)
	}

	return GetStoolByID(db, babyID, stoolID)
}

// DeleteStool hard-deletes a stool entry and its linked fluid_log entry.
func DeleteStool(db *sql.DB, babyID, stoolID string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("delete stool: begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := deleteFluidLogBySourceTx(tx, "stool", stoolID); err != nil {
		return fmt.Errorf("delete stool: fluid_log: %w", err)
	}

	res, err := tx.Exec("DELETE FROM stools WHERE id = ? AND baby_id = ?", stoolID, babyID)
	if err != nil {
		return fmt.Errorf("delete stool: %w", err)
	}
	if err := checkRowsAffected(res, "delete stool"); err != nil {
		return err
	}

	return tx.Commit()
}
