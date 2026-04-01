package store

import (
	"database/sql"
	"fmt"
	"time"
)

// FeedingByType holds per-type feed counts for a day.
type FeedingByType struct {
	BreastMilk int `json:"breast_milk"`
	Formula    int `json:"formula"`
	Solid      int `json:"solid"`
	Other      int `json:"other"`
}

// FeedingDailyEntry holds aggregated feeding data for a single day.
type FeedingDailyEntry struct {
	Date          string        `json:"date"`
	TotalVolumeMl float64       `json:"total_volume_ml"`
	TotalCalories float64       `json:"total_calories"`
	FeedCount     int           `json:"feed_count"`
	ByType        FeedingByType `json:"by_type"`
}

// DiaperDailyEntry holds aggregated diaper data for a single day.
type DiaperDailyEntry struct {
	Date       string `json:"date"`
	WetCount   int    `json:"wet_count"`
	StoolCount int    `json:"stool_count"`
}

// TemperatureSeriesEntry holds an individual temperature reading.
type TemperatureSeriesEntry struct {
	Timestamp string  `json:"timestamp"`
	Value     float64 `json:"value"`
	Method    string  `json:"method"`
}

// WeightSeriesEntry holds an individual weight reading.
type WeightSeriesEntry struct {
	Timestamp         string  `json:"timestamp"`
	WeightKg          float64 `json:"weight_kg"`
	MeasurementSource *string `json:"measurement_source"`
}

// AbdomenGirthEntry holds an individual abdomen girth reading.
type AbdomenGirthEntry struct {
	Timestamp string  `json:"timestamp"`
	GirthCm   float64 `json:"girth_cm"`
}

// HeadCircumferenceSeriesEntry holds an individual head circumference reading.
type HeadCircumferenceSeriesEntry struct {
	Timestamp       string  `json:"timestamp"`
	CircumferenceCm float64 `json:"circumference_cm"`
}

// UpperArmCircumferenceSeriesEntry holds an individual upper arm circumference reading.
type UpperArmCircumferenceSeriesEntry struct {
	Timestamp       string  `json:"timestamp"`
	CircumferenceCm float64 `json:"circumference_cm"`
}

// StoolColorSeriesEntry holds an individual stool color reading.
type StoolColorSeriesEntry struct {
	Timestamp  string `json:"timestamp"`
	ColorScore int    `json:"color_score"`
}

// LabTrendEntry holds an individual lab result for chart series.
type LabTrendEntry struct {
	Timestamp   string  `json:"timestamp"`
	TestName    string  `json:"test_name"`
	Value       string  `json:"value"`
	Unit        *string `json:"unit"`
	NormalRange *string `json:"normal_range"`
}

// GetFeedingDaily returns daily aggregated feeding data within the given date range.
// loc specifies the timezone for date interpretation and daily grouping.
func GetFeedingDaily(db *sql.DB, babyID, from, to string, loc *time.Location) ([]FeedingDailyEntry, error) {
	fromTime, toTime, err := ParseDateRangeInLocation(from, to, loc)
	if err != nil {
		return nil, err
	}

	offsetSec := tzOffsetSeconds(from, loc)

	rows, err := db.Query(
		`SELECT DATE(datetime(timestamp, ? || ' seconds')) as date,
			COALESCE(SUM(volume_ml), 0) as total_volume_ml,
			COALESCE(SUM(calories), 0) as total_calories,
			COUNT(*) as feed_count,
			SUM(CASE WHEN feed_type = 'breast_milk' OR feed_type = 'fortified_breast_milk' THEN 1 ELSE 0 END) as breast_milk,
			SUM(CASE WHEN feed_type = 'formula' THEN 1 ELSE 0 END) as formula,
			SUM(CASE WHEN feed_type = 'solid' THEN 1 ELSE 0 END) as solid,
			SUM(CASE WHEN feed_type = 'other' THEN 1 ELSE 0 END) as other_type
		 FROM feedings
		 WHERE baby_id = ? AND timestamp >= ? AND timestamp < ?
		 GROUP BY DATE(datetime(timestamp, ? || ' seconds'))
		 ORDER BY date ASC`,
		offsetSec, babyID, fromTime, toTime, offsetSec,
	)
	if err != nil {
		return nil, fmt.Errorf("query feeding daily: %w", err)
	}
	defer rows.Close()

	var entries []FeedingDailyEntry
	for rows.Next() {
		var e FeedingDailyEntry
		if err := rows.Scan(&e.Date, &e.TotalVolumeMl, &e.TotalCalories, &e.FeedCount,
			&e.ByType.BreastMilk, &e.ByType.Formula, &e.ByType.Solid, &e.ByType.Other); err != nil {
			return nil, fmt.Errorf("scan feeding daily: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return emptySliceIfNil(entries), nil
}

// GetDiaperDaily returns daily aggregated diaper data (wet + stool counts) within the given date range.
// loc specifies the timezone for date interpretation and daily grouping.
func GetDiaperDaily(db *sql.DB, babyID, from, to string, loc *time.Location) ([]DiaperDailyEntry, error) {
	fromTime, toTime, err := ParseDateRangeInLocation(from, to, loc)
	if err != nil {
		return nil, err
	}

	offsetSec := tzOffsetSeconds(from, loc)

	// Use UNION ALL to combine urine and stool counts by date
	rows, err := db.Query(
		`SELECT date, SUM(wet) as wet_count, SUM(stool) as stool_count FROM (
			SELECT DATE(datetime(timestamp, ? || ' seconds')) as date, 1 as wet, 0 as stool
			FROM urine WHERE baby_id = ? AND timestamp >= ? AND timestamp < ?
			UNION ALL
			SELECT DATE(datetime(timestamp, ? || ' seconds')) as date, 0 as wet, 1 as stool
			FROM stools WHERE baby_id = ? AND timestamp >= ? AND timestamp < ?
		) combined
		GROUP BY date
		ORDER BY date ASC`,
		offsetSec, babyID, fromTime, toTime,
		offsetSec, babyID, fromTime, toTime,
	)
	if err != nil {
		return nil, fmt.Errorf("query diaper daily: %w", err)
	}
	defer rows.Close()

	var entries []DiaperDailyEntry
	for rows.Next() {
		var e DiaperDailyEntry
		if err := rows.Scan(&e.Date, &e.WetCount, &e.StoolCount); err != nil {
			return nil, fmt.Errorf("scan diaper daily: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return emptySliceIfNil(entries), nil
}

// GetTemperatureSeries returns individual temperature readings within the given date range.
// loc specifies the timezone for date interpretation.
func GetTemperatureSeries(db *sql.DB, babyID, from, to string, loc *time.Location) ([]TemperatureSeriesEntry, error) {
	fromTime, toTime, err := ParseDateRangeInLocation(from, to, loc)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(
		`SELECT timestamp, value, method
		 FROM temperatures
		 WHERE baby_id = ? AND timestamp >= ? AND timestamp < ?
		 ORDER BY timestamp ASC`,
		babyID, fromTime, toTime,
	)
	if err != nil {
		return nil, fmt.Errorf("query temperature series: %w", err)
	}
	defer rows.Close()

	var entries []TemperatureSeriesEntry
	for rows.Next() {
		var e TemperatureSeriesEntry
		if err := rows.Scan(&e.Timestamp, &e.Value, &e.Method); err != nil {
			return nil, fmt.Errorf("scan temperature series: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return emptySliceIfNil(entries), nil
}

// GetWeightSeries returns individual weight readings within the given date range.
// loc specifies the timezone for date interpretation.
func GetWeightSeries(db *sql.DB, babyID, from, to string, loc *time.Location) ([]WeightSeriesEntry, error) {
	fromTime, toTime, err := ParseDateRangeInLocation(from, to, loc)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(
		`SELECT timestamp, weight_kg, measurement_source
		 FROM weights
		 WHERE baby_id = ? AND timestamp >= ? AND timestamp < ?
		 ORDER BY timestamp ASC`,
		babyID, fromTime, toTime,
	)
	if err != nil {
		return nil, fmt.Errorf("query weight series: %w", err)
	}
	defer rows.Close()

	var entries []WeightSeriesEntry
	for rows.Next() {
		var e WeightSeriesEntry
		var src sql.NullString
		if err := rows.Scan(&e.Timestamp, &e.WeightKg, &src); err != nil {
			return nil, fmt.Errorf("scan weight series: %w", err)
		}
		e.MeasurementSource = nullStr(src)
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return emptySliceIfNil(entries), nil
}

// GetAbdomenGirthSeries returns individual abdomen girth readings within the given date range.
// Only entries with a non-null girth_cm are included. loc specifies the timezone for date interpretation.
func GetAbdomenGirthSeries(db *sql.DB, babyID, from, to string, loc *time.Location) ([]AbdomenGirthEntry, error) {
	fromTime, toTime, err := ParseDateRangeInLocation(from, to, loc)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(
		`SELECT timestamp, girth_cm
		 FROM abdomen_observations
		 WHERE baby_id = ? AND timestamp >= ? AND timestamp < ? AND girth_cm IS NOT NULL
		 ORDER BY timestamp ASC`,
		babyID, fromTime, toTime,
	)
	if err != nil {
		return nil, fmt.Errorf("query abdomen girth series: %w", err)
	}
	defer rows.Close()

	var entries []AbdomenGirthEntry
	for rows.Next() {
		var e AbdomenGirthEntry
		if err := rows.Scan(&e.Timestamp, &e.GirthCm); err != nil {
			return nil, fmt.Errorf("scan abdomen girth series: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return emptySliceIfNil(entries), nil
}

// GetStoolColorSeries returns individual stool color readings within the given date range.
// loc specifies the timezone for date interpretation.
func GetStoolColorSeries(db *sql.DB, babyID, from, to string, loc *time.Location) ([]StoolColorSeriesEntry, error) {
	fromTime, toTime, err := ParseDateRangeInLocation(from, to, loc)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(
		`SELECT timestamp, color_rating
		 FROM stools
		 WHERE baby_id = ? AND timestamp >= ? AND timestamp < ?
		 ORDER BY timestamp ASC`,
		babyID, fromTime, toTime,
	)
	if err != nil {
		return nil, fmt.Errorf("query stool color series: %w", err)
	}
	defer rows.Close()

	var entries []StoolColorSeriesEntry
	for rows.Next() {
		var e StoolColorSeriesEntry
		if err := rows.Scan(&e.Timestamp, &e.ColorScore); err != nil {
			return nil, fmt.Errorf("scan stool color series: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return emptySliceIfNil(entries), nil
}

// GetLabTrends returns lab results grouped by test_name within the given date range.
// loc specifies the timezone for date interpretation.
func GetLabTrends(db *sql.DB, babyID, from, to string, loc *time.Location) (map[string][]LabTrendEntry, error) {
	fromTime, toTime, err := ParseDateRangeInLocation(from, to, loc)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(
		`SELECT timestamp, test_name, value, unit, normal_range
		 FROM lab_results
		 WHERE baby_id = ? AND timestamp >= ? AND timestamp < ?
		 ORDER BY test_name ASC, timestamp ASC`,
		babyID, fromTime, toTime,
	)
	if err != nil {
		return nil, fmt.Errorf("query lab trends: %w", err)
	}
	defer rows.Close()

	trends := make(map[string][]LabTrendEntry)
	for rows.Next() {
		var e LabTrendEntry
		var unit, normalRange sql.NullString
		if err := rows.Scan(&e.Timestamp, &e.TestName, &e.Value, &unit, &normalRange); err != nil {
			return nil, fmt.Errorf("scan lab trend: %w", err)
		}
		e.Unit = nullStr(unit)
		e.NormalRange = nullStr(normalRange)
		trends[e.TestName] = append(trends[e.TestName], e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return trends, nil
}

// GetHeadCircumferenceSeries returns individual head circumference readings within the given date range.
func GetHeadCircumferenceSeries(db *sql.DB, babyID, from, to string, loc *time.Location) ([]HeadCircumferenceSeriesEntry, error) {
	fromTime, toTime, err := ParseDateRangeInLocation(from, to, loc)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(
		`SELECT timestamp, circumference_cm
		 FROM head_circumferences
		 WHERE baby_id = ? AND timestamp >= ? AND timestamp < ?
		 ORDER BY timestamp ASC`,
		babyID, fromTime, toTime,
	)
	if err != nil {
		return nil, fmt.Errorf("query head circumference series: %w", err)
	}
	defer rows.Close()

	var entries []HeadCircumferenceSeriesEntry
	for rows.Next() {
		var e HeadCircumferenceSeriesEntry
		if err := rows.Scan(&e.Timestamp, &e.CircumferenceCm); err != nil {
			return nil, fmt.Errorf("scan head circumference series: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return emptySliceIfNil(entries), nil
}

// GetUpperArmCircumferenceSeries returns individual upper arm circumference readings within the given date range.
func GetUpperArmCircumferenceSeries(db *sql.DB, babyID, from, to string, loc *time.Location) ([]UpperArmCircumferenceSeriesEntry, error) {
	fromTime, toTime, err := ParseDateRangeInLocation(from, to, loc)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(
		`SELECT timestamp, circumference_cm
		 FROM upper_arm_circumferences
		 WHERE baby_id = ? AND timestamp >= ? AND timestamp < ?
		 ORDER BY timestamp ASC`,
		babyID, fromTime, toTime,
	)
	if err != nil {
		return nil, fmt.Errorf("query upper arm circumference series: %w", err)
	}
	defer rows.Close()

	var entries []UpperArmCircumferenceSeriesEntry
	for rows.Next() {
		var e UpperArmCircumferenceSeriesEntry
		if err := rows.Scan(&e.Timestamp, &e.CircumferenceCm); err != nil {
			return nil, fmt.Errorf("scan upper arm circumference series: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return emptySliceIfNil(entries), nil
}
