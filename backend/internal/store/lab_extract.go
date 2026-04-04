package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

// FindDuplicateLabResult finds an existing lab result for the same baby where test_name
// matches (case-insensitive) and value matches, within a +-3 day window of referenceDate.
// Returns nil if no match is found.
func FindDuplicateLabResult(db *sql.DB, babyID, testName, value string, referenceDate time.Time) (*model.LabResult, error) {
	windowStart := referenceDate.AddDate(0, 0, -3).Format(model.DateTimeFormat)
	windowEnd := referenceDate.AddDate(0, 0, 3).Format(model.DateTimeFormat)

	row := db.QueryRow(
		`SELECT `+labResultColumns+` FROM lab_results
		 WHERE baby_id = ?
		   AND LOWER(test_name) = LOWER(?)
		   AND value = ?
		   AND timestamp >= ?
		   AND timestamp <= ?
		 ORDER BY timestamp DESC
		 LIMIT 1`,
		babyID, strings.ToLower(testName), value, windowStart, windowEnd,
	)

	result, err := scanLabResult(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find duplicate lab result: %w", err)
	}
	return result, nil
}
