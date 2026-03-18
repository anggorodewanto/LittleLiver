package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
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

// listMetricWithTZ is a generic helper for paginated, timezone-aware listing of metric rows.
// It builds the WHERE clause, executes the query, scans rows, and handles cursor pagination.
func listMetricWithTZ[T any](
	db *sql.DB, table, columns, babyID string,
	from, to, cursor *string, limit int, loc *time.Location,
	scan func(scanner) (*T, error),
	getID func(*T) string,
) (*model.MetricPage[T], error) {
	var conditions []string
	var args []any

	conditions = append(conditions, "baby_id = ?")
	args = append(args, babyID)

	if from != nil {
		t, err := time.ParseInLocation(model.DateFormat, *from, loc)
		if err != nil {
			return nil, fmt.Errorf("parse from date: %w", err)
		}
		utcFrom := t.UTC().Format(model.DateTimeFormat)
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, utcFrom)
	}

	if to != nil {
		t, err := time.ParseInLocation(model.DateFormat, *to, loc)
		if err != nil {
			return nil, fmt.Errorf("parse to date: %w", err)
		}
		utcTo := t.Add(24 * time.Hour).UTC().Format(model.DateTimeFormat)
		conditions = append(conditions, "timestamp < ?")
		args = append(args, utcTo)
	}

	if cursor != nil {
		conditions = append(conditions, "id < ?")
		args = append(args, *cursor)
	}

	query := fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s ORDER BY id DESC LIMIT ?",
		columns, table,
		strings.Join(conditions, " AND "),
	)
	args = append(args, limit+1)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", table, err)
	}
	defer rows.Close()

	var items []T
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan %s: %w", table, err)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	page := &model.MetricPage[T]{}

	if len(items) > limit {
		page.Data = items[:limit]
		nc := getID(&items[limit-1])
		page.NextCursor = &nc
	} else {
		page.Data = items
	}

	if page.Data == nil {
		page.Data = make([]T, 0)
	}

	return page, nil
}

// ParseDateRange parses from/to date strings (YYYY-MM-DD) and returns datetime boundaries
// suitable for SQL WHERE clauses: fromTime is start of from-date, toTime is start of day after to-date.
func ParseDateRange(from, to string) (string, string, error) {
	fromDate, err := time.Parse(model.DateFormat, from)
	if err != nil {
		return "", "", fmt.Errorf("parse from date: %w", err)
	}
	toDate, err := time.Parse(model.DateFormat, to)
	if err != nil {
		return "", "", fmt.Errorf("parse to date: %w", err)
	}
	fromTime := fromDate.Format(model.DateTimeFormat)
	toTime := toDate.Add(24 * time.Hour).Format(model.DateTimeFormat)
	return fromTime, toTime, nil
}

// emptySliceIfNil returns an empty slice if s is nil, otherwise returns s unchanged.
// This ensures JSON serialization produces [] instead of null.
func emptySliceIfNil[T any](s []T) []T {
	if s == nil {
		return make([]T, 0)
	}
	return s
}

// parseMetricTimes parses the three standard time strings (timestamp, created_at, updated_at).
func parseMetricTimes(tsStr, createdStr, updatedStr string) (timestamp, createdAt, updatedAt time.Time, err error) {
	timestamp, err = ParseTime(tsStr)
	if err != nil {
		err = fmt.Errorf("parse timestamp: %w", err)
		return
	}
	createdAt, err = ParseTime(createdStr)
	if err != nil {
		err = fmt.Errorf("parse created_at: %w", err)
		return
	}
	updatedAt, err = ParseTime(updatedStr)
	if err != nil {
		err = fmt.Errorf("parse updated_at: %w", err)
		return
	}
	return
}
