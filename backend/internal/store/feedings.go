package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

// scanFeeding scans a single feeding row from the given scanner.
func scanFeeding(s scanner) (*model.Feeding, error) {
	var f model.Feeding
	var updatedBy, notes sql.NullString
	var volMl, calDensity, calories sql.NullFloat64
	var durMin sql.NullInt64
	var tsStr, createdStr, updatedStr string

	err := s.Scan(
		&f.ID, &f.BabyID, &f.LoggedBy, &updatedBy, &tsStr,
		&f.FeedType, &volMl, &calDensity, &calories,
		&f.UsedDefaultCal, &durMin, &notes, &createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}

	f.Timestamp, err = parseTime(tsStr)
	if err != nil {
		return nil, fmt.Errorf("parse timestamp: %w", err)
	}
	f.CreatedAt, err = parseTime(createdStr)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	f.UpdatedAt, err = parseTime(updatedStr)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at: %w", err)
	}

	if updatedBy.Valid {
		f.UpdatedBy = &updatedBy.String
	}
	if volMl.Valid {
		f.VolumeMl = &volMl.Float64
	}
	if calDensity.Valid {
		f.CalDensity = &calDensity.Float64
	}
	if calories.Valid {
		f.Calories = &calories.Float64
	}
	if durMin.Valid {
		v := int(durMin.Int64)
		f.DurationMin = &v
	}
	if notes.Valid {
		f.Notes = &notes.String
	}

	return &f, nil
}

const feedingColumns = `id, baby_id, logged_by, updated_by, timestamp,
	feed_type, volume_ml, cal_density, calories,
	used_default_cal, duration_min, notes, created_at, updated_at`

// CreateFeeding inserts a new feeding entry and returns it.
func CreateFeeding(db *sql.DB, babyID, loggedBy, timestamp, feedType string, volumeMl, calDensity *float64, durationMin *int, notes *string) (*model.Feeding, error) {
	id := model.NewULID()

	_, err := db.Exec(
		`INSERT INTO feedings (id, baby_id, logged_by, timestamp, feed_type, volume_ml, cal_density, duration_min, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, babyID, loggedBy, timestamp, feedType, volumeMl, calDensity, durationMin, notes,
	)
	if err != nil {
		return nil, fmt.Errorf("create feeding: %w", err)
	}

	return GetFeedingByID(db, babyID, id)
}

// GetFeedingByID retrieves a feeding by its ID, scoped to the given baby.
// Returns sql.ErrNoRows if not found or baby_id doesn't match.
func GetFeedingByID(db *sql.DB, babyID, feedingID string) (*model.Feeding, error) {
	row := db.QueryRow(
		`SELECT `+feedingColumns+` FROM feedings WHERE id = ? AND baby_id = ?`,
		feedingID, babyID,
	)
	return scanFeeding(row)
}

// ListFeedings returns a paginated list of feedings for a baby in ULID descending order.
// Uses UTC for date filtering. For timezone-aware filtering, use ListFeedingsWithTZ.
func ListFeedings(db *sql.DB, babyID string, from, to, cursor *string, limit int) (*model.MetricPage[model.Feeding], error) {
	return ListFeedingsWithTZ(db, babyID, from, to, cursor, limit, time.UTC)
}

// ListFeedingsWithTZ returns a paginated list of feedings with timezone-aware date filtering.
func ListFeedingsWithTZ(db *sql.DB, babyID string, from, to, cursor *string, limit int, loc *time.Location) (*model.MetricPage[model.Feeding], error) {
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
		// End of day: 23:59:59
		utcTo := t.Add(24*time.Hour - time.Second).UTC().Format(model.DateTimeFormat)
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, utcTo)
	}

	if cursor != nil {
		conditions = append(conditions, "id < ?")
		args = append(args, *cursor)
	}

	query := fmt.Sprintf(
		"SELECT %s FROM feedings WHERE %s ORDER BY id DESC LIMIT ?",
		feedingColumns,
		strings.Join(conditions, " AND "),
	)
	// Fetch limit+1 to determine if there's a next page
	args = append(args, limit+1)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list feedings: %w", err)
	}
	defer rows.Close()

	var feedings []model.Feeding
	for rows.Next() {
		f, err := scanFeeding(rows)
		if err != nil {
			return nil, fmt.Errorf("scan feeding: %w", err)
		}
		feedings = append(feedings, *f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	page := &model.MetricPage[model.Feeding]{
		Data: make([]model.Feeding, 0),
	}

	if len(feedings) > limit {
		page.Data = feedings[:limit]
		nextCursor := feedings[limit-1].ID
		page.NextCursor = &nextCursor
	} else {
		page.Data = feedings
	}

	// Ensure Data is never nil
	if page.Data == nil {
		page.Data = make([]model.Feeding, 0)
	}

	return page, nil
}

// UpdateFeeding updates a feeding entry. Sets updated_at = CURRENT_TIMESTAMP.
// Returns sql.ErrNoRows if the feeding doesn't exist for the given baby.
func UpdateFeeding(db *sql.DB, babyID, feedingID, updatedBy, timestamp, feedType string, volumeMl, calDensity *float64, durationMin *int, notes *string) (*model.Feeding, error) {
	res, err := db.Exec(
		`UPDATE feedings SET
			updated_by = ?, timestamp = ?, feed_type = ?,
			volume_ml = ?, cal_density = ?, duration_min = ?, notes = ?,
			updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND baby_id = ?`,
		updatedBy, timestamp, feedType,
		volumeMl, calDensity, durationMin, notes,
		feedingID, babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("update feeding: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("update feeding: rows affected: %w", err)
	}
	if affected == 0 {
		return nil, fmt.Errorf("update feeding: %w", sql.ErrNoRows)
	}

	return GetFeedingByID(db, babyID, feedingID)
}

// DeleteFeeding hard-deletes a feeding entry.
// Returns an error wrapping sql.ErrNoRows if the feeding doesn't exist for the given baby.
func DeleteFeeding(db *sql.DB, babyID, feedingID string) error {
	res, err := db.Exec(
		"DELETE FROM feedings WHERE id = ? AND baby_id = ?",
		feedingID, babyID,
	)
	if err != nil {
		return fmt.Errorf("delete feeding: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete feeding: rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("delete feeding: %w", sql.ErrNoRows)
	}

	return nil
}
