package integration_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

// TestMedicationStockFlow is the end-to-end integration test for the medicine
// stock tracking feature. Mirrors the scenario described in the implementation
// plan and exercises every layer (handlers, store, alerts) through the public
// HTTP API. Each numbered step corresponds to step n. of the plan's TDD §10.
func TestMedicationStockFlow(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := setupIntegrationServer(t)
	defer cleanup()

	client := newTestClient(t, srv, db)
	babyID := createBabyViaAPI(t, client, "Stock Tracking Baby")

	// 1. Create medication with structured dose info.
	status, medResp := client.doJSON(http.MethodPost,
		"/api/babies/"+babyID+"/medications",
		map[string]any{
			"name":         "UDCA",
			"dose":         "5mL",
			"frequency":    "twice_daily",
			"dose_amount":  5,
			"dose_unit":    "ml",
			// Default thresholds (3 doses, 3 days) are intentionally left unset
			// so we exercise the default code path.
		},
	)
	if status != http.StatusCreated {
		t.Fatalf("create medication: status %d, resp %v", status, medResp)
	}
	medID := medResp["id"].(string)
	if medResp["dose_amount"].(float64) != 5 {
		t.Errorf("dose_amount round-trip = %v, want 5", medResp["dose_amount"])
	}

	// 2. Add two containers: A (older, 100 mL, no expiry), B (newer, 100 mL,
	// expires in 5 days — far enough to not fire near-expiry yet).
	openedA := time.Now().UTC().Add(-2 * time.Hour).Format(model.DateTimeFormat)
	openedB := time.Now().UTC().Add(-1 * time.Hour).Format(model.DateTimeFormat)
	expiryB := time.Now().UTC().AddDate(0, 0, 5).Format(model.DateFormat)

	statusA, aResp := client.doJSON(http.MethodPost,
		"/api/babies/"+babyID+"/medications/"+medID+"/containers",
		map[string]any{
			"kind":             "bottle",
			"unit":             "ml",
			"quantity_initial": 100,
			"opened_at":        openedA,
		},
	)
	if statusA != http.StatusCreated {
		t.Fatalf("create container A: status %d, resp %v", statusA, aResp)
	}
	bottleA := aResp["id"].(string)

	statusB, bResp := client.doJSON(http.MethodPost,
		"/api/babies/"+babyID+"/medications/"+medID+"/containers",
		map[string]any{
			"kind":             "bottle",
			"unit":             "ml",
			"quantity_initial": 100,
			"opened_at":        openedB,
			"expiration_date":  expiryB,
		},
	)
	if statusB != http.StatusCreated {
		t.Fatalf("create container B: status %d, resp %v", statusB, bResp)
	}
	bottleB := bResp["id"].(string)

	// 3. Log a dose with no container_id → FIFO picks A; A drops to 95.
	status, logResp := client.doJSON(http.MethodPost,
		"/api/babies/"+babyID+"/med-logs",
		map[string]any{
			"medication_id": medID,
			"skipped":       false,
		},
	)
	if status != http.StatusCreated {
		t.Fatalf("create med-log: status %d, resp %v", status, logResp)
	}
	if logResp["container_id"] != bottleA {
		t.Errorf("FIFO container_id = %v, want %s", logResp["container_id"], bottleA)
	}
	a := getContainer(t, client, babyID, medID, bottleA)
	if a.QuantityRemaining != 95 {
		t.Errorf("A.QuantityRemaining = %v, want 95", a.QuantityRemaining)
	}

	// 4. Log another dose with container_id=B → deducts from B; A unchanged.
	status, logResp2 := client.doJSON(http.MethodPost,
		"/api/babies/"+babyID+"/med-logs",
		map[string]any{
			"medication_id": medID,
			"skipped":       false,
			"container_id":  bottleB,
		},
	)
	if status != http.StatusCreated {
		t.Fatalf("create med-log B: status %d, resp %v", status, logResp2)
	}
	if logResp2["container_id"] != bottleB {
		t.Errorf("override container_id = %v, want %s", logResp2["container_id"], bottleB)
	}
	a = getContainer(t, client, babyID, medID, bottleA)
	b := getContainer(t, client, babyID, medID, bottleB)
	if a.QuantityRemaining != 95 {
		t.Errorf("A.QuantityRemaining = %v, want 95 (untouched)", a.QuantityRemaining)
	}
	if b.QuantityRemaining != 95 {
		t.Errorf("B.QuantityRemaining = %v, want 95", b.QuantityRemaining)
	}

	// 5. Manual adjust on A: -50, reason "spilled" → A=45 + audit row exists.
	status, adjResp := client.doJSON(http.MethodPost,
		"/api/babies/"+babyID+"/medications/"+medID+"/containers/"+bottleA+"/adjust",
		map[string]any{
			"delta":  -50,
			"reason": "spilled",
		},
	)
	if status != http.StatusCreated {
		t.Fatalf("adjust A: status %d, resp %v", status, adjResp)
	}
	a = getContainer(t, client, babyID, medID, bottleA)
	if a.QuantityRemaining != 45 {
		t.Errorf("A.QuantityRemaining after spill = %v, want 45", a.QuantityRemaining)
	}
	adjs, err := store.ListMedicationStockAdjustments(db, bottleA)
	if err != nil {
		t.Fatalf("ListMedicationStockAdjustments: %v", err)
	}
	if len(adjs) != 1 {
		t.Errorf("expected 1 adjustment row, got %d", len(adjs))
	}

	// 6. Loop-log doses until total remaining doses ≤ default threshold (3).
	// Currently A=45, B=95 → 28 doses total. Drain doses until ≤ 3 (i.e. ≤ 15 mL).
	for i := 0; i < 26; i++ {
		status, _ := client.doJSON(http.MethodPost,
			"/api/babies/"+babyID+"/med-logs",
			map[string]any{
				"medication_id": medID,
				"skipped":       false,
			},
		)
		if status != http.StatusCreated {
			t.Fatalf("log dose iter %d: status %d", i, status)
		}
	}
	alerts, err := store.GetActiveAlerts(db, babyID)
	if err != nil {
		t.Fatalf("GetActiveAlerts: %v", err)
	}
	if findStockAlert(alerts, store.AlertTypeLowStock) == nil {
		t.Errorf("expected low_stock alert after draining; got %v", alertTypes(alerts))
	}

	// 7. Near-expiry alert. Update B's expiration to 1 day out — within
	// the default 3-day window — and confirm the alert fires.
	soon := time.Now().UTC().AddDate(0, 0, 1).Format(model.DateFormat)
	status, _ = client.doJSON(http.MethodPut,
		"/api/babies/"+babyID+"/medications/"+medID+"/containers/"+bottleB,
		map[string]any{
			"kind":             "bottle",
			"unit":             "ml",
			"quantity_initial": 100,
			"expiration_date":  soon,
		},
	)
	if status != http.StatusOK {
		t.Fatalf("update B expiration: status %d", status)
	}
	alerts, _ = store.GetActiveAlerts(db, babyID)
	if findStockAlert(alerts, store.AlertTypeNearExpiry) == nil {
		t.Errorf("expected near_expiry alert; got %v", alertTypes(alerts))
	}

	// 8. Add a fresh container with plenty of stock → low_stock alert clears.
	farExpiry := time.Now().UTC().AddDate(1, 0, 0).Format(model.DateFormat)
	status, freshResp := client.doJSON(http.MethodPost,
		"/api/babies/"+babyID+"/medications/"+medID+"/containers",
		map[string]any{
			"kind":             "bottle",
			"unit":             "ml",
			"quantity_initial": 500,
			"expiration_date":  farExpiry,
		},
	)
	if status != http.StatusCreated {
		t.Fatalf("create fresh container: status %d, resp %v", status, freshResp)
	}
	alerts, _ = store.GetActiveAlerts(db, babyID)
	if findStockAlert(alerts, store.AlertTypeLowStock) != nil {
		t.Errorf("expected low_stock alert to clear after restock; got %v", alertTypes(alerts))
	}

	// 9. Backfill case: a legacy medication with no dose_amount/dose_unit set
	// must still accept dose-logging without error and without producing
	// container/stock side-effects.
	status, legacyResp := client.doJSON(http.MethodPost,
		"/api/babies/"+babyID+"/medications",
		map[string]any{
			"name":      "Legacy",
			"dose":      "10mg",
			"frequency": "once_daily",
		},
	)
	if status != http.StatusCreated {
		t.Fatalf("create legacy medication: status %d, resp %v", status, legacyResp)
	}
	legacyID := legacyResp["id"].(string)

	status, legacyLog := client.doJSON(http.MethodPost,
		"/api/babies/"+babyID+"/med-logs",
		map[string]any{
			"medication_id": legacyID,
			"skipped":       false,
		},
	)
	if status != http.StatusCreated {
		t.Fatalf("legacy med-log: status %d, resp %v", status, legacyLog)
	}
	if legacyLog["container_id"] != nil {
		t.Errorf("legacy log should have nil container_id, got %v", legacyLog["container_id"])
	}
	if legacyLog["stock_deducted"] != nil {
		t.Errorf("legacy log should have nil stock_deducted, got %v", legacyLog["stock_deducted"])
	}
}

// containerSnapshot captures just the fields the test cares about.
type containerSnapshot struct {
	QuantityRemaining float64
}

func getContainer(t *testing.T, client *testClient, babyID, medID, containerID string) containerSnapshot {
	t.Helper()
	status, resp := client.doJSON(http.MethodGet,
		"/api/babies/"+babyID+"/medications/"+medID+"/containers/"+containerID, nil)
	if status != http.StatusOK {
		t.Fatalf("get container %s: status %d, resp %v", containerID, status, resp)
	}
	q, _ := resp["quantity_remaining"].(float64)
	return containerSnapshot{QuantityRemaining: q}
}

func findStockAlert(alerts []store.Alert, alertType string) *store.Alert {
	for i := range alerts {
		if alerts[i].AlertType == alertType {
			return &alerts[i]
		}
	}
	return nil
}

func alertTypes(alerts []store.Alert) string {
	types := make([]string, 0, len(alerts))
	for _, a := range alerts {
		types = append(types, a.AlertType)
	}
	return strings.Join(types, ",")
}
