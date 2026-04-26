package store_test

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

// medWithDose creates a medication and configures its structured dose fields.
func medWithDose(t *testing.T, db *sql.DB, babyID, userID string, amount float64, unit string) *model.Medication {
	t.Helper()
	med, err := store.CreateMedication(db, babyID, userID, "TestMed", "5mL", "twice_daily", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication: %v", err)
	}
	_, err = store.SetMedicationStockFields(db, babyID, med.ID, userID, store.MedicationStockFields{
		DoseAmount: &amount,
		DoseUnit:   &unit,
	})
	if err != nil {
		t.Fatalf("SetMedicationStockFields: %v", err)
	}
	return med
}

// addContainer creates an opened container with given remaining quantity.
// openedOffset adjusts the opened_at timestamp; older = more negative.
func addContainer(t *testing.T, db *sql.DB, babyID, medID, userID string, remaining float64, openedOffset time.Duration) *model.MedicationContainer {
	t.Helper()
	openedAt := time.Now().UTC().Add(openedOffset).Format(model.DateTimeFormat)
	c, err := store.CreateMedicationContainer(db, store.CreateContainerParams{
		MedicationID:    medID,
		BabyID:          babyID,
		Kind:            "bottle",
		Unit:            "ml",
		QuantityInitial: remaining,
		OpenedAt:        &openedAt,
		CreatedBy:       userID,
	})
	if err != nil {
		t.Fatalf("CreateMedicationContainer: %v", err)
	}
	return c
}

func nowUTCStr() string {
	return time.Now().UTC().Format(model.DateTimeFormat)
}

func TestCreateMedLog_Skipped_NoDecrement(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	med := medWithDose(t, db, baby.ID, user.ID, 5, "ml")
	c := addContainer(t, db, baby.ID, med.ID, user.ID, 100, -time.Hour)

	reason := "vomited"
	ml, err := store.CreateMedLogWithStock(db, store.CreateMedLogParams{
		BabyID:       baby.ID,
		MedicationID: med.ID,
		LoggedBy:     user.ID,
		Skipped:      true,
		SkipReason:   &reason,
	})
	if err != nil {
		t.Fatalf("CreateMedLogWithStock: %v", err)
	}
	if ml.ContainerID != nil {
		t.Errorf("expected nil ContainerID for skipped log, got %v", *ml.ContainerID)
	}
	if ml.StockDeducted != nil {
		t.Errorf("expected nil StockDeducted for skipped log, got %v", *ml.StockDeducted)
	}

	got, _ := store.GetMedicationContainerByID(db, baby.ID, c.ID)
	if got.QuantityRemaining != 100 {
		t.Errorf("QuantityRemaining = %v, want 100 (no decrement on skip)", got.QuantityRemaining)
	}
}

func TestCreateMedLog_NoStructuredDose_NoDecrement(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	// medication has no DoseAmount/DoseUnit configured
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "Legacy", "5mL", "twice_daily", nil, nil, nil, nil)

	givenAt := nowUTCStr()
	ml, err := store.CreateMedLogWithStock(db, store.CreateMedLogParams{
		BabyID:       baby.ID,
		MedicationID: med.ID,
		LoggedBy:     user.ID,
		GivenAt:      &givenAt,
		Skipped:      false,
	})
	if err != nil {
		t.Fatalf("CreateMedLogWithStock: %v", err)
	}
	if ml.ContainerID != nil {
		t.Errorf("expected nil ContainerID for legacy med, got %v", *ml.ContainerID)
	}
}

func TestCreateMedLog_FIFO_PicksOldestOpenedContainer(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	med := medWithDose(t, db, baby.ID, user.ID, 5, "ml")

	older := addContainer(t, db, baby.ID, med.ID, user.ID, 100, -2*time.Hour)
	newer := addContainer(t, db, baby.ID, med.ID, user.ID, 100, -1*time.Hour)

	givenAt := nowUTCStr()
	ml, err := store.CreateMedLogWithStock(db, store.CreateMedLogParams{
		BabyID:       baby.ID,
		MedicationID: med.ID,
		LoggedBy:     user.ID,
		GivenAt:      &givenAt,
	})
	if err != nil {
		t.Fatalf("CreateMedLogWithStock: %v", err)
	}
	if ml.ContainerID == nil || *ml.ContainerID != older.ID {
		t.Errorf("expected ContainerID = %s (older), got %v", older.ID, ml.ContainerID)
	}

	gotOlder, _ := store.GetMedicationContainerByID(db, baby.ID, older.ID)
	gotNewer, _ := store.GetMedicationContainerByID(db, baby.ID, newer.ID)
	if gotOlder.QuantityRemaining != 95 {
		t.Errorf("older.QuantityRemaining = %v, want 95", gotOlder.QuantityRemaining)
	}
	if gotNewer.QuantityRemaining != 100 {
		t.Errorf("newer.QuantityRemaining = %v, want 100 (untouched)", gotNewer.QuantityRemaining)
	}
}

func TestCreateMedLog_AutoOpensSealedWhenNoOpenContainer(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	med := medWithDose(t, db, baby.ID, user.ID, 5, "ml")

	// Create sealed container (no opened_at)
	sealed, err := store.CreateMedicationContainer(db, store.CreateContainerParams{
		MedicationID:    med.ID,
		BabyID:          baby.ID,
		Kind:            "bottle",
		Unit:            "ml",
		QuantityInitial: 100,
		CreatedBy:       user.ID,
	})
	if err != nil {
		t.Fatalf("CreateMedicationContainer: %v", err)
	}

	givenAt := nowUTCStr()
	ml, err := store.CreateMedLogWithStock(db, store.CreateMedLogParams{
		BabyID:       baby.ID,
		MedicationID: med.ID,
		LoggedBy:     user.ID,
		GivenAt:      &givenAt,
	})
	if err != nil {
		t.Fatalf("CreateMedLogWithStock: %v", err)
	}
	if ml.ContainerID == nil || *ml.ContainerID != sealed.ID {
		t.Errorf("expected sealed container to be auto-opened, got %v", ml.ContainerID)
	}

	got, _ := store.GetMedicationContainerByID(db, baby.ID, sealed.ID)
	if got.OpenedAt == nil {
		t.Error("expected OpenedAt set after auto-open")
	}
	if got.QuantityRemaining != 95 {
		t.Errorf("QuantityRemaining = %v, want 95", got.QuantityRemaining)
	}
}

func TestCreateMedLog_ExplicitContainerOverride(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	med := medWithDose(t, db, baby.ID, user.ID, 5, "ml")

	older := addContainer(t, db, baby.ID, med.ID, user.ID, 100, -2*time.Hour)
	newer := addContainer(t, db, baby.ID, med.ID, user.ID, 100, -1*time.Hour)

	givenAt := nowUTCStr()
	ml, err := store.CreateMedLogWithStock(db, store.CreateMedLogParams{
		BabyID:       baby.ID,
		MedicationID: med.ID,
		LoggedBy:     user.ID,
		GivenAt:      &givenAt,
		ContainerID:  &newer.ID,
	})
	if err != nil {
		t.Fatalf("CreateMedLogWithStock: %v", err)
	}
	if ml.ContainerID == nil || *ml.ContainerID != newer.ID {
		t.Errorf("expected override to %s, got %v", newer.ID, ml.ContainerID)
	}
	gotOlder, _ := store.GetMedicationContainerByID(db, baby.ID, older.ID)
	gotNewer, _ := store.GetMedicationContainerByID(db, baby.ID, newer.ID)
	if gotOlder.QuantityRemaining != 100 {
		t.Errorf("older.QuantityRemaining = %v, want 100 (untouched)", gotOlder.QuantityRemaining)
	}
	if gotNewer.QuantityRemaining != 95 {
		t.Errorf("newer.QuantityRemaining = %v, want 95", gotNewer.QuantityRemaining)
	}
}

func TestCreateMedLog_EmptyInventory_LogsWithoutContainer(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	med := medWithDose(t, db, baby.ID, user.ID, 5, "ml")

	givenAt := nowUTCStr()
	ml, err := store.CreateMedLogWithStock(db, store.CreateMedLogParams{
		BabyID:       baby.ID,
		MedicationID: med.ID,
		LoggedBy:     user.ID,
		GivenAt:      &givenAt,
	})
	if err != nil {
		t.Fatalf("expected no error logging into empty inventory, got %v", err)
	}
	if ml.ContainerID != nil {
		t.Errorf("expected nil ContainerID with no containers, got %v", *ml.ContainerID)
	}
}

func TestCreateMedLog_DepletesAtZero(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	med := medWithDose(t, db, baby.ID, user.ID, 5, "ml")
	c := addContainer(t, db, baby.ID, med.ID, user.ID, 5, -time.Hour) // exactly one dose left

	givenAt := nowUTCStr()
	_, err := store.CreateMedLogWithStock(db, store.CreateMedLogParams{
		BabyID:       baby.ID,
		MedicationID: med.ID,
		LoggedBy:     user.ID,
		GivenAt:      &givenAt,
	})
	if err != nil {
		t.Fatalf("CreateMedLogWithStock: %v", err)
	}
	got, _ := store.GetMedicationContainerByID(db, baby.ID, c.ID)
	if got.QuantityRemaining != 0 {
		t.Errorf("QuantityRemaining = %v, want 0", got.QuantityRemaining)
	}
	if !got.Depleted {
		t.Error("expected Depleted=true when remaining hits 0")
	}
}

func TestCreateMedLog_SkipsDepletedContainer(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	med := medWithDose(t, db, baby.ID, user.ID, 5, "ml")

	depleted := addContainer(t, db, baby.ID, med.ID, user.ID, 0, -2*time.Hour)
	// Mark depleted explicitly
	_, err := store.UpdateMedicationContainer(db, baby.ID, depleted.ID, store.UpdateContainerParams{
		UpdatedBy:         user.ID,
		Kind:              depleted.Kind,
		Unit:              depleted.Unit,
		QuantityInitial:   depleted.QuantityInitial,
		QuantityRemaining: 0,
		Depleted:          true,
	})
	if err != nil {
		t.Fatalf("UpdateMedicationContainer: %v", err)
	}
	fresh := addContainer(t, db, baby.ID, med.ID, user.ID, 100, -time.Hour)

	givenAt := nowUTCStr()
	ml, err := store.CreateMedLogWithStock(db, store.CreateMedLogParams{
		BabyID:       baby.ID,
		MedicationID: med.ID,
		LoggedBy:     user.ID,
		GivenAt:      &givenAt,
	})
	if err != nil {
		t.Fatalf("CreateMedLogWithStock: %v", err)
	}
	if ml.ContainerID == nil || *ml.ContainerID != fresh.ID {
		t.Errorf("expected to skip depleted, got %v", ml.ContainerID)
	}
}

func TestCreateMedLog_RejectsContainerFromOtherMed(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	medA := medWithDose(t, db, baby.ID, user.ID, 5, "ml")
	medB, _ := store.CreateMedication(db, baby.ID, user.ID, "Other", "5mL", "twice_daily", nil, nil, nil, nil)

	containerOfB := addContainer(t, db, baby.ID, medB.ID, user.ID, 100, -time.Hour)

	givenAt := nowUTCStr()
	_, err := store.CreateMedLogWithStock(db, store.CreateMedLogParams{
		BabyID:       baby.ID,
		MedicationID: medA.ID,
		LoggedBy:     user.ID,
		GivenAt:      &givenAt,
		ContainerID:  &containerOfB.ID,
	})
	if err == nil {
		t.Fatal("expected error using container from a different medication")
	}
	if !strings.Contains(err.Error(), "container") {
		t.Errorf("expected error to mention container, got %v", err)
	}
}

func TestUpdateMedLog_SkipToGivenDeducts(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	med := medWithDose(t, db, baby.ID, user.ID, 5, "ml")
	c := addContainer(t, db, baby.ID, med.ID, user.ID, 100, -time.Hour)

	reason := "fell"
	ml, err := store.CreateMedLogWithStock(db, store.CreateMedLogParams{
		BabyID:       baby.ID,
		MedicationID: med.ID,
		LoggedBy:     user.ID,
		Skipped:      true,
		SkipReason:   &reason,
	})
	if err != nil {
		t.Fatalf("CreateMedLogWithStock: %v", err)
	}

	// Now flip to given.
	givenAt := nowUTCStr()
	updated, err := store.UpdateMedLogWithStock(db, store.UpdateMedLogParams{
		BabyID:    baby.ID,
		LogID:     ml.ID,
		UpdatedBy: user.ID,
		GivenAt:   &givenAt,
		Skipped:   false,
	})
	if err != nil {
		t.Fatalf("UpdateMedLogWithStock: %v", err)
	}
	if updated.ContainerID == nil || *updated.ContainerID != c.ID {
		t.Errorf("expected ContainerID = %s, got %v", c.ID, updated.ContainerID)
	}
	got, _ := store.GetMedicationContainerByID(db, baby.ID, c.ID)
	if got.QuantityRemaining != 95 {
		t.Errorf("QuantityRemaining = %v, want 95", got.QuantityRemaining)
	}
}

func TestUpdateMedLog_GivenToSkipRestores(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	med := medWithDose(t, db, baby.ID, user.ID, 5, "ml")
	c := addContainer(t, db, baby.ID, med.ID, user.ID, 100, -time.Hour)

	givenAt := nowUTCStr()
	ml, _ := store.CreateMedLogWithStock(db, store.CreateMedLogParams{
		BabyID: baby.ID, MedicationID: med.ID, LoggedBy: user.ID, GivenAt: &givenAt,
	})

	reason := "vomited"
	updated, err := store.UpdateMedLogWithStock(db, store.UpdateMedLogParams{
		BabyID:     baby.ID,
		LogID:      ml.ID,
		UpdatedBy:  user.ID,
		Skipped:    true,
		SkipReason: &reason,
	})
	if err != nil {
		t.Fatalf("UpdateMedLogWithStock: %v", err)
	}
	if updated.ContainerID != nil {
		t.Errorf("expected ContainerID cleared on skip, got %v", *updated.ContainerID)
	}
	got, _ := store.GetMedicationContainerByID(db, baby.ID, c.ID)
	if got.QuantityRemaining != 100 {
		t.Errorf("QuantityRemaining = %v, want 100 (restored)", got.QuantityRemaining)
	}
}

func TestUpdateMedLog_ContainerSwapMovesDeduction(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	med := medWithDose(t, db, baby.ID, user.ID, 5, "ml")

	a := addContainer(t, db, baby.ID, med.ID, user.ID, 100, -2*time.Hour)
	b := addContainer(t, db, baby.ID, med.ID, user.ID, 100, -1*time.Hour)

	givenAt := nowUTCStr()
	ml, _ := store.CreateMedLogWithStock(db, store.CreateMedLogParams{
		BabyID: baby.ID, MedicationID: med.ID, LoggedBy: user.ID, GivenAt: &givenAt,
	})
	if *ml.ContainerID != a.ID {
		t.Fatalf("expected initial deduction from A")
	}

	updated, err := store.UpdateMedLogWithStock(db, store.UpdateMedLogParams{
		BabyID:      baby.ID,
		LogID:       ml.ID,
		UpdatedBy:   user.ID,
		GivenAt:     &givenAt,
		ContainerID: &b.ID,
	})
	if err != nil {
		t.Fatalf("UpdateMedLogWithStock: %v", err)
	}
	if *updated.ContainerID != b.ID {
		t.Errorf("expected container moved to B, got %v", *updated.ContainerID)
	}
	gotA, _ := store.GetMedicationContainerByID(db, baby.ID, a.ID)
	gotB, _ := store.GetMedicationContainerByID(db, baby.ID, b.ID)
	if gotA.QuantityRemaining != 100 {
		t.Errorf("A.QuantityRemaining = %v, want 100 (restored)", gotA.QuantityRemaining)
	}
	if gotB.QuantityRemaining != 95 {
		t.Errorf("B.QuantityRemaining = %v, want 95 (deducted)", gotB.QuantityRemaining)
	}
}

func TestDeleteMedLog_RestoresStock(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	med := medWithDose(t, db, baby.ID, user.ID, 5, "ml")
	c := addContainer(t, db, baby.ID, med.ID, user.ID, 100, -time.Hour)

	givenAt := nowUTCStr()
	ml, _ := store.CreateMedLogWithStock(db, store.CreateMedLogParams{
		BabyID: baby.ID, MedicationID: med.ID, LoggedBy: user.ID, GivenAt: &givenAt,
	})

	if err := store.DeleteMedLogWithStock(db, baby.ID, ml.ID); err != nil {
		t.Fatalf("DeleteMedLogWithStock: %v", err)
	}
	got, _ := store.GetMedicationContainerByID(db, baby.ID, c.ID)
	if got.QuantityRemaining != 100 {
		t.Errorf("QuantityRemaining = %v, want 100 (restored on delete)", got.QuantityRemaining)
	}
}
