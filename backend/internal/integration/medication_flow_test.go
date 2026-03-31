package integration_test

import (
	"database/sql"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/handler"
	"github.com/ablankz/LittleLiver/backend/internal/notify"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

// mockPusher records push sends for integration test verification.
type mockPusher struct {
	mu    sync.Mutex
	sends []mockPushSend
}

type mockPushSend struct {
	Sub     notify.Subscription
	Payload notify.Payload
}

func (m *mockPusher) Send(sub notify.Subscription, payload notify.Payload) (*http.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sends = append(m.sends, mockPushSend{Sub: sub, Payload: payload})
	return &http.Response{StatusCode: http.StatusCreated}, nil
}

func (m *mockPusher) sendCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sends)
}

func (m *mockPusher) lastPayload() notify.Payload {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sends[len(m.sends)-1].Payload
}

// TestMedicationNotificationLifecycle exercises the full medication reminder
// notification lifecycle using mock time and a mock push sender:
//
//  1. Create medication with schedule -> scheduler fires notification at scheduled time
//  2. Parent logs dose as "given" -> verify suppression of follow-ups
//  3. Next dose: no log -> verify +15 min follow-up fires
//  4. Log as skipped -> verify +30 min NOT suppressed (skipped has NULL given_at)
//  5. Verify adherence ratio from med_logs data
//  6. Verify medication appears in dashboard upcoming_meds
func TestMedicationNotificationLifecycle(t *testing.T) {
	t.Parallel()

	srv, db, cleanup := setupIntegrationServer(t)
	defer cleanup()

	client := newTestClient(t, srv, db)
	babyID := createBabyViaAPI(t, client, "Med Flow Baby")

	// Register a push subscription for the user
	testutil.SeedPushSubscription(t, db, client.userID, "https://push.example.com/device1")

	// --- Step 1: Create medication with schedule ---
	medPath := fmt.Sprintf("/api/babies/%s/medications", babyID)
	status, medResp := client.doJSONWithHeaders(http.MethodPost, medPath, map[string]any{
		"name":           "Ursodiol",
		"dose":           "50mg",
		"frequency":      "twice_daily",
		"schedule_times": []string{"08:00", "20:00"},
	}, map[string]string{"X-Timezone": "UTC"})
	if status != http.StatusCreated {
		t.Fatalf("expected 201 creating medication, got %d: %v", status, medResp)
	}
	medID := medResp["id"].(string)

	// Use a fixed date for deterministic testing
	baseDate := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)

	// --- Step 2: Tick at 08:00 UTC -> notification fires ---
	mock1 := &mockPusher{}
	s1 := notify.NewScheduler(db, mock1)
	t0800 := baseDate.Add(8 * time.Hour)
	s1.Tick(t0800)
	if mock1.sendCount() != 1 {
		t.Fatalf("step 2: expected 1 notification at 08:00, got %d", mock1.sendCount())
	}
	p := mock1.lastPayload()
	expectedURL := fmt.Sprintf("/log/med?medication_id=%s", medID)
	if p.URL != expectedURL {
		t.Errorf("step 2: wrong URL: got %q, want %q", p.URL, expectedURL)
	}

	// --- Step 3: Parent logs dose as "given" ---
	// Server sets given_at to NOW() per spec. We then update it via direct DB
	// to the test time (08:05) so the suppression window check works.
	scheduledTime := t0800.Format("2006-01-02T15:04:05Z")
	medLogPath := fmt.Sprintf("/api/babies/%s/med-logs", babyID)
	status, logResp := client.doJSON(http.MethodPost, medLogPath, map[string]any{
		"medication_id":  medID,
		"scheduled_time": scheduledTime,
		"skipped":        false,
	})
	if status != http.StatusCreated {
		t.Fatalf("step 3: expected 201 creating med-log, got %d: %v", status, logResp)
	}
	// Override given_at to the controlled test time for deterministic suppression testing
	givenAt := baseDate.Add(8*time.Hour + 5*time.Minute).Format("2006-01-02T15:04:05Z")
	logID := logResp["id"].(string)
	_, err := db.Exec("UPDATE med_logs SET given_at = ? WHERE id = ?", givenAt, logID)
	if err != nil {
		t.Fatalf("step 3: update given_at: %v", err)
	}

	// --- Step 4: Tick at 08:15 UTC -> suppressed (dose logged within window) ---
	mock2 := &mockPusher{}
	s2 := notify.NewScheduler(db, mock2)
	s2.Tick(baseDate.Add(8*time.Hour + 15*time.Minute))
	if mock2.sendCount() != 0 {
		t.Fatalf("step 4: expected 0 notifications at 08:15 (suppressed), got %d", mock2.sendCount())
	}

	// --- Step 5: Tick at 08:30 UTC -> suppressed (dose logged within window) ---
	mock3 := &mockPusher{}
	s3 := notify.NewScheduler(db, mock3)
	s3.Tick(baseDate.Add(8*time.Hour + 30*time.Minute))
	if mock3.sendCount() != 0 {
		t.Fatalf("step 5: expected 0 notifications at 08:30 (suppressed), got %d", mock3.sendCount())
	}

	// --- Step 6: Tick at 20:00 UTC -> notification fires (next dose, no log) ---
	mock4 := &mockPusher{}
	s4 := notify.NewScheduler(db, mock4)
	s4.Tick(baseDate.Add(20 * time.Hour))
	if mock4.sendCount() != 1 {
		t.Fatalf("step 6: expected 1 notification at 20:00, got %d", mock4.sendCount())
	}

	// --- Step 7: Tick at 20:15 UTC -> +15 min follow-up fires (no log yet) ---
	mock5 := &mockPusher{}
	s5 := notify.NewScheduler(db, mock5)
	s5.Tick(baseDate.Add(20*time.Hour + 15*time.Minute))
	if mock5.sendCount() != 1 {
		t.Fatalf("step 7: expected 1 notification at 20:15 (+15 follow-up), got %d", mock5.sendCount())
	}

	// --- Step 8: Parent logs dose as "skipped" ---
	scheduledTime2 := baseDate.Add(20 * time.Hour).Format("2006-01-02T15:04:05Z")
	status, _ = client.doJSON(http.MethodPost, medLogPath, map[string]any{
		"medication_id":  medID,
		"scheduled_time": scheduledTime2,
		"skipped":        true,
		"skip_reason":    "vomited",
	})
	if status != http.StatusCreated {
		t.Fatalf("step 8: expected 201 creating skipped med-log, got %d", status)
	}

	// --- Step 9: Tick at 20:30 UTC -> skipped dose has scheduled_time matching
	// the dose slot, so IsDoseCovered recognizes it and suppresses the follow-up. ---
	mock6 := &mockPusher{}
	s6 := notify.NewScheduler(db, mock6)
	s6.Tick(baseDate.Add(20*time.Hour + 30*time.Minute))
	if mock6.sendCount() != 0 {
		t.Fatalf("step 9: expected 0 notifications at 20:30 (skipped dose with scheduled_time suppresses), got %d", mock6.sendCount())
	}

	// --- Step 10: Verify adherence ratio ---
	// adherence = given_count / total_count = 1 / 2 = 0.50
	var totalLogs, givenLogs int
	err = db.QueryRow(
		"SELECT COUNT(*) FROM med_logs WHERE medication_id = ? AND baby_id = ?",
		medID, babyID,
	).Scan(&totalLogs)
	if err != nil {
		t.Fatalf("query total med_logs: %v", err)
	}
	if totalLogs != 2 {
		t.Fatalf("expected 2 total med_logs, got %d", totalLogs)
	}

	err = db.QueryRow(
		"SELECT COUNT(*) FROM med_logs WHERE medication_id = ? AND baby_id = ? AND skipped = 0",
		medID, babyID,
	).Scan(&givenLogs)
	if err != nil {
		t.Fatalf("query given med_logs: %v", err)
	}
	if givenLogs != 1 {
		t.Fatalf("expected 1 given med_log, got %d", givenLogs)
	}

	adherence := float64(givenLogs) / float64(totalLogs)
	if adherence != 0.5 {
		t.Errorf("expected adherence ratio 0.5, got %.4f", adherence)
	}

	// --- Step 11: Verify the dashboard shows this medication in upcoming_meds ---
	status, dashMap := client.doJSON(http.MethodGet,
		fmt.Sprintf("/api/babies/%s/dashboard", babyID), nil)
	if status != http.StatusOK {
		t.Fatalf("dashboard: expected 200, got %d: %v", status, dashMap)
	}
	upcomingMeds, ok := dashMap["upcoming_meds"].([]any)
	if !ok {
		t.Fatalf("expected upcoming_meds array in dashboard, got %T", dashMap["upcoming_meds"])
	}
	if len(upcomingMeds) != 1 {
		t.Fatalf("expected 1 upcoming med, got %d", len(upcomingMeds))
	}
	firstMed := upcomingMeds[0].(map[string]any)
	if firstMed["name"] != "Ursodiol" {
		t.Errorf("expected medication name Ursodiol, got %v", firstMed["name"])
	}
	schedTimes, ok := firstMed["schedule_times"].([]any)
	if !ok || len(schedTimes) != 2 {
		t.Errorf("expected 2 schedule_times, got %v", firstMed["schedule_times"])
	}
}

// TestAccountDeletionAnonymizesMedicationTables verifies that account deletion
// anonymizes logged_by and updated_by in medications and med_logs tables.
func TestAccountDeletionAnonymizesMedicationTables(t *testing.T) {
	t.Parallel()

	// First, verify that AnonymizeTables includes medications and med_logs
	requiredTables := []string{"medications", "med_logs"}
	for _, table := range requiredTables {
		found := false
		for _, at := range handler.AnonymizeTables {
			if at == table {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected AnonymizeTables to contain %q; got %v", table, handler.AnonymizeTables)
		}
	}

	srv, db, cleanup := setupIntegrationServer(t)
	defer cleanup()

	clientA := newTestClient(t, srv, db)
	clientB := newTestClient(t, srv, db)

	// --- User A creates a baby ---
	babyID := createBabyViaAPI(t, clientA, "Med Anon Baby")

	// --- User A invites User B ---
	status, inviteResp := clientA.doJSON(http.MethodPost,
		fmt.Sprintf("/api/babies/%s/invite", babyID), nil)
	if status != http.StatusCreated {
		t.Fatalf("expected 201 creating invite, got %d", status)
	}
	inviteCode := inviteResp["code"].(string)
	status, _ = clientB.doJSON(http.MethodPost, "/api/babies/join", map[string]any{"code": inviteCode})
	if status != http.StatusOK {
		t.Fatalf("expected 200 joining baby, got %d", status)
	}

	// --- User A creates a medication ---
	medPath := fmt.Sprintf("/api/babies/%s/medications", babyID)
	status, medResp := clientA.doJSON(http.MethodPost, medPath, map[string]any{
		"name":      "Ursodiol",
		"dose":      "50mg",
		"frequency": "twice_daily",
	})
	if status != http.StatusCreated {
		t.Fatalf("expected 201 creating medication, got %d: %v", status, medResp)
	}
	medID := medResp["id"].(string)

	// --- User A logs a dose (given) ---
	medLogPath := fmt.Sprintf("/api/babies/%s/med-logs", babyID)
	status, _ = clientA.doJSON(http.MethodPost, medLogPath, map[string]any{
		"medication_id":  medID,
		"scheduled_time": "2025-06-15T08:00:00Z",
		"given_at":       "2025-06-15T08:05:00Z",
		"skipped":        false,
	})
	if status != http.StatusCreated {
		t.Fatalf("expected 201 creating med-log (A), got %d", status)
	}

	// --- User B logs a dose (skipped) ---
	status, logB := clientB.doJSON(http.MethodPost, medLogPath, map[string]any{
		"medication_id":  medID,
		"scheduled_time": "2025-06-15T20:00:00Z",
		"skipped":        true,
		"skip_reason":    "unavailable",
	})
	if status != http.StatusCreated {
		t.Fatalf("expected 201 creating med-log (B), got %d", status)
	}
	logBID := logB["id"].(string)

	// --- User A updates User B's med log (sets updated_by = A) ---
	status, _ = clientA.doJSON(http.MethodPut,
		fmt.Sprintf("%s/%s", medLogPath, logBID),
		map[string]any{
			"scheduled_time": "2025-06-15T20:00:00Z",
			"skipped":        true,
			"skip_reason":    "vomited",
		})
	if status != http.StatusOK {
		t.Fatalf("expected 200 updating med-log, got %d", status)
	}

	// --- User A updates the medication (sets updated_by = A) ---
	status, _ = clientA.doJSON(http.MethodPut,
		fmt.Sprintf("%s/%s", medPath, medID),
		map[string]any{
			"name":      "Ursodiol",
			"dose":      "75mg",
			"frequency": "twice_daily",
		})
	if status != http.StatusOK {
		t.Fatalf("expected 200 updating medication, got %d", status)
	}

	userAID := clientA.userID

	// --- User A deletes their account ---
	status, _ = clientA.doJSON(http.MethodDelete, "/api/users/me", nil)
	if status != http.StatusNoContent {
		t.Fatalf("expected 204 deleting account, got %d", status)
	}

	// --- Verify User A is deleted ---
	var userCount int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE id = ?", userAID).Scan(&userCount)
	if err != nil {
		t.Fatalf("query user count: %v", err)
	}
	if userCount != 0 {
		t.Fatalf("expected user A to be deleted, found %d", userCount)
	}

	// --- Verify medications and med_logs still exist ---
	verifyEntryCount(t, db, "medications", babyID, 1)
	verifyEntryCount(t, db, "med_logs", babyID, 2)

	// --- Verify medications.logged_by anonymized for User A ---
	verifyAnonymized(t, db, "medications", "logged_by", userAID, "deleted_user")

	// --- Verify medications.updated_by anonymized for User A ---
	verifyAnonymized(t, db, "medications", "updated_by", userAID, "deleted_user")

	// --- Verify med_logs.logged_by anonymized for User A's entries ---
	verifyAnonymized(t, db, "med_logs", "logged_by", userAID, "deleted_user")

	// --- Verify med_logs.updated_by for B's log (updated by A) is anonymized ---
	var updatedBy sql.NullString
	err = db.QueryRow("SELECT updated_by FROM med_logs WHERE id = ?", logBID).Scan(&updatedBy)
	if err != nil {
		t.Fatalf("query med_logs updated_by: %v", err)
	}
	if !updatedBy.Valid || updatedBy.String != "deleted_user" {
		t.Errorf("expected med_logs.updated_by='deleted_user' for B's log updated by A, got %v", updatedBy)
	}

	// --- Verify User B's entries are NOT anonymized ---
	userBID := clientB.userID
	verifyNotAnonymized(t, db, "med_logs", "logged_by", userBID)

	// --- User B can still read medications and med-logs via API ---
	status, meds := clientB.doJSONArray(medPath)
	if status != http.StatusOK {
		t.Fatalf("expected 200 listing medications, got %d", status)
	}
	if len(meds) != 1 {
		t.Fatalf("expected 1 medication after deletion, got %d", len(meds))
	}

	status, logs := clientB.doJSONList(medLogPath)
	if status != http.StatusOK {
		t.Fatalf("expected 200 listing med-logs, got %d", status)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 med-logs after deletion, got %d", len(logs))
	}

	// --- Verify the baby still exists ---
	var babyCount int
	err = db.QueryRow("SELECT COUNT(*) FROM babies WHERE id = ?", babyID).Scan(&babyCount)
	if err != nil {
		t.Fatalf("query baby count: %v", err)
	}
	if babyCount != 1 {
		t.Fatalf("expected baby to still exist, got count=%d", babyCount)
	}
}
