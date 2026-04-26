package store_test

import (
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

func TestCreateContainer_Basic(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	med, err := store.CreateMedication(db, baby.ID, user.ID, "UDCA", "5mL", "twice_daily", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication: %v", err)
	}

	c, err := store.CreateMedicationContainer(db, store.CreateContainerParams{
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
	if c.ID == "" {
		t.Error("expected non-empty ID")
	}
	if c.QuantityRemaining != 100 {
		t.Errorf("QuantityRemaining = %v, want 100", c.QuantityRemaining)
	}
	if c.Depleted {
		t.Error("expected not depleted")
	}
	if c.MedicationID != med.ID {
		t.Errorf("MedicationID = %s, want %s", c.MedicationID, med.ID)
	}
}

func TestCreateContainer_RejectsInvalidUnit(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "UDCA", "5mL", "twice_daily", nil, nil, nil, nil)

	_, err := store.CreateMedicationContainer(db, store.CreateContainerParams{
		MedicationID:    med.ID,
		BabyID:          baby.ID,
		Kind:            "bottle",
		Unit:            "gallons",
		QuantityInitial: 1,
		CreatedBy:       user.ID,
	})
	if err == nil {
		t.Fatal("expected error for invalid unit")
	}
}

func TestCreateContainer_RejectsInvalidKind(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "UDCA", "5mL", "twice_daily", nil, nil, nil, nil)

	_, err := store.CreateMedicationContainer(db, store.CreateContainerParams{
		MedicationID:    med.ID,
		BabyID:          baby.ID,
		Kind:            "spaceship",
		Unit:            "ml",
		QuantityInitial: 1,
		CreatedBy:       user.ID,
	})
	if err == nil {
		t.Fatal("expected error for invalid kind")
	}
}

func TestGetContainerByID(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "UDCA", "5mL", "twice_daily", nil, nil, nil, nil)

	c, _ := store.CreateMedicationContainer(db, store.CreateContainerParams{
		MedicationID:    med.ID,
		BabyID:          baby.ID,
		Kind:            "bottle",
		Unit:            "ml",
		QuantityInitial: 100,
		CreatedBy:       user.ID,
	})

	got, err := store.GetMedicationContainerByID(db, baby.ID, c.ID)
	if err != nil {
		t.Fatalf("GetMedicationContainerByID: %v", err)
	}
	if got.ID != c.ID {
		t.Errorf("ID = %s, want %s", got.ID, c.ID)
	}
}

func TestListContainersByMedication(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "UDCA", "5mL", "twice_daily", nil, nil, nil, nil)

	for i := 0; i < 3; i++ {
		_, err := store.CreateMedicationContainer(db, store.CreateContainerParams{
			MedicationID:    med.ID,
			BabyID:          baby.ID,
			Kind:            "bottle",
			Unit:            "ml",
			QuantityInitial: 100,
			CreatedBy:       user.ID,
		})
		if err != nil {
			t.Fatalf("CreateMedicationContainer #%d: %v", i, err)
		}
	}

	list, err := store.ListMedicationContainers(db, baby.ID, med.ID)
	if err != nil {
		t.Fatalf("ListMedicationContainers: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("len(list) = %d, want 3", len(list))
	}
}

func TestUpdateContainer_OpensAndSetsExpiry(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "UDCA", "5mL", "twice_daily", nil, nil, nil, nil)

	c, _ := store.CreateMedicationContainer(db, store.CreateContainerParams{
		MedicationID:    med.ID,
		BabyID:          baby.ID,
		Kind:            "bottle",
		Unit:            "ml",
		QuantityInitial: 100,
		CreatedBy:       user.ID,
	})

	openedAt := time.Now().UTC().Format(model.DateTimeFormat)
	maxDays := 30
	expiry := "2027-12-31"
	updated, err := store.UpdateMedicationContainer(db, baby.ID, c.ID, store.UpdateContainerParams{
		UpdatedBy:           user.ID,
		Kind:                "bottle",
		Unit:                "ml",
		QuantityInitial:     100,
		QuantityRemaining:   90,
		OpenedAt:            &openedAt,
		MaxDaysAfterOpening: &maxDays,
		ExpirationDate:      &expiry,
		Depleted:            false,
	})
	if err != nil {
		t.Fatalf("UpdateMedicationContainer: %v", err)
	}
	if updated.OpenedAt == nil {
		t.Error("expected OpenedAt set")
	}
	if updated.MaxDaysAfterOpening == nil || *updated.MaxDaysAfterOpening != 30 {
		t.Errorf("MaxDaysAfterOpening = %v, want 30", updated.MaxDaysAfterOpening)
	}
	if updated.ExpirationDate == nil || *updated.ExpirationDate != "2027-12-31" {
		t.Errorf("ExpirationDate = %v, want 2027-12-31", updated.ExpirationDate)
	}
	if updated.QuantityRemaining != 90 {
		t.Errorf("QuantityRemaining = %v, want 90", updated.QuantityRemaining)
	}
}

func TestDeleteContainer(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "UDCA", "5mL", "twice_daily", nil, nil, nil, nil)

	c, _ := store.CreateMedicationContainer(db, store.CreateContainerParams{
		MedicationID:    med.ID,
		BabyID:          baby.ID,
		Kind:            "bottle",
		Unit:            "ml",
		QuantityInitial: 100,
		CreatedBy:       user.ID,
	})

	if err := store.DeleteMedicationContainer(db, baby.ID, c.ID); err != nil {
		t.Fatalf("DeleteMedicationContainer: %v", err)
	}

	_, err := store.GetMedicationContainerByID(db, baby.ID, c.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestAdjustContainerStock_AppliesDeltaAndAuditTrail(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "UDCA", "5mL", "twice_daily", nil, nil, nil, nil)

	c, _ := store.CreateMedicationContainer(db, store.CreateContainerParams{
		MedicationID:    med.ID,
		BabyID:          baby.ID,
		Kind:            "bottle",
		Unit:            "ml",
		QuantityInitial: 100,
		CreatedBy:       user.ID,
	})

	reason := "spilled"
	adj, err := store.AdjustMedicationContainer(db, baby.ID, c.ID, store.AdjustContainerParams{
		AdjustedBy: user.ID,
		Delta:      -50,
		Reason:     &reason,
	})
	if err != nil {
		t.Fatalf("AdjustMedicationContainer: %v", err)
	}
	if adj.Delta != -50 {
		t.Errorf("Delta = %v, want -50", adj.Delta)
	}

	got, _ := store.GetMedicationContainerByID(db, baby.ID, c.ID)
	if got.QuantityRemaining != 50 {
		t.Errorf("QuantityRemaining = %v, want 50", got.QuantityRemaining)
	}

	// Adjustment row exists in audit trail
	adjs, err := store.ListMedicationStockAdjustments(db, c.ID)
	if err != nil {
		t.Fatalf("ListMedicationStockAdjustments: %v", err)
	}
	if len(adjs) != 1 {
		t.Errorf("len(adjs) = %d, want 1", len(adjs))
	}
}

func TestAdjustContainerStock_AddIncreases(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "UDCA", "5mL", "twice_daily", nil, nil, nil, nil)

	c, _ := store.CreateMedicationContainer(db, store.CreateContainerParams{
		MedicationID:    med.ID,
		BabyID:          baby.ID,
		Kind:            "bottle",
		Unit:            "ml",
		QuantityInitial: 100,
		CreatedBy:       user.ID,
	})

	// First reduce so we can add back
	_, _ = store.AdjustMedicationContainer(db, baby.ID, c.ID, store.AdjustContainerParams{
		AdjustedBy: user.ID,
		Delta:      -30,
	})

	_, err := store.AdjustMedicationContainer(db, baby.ID, c.ID, store.AdjustContainerParams{
		AdjustedBy: user.ID,
		Delta:      +10,
	})
	if err != nil {
		t.Fatalf("AdjustMedicationContainer: %v", err)
	}

	got, _ := store.GetMedicationContainerByID(db, baby.ID, c.ID)
	if got.QuantityRemaining != 80 {
		t.Errorf("QuantityRemaining = %v, want 80", got.QuantityRemaining)
	}
}
