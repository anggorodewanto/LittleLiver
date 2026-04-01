package integration_test

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/handler"
	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/storage"
)

// seedDaysOfData inserts 30 days of varied health data for a baby via the API
// and directly into the DB for data types without API seeding convenience.
func seedDaysOfData(t *testing.T, db *sql.DB, babyID, userID string, objStore *storage.MemoryStore) {
	t.Helper()
	baseDate := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()

	for day := 0; day < 30; day++ {
		date := baseDate.AddDate(0, 0, day)

		// Feedings: 4 per day, alternating types
		feedTypes := []string{"formula", "breast_milk", "formula", "breast_milk"}
		for i, ft := range feedTypes {
			ts := date.Add(time.Duration(6+i*4) * time.Hour).Format(model.DateTimeFormat)
			vol := 100.0 + float64(day)*2 + float64(i)*10
			cal := vol * 0.67
			_, err := tx.Exec(
				`INSERT INTO feedings (id, baby_id, logged_by, timestamp, feed_type, volume_ml, calories)
				 VALUES (?, ?, ?, ?, ?, ?, ?)`,
				model.NewULID(), babyID, userID, ts, ft, vol, cal,
			)
			if err != nil {
				t.Fatalf("insert feeding day %d feed %d: %v", day, i, err)
			}
		}

		// Stools: 1-2 per day with varied color ratings
		stoolCount := 1 + (day % 2)
		for i := 0; i < stoolCount; i++ {
			ts := date.Add(time.Duration(9+i*5) * time.Hour).Format(model.DateTimeFormat)
			colorRating := (day%7 + 1) // 1-7 color scale
			_, err := tx.Exec(
				`INSERT INTO stools (id, baby_id, logged_by, timestamp, color_rating, color_label)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				model.NewULID(), babyID, userID, ts, colorRating, "varied",
			)
			if err != nil {
				t.Fatalf("insert stool day %d: %v", day, err)
			}
		}

		// Urine: 3 per day
		for i := 0; i < 3; i++ {
			ts := date.Add(time.Duration(7+i*4) * time.Hour).Format(model.DateTimeFormat)
			_, err := tx.Exec(
				`INSERT INTO urine (id, baby_id, logged_by, timestamp)
				 VALUES (?, ?, ?, ?)`,
				model.NewULID(), babyID, userID, ts,
			)
			if err != nil {
				t.Fatalf("insert urine day %d: %v", day, err)
			}
		}

		// Temperatures: 2 per day, occasional fevers
		for i, val := range []float64{36.5 + float64(day%3)*0.3, 37.0 + float64(day%5)*0.4} {
			ts := date.Add(time.Duration(8+i*8) * time.Hour).Format(model.DateTimeFormat)
			_, err := tx.Exec(
				`INSERT INTO temperatures (id, baby_id, logged_by, timestamp, value, method)
				 VALUES (?, ?, ?, ?, ?, 'axillary')`,
				model.NewULID(), babyID, userID, ts, val,
			)
			if err != nil {
				t.Fatalf("insert temperature day %d: %v", day, err)
			}
		}

		// Weights: once every 3 days
		if day%3 == 0 {
			ts := date.Add(8 * time.Hour).Format(model.DateTimeFormat)
			weight := 3.5 + float64(day)*0.05
			_, err := tx.Exec(
				`INSERT INTO weights (id, baby_id, logged_by, timestamp, weight_kg)
				 VALUES (?, ?, ?, ?, ?)`,
				model.NewULID(), babyID, userID, ts, weight,
			)
			if err != nil {
				t.Fatalf("insert weight day %d: %v", day, err)
			}
		}
	}

	// Lab results: a few entries at different dates
	labTests := []struct {
		day      int
		testName string
		value    string
		unit     string
	}{
		{0, "Bilirubin", "3.5", "mg/dL"},
		{7, "Bilirubin", "2.8", "mg/dL"},
		{14, "Bilirubin", "2.1", "mg/dL"},
		{21, "Bilirubin", "1.8", "mg/dL"},
		{0, "ALT", "55", "U/L"},
		{14, "ALT", "42", "U/L"},
		{28, "ALT", "38", "U/L"},
		{0, "GGT", "120", "U/L"},
		{14, "GGT", "95", "U/L"},
	}
	for _, lt := range labTests {
		ts := baseDate.AddDate(0, 0, lt.day).Add(10 * time.Hour).Format(model.DateTimeFormat)
		_, err := tx.Exec(
			`INSERT INTO lab_results (id, baby_id, logged_by, timestamp, test_name, value, unit)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			model.NewULID(), babyID, userID, ts, lt.testName, lt.value, lt.unit,
		)
		if err != nil {
			t.Fatalf("insert lab result %s day %d: %v", lt.testName, lt.day, err)
		}
	}

	// General notes
	notes := []struct {
		day      int
		content  string
		category string
	}{
		{1, "Baby seemed extra fussy today", "observation"},
		{5, "Jaundice appears to be improving", "clinical"},
		{15, "Good appetite, no vomiting", "feeding"},
		{25, "Mild rash on left arm", "skin"},
	}
	for _, n := range notes {
		ts := baseDate.AddDate(0, 0, n.day).Add(12 * time.Hour).Format(model.DateTimeFormat)
		_, err := tx.Exec(
			`INSERT INTO general_notes (id, baby_id, logged_by, timestamp, content, category)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			model.NewULID(), babyID, userID, ts, n.content, n.category,
		)
		if err != nil {
			t.Fatalf("insert note day %d: %v", n.day, err)
		}
	}

	// Medications and med logs
	medID := model.NewULID()
	_, err = tx.Exec(
		`INSERT INTO medications (id, baby_id, logged_by, name, dose, frequency, active)
		 VALUES (?, ?, ?, 'UDCA', '25mg', 'twice_daily', 1)`,
		medID, babyID, userID,
	)
	if err != nil {
		t.Fatalf("insert medication: %v", err)
	}

	med2ID := model.NewULID()
	_, err = tx.Exec(
		`INSERT INTO medications (id, baby_id, logged_by, name, dose, frequency, active)
		 VALUES (?, ?, ?, 'Vitamin K', '1mg', 'once_daily', 1)`,
		med2ID, babyID, userID,
	)
	if err != nil {
		t.Fatalf("insert medication 2: %v", err)
	}

	// Med logs: UDCA twice daily for 30 days, with occasional skips
	for day := 0; day < 30; day++ {
		for _, hour := range []int{8, 20} {
			ts := baseDate.AddDate(0, 0, day).Add(time.Duration(hour) * time.Hour).Format(model.DateTimeFormat)
			skipped := (day == 5 && hour == 20) || (day == 12 && hour == 8)
			var givenAt *string
			if !skipped {
				givenAt = &ts
			}
			_, err := tx.Exec(
				`INSERT INTO med_logs (id, medication_id, baby_id, logged_by, scheduled_time, given_at, skipped, created_at)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				model.NewULID(), medID, babyID, userID, ts, givenAt, skipped, ts,
			)
			if err != nil {
				t.Fatalf("insert med_log day %d hour %d: %v", day, hour, err)
			}
		}
	}

	// Vitamin K daily
	for day := 0; day < 30; day++ {
		ts := baseDate.AddDate(0, 0, day).Add(9 * time.Hour).Format(model.DateTimeFormat)
		_, err := tx.Exec(
			`INSERT INTO med_logs (id, medication_id, baby_id, logged_by, scheduled_time, given_at, skipped, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, 0, ?)`,
			model.NewULID(), med2ID, babyID, userID, ts, ts, ts,
		)
		if err != nil {
			t.Fatalf("insert vitamin K med_log day %d: %v", day, err)
		}
	}

	// Photos with thumbnails
	if objStore != nil {
		thumbPNG := createTestJPEGData(t, 20, 20)
		for i := 0; i < 3; i++ {
			thumbKey := fmt.Sprintf("photos/report_thumb_%d.jpg", i)
			r2Key := fmt.Sprintf("photos/report_photo_%d.jpg", i)
			ctx := context.Background()
			if err := objStore.Put(ctx, r2Key, bytes.NewReader(thumbPNG), "image/jpeg"); err != nil {
				t.Fatalf("put photo %d: %v", i, err)
			}
			if err := objStore.Put(ctx, thumbKey, bytes.NewReader(thumbPNG), "image/jpeg"); err != nil {
				t.Fatalf("put thumbnail %d: %v", i, err)
			}
			uploadedAt := baseDate.AddDate(0, 0, i*10).Add(10 * time.Hour).Format(model.DateTimeFormat)
			_, err := tx.Exec(
				`INSERT INTO photo_uploads (id, baby_id, r2_key, thumbnail_key, uploaded_at, linked_at)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				model.NewULID(), babyID, r2Key, thumbKey, uploadedAt, uploadedAt,
			)
			if err != nil {
				t.Fatalf("insert photo_upload %d: %v", i, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit transaction: %v", err)
	}
}


// TestReportGeneration_FullIntegration seeds 30 days of varied data (all metric
// types, photos, medications) then generates a PDF report and validates that
// it is structurally valid and contains all expected sections and key values.
func TestReportGeneration_FullIntegration(t *testing.T) {
	t.Parallel()

	objStore := storage.NewMemoryStore()
	srv, db, cleanup := setupIntegrationServer(t, handler.WithObjectStore(objStore))
	defer cleanup()

	client := newTestClient(t, srv, db)

	// Create baby with kasai_date via API
	status, babyResp := client.doJSON(http.MethodPost, "/api/babies", map[string]any{
		"name":          "Report Test Baby",
		"sex":           "female",
		"date_of_birth": "2025-06-15",
		"kasai_date":    "2025-07-01",
	})
	if status != http.StatusCreated {
		t.Fatalf("expected 201 creating baby, got %d: %v", status, babyResp)
	}
	babyID := babyResp["id"].(string)

	// Seed 30 days of comprehensive data
	seedDaysOfData(t, db, babyID, client.userID, objStore)

	// Generate report for the full 30-day range
	reportPath := fmt.Sprintf("/api/babies/%s/report?from=2025-08-01&to=2025-08-30", babyID)
	respStatus, pdfBytes := client.doRawBytes(reportPath)

	// === Verify HTTP response ===
	if respStatus != http.StatusOK {
		t.Fatalf("expected 200 from report endpoint, got %d: %s", respStatus, string(pdfBytes))
	}

	// === Verify valid PDF ===
	if len(pdfBytes) < 100 {
		t.Fatalf("PDF too small: %d bytes", len(pdfBytes))
	}
	if string(pdfBytes[:5]) != "%PDF-" {
		t.Fatalf("response is not a valid PDF, starts with: %q", string(pdfBytes[:20]))
	}

	pdfText := string(pdfBytes)

	// === Verify baby header info ===
	headerChecks := []string{
		"Report Test Baby",           // baby name
		"LittleLiver Health Report",  // report title
		"female",                     // sex (may appear in header or within text)
		"2025-06-15",                 // DOB
		"2025-08-01",                 // period start
		"2025-08-30",                 // period end
	}
	for _, expected := range headerChecks {
		if !strings.Contains(pdfText, expected) {
			t.Errorf("PDF missing header info: %q", expected)
		}
	}

	// === Verify Kasai date info ===
	if !strings.Contains(pdfText, "Post-Kasai") && !strings.Contains(pdfText, "Kasai") {
		t.Error("PDF should contain Kasai date info")
	}

	// === Verify section headers ===
	sections := []string{
		"Summary",
		"Stool Color Log",
		"Temperature Log",
		"Feeding Summary",
		"Medication Adherence",
		"Notable Observations",
		"Stool Color Distribution",
		"Weight Chart",
		"Lab Trends:",
		"Photo Appendix",
	}
	for _, section := range sections {
		if !strings.Contains(pdfText, section) {
			t.Errorf("PDF missing section: %q", section)
		}
	}

	// === Verify stool data present ===
	// We inserted stools with color ratings 1-7 across 30 days
	// The stool color log table should contain the entries
	// Check that at least some color score values appear
	for _, score := range []string{"1", "2", "3", "4", "5"} {
		// These appear as text in table cells
		if !strings.Contains(pdfText, score) {
			t.Errorf("PDF missing stool color score: %s", score)
		}
	}

	// === Verify feeding summary data ===
	// We inserted 4 feedings per day at ~100-160ml each
	// The feeding summary should show volume and calorie data
	if !strings.Contains(pdfText, "mL") || !strings.Contains(pdfText, "kcal") {
		t.Error("PDF should contain feeding volume (mL) and calorie (kcal) data")
	}

	// === Verify medication names appear ===
	if !strings.Contains(pdfText, "UDCA") {
		t.Error("PDF should contain medication name 'UDCA'")
	}
	if !strings.Contains(pdfText, "Vitamin K") {
		t.Error("PDF should contain medication name 'Vitamin K'")
	}

	// === Verify medication adherence data ===
	// UDCA had 2 skips out of 60 doses = ~97% adherence
	if !strings.Contains(pdfText, "25mg") {
		t.Error("PDF should contain UDCA dose '25mg'")
	}

	// === Verify lab chart section is present ===
	// Lab values are rendered as chart images, so we verify the section header
	// exists (already checked above in sections list) and that the chart data
	// was embedded (the PDF size should be substantial with chart PNGs).
	if len(pdfBytes) < 5000 {
		t.Errorf("PDF with charts and data should be substantial, got only %d bytes", len(pdfBytes))
	}

	// === Verify notes content ===
	noteFragments := []string{
		"fussy",
		"Jaundice",
		"appetite",
		"rash",
	}
	for _, frag := range noteFragments {
		if !strings.Contains(pdfText, frag) {
			t.Errorf("PDF missing note content fragment: %q", frag)
		}
	}

	// === Verify weight data contributed to chart ===
	if !strings.Contains(pdfText, "Weight Chart") {
		t.Error("PDF should contain 'Weight Chart' section with weight data")
	}

	// === Verify PDF ends properly (should contain %%EOF marker) ===
	if !strings.Contains(pdfText[len(pdfText)-100:], "%%EOF") {
		t.Error("PDF should end with EOF marker")
	}
}

// TestReportGeneration_EmptyRange verifies that generating a report for a
// date range with no data produces a valid PDF with section headers but
// empty-data messages.
func TestReportGeneration_EmptyRange(t *testing.T) {
	t.Parallel()

	srv, db, cleanup := setupIntegrationServer(t)
	defer cleanup()

	client := newTestClient(t, srv, db)

	status, babyResp := client.doJSON(http.MethodPost, "/api/babies", map[string]any{
		"name":          "Empty Report Baby",
		"sex":           "male",
		"date_of_birth": "2025-06-01",
	})
	if status != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %v", status, babyResp)
	}
	babyID := babyResp["id"].(string)

	reportPath := fmt.Sprintf("/api/babies/%s/report?from=2025-09-01&to=2025-09-30", babyID)
	respStatus, pdfBytes := client.doRawBytes(reportPath)

	if respStatus != http.StatusOK {
		t.Fatalf("expected 200, got %d", respStatus)
	}

	if len(pdfBytes) < 5 || string(pdfBytes[:5]) != "%PDF-" {
		t.Fatal("response is not a valid PDF")
	}

	pdfText := string(pdfBytes)

	// Should still contain section headers
	if !strings.Contains(pdfText, "LittleLiver Health Report") {
		t.Error("empty-range PDF missing report title")
	}
	if !strings.Contains(pdfText, "Empty Report Baby") {
		t.Error("empty-range PDF missing baby name")
	}

	// Should not contain photo appendix when no photos
	if strings.Contains(pdfText, "Photo Appendix") {
		t.Error("empty-range PDF should not have Photo Appendix")
	}
}

// TestReportGeneration_WithoutPhotos verifies that a report without photos
// still generates correctly and omits the Photo Appendix section.
func TestReportGeneration_WithoutPhotos(t *testing.T) {
	t.Parallel()

	srv, db, cleanup := setupIntegrationServer(t)
	defer cleanup()

	client := newTestClient(t, srv, db)

	status, babyResp := client.doJSON(http.MethodPost, "/api/babies", map[string]any{
		"name":          "No Photo Baby",
		"sex":           "female",
		"date_of_birth": "2025-06-15",
		"kasai_date":    "2025-07-01",
	})
	if status != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %v", status, babyResp)
	}
	babyID := babyResp["id"].(string)

	// Seed data without photos (nil objStore)
	seedDaysOfData(t, db, babyID, client.userID, nil)

	reportPath := fmt.Sprintf("/api/babies/%s/report?from=2025-08-01&to=2025-08-30", babyID)
	respStatus, pdfBytes := client.doRawBytes(reportPath)

	if respStatus != http.StatusOK {
		t.Fatalf("expected 200, got %d", respStatus)
	}

	pdfText := string(pdfBytes)
	if string(pdfBytes[:5]) != "%PDF-" {
		t.Fatal("not a valid PDF")
	}

	// All sections except Photo Appendix should be present
	for _, section := range []string{"Summary", "Stool Color Log", "Feeding Summary", "Medication Adherence"} {
		if !strings.Contains(pdfText, section) {
			t.Errorf("PDF missing section: %q", section)
		}
	}

	// Photo Appendix should NOT be present (no object store = no photos)
	if strings.Contains(pdfText, "Photo Appendix") {
		t.Error("PDF should not contain Photo Appendix when no photos exist")
	}
}
