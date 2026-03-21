package report_test

import (
	"bytes"
	"context"
	"database/sql"
	"image"
	"image/color"
	"image/png"
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/report"
	"github.com/ablankz/LittleLiver/backend/internal/storage"
	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

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
	baby := testutil.SeedBabyWithKasai(t, db, user.ID)
	seedReportData(t, db, baby.ID, user.ID)

	var buf bytes.Buffer
	err := report.Generate(db, nil, baby, "2025-08-01", "2025-08-01", &buf, nil)
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
	baby := testutil.SeedBabyWithKasai(t, db, user.ID)
	// No data seeded — empty date range

	var buf bytes.Buffer
	err := report.Generate(db, nil, baby, "2025-09-01", "2025-09-01", &buf, nil)
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
	baby := testutil.SeedBabyWithKasai(t, db, user.ID)
	seedReportData(t, db, baby.ID, user.ID)

	var buf bytes.Buffer
	err := report.Generate(db, nil, baby, "2025-08-01", "2025-08-01", &buf, nil)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	// The PDF should contain text strings for the baby name, sections, etc.
	// We check the raw PDF bytes for expected string fragments.
	data := string(buf.Bytes())

	expectedFragments := []string{
		"Report Baby",     // baby name
		"Stool Color Log", // section header
		"Temperature Log", // section header
		"Feeding Summary", // section header
		"Medication",      // medication section
		"UDCA",            // medication name
		"Notable",         // observations section
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
	err := report.Generate(db, nil, baby, "2025-08-01", "2025-08-01", &buf, nil)
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
	babies, err := store.GetBabiesByUserID(db, user.ID)
	if err != nil {
		t.Fatalf("get babies: %v", err)
	}
	var baby *model.Baby
	for i := range babies {
		if babies[i].ID == babyID {
			baby = &babies[i]
			break
		}
	}
	if baby == nil {
		t.Fatalf("baby %s not found", babyID)
	}

	var buf bytes.Buffer
	err = report.Generate(db, nil, baby, "2025-08-01", "2025-08-01", &buf, nil)
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
	err := report.Generate(db, nil, baby, "invalid", "2025-08-01", &buf, nil)
	if err == nil {
		t.Fatal("expected error for invalid date range")
	}
}

func TestGeneratePDF_NoteWithoutCategory(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.SeedBabyWithKasai(t, db, user.ID)

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
	err = report.Generate(db, nil, baby, "2025-08-01", "2025-08-01", &buf, nil)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	data := string(buf.Bytes())
	if !containsText(data, "Note without category") {
		t.Error("PDF should contain note content")
	}
}

// createTestPNG generates a small valid PNG image for testing.
func createTestPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for x := 0; x < 10; x++ {
		for y := 0; y < 10; y++ {
			img.Set(x, y, color.RGBA{R: 200, G: 150, B: 100, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode test PNG: %v", err)
	}
	return buf.Bytes()
}

// seedFullReportData seeds all types of data including weights, labs, and photos.
func seedFullReportData(t *testing.T, db *sql.DB, babyID, userID string, objStore *storage.MemoryStore) {
	t.Helper()
	seedReportData(t, db, babyID, userID)

	// Weights
	for i, w := range []float64{4.2, 4.5, 4.8} {
		ts := time.Date(2025, 8, 1, 8+i*4, 0, 0, 0, time.UTC).Format(model.DateTimeFormat)
		_, err := db.Exec(
			`INSERT INTO weights (id, baby_id, logged_by, timestamp, weight_kg)
			 VALUES (?, ?, ?, ?, ?)`,
			model.NewULID(), babyID, userID, ts, w,
		)
		if err != nil {
			t.Fatalf("insert weight: %v", err)
		}
	}

	// Lab results
	for i, val := range []string{"2.5", "2.1"} {
		ts := time.Date(2025, 8, 1, 8+i*6, 0, 0, 0, time.UTC).Format(model.DateTimeFormat)
		_, err := db.Exec(
			`INSERT INTO lab_results (id, baby_id, logged_by, timestamp, test_name, value, unit)
			 VALUES (?, ?, ?, ?, 'Bilirubin', ?, 'mg/dL')`,
			model.NewULID(), babyID, userID, ts, val,
		)
		if err != nil {
			t.Fatalf("insert lab result: %v", err)
		}
	}

	// Photo uploads with thumbnails
	if objStore != nil {
		thumbPNG := createTestPNG(t)
		thumbKey := "photos/thumb_test1.png"
		ctx := context.Background()
		if err := objStore.Put(ctx, thumbKey, bytes.NewReader(thumbPNG), "image/png"); err != nil {
			t.Fatalf("put thumbnail: %v", err)
		}
		uploadedAt := time.Date(2025, 8, 1, 10, 0, 0, 0, time.UTC).Format(model.DateTimeFormat)
		_, err := db.Exec(
			`INSERT INTO photo_uploads (id, baby_id, r2_key, thumbnail_key, uploaded_at, linked_at)
			 VALUES (?, ?, 'photos/test1.jpg', ?, ?, ?)`,
			model.NewULID(), babyID, thumbKey, uploadedAt, uploadedAt,
		)
		if err != nil {
			t.Fatalf("insert photo_upload: %v", err)
		}
	}
}

func TestGeneratePDF_WithCharts(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.SeedBabyWithKasai(t, db, user.ID)
	seedFullReportData(t, db, baby.ID, user.ID, nil)

	var buf bytes.Buffer
	err := report.Generate(db, nil, baby, "2025-08-01", "2025-08-01", &buf, nil)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	data := buf.Bytes()
	if len(data) < 5 {
		t.Fatal("PDF output too small")
	}
	if string(data[:5]) != "%PDF-" {
		t.Fatalf("output does not start with %%PDF-")
	}

	pdfStr := string(data)
	// PDF with charts should contain chart section headers
	if !containsText(pdfStr, "Stool Color Distribution") {
		t.Error("PDF should contain 'Stool Color Distribution' chart section")
	}
	if !containsText(pdfStr, "Weight Chart") {
		t.Error("PDF should contain 'Weight Chart' section")
	}
}

func TestGeneratePDF_WithLabTrendsChart(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.SeedBabyWithKasai(t, db, user.ID)
	seedFullReportData(t, db, baby.ID, user.ID, nil)

	var buf bytes.Buffer
	err := report.Generate(db, nil, baby, "2025-08-01", "2025-08-01", &buf, nil)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	pdfStr := string(buf.Bytes())
	if !containsText(pdfStr, "Lab Results Trends") {
		t.Error("PDF should contain 'Lab Results Trends' chart section")
	}
}

func TestGeneratePDF_WithPhotos(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.SeedBabyWithKasai(t, db, user.ID)
	objStore := storage.NewMemoryStore()
	seedFullReportData(t, db, baby.ID, user.ID, objStore)

	var buf bytes.Buffer
	err := report.Generate(db, objStore, baby, "2025-08-01", "2025-08-01", &buf, nil)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	pdfStr := string(buf.Bytes())
	if !containsText(pdfStr, "Photo Appendix") {
		t.Error("PDF should contain 'Photo Appendix' section when photos exist")
	}
}

func TestGeneratePDF_WithPhotos_NoStorage(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.SeedBabyWithKasai(t, db, user.ID)
	seedReportData(t, db, baby.ID, user.ID)

	// Pass nil storage — should not error, just skip photos
	var buf bytes.Buffer
	err := report.Generate(db, nil, baby, "2025-08-01", "2025-08-01", &buf, nil)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	pdfStr := string(buf.Bytes())
	// Without photos, no Photo Appendix
	if containsText(pdfStr, "Photo Appendix") {
		t.Error("PDF should not contain 'Photo Appendix' when no photos exist")
	}
}

func TestGeneratePDF_FullReport_ValidPDF(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.SeedBabyWithKasai(t, db, user.ID)
	objStore := storage.NewMemoryStore()
	seedFullReportData(t, db, baby.ID, user.ID, objStore)

	var buf bytes.Buffer
	err := report.Generate(db, objStore, baby, "2025-08-01", "2025-08-01", &buf, nil)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	data := buf.Bytes()
	if len(data) < 100 {
		t.Fatalf("full report PDF too small: %d bytes", len(data))
	}
	if string(data[:5]) != "%PDF-" {
		t.Fatal("output is not a valid PDF")
	}

	pdfStr := string(data)
	expected := []string{
		"Report Baby",
		"Summary",
		"Stool Color Log",
		"Temperature Log",
		"Feeding Summary",
		"Medication",
		"Stool Color Distribution",
		"Weight Chart",
		"Lab Results Trends",
		"Photo Appendix",
	}
	for _, frag := range expected {
		if !containsText(pdfStr, frag) {
			t.Errorf("full PDF missing expected section: %q", frag)
		}
	}
}
