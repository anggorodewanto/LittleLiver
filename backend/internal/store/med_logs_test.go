package store_test

import (
	"database/sql"
	"testing"

	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

func createTestMed(t *testing.T, db *sql.DB, babyID, userID string) string {
	t.Helper()
	med, err := store.CreateMedication(db, babyID, userID, "Ursodiol", "50mg", "twice_daily", nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication: %v", err)
	}
	return med.ID
}

func strPtr(s string) *string { return &s }

func TestCreateMedLog_Given(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	medID := createTestMed(t, db, baby.ID, user.ID)

	ml, err := store.CreateMedLog(db, baby.ID, medID, user.ID, nil, strPtr("2026-03-17T08:00:00Z"), false, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedLog: %v", err)
	}
	if ml.Skipped {
		t.Error("expected skipped=false")
	}
	if ml.GivenAt == nil {
		t.Error("expected non-nil given_at")
	}
	if ml.MedicationID != medID {
		t.Errorf("expected medication_id=%s, got %s", medID, ml.MedicationID)
	}
	if ml.LoggedBy != user.ID {
		t.Errorf("expected logged_by=%s, got %s", user.ID, ml.LoggedBy)
	}
}

func TestCreateMedLog_Skipped(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	medID := createTestMed(t, db, baby.ID, user.ID)

	ml, err := store.CreateMedLog(db, baby.ID, medID, user.ID, nil, nil, true, strPtr("vomited"), nil)
	if err != nil {
		t.Fatalf("CreateMedLog: %v", err)
	}
	if !ml.Skipped {
		t.Error("expected skipped=true")
	}
	if ml.GivenAt != nil {
		t.Errorf("expected given_at=nil, got %v", ml.GivenAt)
	}
	if ml.SkipReason == nil || *ml.SkipReason != "vomited" {
		t.Errorf("expected skip_reason=vomited, got %v", ml.SkipReason)
	}
}

func TestGetMedLogByID_NotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	_, err := store.GetMedLogByID(db, baby.ID, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent med-log")
	}
}

func TestListMedLogs_Empty(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	page, err := store.ListMedLogs(db, baby.ID, nil, nil, nil, nil, 50)
	if err != nil {
		t.Fatalf("ListMedLogs: %v", err)
	}
	if len(page.Data) != 0 {
		t.Errorf("expected 0, got %d", len(page.Data))
	}
}

func TestListMedLogs_FilterByMedicationID(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	med1 := createTestMed(t, db, baby.ID, user.ID)

	med2, err := store.CreateMedication(db, baby.ID, user.ID, "VitD", "400IU", "once_daily", nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication: %v", err)
	}

	_, _ = store.CreateMedLog(db, baby.ID, med1, user.ID, nil, strPtr("2026-03-17T08:00:00Z"), false, nil, nil)
	_, _ = store.CreateMedLog(db, baby.ID, med2.ID, user.ID, nil, strPtr("2026-03-17T09:00:00Z"), false, nil, nil)

	page, err := store.ListMedLogs(db, baby.ID, &med1, nil, nil, nil, 50)
	if err != nil {
		t.Fatalf("ListMedLogs: %v", err)
	}
	if len(page.Data) != 1 {
		t.Errorf("expected 1, got %d", len(page.Data))
	}
}

func TestListMedLogs_FilterByFromTo(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	medID := createTestMed(t, db, baby.ID, user.ID)

	_, _ = store.CreateMedLog(db, baby.ID, medID, user.ID, nil, strPtr("2026-03-17T08:00:00Z"), false, nil, nil)

	// Use from date that is today — created_at is CURRENT_TIMESTAMP which is "now"
	from := "2026-03-01"
	to := "2026-03-18"
	page, err := store.ListMedLogs(db, baby.ID, nil, &from, &to, nil, 50)
	if err != nil {
		t.Fatalf("ListMedLogs: %v", err)
	}
	if len(page.Data) != 1 {
		t.Errorf("expected 1, got %d", len(page.Data))
	}
}

func TestListMedLogs_Pagination(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	medID := createTestMed(t, db, baby.ID, user.ID)

	// Create 3 logs
	for i := 0; i < 3; i++ {
		_, err := store.CreateMedLog(db, baby.ID, medID, user.ID, nil, strPtr("2026-03-17T08:00:00Z"), false, nil, nil)
		if err != nil {
			t.Fatalf("CreateMedLog: %v", err)
		}
	}

	// Fetch with limit=2 to trigger pagination
	page, err := store.ListMedLogs(db, baby.ID, nil, nil, nil, nil, 2)
	if err != nil {
		t.Fatalf("ListMedLogs: %v", err)
	}
	if len(page.Data) != 2 {
		t.Errorf("expected 2, got %d", len(page.Data))
	}
	if page.NextCursor == nil {
		t.Fatal("expected non-nil next_cursor")
	}

	// Fetch next page using cursor
	page2, err := store.ListMedLogs(db, baby.ID, nil, nil, nil, page.NextCursor, 2)
	if err != nil {
		t.Fatalf("ListMedLogs page2: %v", err)
	}
	if len(page2.Data) != 1 {
		t.Errorf("expected 1, got %d", len(page2.Data))
	}
	if page2.NextCursor != nil {
		t.Error("expected nil next_cursor on last page")
	}
}

func TestUpdateMedLog_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	medID := createTestMed(t, db, baby.ID, user.ID)

	ml, err := store.CreateMedLog(db, baby.ID, medID, user.ID, nil, strPtr("2026-03-17T08:00:00Z"), false, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedLog: %v", err)
	}

	updated, err := store.UpdateMedLog(db, baby.ID, ml.ID, user.ID, nil, strPtr("2026-03-17T08:15:00Z"), false, nil, strPtr("late"))
	if err != nil {
		t.Fatalf("UpdateMedLog: %v", err)
	}
	if updated.UpdatedBy == nil || *updated.UpdatedBy != user.ID {
		t.Errorf("expected updated_by=%s, got %v", user.ID, updated.UpdatedBy)
	}
	if updated.Notes == nil || *updated.Notes != "late" {
		t.Errorf("expected notes=late, got %v", updated.Notes)
	}
}

func TestUpdateMedLog_NotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	_, err := store.UpdateMedLog(db, baby.ID, "nonexistent", user.ID, nil, strPtr("2026-03-17T08:00:00Z"), false, nil, nil)
	if err == nil {
		t.Error("expected error for nonexistent med-log")
	}
}

func TestDeleteMedLog_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	medID := createTestMed(t, db, baby.ID, user.ID)

	ml, err := store.CreateMedLog(db, baby.ID, medID, user.ID, nil, strPtr("2026-03-17T08:00:00Z"), false, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedLog: %v", err)
	}

	err = store.DeleteMedLog(db, baby.ID, ml.ID)
	if err != nil {
		t.Fatalf("DeleteMedLog: %v", err)
	}

	_, err = store.GetMedLogByID(db, baby.ID, ml.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestDeleteMedLog_NotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	err := store.DeleteMedLog(db, baby.ID, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent med-log")
	}
}

func TestGetMedicationBabyID_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	medID := createTestMed(t, db, baby.ID, user.ID)

	babyID, err := store.GetMedicationBabyID(db, medID)
	if err != nil {
		t.Fatalf("GetMedicationBabyID: %v", err)
	}
	if babyID != baby.ID {
		t.Errorf("expected %s, got %s", baby.ID, babyID)
	}
}

func TestGetMedicationBabyID_NotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	_, err := store.GetMedicationBabyID(db, "nonexistent")
	if err == nil {
		t.Error("expected error")
	}
}

func TestCreateMedLog_WithScheduledTime(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	medID := createTestMed(t, db, baby.ID, user.ID)

	sched := "2026-03-17T08:00:00Z"
	given := "2026-03-17T08:05:00Z"
	ml, err := store.CreateMedLog(db, baby.ID, medID, user.ID, &sched, &given, false, nil, strPtr("on time"))
	if err != nil {
		t.Fatalf("CreateMedLog: %v", err)
	}
	if ml.ScheduledTime == nil {
		t.Error("expected non-nil scheduled_time")
	}
	if ml.Notes == nil || *ml.Notes != "on time" {
		t.Errorf("expected notes='on time', got %v", ml.Notes)
	}
}

func TestListMedLogs_InvalidFromDate(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	badDate := "not-a-date"
	_, err := store.ListMedLogs(db, baby.ID, nil, &badDate, nil, nil, 50)
	if err == nil {
		t.Error("expected error for invalid from date")
	}
}

func TestListMedLogs_InvalidToDate(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	badDate := "not-a-date"
	_, err := store.ListMedLogs(db, baby.ID, nil, nil, &badDate, nil, 50)
	if err == nil {
		t.Error("expected error for invalid to date")
	}
}
