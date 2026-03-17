package integration_test

import (
	"database/sql"
	"fmt"
	"net/http"
	"testing"

	"github.com/ablankz/LittleLiver/backend/internal/handler"
)

// TestAccountDeletionAnonymization exercises the full account deletion anonymization flow:
// User A creates baby -> User B joins -> both log entries across feedings, stools,
// temperatures, weights -> User A deletes account -> verify all logged_by/updated_by
// fields are anonymized to "deleted_user" across all metric tables while entries remain intact.
func TestAccountDeletionAnonymization(t *testing.T) {
	t.Parallel()

	// Ensure AnonymizeTables is populated for this test.
	// In production this is set at startup; we verify it includes metric tables.
	requiredTables := []string{
		"feedings", "stools", "urine", "weights", "temperatures",
		"abdomen_observations", "skin_observations", "bruising",
		"lab_results", "general_notes",
	}
	for _, table := range requiredTables {
		found := false
		for _, at := range handler.AnonymizeTables {
			if at == table {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected AnonymizeTables to contain %q but it does not; got %v", table, handler.AnonymizeTables)
		}
	}

	srv, db, cleanup := setupMultiParentServer(t)
	defer cleanup()

	clientA := newTestClient(t, srv, db)
	clientB := newTestClient(t, srv, db)

	// --- Step 1: User A creates a baby ---
	status, babyResp := clientA.doJSON(http.MethodPost, "/api/babies", map[string]any{
		"name":          "Anonymization Baby",
		"sex":           "female",
		"date_of_birth": "2025-06-01",
	})
	if status != http.StatusCreated {
		t.Fatalf("expected 201 creating baby, got %d: %v", status, babyResp)
	}
	babyID := babyResp["id"].(string)

	// --- Step 2: User A generates invite, User B joins ---
	status, inviteResp := clientA.doJSON(http.MethodPost, fmt.Sprintf("/api/babies/%s/invite", babyID), nil)
	if status != http.StatusCreated {
		t.Fatalf("expected 201 creating invite, got %d: %v", status, inviteResp)
	}
	inviteCode := inviteResp["code"].(string)

	status, _ = clientB.doJSON(http.MethodPost, "/api/babies/join", map[string]any{
		"code": inviteCode,
	})
	if status != http.StatusOK {
		t.Fatalf("expected 200 joining baby, got %d", status)
	}

	// --- Step 3: User A logs entries across multiple metric types ---
	feedingPath := fmt.Sprintf("/api/babies/%s/feedings", babyID)
	stoolPath := fmt.Sprintf("/api/babies/%s/stools", babyID)
	tempPath := fmt.Sprintf("/api/babies/%s/temperatures", babyID)
	weightPath := fmt.Sprintf("/api/babies/%s/weights", babyID)

	// User A logs a feeding
	status, feedA := clientA.doJSON(http.MethodPost, feedingPath, map[string]any{
		"timestamp":    "2025-06-15T10:00:00Z",
		"feed_type":    "breast_milk",
		"duration_min": 15,
	})
	if status != http.StatusCreated {
		t.Fatalf("expected 201 creating feeding (A), got %d: %v", status, feedA)
	}

	// User A logs a stool
	status, stoolA := clientA.doJSON(http.MethodPost, stoolPath, map[string]any{
		"timestamp":    "2025-06-15T11:00:00Z",
		"color_rating": 3,
		"consistency":  "soft",
	})
	if status != http.StatusCreated {
		t.Fatalf("expected 201 creating stool (A), got %d: %v", status, stoolA)
	}

	// User A logs a temperature
	status, tempA := clientA.doJSON(http.MethodPost, tempPath, map[string]any{
		"timestamp": "2025-06-15T12:00:00Z",
		"value":     37.2,
		"method":    "axillary",
	})
	if status != http.StatusCreated {
		t.Fatalf("expected 201 creating temperature (A), got %d: %v", status, tempA)
	}

	// User A logs a weight
	status, weightA := clientA.doJSON(http.MethodPost, weightPath, map[string]any{
		"timestamp": "2025-06-15T08:00:00Z",
		"weight_kg": 4.5,
	})
	if status != http.StatusCreated {
		t.Fatalf("expected 201 creating weight (A), got %d: %v", status, weightA)
	}

	// --- Step 4: User B also logs entries ---
	// User B logs a feeding
	status, feedB := clientB.doJSON(http.MethodPost, feedingPath, map[string]any{
		"timestamp":   "2025-06-15T14:00:00Z",
		"feed_type":   "formula",
		"volume_ml":   120.0,
		"cal_density": 0.67,
	})
	if status != http.StatusCreated {
		t.Fatalf("expected 201 creating feeding (B), got %d: %v", status, feedB)
	}

	// User B logs a stool
	status, stoolB := clientB.doJSON(http.MethodPost, stoolPath, map[string]any{
		"timestamp":    "2025-06-15T15:00:00Z",
		"color_rating": 5,
		"consistency":  "loose",
	})
	if status != http.StatusCreated {
		t.Fatalf("expected 201 creating stool (B), got %d: %v", status, stoolB)
	}

	// User B logs a temperature
	status, tempB := clientB.doJSON(http.MethodPost, tempPath, map[string]any{
		"timestamp": "2025-06-15T16:00:00Z",
		"value":     36.8,
		"method":    "rectal",
	})
	if status != http.StatusCreated {
		t.Fatalf("expected 201 creating temperature (B), got %d: %v", status, tempB)
	}

	// User B logs a weight
	status, weightB := clientB.doJSON(http.MethodPost, weightPath, map[string]any{
		"timestamp": "2025-06-16T08:00:00Z",
		"weight_kg": 4.6,
	})
	if status != http.StatusCreated {
		t.Fatalf("expected 201 creating weight (B), got %d: %v", status, weightB)
	}

	// User A also updates one of B's feedings (so updated_by = A on B's entry)
	feedBID := feedB["id"].(string)
	status, _ = clientA.doJSON(http.MethodPut, fmt.Sprintf("%s/%s", feedingPath, feedBID), map[string]any{
		"timestamp":   "2025-06-15T14:00:00Z",
		"feed_type":   "formula",
		"volume_ml":   130.0,
		"cal_density": 0.67,
	})
	if status != http.StatusOK {
		t.Fatalf("expected 200 updating feeding (A updates B's), got %d", status)
	}

	userAID := clientA.userID

	// --- Step 5: User A deletes their account ---
	status, _ = clientA.doJSON(http.MethodDelete, "/api/users/me", nil)
	if status != http.StatusNoContent {
		t.Fatalf("expected 204 deleting account, got %d", status)
	}

	// --- Step 6: Verify User A is deleted ---
	var userCount int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE id = ?", userAID).Scan(&userCount)
	if err != nil {
		t.Fatalf("query user count: %v", err)
	}
	if userCount != 0 {
		t.Fatalf("expected user A to be deleted, but found %d records", userCount)
	}

	// --- Step 7: Verify entries remain intact (not deleted) ---
	verifyEntryCount(t, db, "feedings", babyID, 2)
	verifyEntryCount(t, db, "stools", babyID, 2)
	verifyEntryCount(t, db, "temperatures", babyID, 2)
	verifyEntryCount(t, db, "weights", babyID, 2)

	// --- Step 8: Verify logged_by anonymization for User A's entries ---
	// User A's feedings should have logged_by = "deleted_user"
	verifyAnonymized(t, db, "feedings", "logged_by", userAID, "deleted_user")
	verifyAnonymized(t, db, "stools", "logged_by", userAID, "deleted_user")
	verifyAnonymized(t, db, "temperatures", "logged_by", userAID, "deleted_user")
	verifyAnonymized(t, db, "weights", "logged_by", userAID, "deleted_user")

	// --- Step 9: Verify updated_by anonymization ---
	// User A updated B's feeding, so that entry's updated_by should be "deleted_user"
	var updatedBy sql.NullString
	err = db.QueryRow("SELECT updated_by FROM feedings WHERE id = ?", feedBID).Scan(&updatedBy)
	if err != nil {
		t.Fatalf("query updated_by for feeding %s: %v", feedBID, err)
	}
	if !updatedBy.Valid || updatedBy.String != "deleted_user" {
		t.Errorf("expected updated_by='deleted_user' for B's feeding updated by A, got %v", updatedBy)
	}

	// --- Step 10: Verify User B's entries are NOT anonymized ---
	userBID := clientB.userID
	verifyNotAnonymized(t, db, "feedings", "logged_by", userBID)
	verifyNotAnonymized(t, db, "stools", "logged_by", userBID)
	verifyNotAnonymized(t, db, "temperatures", "logged_by", userBID)
	verifyNotAnonymized(t, db, "weights", "logged_by", userBID)

	// --- Step 11: User B can still read all entries via API ---
	status, feedings := clientB.doJSONList(feedingPath)
	if status != http.StatusOK {
		t.Fatalf("expected 200 listing feedings after A's deletion, got %d", status)
	}
	if len(feedings) != 2 {
		t.Fatalf("expected 2 feedings after A's deletion, got %d", len(feedings))
	}

	status, stools := clientB.doJSONList(stoolPath)
	if status != http.StatusOK {
		t.Fatalf("expected 200 listing stools after A's deletion, got %d", status)
	}
	if len(stools) != 2 {
		t.Fatalf("expected 2 stools after A's deletion, got %d", len(stools))
	}

	status, temps := clientB.doJSONList(tempPath)
	if status != http.StatusOK {
		t.Fatalf("expected 200 listing temperatures after A's deletion, got %d", status)
	}
	if len(temps) != 2 {
		t.Fatalf("expected 2 temperatures after A's deletion, got %d", len(temps))
	}

	status, weights := clientB.doJSONList(weightPath)
	if status != http.StatusOK {
		t.Fatalf("expected 200 listing weights after A's deletion, got %d", status)
	}
	if len(weights) != 2 {
		t.Fatalf("expected 2 weights after A's deletion, got %d", len(weights))
	}

	// --- Step 12: Verify the baby still exists ---
	var babyCount int
	err = db.QueryRow("SELECT COUNT(*) FROM babies WHERE id = ?", babyID).Scan(&babyCount)
	if err != nil {
		t.Fatalf("query baby count: %v", err)
	}
	if babyCount != 1 {
		t.Fatalf("expected baby to still exist (B is still linked), got count=%d", babyCount)
	}
}

// verifyEntryCount checks that the given table has the expected number of rows for a baby.
func verifyEntryCount(t *testing.T, db *sql.DB, table, babyID string, expected int) {
	t.Helper()
	var count int
	err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE baby_id = ?", table), babyID).Scan(&count)
	if err != nil {
		t.Fatalf("query %s count: %v", table, err)
	}
	if count != expected {
		t.Errorf("expected %d entries in %s, got %d", expected, table, count)
	}
}

// verifyAnonymized checks that no rows in the table still have the original user ID
// in the specified column, and at least one row has "deleted_user".
func verifyAnonymized(t *testing.T, db *sql.DB, table, column, originalUserID, sentinel string) {
	t.Helper()

	// No rows should have the original user ID
	var remaining int
	err := db.QueryRow(
		fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = ?", table, column),
		originalUserID,
	).Scan(&remaining)
	if err != nil {
		t.Fatalf("query %s.%s remaining: %v", table, column, err)
	}
	if remaining != 0 {
		t.Errorf("expected 0 rows with %s.%s=%q after anonymization, got %d", table, column, originalUserID, remaining)
	}

	// At least one row should have the sentinel value
	var anonymized int
	err = db.QueryRow(
		fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = ?", table, column),
		sentinel,
	).Scan(&anonymized)
	if err != nil {
		t.Fatalf("query %s.%s anonymized: %v", table, column, err)
	}
	if anonymized == 0 {
		t.Errorf("expected at least 1 row with %s.%s=%q after anonymization, got 0", table, column, sentinel)
	}
}

// verifyNotAnonymized checks that the specified user's entries still have the
// original user ID in the specified column (not anonymized).
func verifyNotAnonymized(t *testing.T, db *sql.DB, table, column, userID string) {
	t.Helper()
	var count int
	err := db.QueryRow(
		fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = ?", table, column),
		userID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("query %s.%s for user %s: %v", table, column, userID, err)
	}
	if count == 0 {
		t.Errorf("expected at least 1 row with %s.%s=%q (should not be anonymized), got 0", table, column, userID)
	}
}
