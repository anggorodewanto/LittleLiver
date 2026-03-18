package store

import (
	"database/sql"
	"fmt"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

const medicationColumns = `id, baby_id, logged_by, updated_by, name, dose,
	frequency, schedule, timezone, active, created_at, updated_at`

// scanMedication scans a single medication row from the given scanner.
func scanMedication(s scanner) (*model.Medication, error) {
	var m model.Medication
	var updatedBy, schedule, timezone sql.NullString
	var createdStr, updatedStr string
	var active bool

	err := s.Scan(
		&m.ID, &m.BabyID, &m.LoggedBy, &updatedBy,
		&m.Name, &m.Dose, &m.Frequency, &schedule,
		&timezone, &active, &createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}

	m.Active = active
	m.UpdatedBy = nullStr(updatedBy)
	m.Schedule = nullStr(schedule)
	m.Timezone = nullStr(timezone)

	m.CreatedAt, err = ParseTime(createdStr)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	m.UpdatedAt, err = ParseTime(updatedStr)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at: %w", err)
	}

	return &m, nil
}

// parseTime is declared in metric_helpers.go; if not, we need a local reference.
// It's already available via the store package.

// CreateMedication inserts a new medication and returns it.
func CreateMedication(db *sql.DB, babyID, loggedBy, name, dose, frequency string, schedule, timezone *string) (*model.Medication, error) {
	id := model.NewULID()

	_, err := db.Exec(
		`INSERT INTO medications (id, baby_id, logged_by, name, dose, frequency, schedule, timezone)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, babyID, loggedBy, name, dose, frequency, schedule, timezone,
	)
	if err != nil {
		return nil, fmt.Errorf("create medication: %w", err)
	}

	return GetMedicationByID(db, babyID, id)
}

// GetMedicationByID retrieves a medication by its ID, scoped to the given baby.
func GetMedicationByID(db *sql.DB, babyID, medID string) (*model.Medication, error) {
	row := db.QueryRow(
		`SELECT `+medicationColumns+` FROM medications WHERE id = ? AND baby_id = ?`,
		medID, babyID,
	)
	return scanMedication(row)
}

// ListMedications returns all medications (active and inactive) for a baby, ordered by created_at DESC.
func ListMedications(db *sql.DB, babyID string) ([]model.Medication, error) {
	rows, err := db.Query(
		`SELECT `+medicationColumns+` FROM medications WHERE baby_id = ? ORDER BY created_at DESC`,
		babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("list medications: %w", err)
	}
	defer rows.Close()

	var meds []model.Medication
	for rows.Next() {
		med, err := scanMedication(rows)
		if err != nil {
			return nil, fmt.Errorf("scan medication: %w", err)
		}
		meds = append(meds, *med)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	if meds == nil {
		meds = make([]model.Medication, 0)
	}
	return meds, nil
}

// UpdateMedication updates a medication's fields. Pass nil for active to leave unchanged.
func UpdateMedication(db *sql.DB, babyID, medID, updatedBy, name, dose, frequency string, schedule, timezone *string, active *bool) (*model.Medication, error) {
	// First get existing to determine current active state if not changing
	existing, err := GetMedicationByID(db, babyID, medID)
	if err != nil {
		return nil, err
	}

	activeVal := existing.Active
	if active != nil {
		activeVal = *active
	}

	res, err := db.Exec(
		`UPDATE medications SET
			updated_by = ?, name = ?, dose = ?, frequency = ?,
			schedule = ?, timezone = ?, active = ?,
			updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND baby_id = ?`,
		updatedBy, name, dose, frequency,
		schedule, timezone, activeVal,
		medID, babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("update medication: %w", err)
	}

	if err := checkRowsAffected(res, "update medication"); err != nil {
		return nil, err
	}

	return GetMedicationByID(db, babyID, medID)
}
