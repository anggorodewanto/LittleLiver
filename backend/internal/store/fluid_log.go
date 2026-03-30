package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

// dbExecer is satisfied by both *sql.DB and *sql.Tx, allowing fluid_log
// helpers to be called within or outside a transaction.
type dbExecer interface {
	Exec(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
	Query(query string, args ...any) (*sql.Rows, error)
}

const fluidLogColumns = `id, baby_id, logged_by, updated_by, timestamp,
	direction, method, volume_ml, source_type, source_id,
	notes, created_at, updated_at`

// scanFluidLog scans a single fluid_log row from the given scanner.
func scanFluidLog(s scanner) (*model.FluidLog, error) {
	var fl model.FluidLog
	var updatedBy, sourceType, sourceID, notes sql.NullString
	var volumeMl sql.NullFloat64
	var tsStr, createdStr, updatedStr string

	err := s.Scan(
		&fl.ID, &fl.BabyID, &fl.LoggedBy, &updatedBy, &tsStr,
		&fl.Direction, &fl.Method, &volumeMl, &sourceType, &sourceID,
		&notes, &createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}

	fl.Timestamp, fl.CreatedAt, fl.UpdatedAt, err = parseMetricTimes(tsStr, createdStr, updatedStr)
	if err != nil {
		return nil, err
	}

	fl.UpdatedBy = nullStr(updatedBy)
	fl.VolumeMl = nullFloat(volumeMl)
	fl.SourceType = nullStr(sourceType)
	fl.SourceID = nullStr(sourceID)
	fl.Notes = nullStr(notes)

	return &fl, nil
}

// createFluidLogTx inserts a fluid_log entry using the given executor (db or tx).
func createFluidLogTx(ex dbExecer, id, babyID, loggedBy, timestamp, direction, method string, volumeMl *float64, sourceType, sourceID, notes *string) error {
	_, err := ex.Exec(
		`INSERT INTO fluid_log (id, baby_id, logged_by, timestamp, direction, method, volume_ml, source_type, source_id, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, babyID, loggedBy, timestamp, direction, method, volumeMl, sourceType, sourceID, notes,
	)
	return err
}

// deleteFluidLogBySourceTx deletes the fluid_log entry linked to a given source.
func deleteFluidLogBySourceTx(ex dbExecer, sourceType, sourceID string) error {
	_, err := ex.Exec(
		`DELETE FROM fluid_log WHERE source_type = ? AND source_id = ?`,
		sourceType, sourceID,
	)
	return err
}

// upsertFluidLogBySourceTx deletes any existing fluid_log for the source, then creates a new one if volumeMl is non-nil.
// For feedings, always creates (volumeMl may be nil but we still track the intake).
func upsertFluidLogBySourceTx(ex dbExecer, babyID, loggedBy, updatedBy, timestamp, direction, method string, volumeMl *float64, sourceType, sourceID string, notes *string) error {
	if err := deleteFluidLogBySourceTx(ex, sourceType, sourceID); err != nil {
		return err
	}
	user := loggedBy
	if updatedBy != "" {
		user = updatedBy
	}
	id := model.NewULID()
	return createFluidLogTx(ex, id, babyID, user, timestamp, direction, method, volumeMl, &sourceType, &sourceID, notes)
}

// CreateFluidLog inserts a standalone fluid_log entry (no source link) and returns it.
func CreateFluidLog(db *sql.DB, babyID, loggedBy, timestamp, direction, method string, volumeMl *float64, notes *string) (*model.FluidLog, error) {
	id := model.NewULID()

	_, err := db.Exec(
		`INSERT INTO fluid_log (id, baby_id, logged_by, timestamp, direction, method, volume_ml, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, babyID, loggedBy, timestamp, direction, method, volumeMl, notes,
	)
	if err != nil {
		return nil, fmt.Errorf("create fluid_log: %w", err)
	}

	return GetFluidLogByID(db, babyID, id)
}

// GetFluidLogByID retrieves a fluid_log entry by its ID, scoped to the given baby.
func GetFluidLogByID(db *sql.DB, babyID, fluidLogID string) (*model.FluidLog, error) {
	row := db.QueryRow(
		`SELECT `+fluidLogColumns+` FROM fluid_log WHERE id = ? AND baby_id = ?`,
		fluidLogID, babyID,
	)
	return scanFluidLog(row)
}

// ListFluidLog returns a paginated list of fluid_log entries for a baby.
func ListFluidLog(db *sql.DB, babyID string, from, to, cursor *string, limit int) (*model.MetricPage[model.FluidLog], error) {
	return ListFluidLogWithTZ(db, babyID, from, to, cursor, limit, time.UTC)
}

// ListFluidLogWithTZ returns a paginated list of fluid_log entries with timezone-aware date filtering.
func ListFluidLogWithTZ(db *sql.DB, babyID string, from, to, cursor *string, limit int, loc *time.Location) (*model.MetricPage[model.FluidLog], error) {
	return listMetricWithTZ(db, "fluid_log", fluidLogColumns, babyID, from, to, cursor, limit, loc, scanFluidLog, func(fl *model.FluidLog) string { return fl.ID })
}

// UpdateFluidLog updates a standalone fluid_log entry. Returns an error if the entry is linked to a source.
func UpdateFluidLog(db *sql.DB, babyID, fluidLogID, updatedBy, timestamp, direction, method string, volumeMl *float64, notes *string) (*model.FluidLog, error) {
	// Check that entry is standalone (not linked)
	existing, err := GetFluidLogByID(db, babyID, fluidLogID)
	if err != nil {
		return nil, fmt.Errorf("update fluid_log: %w", err)
	}
	if existing.SourceType != nil {
		return nil, fmt.Errorf("update fluid_log: cannot update linked entry")
	}

	res, err := db.Exec(
		`UPDATE fluid_log SET
			updated_by = ?, timestamp = ?,
			direction = ?, method = ?, volume_ml = ?,
			notes = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND baby_id = ?`,
		updatedBy, timestamp,
		direction, method, volumeMl,
		notes,
		fluidLogID, babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("update fluid_log: %w", err)
	}

	if err := checkRowsAffected(res, "update fluid_log"); err != nil {
		return nil, err
	}

	return GetFluidLogByID(db, babyID, fluidLogID)
}

// DeleteFluidLog hard-deletes a standalone fluid_log entry. Returns an error if linked to a source.
func DeleteFluidLog(db *sql.DB, babyID, fluidLogID string) error {
	existing, err := GetFluidLogByID(db, babyID, fluidLogID)
	if err != nil {
		return fmt.Errorf("delete fluid_log: %w", err)
	}
	if existing.SourceType != nil {
		return fmt.Errorf("delete fluid_log: cannot delete linked entry")
	}

	return deleteByID(db, "fluid_log", babyID, fluidLogID)
}
