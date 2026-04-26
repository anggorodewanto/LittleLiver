package store_test

import (
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

// findAlert returns the first alert matching alertType, or nil.
func findAlert(alerts []store.Alert, alertType string) *store.Alert {
	for i := range alerts {
		if alerts[i].AlertType == alertType {
			return &alerts[i]
		}
	}
	return nil
}

func TestLowStockAlert_FiresWhenDosesAtOrBelowThreshold(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	med := medWithDose(t, db, baby.ID, user.ID, 5, "ml")
	low := 3
	_, err := store.SetMedicationStockFields(db, baby.ID, med.ID, user.ID, store.MedicationStockFields{
		LowStockThreshold: &low,
	})
	if err != nil {
		t.Fatalf("SetMedicationStockFields: %v", err)
	}

	// 15 mL / 5 mL/dose = 3 doses left → should fire (<=)
	addContainer(t, db, baby.ID, med.ID, user.ID, 15, -time.Hour)

	alerts, err := store.GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("GetActiveAlerts: %v", err)
	}
	a := findAlert(alerts, store.AlertTypeLowStock)
	if a == nil {
		t.Fatalf("expected low_stock alert, got %d alerts", len(alerts))
	}
	if a.MedicationID != med.ID {
		t.Errorf("MedicationID = %s, want %s", a.MedicationID, med.ID)
	}
}

func TestLowStockAlert_DefaultsToThreeDoses(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	med := medWithDose(t, db, baby.ID, user.ID, 5, "ml") // no LowStockThreshold set → default 3

	// 16 mL / 5 = 3.2 doses → ceil/floor: depends on impl. Use 14 mL for clear <= 3 (2.8 doses).
	addContainer(t, db, baby.ID, med.ID, user.ID, 14, -time.Hour)

	alerts, _ := store.GetActiveAlerts(db, baby.ID)
	if findAlert(alerts, store.AlertTypeLowStock) == nil {
		t.Errorf("expected default-threshold low_stock alert; got %v", alerts)
	}
}

func TestLowStockAlert_DoesNotFireAboveThreshold(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	med := medWithDose(t, db, baby.ID, user.ID, 5, "ml")

	// 100 mL / 5 = 20 doses, well above default 3.
	addContainer(t, db, baby.ID, med.ID, user.ID, 100, -time.Hour)

	alerts, _ := store.GetActiveAlerts(db, baby.ID)
	if findAlert(alerts, store.AlertTypeLowStock) != nil {
		t.Errorf("expected no low_stock alert with 20 doses; got %v", alerts)
	}
}

func TestLowStockAlert_SkipsMedsWithoutStructuredDose(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	// Legacy medication: no DoseAmount
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "Legacy", "5mL", "twice_daily", nil, nil, nil, nil)
	addContainer(t, db, baby.ID, med.ID, user.ID, 1, -time.Hour) // tiny, but no dose info

	alerts, _ := store.GetActiveAlerts(db, baby.ID)
	if findAlert(alerts, store.AlertTypeLowStock) != nil {
		t.Errorf("expected no low_stock alert without dose_amount; got %v", alerts)
	}
}

func TestNearExpiryAlert_FiresWhenManufacturerExpiryWithinDefault(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	med := medWithDose(t, db, baby.ID, user.ID, 5, "ml")

	// Default warning days = 3. Set expiry 2 days from now → within window.
	expiry := time.Now().UTC().Add(48 * time.Hour).Format(model.DateFormat)
	c, err := store.CreateMedicationContainer(db, store.CreateContainerParams{
		MedicationID:    med.ID,
		BabyID:          baby.ID,
		Kind:            "bottle",
		Unit:            "ml",
		QuantityInitial: 100,
		ExpirationDate:  &expiry,
		CreatedBy:       user.ID,
	})
	if err != nil {
		t.Fatalf("CreateMedicationContainer: %v", err)
	}

	alerts, _ := store.GetActiveAlerts(db, baby.ID)
	a := findAlert(alerts, store.AlertTypeNearExpiry)
	if a == nil {
		t.Fatalf("expected near_expiry alert, got %d alerts", len(alerts))
	}
	if a.EntryID == "" {
		t.Errorf("expected EntryID set, got empty")
	}
	_ = c
}

func TestNearExpiryAlert_FiresWhenOpenedPlusMaxDaysExpiresSoonest(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	med := medWithDose(t, db, baby.ID, user.ID, 5, "ml")

	// Manufacturer expiry far away, but opened 28 days ago with 30-day rule
	// → effective expiry in 2 days, within 3-day default warning.
	openedAt := time.Now().UTC().Add(-28 * 24 * time.Hour).Format(model.DateTimeFormat)
	maxDays := 30
	farExpiry := time.Now().UTC().Add(365 * 24 * time.Hour).Format(model.DateFormat)
	_, err := store.CreateMedicationContainer(db, store.CreateContainerParams{
		MedicationID:        med.ID,
		BabyID:              baby.ID,
		Kind:                "bottle",
		Unit:                "ml",
		QuantityInitial:     100,
		OpenedAt:            &openedAt,
		MaxDaysAfterOpening: &maxDays,
		ExpirationDate:      &farExpiry,
		CreatedBy:           user.ID,
	})
	if err != nil {
		t.Fatalf("CreateMedicationContainer: %v", err)
	}

	alerts, _ := store.GetActiveAlerts(db, baby.ID)
	if findAlert(alerts, store.AlertTypeNearExpiry) == nil {
		t.Errorf("expected near_expiry alert based on opened+max_days; got %v", alerts)
	}
}

func TestNearExpiryAlert_FiresWhenAlreadyExpired(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	med := medWithDose(t, db, baby.ID, user.ID, 5, "ml")

	// Yesterday's date.
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format(model.DateFormat)
	_, err := store.CreateMedicationContainer(db, store.CreateContainerParams{
		MedicationID:    med.ID,
		BabyID:          baby.ID,
		Kind:            "bottle",
		Unit:            "ml",
		QuantityInitial: 100,
		ExpirationDate:  &yesterday,
		CreatedBy:       user.ID,
	})
	if err != nil {
		t.Fatalf("CreateMedicationContainer: %v", err)
	}

	alerts, _ := store.GetActiveAlerts(db, baby.ID)
	if findAlert(alerts, store.AlertTypeNearExpiry) == nil {
		t.Errorf("expected near_expiry alert for already-expired container; got %v", alerts)
	}
}

func TestNearExpiryAlert_DoesNotFireWhenFar(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	med := medWithDose(t, db, baby.ID, user.ID, 5, "ml")

	farExpiry := time.Now().UTC().AddDate(0, 0, 30).Format(model.DateFormat)
	_, err := store.CreateMedicationContainer(db, store.CreateContainerParams{
		MedicationID:    med.ID,
		BabyID:          baby.ID,
		Kind:            "bottle",
		Unit:            "ml",
		QuantityInitial: 100,
		ExpirationDate:  &farExpiry,
		CreatedBy:       user.ID,
	})
	if err != nil {
		t.Fatalf("CreateMedicationContainer: %v", err)
	}

	alerts, _ := store.GetActiveAlerts(db, baby.ID)
	if findAlert(alerts, store.AlertTypeNearExpiry) != nil {
		t.Errorf("expected no near_expiry alert for 30-day-out container; got %v", alerts)
	}
}

func TestNearExpiryAlert_RespectsCustomWarningDays(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	med := medWithDose(t, db, baby.ID, user.ID, 5, "ml")
	custom := 14
	_, _ = store.SetMedicationStockFields(db, baby.ID, med.ID, user.ID, store.MedicationStockFields{
		ExpiryWarningDays: &custom,
	})

	// 10 days away — within 14-day window.
	expiry := time.Now().UTC().AddDate(0, 0, 10).Format(model.DateFormat)
	_, _ = store.CreateMedicationContainer(db, store.CreateContainerParams{
		MedicationID:    med.ID,
		BabyID:          baby.ID,
		Kind:            "bottle",
		Unit:            "ml",
		QuantityInitial: 100,
		ExpirationDate:  &expiry,
		CreatedBy:       user.ID,
	})

	alerts, _ := store.GetActiveAlerts(db, baby.ID)
	if findAlert(alerts, store.AlertTypeNearExpiry) == nil {
		t.Errorf("expected near_expiry alert with 14-day custom window; got %v", alerts)
	}
}

func TestNearExpiryAlert_SkipsDepletedContainer(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)
	med := medWithDose(t, db, baby.ID, user.ID, 5, "ml")

	expiry := time.Now().UTC().AddDate(0, 0, 1).Format(model.DateFormat)
	c, _ := store.CreateMedicationContainer(db, store.CreateContainerParams{
		MedicationID:    med.ID,
		BabyID:          baby.ID,
		Kind:            "bottle",
		Unit:            "ml",
		QuantityInitial: 100,
		ExpirationDate:  &expiry,
		CreatedBy:       user.ID,
	})
	// Mark depleted explicitly.
	_, _ = store.UpdateMedicationContainer(db, baby.ID, c.ID, store.UpdateContainerParams{
		UpdatedBy:         user.ID,
		Kind:              c.Kind,
		Unit:              c.Unit,
		QuantityInitial:   c.QuantityInitial,
		QuantityRemaining: 0,
		ExpirationDate:    &expiry,
		Depleted:          true,
	})

	alerts, _ := store.GetActiveAlerts(db, baby.ID)
	if findAlert(alerts, store.AlertTypeNearExpiry) != nil {
		t.Errorf("expected no near_expiry alert for depleted container; got %v", alerts)
	}
}
