package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

const bruisingColumns = `id, baby_id, logged_by, updated_by, timestamp,
	location, size_estimate, size_cm, color, photo_keys, notes, created_at, updated_at`

// scanBruising scans a single bruising observation row from the given scanner.
func scanBruising(s scanner) (*model.BruisingObservation, error) {
	var b model.BruisingObservation
	var updatedBy, color, photoKeys, notes sql.NullString
	var sizeCm sql.NullFloat64
	var tsStr, createdStr, updatedStr string

	err := s.Scan(
		&b.ID, &b.BabyID, &b.LoggedBy, &updatedBy, &tsStr,
		&b.Location, &b.SizeEstimate, &sizeCm, &color, &photoKeys, &notes,
		&createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}

	b.Timestamp, b.CreatedAt, b.UpdatedAt, err = parseMetricTimes(tsStr, createdStr, updatedStr)
	if err != nil {
		return nil, err
	}

	b.UpdatedBy = nullStr(updatedBy)
	b.SizeCm = nullFloat(sizeCm)
	b.Color = nullStr(color)
	b.PhotoKeys = nullStr(photoKeys)
	b.Notes = nullStr(notes)

	return &b, nil
}

// CreateBruising inserts a new bruising observation and returns it.
func CreateBruising(db *sql.DB, babyID, loggedBy, timestamp, location, sizeEstimate string, sizeCm *float64, color, notes *string) (*model.BruisingObservation, error) {
	return CreateBruisingWithPhotos(db, babyID, loggedBy, timestamp, location, sizeEstimate, sizeCm, color, nil, notes)
}

// CreateBruisingWithPhotos inserts a new bruising observation with optional photo keys and returns it.
func CreateBruisingWithPhotos(db *sql.DB, babyID, loggedBy, timestamp, location, sizeEstimate string, sizeCm *float64, color, photoKeys, notes *string) (*model.BruisingObservation, error) {
	id := model.NewULID()

	_, err := db.Exec(
		`INSERT INTO bruising (id, baby_id, logged_by, timestamp, location, size_estimate, size_cm, color, photo_keys, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, babyID, loggedBy, timestamp, location, sizeEstimate, sizeCm, color, photoKeys, notes,
	)
	if err != nil {
		return nil, fmt.Errorf("create bruising: %w", err)
	}

	return GetBruisingByID(db, babyID, id)
}

// GetBruisingByID retrieves a bruising observation by its ID, scoped to the given baby.
func GetBruisingByID(db *sql.DB, babyID, bruisingID string) (*model.BruisingObservation, error) {
	row := db.QueryRow(
		`SELECT `+bruisingColumns+` FROM bruising WHERE id = ? AND baby_id = ?`,
		bruisingID, babyID,
	)
	return scanBruising(row)
}

// ListBruising returns a paginated list of bruising observations for a baby in ULID descending order.
func ListBruising(db *sql.DB, babyID string, from, to, cursor *string, limit int) (*model.MetricPage[model.BruisingObservation], error) {
	return ListBruisingWithTZ(db, babyID, from, to, cursor, limit, time.UTC)
}

// ListBruisingWithTZ returns a paginated list of bruising observations with timezone-aware date filtering.
func ListBruisingWithTZ(db *sql.DB, babyID string, from, to, cursor *string, limit int, loc *time.Location) (*model.MetricPage[model.BruisingObservation], error) {
	return listMetricWithTZ(db, "bruising", bruisingColumns, babyID, from, to, cursor, limit, loc, scanBruising, func(b *model.BruisingObservation) string { return b.ID })
}

// UpdateBruising updates a bruising observation.
func UpdateBruising(db *sql.DB, babyID, bruisingID, updatedBy, timestamp, location, sizeEstimate string, sizeCm *float64, color, notes *string) (*model.BruisingObservation, error) {
	return UpdateBruisingWithPhotos(db, babyID, bruisingID, updatedBy, timestamp, location, sizeEstimate, sizeCm, color, nil, notes)
}

// UpdateBruisingWithPhotos updates a bruising observation with optional photo keys.
func UpdateBruisingWithPhotos(db *sql.DB, babyID, bruisingID, updatedBy, timestamp, location, sizeEstimate string, sizeCm *float64, color, photoKeys, notes *string) (*model.BruisingObservation, error) {
	res, err := db.Exec(
		`UPDATE bruising SET
			updated_by = ?, timestamp = ?, location = ?,
			size_estimate = ?, size_cm = ?, color = ?, photo_keys = ?, notes = ?,
			updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND baby_id = ?`,
		updatedBy, timestamp, location,
		sizeEstimate, sizeCm, color, photoKeys, notes,
		bruisingID, babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("update bruising: %w", err)
	}

	if err := checkRowsAffected(res, "update bruising"); err != nil {
		return nil, err
	}

	return GetBruisingByID(db, babyID, bruisingID)
}

// DeleteBruising hard-deletes a bruising observation.
func DeleteBruising(db *sql.DB, babyID, bruisingID string) error {
	return deleteByID(db, "bruising", babyID, bruisingID)
}
