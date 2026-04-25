package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

const medLogColumns = `id, medication_id, baby_id, logged_by, updated_by,
	scheduled_time, given_at, skipped, skip_reason, notes,
	container_id, stock_deducted,
	created_at, updated_at`

// scanMedLog scans a single med_log row from the given scanner.
func scanMedLog(s scanner) (*model.MedLog, error) {
	var m model.MedLog
	var updatedBy, scheduledTimeStr, givenAtStr, skipReason, notes, containerID sql.NullString
	var stockDeducted sql.NullFloat64
	var createdStr, updatedStr string
	var skipped bool

	err := s.Scan(
		&m.ID, &m.MedicationID, &m.BabyID, &m.LoggedBy, &updatedBy,
		&scheduledTimeStr, &givenAtStr, &skipped, &skipReason, &notes,
		&containerID, &stockDeducted,
		&createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}

	m.Skipped = skipped
	m.UpdatedBy = nullStr(updatedBy)
	m.SkipReason = nullStr(skipReason)
	m.Notes = nullStr(notes)
	m.ContainerID = nullStr(containerID)
	m.StockDeducted = nullFloat(stockDeducted)

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

// CreateMedLogParams bundles all inputs for a med-log creation, including the
// optional ContainerID override for stock auto-decrement.
type CreateMedLogParams struct {
	BabyID        string
	MedicationID  string
	LoggedBy      string
	ScheduledTime *string
	GivenAt       *string
	Skipped       bool
	SkipReason    *string
	Notes         *string
	// ContainerID overrides the FIFO auto-pick. Nil means auto-select.
	ContainerID *string
}

// UpdateMedLogParams bundles inputs for updating a med-log entry.
type UpdateMedLogParams struct {
	BabyID        string
	LogID         string
	UpdatedBy     string
	ScheduledTime *string
	GivenAt       *string
	Skipped       bool
	SkipReason    *string
	Notes         *string
	// ContainerID overrides the FIFO auto-pick when transitioning to given.
	// Nil means auto-select.
	ContainerID *string
}

// CreateMedLog is the legacy positional wrapper around CreateMedLogWithStock,
// preserved so existing callers don't need to migrate. New code should prefer
// CreateMedLogWithStock so the ContainerID override is reachable.
func CreateMedLog(db *sql.DB, babyID, medicationID, loggedBy string, scheduledTime, givenAt *string, skipped bool, skipReason, notes *string) (*model.MedLog, error) {
	return CreateMedLogWithStock(db, CreateMedLogParams{
		BabyID:        babyID,
		MedicationID:  medicationID,
		LoggedBy:      loggedBy,
		ScheduledTime: scheduledTime,
		GivenAt:       givenAt,
		Skipped:       skipped,
		SkipReason:    skipReason,
		Notes:         notes,
	})
}

// CreateMedLogWithStock inserts a new med-log entry and, if applicable,
// auto-decrements stock from a chosen container in the same transaction.
//
// Auto-decrement applies only when:
//   - p.Skipped == false (skipped doses do not deduct), AND
//   - the medication has both DoseAmount and DoseUnit set, AND
//   - a container exists (explicit override or FIFO/sealed auto-pick).
//
// Otherwise the log is written with container_id and stock_deducted left NULL.
func CreateMedLogWithStock(db *sql.DB, p CreateMedLogParams) (*model.MedLog, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	id := model.NewULID()
	containerID, deducted, err := pickAndDeductForLog(tx, p.BabyID, p.MedicationID, p.Skipped, p.ContainerID)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(
		`INSERT INTO med_logs
			(id, medication_id, baby_id, logged_by,
			 scheduled_time, given_at, skipped, skip_reason, notes,
			 container_id, stock_deducted)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, p.MedicationID, p.BabyID, p.LoggedBy,
		p.ScheduledTime, p.GivenAt, p.Skipped, p.SkipReason, p.Notes,
		containerID, deducted,
	)
	if err != nil {
		return nil, fmt.Errorf("create med_log: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return GetMedLogByID(db, p.BabyID, id)
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

// UpdateMedLog is the legacy positional wrapper around UpdateMedLogWithStock.
func UpdateMedLog(db *sql.DB, babyID, logID, updatedBy string, scheduledTime, givenAt *string, skipped bool, skipReason, notes *string) (*model.MedLog, error) {
	return UpdateMedLogWithStock(db, UpdateMedLogParams{
		BabyID:        babyID,
		LogID:         logID,
		UpdatedBy:     updatedBy,
		ScheduledTime: scheduledTime,
		GivenAt:       givenAt,
		Skipped:       skipped,
		SkipReason:    skipReason,
		Notes:         notes,
	})
}

// UpdateMedLogWithStock updates a med-log entry. If the previous state had
// stock deducted, that stock is restored before the new state's deduction
// (if any) is applied. Restore + deduct happen atomically.
func UpdateMedLogWithStock(db *sql.DB, p UpdateMedLogParams) (*model.MedLog, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	prev, err := loadMedLogForTx(tx, p.BabyID, p.LogID)
	if err != nil {
		return nil, err
	}

	if prev.ContainerID != nil && prev.StockDeducted != nil {
		if err := restoreStockTx(tx, *prev.ContainerID, *prev.StockDeducted); err != nil {
			return nil, err
		}
	}

	containerID, deducted, err := pickAndDeductForLog(tx, p.BabyID, prev.MedicationID, p.Skipped, p.ContainerID)
	if err != nil {
		return nil, err
	}

	res, err := tx.Exec(
		`UPDATE med_logs SET
			updated_by = ?, scheduled_time = ?, given_at = ?,
			skipped = ?, skip_reason = ?, notes = ?,
			container_id = ?, stock_deducted = ?,
			updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND baby_id = ?`,
		p.UpdatedBy, p.ScheduledTime, p.GivenAt,
		p.Skipped, p.SkipReason, p.Notes,
		containerID, deducted,
		p.LogID, p.BabyID,
	)
	if err != nil {
		return nil, fmt.Errorf("update med_log: %w", err)
	}
	if err := checkRowsAffected(res, "update med_log"); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return GetMedLogByID(db, p.BabyID, p.LogID)
}

// DeleteMedLog hard-deletes a med-log entry and, if it had auto-decremented
// stock, restores that quantity to the container.
func DeleteMedLog(db *sql.DB, babyID, logID string) error {
	return DeleteMedLogWithStock(db, babyID, logID)
}

// DeleteMedLogWithStock is an explicit alias for DeleteMedLog.
func DeleteMedLogWithStock(db *sql.DB, babyID, logID string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	prev, err := loadMedLogForTx(tx, babyID, logID)
	if err != nil {
		return err
	}

	if prev.ContainerID != nil && prev.StockDeducted != nil {
		if err := restoreStockTx(tx, *prev.ContainerID, *prev.StockDeducted); err != nil {
			return err
		}
	}

	res, err := tx.Exec(
		`DELETE FROM med_logs WHERE id = ? AND baby_id = ?`,
		logID, babyID,
	)
	if err != nil {
		return fmt.Errorf("delete med_log: %w", err)
	}
	if err := checkRowsAffected(res, "delete med_log"); err != nil {
		return err
	}
	return tx.Commit()
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

// loadMedLogForTx fetches a med-log inside an open transaction so the row's
// previous state is consistent with the subsequent updates.
func loadMedLogForTx(tx *sql.Tx, babyID, logID string) (*model.MedLog, error) {
	row := tx.QueryRow(
		`SELECT `+medLogColumns+` FROM med_logs WHERE id = ? AND baby_id = ?`,
		logID, babyID,
	)
	return scanMedLog(row)
}

// pickAndDeductForLog picks the container that should absorb this dose's
// deduction (if any) and applies the deduction inside the open transaction.
// Returns the chosen container_id and the amount deducted (both nil when no
// deduction applies).
//
// Selection order:
//  1. p.Skipped or no DoseAmount/DoseUnit -> nothing deducted, returns (nil, nil).
//  2. explicit override -> validate same medication and not depleted.
//  3. FIFO: oldest opened, non-depleted container.
//  4. Auto-open the oldest sealed container if none are open.
//  5. Empty inventory -> nothing deducted, returns (nil, nil).
func pickAndDeductForLog(tx *sql.Tx, babyID, medicationID string, skipped bool, override *string) (*string, *float64, error) {
	if skipped {
		return nil, nil, nil
	}

	doseAmount, doseUnit, err := loadMedDoseInfo(tx, babyID, medicationID)
	if err != nil {
		return nil, nil, err
	}
	if doseAmount == nil || doseUnit == nil {
		return nil, nil, nil
	}

	containerID, err := selectContainerForDeduction(tx, babyID, medicationID, *doseUnit, override)
	if err != nil {
		return nil, nil, err
	}
	if containerID == "" {
		return nil, nil, nil
	}

	if err := deductFromContainerTx(tx, containerID, *doseAmount); err != nil {
		return nil, nil, err
	}
	return &containerID, doseAmount, nil
}

func loadMedDoseInfo(tx *sql.Tx, babyID, medicationID string) (*float64, *string, error) {
	var amount sql.NullFloat64
	var unit sql.NullString
	err := tx.QueryRow(
		`SELECT dose_amount, dose_unit FROM medications WHERE id = ? AND baby_id = ?`,
		medicationID, babyID,
	).Scan(&amount, &unit)
	if err != nil {
		return nil, nil, fmt.Errorf("load medication dose info: %w", err)
	}
	return nullFloat(amount), nullStr(unit), nil
}

// selectContainerForDeduction picks a container to deduct from (or returns ""
// if no container should be used). It also auto-opens a sealed container if
// no opened ones exist.
func selectContainerForDeduction(tx *sql.Tx, babyID, medicationID, doseUnit string, override *string) (string, error) {
	if override != nil {
		var medID, unit string
		var depleted bool
		err := tx.QueryRow(
			`SELECT medication_id, unit, depleted FROM medication_containers
			 WHERE id = ? AND baby_id = ?`,
			*override, babyID,
		).Scan(&medID, &unit, &depleted)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", fmt.Errorf("container %s not found", *override)
			}
			return "", fmt.Errorf("load override container: %w", err)
		}
		if medID != medicationID {
			return "", fmt.Errorf("container %s belongs to a different medication", *override)
		}
		if depleted {
			return "", fmt.Errorf("container %s is depleted", *override)
		}
		if unit != doseUnit {
			return "", fmt.Errorf("container unit %s does not match medication dose unit %s", unit, doseUnit)
		}
		return *override, nil
	}

	// FIFO: oldest opened, non-depleted, matching unit.
	var openedID string
	err := tx.QueryRow(
		`SELECT id FROM medication_containers
		 WHERE baby_id = ? AND medication_id = ?
		   AND depleted = 0 AND opened_at IS NOT NULL
		   AND unit = ?
		 ORDER BY opened_at ASC, created_at ASC, id ASC
		 LIMIT 1`,
		babyID, medicationID, doseUnit,
	).Scan(&openedID)
	if err == nil {
		return openedID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("query opened containers: %w", err)
	}

	// No opened container — auto-open the oldest sealed one.
	var sealedID string
	err = tx.QueryRow(
		`SELECT id FROM medication_containers
		 WHERE baby_id = ? AND medication_id = ?
		   AND depleted = 0 AND opened_at IS NULL
		   AND unit = ?
		 ORDER BY created_at ASC, id ASC
		 LIMIT 1`,
		babyID, medicationID, doseUnit,
	).Scan(&sealedID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil // truly empty inventory
		}
		return "", fmt.Errorf("query sealed containers: %w", err)
	}

	openedAt := time.Now().UTC().Format(model.DateTimeFormat)
	if _, err := tx.Exec(
		`UPDATE medication_containers
		 SET opened_at = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		openedAt, sealedID,
	); err != nil {
		return "", fmt.Errorf("auto-open container: %w", err)
	}
	return sealedID, nil
}

func deductFromContainerTx(tx *sql.Tx, containerID string, amount float64) error {
	var remaining float64
	err := tx.QueryRow(
		`SELECT quantity_remaining FROM medication_containers WHERE id = ?`,
		containerID,
	).Scan(&remaining)
	if err != nil {
		return fmt.Errorf("load container remaining: %w", err)
	}
	newRem := remaining - amount
	if newRem < 0 {
		newRem = 0
	}
	depleted := newRem == 0
	_, err = tx.Exec(
		`UPDATE medication_containers
		 SET quantity_remaining = ?, depleted = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		newRem, depleted, containerID,
	)
	if err != nil {
		return fmt.Errorf("deduct from container: %w", err)
	}
	return nil
}

// restoreStockTx adds back a previously deducted amount to a container, lifting
// the depleted flag if applicable. If the container has been deleted, this is
// a no-op (orphaned restore is safe to skip).
func restoreStockTx(tx *sql.Tx, containerID string, amount float64) error {
	var remaining float64
	err := tx.QueryRow(
		`SELECT quantity_remaining FROM medication_containers WHERE id = ?`,
		containerID,
	).Scan(&remaining)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("load container for restore: %w", err)
	}
	newRem := remaining + amount
	depleted := newRem == 0
	_, err = tx.Exec(
		`UPDATE medication_containers
		 SET quantity_remaining = ?, depleted = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		newRem, depleted, containerID,
	)
	if err != nil {
		return fmt.Errorf("restore container stock: %w", err)
	}
	return nil
}
