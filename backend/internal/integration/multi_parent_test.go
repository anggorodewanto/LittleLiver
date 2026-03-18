package integration_test

import (
	"fmt"
	"net/http"
	"testing"
)

// TestMultiParentBabyLifecycle exercises the full multi-parent lifecycle:
// User A creates baby -> generates invite -> User B joins -> both log entries ->
// verify both can read all entries -> User B unlinks -> verify B loses access ->
// User A unlinks (last parent) -> verify baby and all data deleted.
func TestMultiParentBabyLifecycle(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := setupIntegrationServer(t)
	defer cleanup()

	clientA := newTestClient(t, srv, db)
	clientB := newTestClient(t, srv, db)

	// --- Step 1: User A creates a baby ---
	status, babyResp := clientA.doJSON(http.MethodPost, "/api/babies", map[string]any{
		"name":          "Integration Baby",
		"sex":           "female",
		"date_of_birth": "2025-06-01",
	})
	if status != http.StatusCreated {
		t.Fatalf("expected 201 creating baby, got %d: %v", status, babyResp)
	}
	babyID, ok := babyResp["id"].(string)
	if !ok || babyID == "" {
		t.Fatalf("expected non-empty baby ID, got %v", babyResp["id"])
	}

	// --- Step 2: User A generates an invite code ---
	status, inviteResp := clientA.doJSON(http.MethodPost, fmt.Sprintf("/api/babies/%s/invite", babyID), nil)
	if status != http.StatusCreated {
		t.Fatalf("expected 201 creating invite, got %d: %v", status, inviteResp)
	}
	inviteCode, ok := inviteResp["code"].(string)
	if !ok || inviteCode == "" {
		t.Fatalf("expected non-empty invite code, got %v", inviteResp["code"])
	}

	// --- Step 3: User B joins with the invite code ---
	status, joinResp := clientB.doJSON(http.MethodPost, "/api/babies/join", map[string]any{
		"code": inviteCode,
	})
	if status != http.StatusOK {
		t.Fatalf("expected 200 joining baby, got %d: %v", status, joinResp)
	}
	if joinResp["baby_id"] != babyID {
		t.Fatalf("expected baby_id=%s in join response, got %v", babyID, joinResp["baby_id"])
	}

	// Verify User B can now see the baby in their list
	status, meResp := clientB.doJSON(http.MethodGet, "/api/me", nil)
	if status != http.StatusOK {
		t.Fatalf("expected 200 from /api/me for User B, got %d", status)
	}
	babies, ok := meResp["babies"].([]any)
	if !ok || len(babies) == 0 {
		t.Fatalf("expected User B to see at least 1 baby, got %v", meResp["babies"])
	}

	// --- Step 4: Both users log feedings ---
	feedingPath := fmt.Sprintf("/api/babies/%s/feedings", babyID)

	// User A logs a breast_milk (no volume) feeding (breast_milk with no volume)
	status, feedA := clientA.doJSON(http.MethodPost, feedingPath, map[string]any{
		"timestamp":    "2025-06-15T10:00:00Z",
		"feed_type":    "breast_milk",
		"duration_min": 15,
	})
	if status != http.StatusCreated {
		t.Fatalf("expected 201 creating feeding (A), got %d: %v", status, feedA)
	}
	feedAID := feedA["id"].(string)

	// User B logs a formula feeding
	status, feedB := clientB.doJSON(http.MethodPost, feedingPath, map[string]any{
		"timestamp":   "2025-06-15T14:00:00Z",
		"feed_type":   "formula",
		"volume_ml":   120.0,
		"cal_density": 0.67,
	})
	if status != http.StatusCreated {
		t.Fatalf("expected 201 creating feeding (B), got %d: %v", status, feedB)
	}
	feedBID := feedB["id"].(string)

	// --- Step 5: Both users log stools ---
	stoolPath := fmt.Sprintf("/api/babies/%s/stools", babyID)

	status, stoolA := clientA.doJSON(http.MethodPost, stoolPath, map[string]any{
		"timestamp":    "2025-06-15T11:00:00Z",
		"color_rating": 3,
		"consistency":  "soft",
	})
	if status != http.StatusCreated {
		t.Fatalf("expected 201 creating stool (A), got %d: %v", status, stoolA)
	}

	status, stoolB := clientB.doJSON(http.MethodPost, stoolPath, map[string]any{
		"timestamp":    "2025-06-15T15:00:00Z",
		"color_rating": 5,
		"consistency":  "loose",
	})
	if status != http.StatusCreated {
		t.Fatalf("expected 201 creating stool (B), got %d: %v", status, stoolB)
	}

	// --- Step 6: Verify both users can read ALL entries ---
	// User A reads feedings
	status, feedingsA := clientA.doJSONList(feedingPath)
	if status != http.StatusOK {
		t.Fatalf("expected 200 listing feedings (A), got %d", status)
	}
	if len(feedingsA) != 2 {
		t.Fatalf("expected 2 feedings visible to A, got %d", len(feedingsA))
	}

	// User B reads feedings
	status, feedingsB := clientB.doJSONList(feedingPath)
	if status != http.StatusOK {
		t.Fatalf("expected 200 listing feedings (B), got %d", status)
	}
	if len(feedingsB) != 2 {
		t.Fatalf("expected 2 feedings visible to B, got %d", len(feedingsB))
	}

	// User A reads stools
	status, stoolsA := clientA.doJSONList(stoolPath)
	if status != http.StatusOK {
		t.Fatalf("expected 200 listing stools (A), got %d", status)
	}
	if len(stoolsA) != 2 {
		t.Fatalf("expected 2 stools visible to A, got %d", len(stoolsA))
	}

	// User B reads stools
	status, stoolsB := clientB.doJSONList(stoolPath)
	if status != http.StatusOK {
		t.Fatalf("expected 200 listing stools (B), got %d", status)
	}
	if len(stoolsB) != 2 {
		t.Fatalf("expected 2 stools visible to B, got %d", len(stoolsB))
	}

	// Cross-user read: User B can read User A's feeding by ID
	status, _ = clientB.doJSON(http.MethodGet, fmt.Sprintf("%s/%s", feedingPath, feedAID), nil)
	if status != http.StatusOK {
		t.Fatalf("expected 200 reading A's feeding from B, got %d", status)
	}

	// Cross-user read: User A can read User B's feeding by ID
	status, _ = clientA.doJSON(http.MethodGet, fmt.Sprintf("%s/%s", feedingPath, feedBID), nil)
	if status != http.StatusOK {
		t.Fatalf("expected 200 reading B's feeding from A, got %d", status)
	}

	// --- Step 7: User B unlinks ---
	unlinkPath := fmt.Sprintf("/api/babies/%s/parents/me", babyID)
	status = clientB.doRaw(http.MethodDelete, unlinkPath)
	if status != http.StatusNoContent {
		t.Fatalf("expected 204 unlinking B, got %d", status)
	}

	// --- Step 8: Verify User B loses access ---
	status = clientB.doRaw(http.MethodGet, fmt.Sprintf("/api/babies/%s", babyID))
	if status != http.StatusForbidden {
		t.Fatalf("expected 403 for B accessing baby after unlink, got %d", status)
	}

	status, _ = clientB.doJSONList(feedingPath)
	if status != http.StatusForbidden {
		t.Fatalf("expected 403 for B listing feedings after unlink, got %d", status)
	}

	status, _ = clientB.doJSONList(stoolPath)
	if status != http.StatusForbidden {
		t.Fatalf("expected 403 for B listing stools after unlink, got %d", status)
	}

	// User A still has access
	status, feedingsA = clientA.doJSONList(feedingPath)
	if status != http.StatusOK {
		t.Fatalf("expected 200 for A listing feedings after B unlinks, got %d", status)
	}
	if len(feedingsA) != 2 {
		t.Fatalf("expected 2 feedings still visible to A, got %d", len(feedingsA))
	}

	// --- Step 9: User A unlinks (last parent) -> baby and data deleted ---
	status = clientA.doRaw(http.MethodDelete, unlinkPath)
	if status != http.StatusNoContent {
		t.Fatalf("expected 204 unlinking A (last parent), got %d", status)
	}

	// Baby should be gone
	status = clientA.doRaw(http.MethodGet, fmt.Sprintf("/api/babies/%s", babyID))
	if status != http.StatusForbidden && status != http.StatusNotFound {
		t.Fatalf("expected 403 or 404 for deleted baby, got %d", status)
	}

	// Verify data is actually deleted from the DB
	var feedCount int
	err := db.QueryRow("SELECT COUNT(*) FROM feedings WHERE baby_id = ?", babyID).Scan(&feedCount)
	if err != nil {
		t.Fatalf("query feedings count: %v", err)
	}
	if feedCount != 0 {
		t.Fatalf("expected 0 feedings after last parent unlinks, got %d", feedCount)
	}

	var stoolCount int
	err = db.QueryRow("SELECT COUNT(*) FROM stools WHERE baby_id = ?", babyID).Scan(&stoolCount)
	if err != nil {
		t.Fatalf("query stools count: %v", err)
	}
	if stoolCount != 0 {
		t.Fatalf("expected 0 stools after last parent unlinks, got %d", stoolCount)
	}

	var babyCount int
	err = db.QueryRow("SELECT COUNT(*) FROM babies WHERE id = ?", babyID).Scan(&babyCount)
	if err != nil {
		t.Fatalf("query baby count: %v", err)
	}
	if babyCount != 0 {
		t.Fatalf("expected baby to be deleted after last parent unlinks, got count=%d", babyCount)
	}
}

// TestRecalculateCaloriesFlow tests the calorie recalculation flow:
// Create baby -> log breast_milk (no volume) feedings (uses default_cal_per_feed) ->
// update baby with new default_cal_per_feed and recalculate_calories=true ->
// verify all feeding calories updated.
func TestRecalculateCaloriesFlow(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := setupIntegrationServer(t)
	defer cleanup()

	client := newTestClient(t, srv, db)

	// --- Step 1: Create baby with default_cal_per_feed ---
	status, babyResp := client.doJSON(http.MethodPost, "/api/babies", map[string]any{
		"name":                 "Calorie Baby",
		"sex":                  "male",
		"date_of_birth":        "2025-05-01",
		"default_cal_per_feed": 50.0,
	})
	if status != http.StatusCreated {
		t.Fatalf("expected 201 creating baby, got %d: %v", status, babyResp)
	}
	babyID := babyResp["id"].(string)

	feedingPath := fmt.Sprintf("/api/babies/%s/feedings", babyID)

	// --- Step 2: Log breast_milk (no volume) feedings (should use default_cal_per_feed) ---
	for i := 0; i < 3; i++ {
		ts := fmt.Sprintf("2025-06-15T%02d:00:00Z", 8+i*2)
		status, resp := client.doJSON(http.MethodPost, feedingPath, map[string]any{
			"timestamp":    ts,
			"feed_type":    "breast_milk",
			"duration_min": 15,
		})
		if status != http.StatusCreated {
			t.Fatalf("expected 201 creating feeding %d, got %d: %v", i, status, resp)
		}
		// Verify it used the default cal
		if resp["used_default_cal"] != true {
			t.Fatalf("expected used_default_cal=true for breast_milk (no volume) feeding %d, got %v", i, resp["used_default_cal"])
		}
		cal, ok := resp["calories"].(float64)
		if !ok || cal != 50.0 {
			t.Fatalf("expected calories=50.0 for feeding %d, got %v", i, resp["calories"])
		}
	}

	// Also log a formula feeding with explicit cal_density (should NOT use default)
	status, formulaResp := client.doJSON(http.MethodPost, feedingPath, map[string]any{
		"timestamp":   "2025-06-15T16:00:00Z",
		"feed_type":   "formula",
		"volume_ml":   120.0,
		"cal_density": 0.67,
	})
	if status != http.StatusCreated {
		t.Fatalf("expected 201 creating formula feeding, got %d: %v", status, formulaResp)
	}
	if formulaResp["used_default_cal"] != false {
		t.Fatalf("expected used_default_cal=false for formula feeding, got %v", formulaResp["used_default_cal"])
	}
	formulaCalories := formulaResp["calories"].(float64)

	// --- Step 3: Update baby with new default_cal_per_feed + recalculate ---
	newCalPerFeed := 75.0
	status, updateResp := client.doJSON(http.MethodPut,
		fmt.Sprintf("/api/babies/%s?recalculate_calories=true", babyID),
		map[string]any{
			"name":                 "Calorie Baby",
			"sex":                  "male",
			"date_of_birth":        "2025-05-01",
			"default_cal_per_feed": newCalPerFeed,
		},
	)
	if status != http.StatusOK {
		t.Fatalf("expected 200 updating baby with recalculate, got %d: %v", status, updateResp)
	}

	// Verify recalculated_count
	recalcCount, ok := updateResp["recalculated_count"].(float64)
	if !ok {
		t.Fatalf("expected recalculated_count in response, got %v", updateResp)
	}
	if int(recalcCount) != 3 {
		t.Fatalf("expected 3 recalculated feedings, got %d", int(recalcCount))
	}

	// Verify baby's new default_cal_per_feed
	babyData, ok := updateResp["baby"].(map[string]any)
	if !ok {
		t.Fatalf("expected baby in response, got %v", updateResp)
	}
	if babyData["default_cal_per_feed"].(float64) != newCalPerFeed {
		t.Fatalf("expected default_cal_per_feed=%f, got %v", newCalPerFeed, babyData["default_cal_per_feed"])
	}

	// --- Step 4: Verify all breast_milk (no volume) feedings now have new calories ---
	status, feedings := client.doJSONList(feedingPath)
	if status != http.StatusOK {
		t.Fatalf("expected 200 listing feedings, got %d", status)
	}
	if len(feedings) != 4 {
		t.Fatalf("expected 4 feedings, got %d", len(feedings))
	}

	for _, item := range feedings {
		f := item.(map[string]any)
		feedType := f["feed_type"].(string)
		cal := f["calories"].(float64)
		usedDefault := f["used_default_cal"].(bool)

		if feedType == "breast_milk" {
			if !usedDefault {
				t.Errorf("expected used_default_cal=true for breast_milk (no volume), got false")
			}
			if cal != newCalPerFeed {
				t.Errorf("expected calories=%f for breast_milk (no volume) after recalculation, got %f", newCalPerFeed, cal)
			}
		} else if feedType == "formula" {
			if usedDefault {
				t.Errorf("expected used_default_cal=false for formula, got true")
			}
			if cal != formulaCalories {
				t.Errorf("formula calories should be unchanged: expected %f, got %f", formulaCalories, cal)
			}
		}
	}

	// --- Step 5: Update WITHOUT recalculate_calories -> no recalculated_count in response ---
	status, noRecalcResp := client.doJSON(http.MethodPut,
		fmt.Sprintf("/api/babies/%s", babyID),
		map[string]any{
			"name":                 "Calorie Baby Updated",
			"sex":                  "male",
			"date_of_birth":        "2025-05-01",
			"default_cal_per_feed": 100.0,
		},
	)
	if status != http.StatusOK {
		t.Fatalf("expected 200 updating baby without recalculate, got %d: %v", status, noRecalcResp)
	}
	// Should NOT have recalculated_count key (plain baby response, not envelope)
	if _, exists := noRecalcResp["recalculated_count"]; exists {
		t.Fatalf("expected no recalculated_count when recalculate_calories is not set, got %v", noRecalcResp)
	}

	// Feedings should still have old calories (75.0), not the new 100.0
	status, feedings = client.doJSONList(feedingPath)
	if status != http.StatusOK {
		t.Fatalf("expected 200 listing feedings, got %d", status)
	}
	for _, item := range feedings {
		f := item.(map[string]any)
		if f["feed_type"].(string) == "breast_milk" {
			cal := f["calories"].(float64)
			if cal != newCalPerFeed {
				t.Errorf("expected breast_milk (no volume) calories unchanged at %f without recalculate, got %f", newCalPerFeed, cal)
			}
		}
	}
}
