package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

const medContainerColumns = `id, medication_id, baby_id, kind, unit,
	quantity_initial, quantity_remaining, opened_at, max_days_after_opening,
	expiration_date, depleted, notes, created_by, updated_by,
	created_at, updated_at`

// CreateContainerParams bundles inputs for CreateMedicationContainer.
type CreateContainerParams struct {
	MedicationID        string
	BabyID              string
	Kind                string
	Unit                string
	QuantityInitial     float64
	OpenedAt            *string
	MaxDaysAfterOpening *int
	ExpirationDate      *string
	Notes               *string
	CreatedBy           string
}

// UpdateContainerParams bundles inputs for UpdateMedicationContainer.
type UpdateContainerParams struct {
	UpdatedBy           string
	Kind                string
	Unit                string
	QuantityInitial     float64
	QuantityRemaining   float64
	OpenedAt            *string
	MaxDaysAfterOpening *int
	ExpirationDate      *string
	Depleted            bool
	Notes               *string
}

// AdjustContainerParams bundles inputs for AdjustMedicationContainer.
type AdjustContainerParams struct {
	AdjustedBy string
	Delta      float64
	Reason     *string
}

func scanMedicationContainer(s scanner) (*model.MedicationContainer, error) {
	var c model.MedicationContainer
	var openedAt, expirationDate, notes, updatedBy sql.NullString
	var maxDays sql.NullInt64
	var createdStr, updatedStr string

	err := s.Scan(
		&c.ID, &c.MedicationID, &c.BabyID, &c.Kind, &c.Unit,
		&c.QuantityInitial, &c.QuantityRemaining,
		&openedAt, &maxDays, &expirationDate, &c.Depleted,
		&notes, &c.CreatedBy, &updatedBy,
		&createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}

	if openedAt.Valid {
		t, err := ParseTime(openedAt.String)
		if err != nil {
			return nil, fmt.Errorf("parse opened_at: %w", err)
		}
		c.OpenedAt = &t
	}
	c.MaxDaysAfterOpening = nullInt(maxDays)
	c.ExpirationDate = nullStr(expirationDate)
	c.Notes = nullStr(notes)
	c.UpdatedBy = nullStr(updatedBy)

	c.CreatedAt, err = ParseTime(createdStr)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	c.UpdatedAt, err = ParseTime(updatedStr)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at: %w", err)
	}

	return &c, nil
}

// CreateMedicationContainer inserts a container and returns it.
func CreateMedicationContainer(db *sql.DB, p CreateContainerParams) (*model.MedicationContainer, error) {
	if !model.ValidContainerKind(p.Kind) {
		return nil, fmt.Errorf("invalid container kind: %s", p.Kind)
	}
	if !model.ValidDoseUnit(p.Unit) {
		return nil, fmt.Errorf("invalid container unit: %s", p.Unit)
	}
	if p.QuantityInitial < 0 {
		return nil, fmt.Errorf("quantity_initial must be >= 0")
	}

	id := model.NewULID()
	_, err := db.Exec(
		`INSERT INTO medication_containers
			(id, medication_id, baby_id, kind, unit,
			 quantity_initial, quantity_remaining, opened_at, max_days_after_opening,
			 expiration_date, notes, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, p.MedicationID, p.BabyID, p.Kind, p.Unit,
		p.QuantityInitial, p.QuantityInitial, p.OpenedAt, p.MaxDaysAfterOpening,
		p.ExpirationDate, p.Notes, p.CreatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("create medication_container: %w", err)
	}
	return GetMedicationContainerByID(db, p.BabyID, id)
}

// GetMedicationContainerByID fetches one container scoped to baby.
func GetMedicationContainerByID(db *sql.DB, babyID, containerID string) (*model.MedicationContainer, error) {
	row := db.QueryRow(
		`SELECT `+medContainerColumns+`
		 FROM medication_containers
		 WHERE id = ? AND baby_id = ?`,
		containerID, babyID,
	)
	return scanMedicationContainer(row)
}

// ListMedicationContainers returns all containers for a medication, oldest first.
// Caller decides ordering when picking for FIFO; this is a simple list.
func ListMedicationContainers(db *sql.DB, babyID, medicationID string) ([]model.MedicationContainer, error) {
	rows, err := db.Query(
		`SELECT `+medContainerColumns+`
		 FROM medication_containers
		 WHERE baby_id = ? AND medication_id = ?
		 ORDER BY created_at ASC, id ASC`,
		babyID, medicationID,
	)
	if err != nil {
		return nil, fmt.Errorf("list medication_containers: %w", err)
	}
	defer rows.Close()

	var out []model.MedicationContainer
	for rows.Next() {
		c, err := scanMedicationContainer(rows)
		if err != nil {
			return nil, fmt.Errorf("scan medication_container: %w", err)
		}
		out = append(out, *c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return emptySliceIfNil(out), nil
}

// UpdateMedicationContainer updates all editable fields on a container.
func UpdateMedicationContainer(db *sql.DB, babyID, containerID string, p UpdateContainerParams) (*model.MedicationContainer, error) {
	if !model.ValidContainerKind(p.Kind) {
		return nil, fmt.Errorf("invalid container kind: %s", p.Kind)
	}
	if !model.ValidDoseUnit(p.Unit) {
		return nil, fmt.Errorf("invalid container unit: %s", p.Unit)
	}
	if p.QuantityRemaining < 0 {
		return nil, fmt.Errorf("quantity_remaining must be >= 0")
	}

	res, err := db.Exec(
		`UPDATE medication_containers SET
			updated_by = ?, kind = ?, unit = ?,
			quantity_initial = ?, quantity_remaining = ?,
			opened_at = ?, max_days_after_opening = ?,
			expiration_date = ?, depleted = ?, notes = ?,
			updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND baby_id = ?`,
		p.UpdatedBy, p.Kind, p.Unit,
		p.QuantityInitial, p.QuantityRemaining,
		p.OpenedAt, p.MaxDaysAfterOpening,
		p.ExpirationDate, p.Depleted, p.Notes,
		containerID, babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("update medication_container: %w", err)
	}
	if err := checkRowsAffected(res, "update medication_container"); err != nil {
		return nil, err
	}
	return GetMedicationContainerByID(db, babyID, containerID)
}

// DeleteMedicationContainer hard-deletes a container.
func DeleteMedicationContainer(db *sql.DB, babyID, containerID string) error {
	return deleteByID(db, "medication_containers", babyID, containerID)
}

// AdjustMedicationContainer applies a manual delta to a container's
// quantity_remaining and writes an audit row to medication_stock_adjustments.
// Negative delta removes stock; positive delta adds.
// Both are applied atomically.
func AdjustMedicationContainer(db *sql.DB, babyID, containerID string, p AdjustContainerParams) (*model.MedicationStockAdjustment, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var remaining float64
	var depleted bool
	err = tx.QueryRow(
		`SELECT quantity_remaining, depleted
		 FROM medication_containers
		 WHERE id = ? AND baby_id = ?`,
		containerID, babyID,
	).Scan(&remaining, &depleted)
	if err != nil {
		return nil, fmt.Errorf("load container: %w", err)
	}

	newRem := remaining + p.Delta
	if newRem < 0 {
		newRem = 0
	}
	newDepleted := newRem == 0

	_, err = tx.Exec(
		`UPDATE medication_containers
		 SET quantity_remaining = ?, depleted = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND baby_id = ?`,
		newRem, newDepleted, containerID, babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("update container quantity: %w", err)
	}

	adjID := model.NewULID()
	adjustedAt := time.Now().UTC().Format(model.DateTimeFormat)
	_, err = tx.Exec(
		`INSERT INTO medication_stock_adjustments
			(id, container_id, delta, reason, adjusted_by, adjusted_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		adjID, containerID, p.Delta, p.Reason, p.AdjustedBy, adjustedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert stock adjustment: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return GetMedicationStockAdjustmentByID(db, adjID)
}

// GetMedicationStockAdjustmentByID fetches one adjustment row.
func GetMedicationStockAdjustmentByID(db *sql.DB, adjID string) (*model.MedicationStockAdjustment, error) {
	row := db.QueryRow(
		`SELECT id, container_id, delta, reason, adjusted_by, adjusted_at, created_at
		 FROM medication_stock_adjustments WHERE id = ?`,
		adjID,
	)
	return scanMedicationStockAdjustment(row)
}

// ListMedicationStockAdjustments returns all adjustment rows for a container,
// most recent first.
func ListMedicationStockAdjustments(db *sql.DB, containerID string) ([]model.MedicationStockAdjustment, error) {
	rows, err := db.Query(
		`SELECT id, container_id, delta, reason, adjusted_by, adjusted_at, created_at
		 FROM medication_stock_adjustments
		 WHERE container_id = ?
		 ORDER BY adjusted_at DESC, id DESC`,
		containerID,
	)
	if err != nil {
		return nil, fmt.Errorf("list medication_stock_adjustments: %w", err)
	}
	defer rows.Close()

	var out []model.MedicationStockAdjustment
	for rows.Next() {
		a, err := scanMedicationStockAdjustment(rows)
		if err != nil {
			return nil, fmt.Errorf("scan medication_stock_adjustment: %w", err)
		}
		out = append(out, *a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return emptySliceIfNil(out), nil
}

func scanMedicationStockAdjustment(s scanner) (*model.MedicationStockAdjustment, error) {
	var a model.MedicationStockAdjustment
	var reason sql.NullString
	var adjustedAtStr, createdStr string

	err := s.Scan(&a.ID, &a.ContainerID, &a.Delta, &reason, &a.AdjustedBy, &adjustedAtStr, &createdStr)
	if err != nil {
		return nil, err
	}
	a.Reason = nullStr(reason)
	a.AdjustedAt, err = ParseTime(adjustedAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse adjusted_at: %w", err)
	}
	a.CreatedAt, err = ParseTime(createdStr)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	return &a, nil
}
