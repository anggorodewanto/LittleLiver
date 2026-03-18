package report_test

import (
	"bytes"
	"database/sql"
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/report"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

// seedBabyWithKasai creates a baby with kasai_date set for report testing.
func seedBabyWithKasai(t *testing.T, db *sql.DB, userID string) *model.Baby {
	t.Helper()
	babyID := model.NewULID()
	_, err := db.Exec(
		`INSERT INTO babies (id, name, sex, date_of_birth, kasai_date)
		 VALUES (?, 'Report Baby', 'female', '2025-06-15', '2025-07-01')`,
		babyID,
	)
	if err != nil {
		t.Fatalf("insert baby: %v", err)
	}
	_, err = db.Exec(
		"INSERT INTO baby_parents (baby_id, user_id) VALUES (?, ?)",
		babyID, userID,
	)
	if err != nil {
		t.Fatalf("insert baby_parents: %v", err)
	}
	baby, err := getBabyByID(db, babyID)
	if err != nil {
		t.Fatalf("get baby: %v", err)
	}
	return baby
}

// getBabyByID fetches a baby by ID for test convenience.
func getBabyByID(db *sql.DB, id string) (*model.Baby, error) {
	var b model.Baby
	var diagDate, kasaiDate, notes sql.NullString
	var dobStr, createdStr string
	err := db.QueryRow(
		`SELECT id, name, sex, date_of_birth, diagnosis_date, kasai_date,
		        default_cal_per_feed, notes, created_at
		 FROM babies WHERE id = ?`, id,
	).Scan(&b.ID, &b.Name, &b.Sex, &dobStr, &diagDate, &kasaiDate,
		&b.DefaultCalPerFeed, &notes, &createdStr)
	if err != nil {
		return nil, err
	}
	b.DateOfBirth, _ = time.Parse(model.DateFormat, dobStr)
	if kasaiDate.Valid {
		t, _ := time.Parse(model.DateFormat, kasaiDate.String)
		b.KasaiDate = &t
	}
	if diagDate.Valid {
		t, _ := time.Parse(model.DateFormat, diagDate.String)
		b.DiagnosisDate = &t
	}
	if notes.Valid {
		b.Notes = &notes.String
	}
	return &b, nil
}

// seedReportData inserts sample metrics for a baby within a given date range.
func seedReportData(t *testing.T, db *sql.DB, babyID, userID string) {
	t.Helper()

	// Feedings
	for i := 0; i < 3; i++ {
		ts := time.Date(2025, 8, 1, 8+i*4, 0, 0, 0, time.UTC).Format(model.DateTimeFormat)
		_, err := db.Exec(
			`INSERT INTO feedings (id, baby_id, logged_by, timestamp, feed_type, volume_ml, calories)
			 VALUES (?, ?, ?, ?, 'formula', 120.0, 80.0)`,
			model.NewULID(), babyID, userID, ts,
		)
		if err != nil {
			t.Fatalf("insert feeding: %v", err)
		}
	}

	// Stools
	for i := 0; i < 2; i++ {
		ts := time.Date(2025, 8, 1, 10+i*3, 0, 0, 0, time.UTC).Format(model.DateTimeFormat)
		_, err := db.Exec(
			`INSERT INTO stools (id, baby_id, logged_by, timestamp, color_rating, color_label)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			model.NewULID(), babyID, userID, ts, 3+i, "yellow",
		)
		if err != nil {
			t.Fatalf("insert stool: %v", err)
		}
	}

	// Temperatures (one normal, one fever)
	for i, val := range []float64{36.8, 38.5} {
		ts := time.Date(2025, 8, 1, 9+i*6, 0, 0, 0, time.UTC).Format(model.DateTimeFormat)
		_, err := db.Exec(
			`INSERT INTO temperatures (id, baby_id, logged_by, timestamp, value, method)
			 VALUES (?, ?, ?, ?, ?, 'axillary')`,
			model.NewULID(), babyID, userID, ts, val,
		)
		if err != nil {
			t.Fatalf("insert temperature: %v", err)
		}
	}

	// Medication + med_logs
	medID := model.NewULID()
	_, err := db.Exec(
		`INSERT INTO medications (id, baby_id, logged_by, name, dose, frequency, active)
		 VALUES (?, ?, ?, 'UDCA', '50mg', 'twice_daily', 1)`,
		medID, babyID, userID,
	)
	if err != nil {
		t.Fatalf("insert medication: %v", err)
	}

	// 3 med logs: 2 given, 1 skipped
	for i := 0; i < 3; i++ {
		logID := model.NewULID()
		ts := time.Date(2025, 8, 1, 8+i*4, 0, 0, 0, time.UTC).Format(model.DateTimeFormat)
		skipped := i == 2
		var givenAt *string
		if !skipped {
			givenAt = &ts
		}
		_, err := db.Exec(
			`INSERT INTO med_logs (id, medication_id, baby_id, logged_by, scheduled_time, given_at, skipped, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			logID, medID, babyID, userID, ts, givenAt, skipped, ts,
		)
		if err != nil {
			t.Fatalf("insert med_log: %v", err)
		}
	}

	// General notes
	ts := time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC).Format(model.DateTimeFormat)
	_, err = db.Exec(
		`INSERT INTO general_notes (id, baby_id, logged_by, timestamp, content, category)
		 VALUES (?, ?, ?, ?, 'Baby seemed fussy after feeding', 'observation')`,
		model.NewULID(), babyID, userID, ts,
	)
	if err != nil {
		t.Fatalf("insert general_note: %v", err)
	}
}

func TestGeneratePDF_WithData(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := seedBabyWithKasai(t, db, user.ID)
	seedReportData(t, db, baby.ID, user.ID)

	var buf bytes.Buffer
	err := report.Generate(db, baby, "2025-08-01", "2025-08-01", &buf)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	// Check that output is a valid PDF (starts with %PDF-)
	data := buf.Bytes()
	if len(data) < 5 {
		t.Fatal("PDF output too small")
	}
	if string(data[:5]) != "%PDF-" {
		t.Fatalf("output does not start with %%PDF- header, got: %q", string(data[:20]))
	}
}

func TestGeneratePDF_EmptyDateRange(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := seedBabyWithKasai(t, db, user.ID)
	// No data seeded — empty date range

	var buf bytes.Buffer
	err := report.Generate(db, baby, "2025-09-01", "2025-09-01", &buf)
	if err != nil {
		t.Fatalf("Generate returned error for empty range: %v", err)
	}

	data := buf.Bytes()
	if len(data) < 5 {
		t.Fatal("PDF output too small for empty range")
	}
	if string(data[:5]) != "%PDF-" {
		t.Fatalf("output does not start with %%PDF- header for empty range")
	}
}

func TestGeneratePDF_ContainsExpectedText(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := seedBabyWithKasai(t, db, user.ID)
	seedReportData(t, db, baby.ID, user.ID)

	var buf bytes.Buffer
	err := report.Generate(db, baby, "2025-08-01", "2025-08-01", &buf)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	// The PDF should contain text strings for the baby name, sections, etc.
	// We check the raw PDF bytes for expected string fragments.
	data := string(buf.Bytes())

	expectedFragments := []string{
		"Report Baby",        // baby name
		"Stool Color Log",    // section header
		"Temperature Log",    // section header
		"Feeding Summary",    // section header
		"Medication",         // medication section
		"UDCA",               // medication name
		"Notable",            // observations section
	}

	for _, frag := range expectedFragments {
		if !containsText(data, frag) {
			t.Errorf("PDF does not contain expected text %q", frag)
		}
	}
}

func TestFormatAge(t *testing.T) {
	t.Parallel()
	tests := []struct {
		days     int
		expected string
	}{
		{0, "0 days"},
		{10, "10 days"},
		{29, "29 days"},
		{30, "1 month, 0 days"},
		{31, "1 month, 1 day"},
		{60, "2 months, 0 days"},
		{350, "11 months, 20 days"},
		{365, "1 year, 0 months"},
		{400, "1 year, 1 month"},
		{730, "2 years, 0 months"},
	}
	for _, tt := range tests {
		got := report.FormatAge(tt.days)
		if got != tt.expected {
			t.Errorf("FormatAge(%d) = %q, want %q", tt.days, got, tt.expected)
		}
	}
}

// containsText checks if the PDF raw bytes contain a text fragment.
// PDF text may appear in various encodings; we do a simple substring check.
func containsText(pdfData, text string) bool {
	return bytes.Contains([]byte(pdfData), []byte(text))
}

func TestGeneratePDF_NoKasaiDate(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	// Create baby without kasai_date
	baby := testutil.CreateTestBaby(t, db, user.ID)

	var buf bytes.Buffer
	err := report.Generate(db, baby, "2025-08-01", "2025-08-01", &buf)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	data := string(buf.Bytes())
	if !containsText(data, "Kasai: N/A") {
		t.Error("PDF should contain 'Kasai: N/A' when no kasai date")
	}
}

func TestGeneratePDF_OldBaby(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	// Create baby born 2 years ago to exercise year-based age formatting
	babyID := model.NewULID()
	_, err := db.Exec(
		`INSERT INTO babies (id, name, sex, date_of_birth)
		 VALUES (?, 'Old Baby', 'male', '2023-01-01')`,
		babyID,
	)
	if err != nil {
		t.Fatalf("insert baby: %v", err)
	}
	_, err = db.Exec(
		"INSERT INTO baby_parents (baby_id, user_id) VALUES (?, ?)",
		babyID, user.ID,
	)
	if err != nil {
		t.Fatalf("insert baby_parents: %v", err)
	}
	baby, err := getBabyByID(db, babyID)
	if err != nil {
		t.Fatalf("get baby: %v", err)
	}

	var buf bytes.Buffer
	err = report.Generate(db, baby, "2025-08-01", "2025-08-01", &buf)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	data := string(buf.Bytes())
	if !containsText(data, "years") {
		t.Error("PDF should contain 'years' in age for old baby")
	}
}

func TestGeneratePDF_InvalidDateRange(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	var buf bytes.Buffer
	err := report.Generate(db, baby, "invalid", "2025-08-01", &buf)
	if err == nil {
		t.Fatal("expected error for invalid date range")
	}
}

func TestGeneratePDF_NoteWithoutCategory(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := seedBabyWithKasai(t, db, user.ID)

	// Insert note without category
	ts := time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC).Format(model.DateTimeFormat)
	_, err := db.Exec(
		`INSERT INTO general_notes (id, baby_id, logged_by, timestamp, content)
		 VALUES (?, ?, ?, ?, 'Note without category')`,
		model.NewULID(), baby.ID, user.ID, ts,
	)
	if err != nil {
		t.Fatalf("insert note: %v", err)
	}

	var buf bytes.Buffer
	err = report.Generate(db, baby, "2025-08-01", "2025-08-01", &buf)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	data := string(buf.Bytes())
	if !containsText(data, "Note without category") {
		t.Error("PDF should contain note content")
	}
}
