package store

import (
	"database/sql"
	"fmt"
	"time"
)

// checkRowsAffected verifies that at least one row was affected by the operation.
// Returns sql.ErrNoRows wrapped with the operation name if no rows were affected.
func checkRowsAffected(res sql.Result, op string) error {
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: rows affected: %w", op, err)
	}
	if affected == 0 {
		return fmt.Errorf("%s: %w", op, sql.ErrNoRows)
	}
	return nil
}

// deleteByID deletes a row from the given table by ID and baby_id.
func deleteByID(db *sql.DB, table, babyID, entryID string) error {
	res, err := db.Exec(
		fmt.Sprintf("DELETE FROM %s WHERE id = ? AND baby_id = ?", table),
		entryID, babyID,
	)
	if err != nil {
		return fmt.Errorf("delete %s: %w", table, err)
	}
	return checkRowsAffected(res, "delete "+table)
}

// nullStr converts a sql.NullString to a *string.
func nullStr(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

// nullFloat converts a sql.NullFloat64 to a *float64.
func nullFloat(nf sql.NullFloat64) *float64 {
	if nf.Valid {
		return &nf.Float64
	}
	return nil
}

// nullInt converts a sql.NullInt64 to a *int.
func nullInt(ni sql.NullInt64) *int {
	if ni.Valid {
		v := int(ni.Int64)
		return &v
	}
	return nil
}

// parseMetricTimes parses the three standard time strings (timestamp, created_at, updated_at).
func parseMetricTimes(tsStr, createdStr, updatedStr string) (timestamp, createdAt, updatedAt time.Time, err error) {
	timestamp, err = parseTime(tsStr)
	if err != nil {
		err = fmt.Errorf("parse timestamp: %w", err)
		return
	}
	createdAt, err = parseTime(createdStr)
	if err != nil {
		err = fmt.Errorf("parse created_at: %w", err)
		return
	}
	updatedAt, err = parseTime(updatedStr)
	if err != nil {
		err = fmt.Errorf("parse updated_at: %w", err)
		return
	}
	return
}
