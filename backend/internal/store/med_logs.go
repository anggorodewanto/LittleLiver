package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

const medLogColumns = `id, medication_id, baby_id, logged_by, updated_by,
	scheduled_time, given_at, skipped, skip_reason, notes, created_at, updated_at`

// scanMedLog scans a single med_log row from the given scanner.
func scanMedLog(s scanner) (*model.MedLog, error) {
	var m model.MedLog
	var updatedBy, scheduledTimeStr, givenAtStr, skipReason, notes sql.NullString
	var createdStr, updatedStr string
	var skipped bool

	err := s.Scan(
		&m.ID, &m.MedicationID, &m.BabyID, &m.LoggedBy, &updatedBy,
		&scheduledTimeStr, &givenAtStr, &skipped, &skipReason, &notes,
		&createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}

	m.Skipped = skipped
	m.UpdatedBy = nullStr(updatedBy)
	m.SkipReason = nullStr(skipReason)
	m.Notes = nullStr(notes)

	if scheduledTimeStr.Valid {
		t, err := ParseTime(scheduledTimeStr.String)
		if err != nil {
			return nil, fmt.Errorf("parse scheduled_time: %w", err)
		}
		m.ScheduledTime = &t
	}

	if givenAtStr.Valid {
		t, err := ParseTime(givenAtStr.String)
		if err != nil {
			return nil, fmt.Errorf("parse given_at: %w", err)
		}
		m.GivenAt = &t
	}

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

// CreateMedLog inserts a new med-log entry and returns it.
// For "given" logs: givenAt should be non-nil and skipped=false.
// For "skipped" logs: skipped=true and givenAt should be nil.
func CreateMedLog(db *sql.DB, babyID, medicationID, loggedBy string, scheduledTime, givenAt *string, skipped bool, skipReason, notes *string) (*model.MedLog, error) {
	id := model.NewULID()

	_, err := db.Exec(
		`INSERT INTO med_logs (id, medication_id, baby_id, logged_by, scheduled_time, given_at, skipped, skip_reason, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, medicationID, babyID, loggedBy, scheduledTime, givenAt, skipped, skipReason, notes,
	)
	if err != nil {
		return nil, fmt.Errorf("create med_log: %w", err)
	}

	return GetMedLogByID(db, babyID, id)
}

// GetMedLogByID retrieves a med-log by its ID, scoped to the given baby.
func GetMedLogByID(db *sql.DB, babyID, logID string) (*model.MedLog, error) {
	row := db.QueryRow(
		`SELECT `+medLogColumns+` FROM med_logs WHERE id = ? AND baby_id = ?`,
		logID, babyID,
	)
	return scanMedLog(row)
}

// ListMedLogs returns a paginated list of med-logs for a baby with optional
// medication_id, from/to date filters, and cursor-based pagination.
func ListMedLogs(db *sql.DB, babyID string, medicationID, from, to, cursor *string, limit int, loc *time.Location) (*model.MetricPage[model.MedLog], error) {
	var conditions []string
	var args []any

	conditions = append(conditions, "baby_id = ?")
	args = append(args, babyID)

	if medicationID != nil {
		conditions = append(conditions, "medication_id = ?")
		args = append(args, *medicationID)
	}

	if loc == nil {
		loc = time.UTC
	}

	// For date filtering, use given_at for given doses and created_at for skipped doses.
	dateField := "CASE WHEN skipped = 1 THEN strftime('%Y-%m-%dT%H:%M:%SZ', created_at) ELSE given_at END"

	if from != nil {
		fromDate, err := time.ParseInLocation(model.DateFormat, *from, loc)
		if err != nil {
			return nil, fmt.Errorf("parse from date: %w", err)
		}
		conditions = append(conditions, dateField+" >= ?")
		args = append(args, fromDate.UTC().Format(model.DateTimeFormat))
	}

	if to != nil {
		toDate, err := time.ParseInLocation(model.DateFormat, *to, loc)
		if err != nil {
			return nil, fmt.Errorf("parse to date: %w", err)
		}
		// to is inclusive of the whole day
		toTime := toDate.Add(24 * time.Hour).UTC().Format(model.DateTimeFormat)
		conditions = append(conditions, dateField+" < ?")
		args = append(args, toTime)
	}

	if cursor != nil {
		conditions = append(conditions, "id < ?")
		args = append(args, *cursor)
	}

	query := fmt.Sprintf(
		"SELECT %s FROM med_logs WHERE %s ORDER BY id DESC LIMIT ?",
		medLogColumns,
		strings.Join(conditions, " AND "),
	)
	args = append(args, limit+1)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list med_logs: %w", err)
	}
	defer rows.Close()

	var logs []model.MedLog
	for rows.Next() {
		ml, err := scanMedLog(rows)
		if err != nil {
			return nil, fmt.Errorf("scan med_log: %w", err)
		}
		logs = append(logs, *ml)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	page := &model.MetricPage[model.MedLog]{}
	if len(logs) > limit {
		page.Data = logs[:limit]
		nc := logs[limit-1].ID
		page.NextCursor = &nc
	} else {
		page.Data = logs
	}

	if page.Data == nil {
		page.Data = make([]model.MedLog, 0)
	}
	return page, nil
}

// UpdateMedLog updates a med-log entry. Sets updated_by and updated_at.
func UpdateMedLog(db *sql.DB, babyID, logID, updatedBy string, scheduledTime, givenAt *string, skipped bool, skipReason, notes *string) (*model.MedLog, error) {
	res, err := db.Exec(
		`UPDATE med_logs SET
			updated_by = ?, scheduled_time = ?, given_at = ?,
			skipped = ?, skip_reason = ?, notes = ?,
			updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND baby_id = ?`,
		updatedBy, scheduledTime, givenAt,
		skipped, skipReason, notes,
		logID, babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("update med_log: %w", err)
	}

	if err := checkRowsAffected(res, "update med_log"); err != nil {
		return nil, err
	}

	return GetMedLogByID(db, babyID, logID)
}

// DeleteMedLog hard-deletes a med-log entry.
func DeleteMedLog(db *sql.DB, babyID, logID string) error {
	return deleteByID(db, "med_logs", babyID, logID)
}

// GetMedicationBabyID retrieves the baby_id for a medication.
// Returns sql.ErrNoRows if the medication doesn't exist.
func GetMedicationBabyID(db *sql.DB, medicationID string) (string, error) {
	var babyID string
	err := db.QueryRow("SELECT baby_id FROM medications WHERE id = ?", medicationID).Scan(&babyID)
	if err != nil {
		return "", err
	}
	return babyID, nil
}
