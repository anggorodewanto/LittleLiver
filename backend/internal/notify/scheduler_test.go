package notify

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

// seedUser inserts a test user and returns their ID.
func seedUser(t *testing.T, db *sql.DB, id, googleID, email string) {
	t.Helper()
	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES (?, ?, ?, ?)",
		id, googleID, email, "Test User")
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

// seedBaby inserts a test baby and links to parent.
func seedBaby(t *testing.T, db *sql.DB, babyID, parentID string) {
	t.Helper()
	_, err := db.Exec("INSERT INTO babies (id, name, sex, date_of_birth) VALUES (?, 'Baby', 'female', '2025-01-01')", babyID)
	if err != nil {
		t.Fatalf("seed baby: %v", err)
	}
	_, err = db.Exec("INSERT INTO baby_parents (baby_id, user_id) VALUES (?, ?)", babyID, parentID)
	if err != nil {
		t.Fatalf("seed baby_parents: %v", err)
	}
}

// seedMedication inserts a medication with given schedule and timezone.
func seedMedication(t *testing.T, db *sql.DB, id, babyID, name string, scheduleTimes []string, tz string, active bool) {
	t.Helper()
	var schedule *string
	if len(scheduleTimes) > 0 {
		b, _ := json.Marshal(scheduleTimes)
		s := string(b)
		schedule = &s
	}
	activeInt := 0
	if active {
		activeInt = 1
	}
	_, err := db.Exec(
		`INSERT INTO medications (id, baby_id, logged_by, name, dose, frequency, schedule, timezone, active)
		 VALUES (?, ?, 'u1', ?, '50mg', 'twice_daily', ?, ?, ?)`,
		id, babyID, name, schedule, tz, activeInt,
	)
	if err != nil {
		t.Fatalf("seed medication: %v", err)
	}
}

// seedMedLog inserts a med_log at a given time for suppression testing.
func seedMedLog(t *testing.T, db *sql.DB, medID, babyID string, scheduledTime, givenAt time.Time) {
	t.Helper()
	stStr := scheduledTime.UTC().Format("2006-01-02T15:04:05Z")
	gaStr := givenAt.UTC().Format("2006-01-02T15:04:05Z")
	_, err := db.Exec(
		`INSERT INTO med_logs (id, medication_id, baby_id, logged_by, scheduled_time, given_at, skipped)
		 VALUES (?, ?, ?, 'u1', ?, ?, 0)`,
		fmt.Sprintf("ml-%d", time.Now().UnixNano()), medID, babyID, stStr, gaStr,
	)
	if err != nil {
		t.Fatalf("seed med_log: %v", err)
	}
}


// TestScheduler_MedicationDueNowTriggersNotification tests that a medication
// due at the current time triggers a push notification.
func TestScheduler_MedicationDueNowTriggersNotification(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	seedUser(t, db, "u1", "g1", "u1@test.com")
	seedBaby(t, db, "b1", "u1")
	testutil.SeedPushSubscription(t, db, "u1", "https://push.example.com/u1")

	// Medication scheduled at 08:00 in America/New_York
	seedMedication(t, db, "med1", "b1", "Ursodiol", []string{"08:00"}, "America/New_York", true)

	// "now" is 08:00 in America/New_York = 13:00 UTC (EST = UTC-5)
	loc, _ := time.LoadLocation("America/New_York")
	now := time.Date(2026, 3, 18, 8, 0, 0, 0, loc).UTC()

	mock := &MockPusher{}
	s := NewScheduler(db, mock)
	s.Tick(now)

	if len(mock.Sends) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(mock.Sends))
	}
	if mock.Sends[0].Payload.URL != "/log/med?medication_id=med1" {
		t.Errorf("wrong URL: %q", mock.Sends[0].Payload.URL)
	}
}

// TestScheduler_LoggedDoseSuppressesNotification tests that a medication
// with a logged dose within +/-30 min of the scheduled time is suppressed.
func TestScheduler_LoggedDoseSuppressesNotification(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	seedUser(t, db, "u1", "g1", "u1@test.com")
	seedBaby(t, db, "b1", "u1")
	testutil.SeedPushSubscription(t, db, "u1", "https://push.example.com/u1")

	seedMedication(t, db, "med1", "b1", "Ursodiol", []string{"08:00"}, "America/New_York", true)

	loc, _ := time.LoadLocation("America/New_York")
	scheduledUTC := time.Date(2026, 3, 18, 8, 0, 0, 0, loc).UTC()
	// Log dose 5 minutes before scheduled time
	seedMedLog(t, db, "med1", "b1", scheduledUTC, scheduledUTC.Add(-5*time.Minute))

	now := scheduledUTC
	mock := &MockPusher{}
	s := NewScheduler(db, mock)
	s.Tick(now)

	if len(mock.Sends) != 0 {
		t.Fatalf("expected 0 notifications (suppressed), got %d", len(mock.Sends))
	}
}

// TestScheduler_FollowUpAt15MinIfNoLog tests that a +15 min follow-up
// fires if no dose has been logged.
func TestScheduler_FollowUpAt15MinIfNoLog(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	seedUser(t, db, "u1", "g1", "u1@test.com")
	seedBaby(t, db, "b1", "u1")
	testutil.SeedPushSubscription(t, db, "u1", "https://push.example.com/u1")

	seedMedication(t, db, "med1", "b1", "Ursodiol", []string{"08:00"}, "America/New_York", true)

	loc, _ := time.LoadLocation("America/New_York")
	scheduledUTC := time.Date(2026, 3, 18, 8, 0, 0, 0, loc).UTC()
	// Now is 15 minutes after scheduled time
	now := scheduledUTC.Add(15 * time.Minute)

	mock := &MockPusher{}
	s := NewScheduler(db, mock)
	s.Tick(now)

	if len(mock.Sends) != 1 {
		t.Fatalf("expected 1 notification (+15 min follow-up), got %d", len(mock.Sends))
	}
}

// TestScheduler_FollowUpAt30MinIfNoLog tests that a +30 min follow-up
// fires if still no dose has been logged.
func TestScheduler_FollowUpAt30MinIfNoLog(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	seedUser(t, db, "u1", "g1", "u1@test.com")
	seedBaby(t, db, "b1", "u1")
	testutil.SeedPushSubscription(t, db, "u1", "https://push.example.com/u1")

	seedMedication(t, db, "med1", "b1", "Ursodiol", []string{"08:00"}, "America/New_York", true)

	loc, _ := time.LoadLocation("America/New_York")
	scheduledUTC := time.Date(2026, 3, 18, 8, 0, 0, 0, loc).UTC()
	// Now is 30 minutes after scheduled time
	now := scheduledUTC.Add(30 * time.Minute)

	mock := &MockPusher{}
	s := NewScheduler(db, mock)
	s.Tick(now)

	if len(mock.Sends) != 1 {
		t.Fatalf("expected 1 notification (+30 min follow-up), got %d", len(mock.Sends))
	}
}

// TestScheduler_LoggedDoseSuppressesFollowUps tests that a logged dose
// suppresses the +15 and +30 min follow-up notifications.
func TestScheduler_LoggedDoseSuppressesFollowUps(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	seedUser(t, db, "u1", "g1", "u1@test.com")
	seedBaby(t, db, "b1", "u1")
	testutil.SeedPushSubscription(t, db, "u1", "https://push.example.com/u1")

	seedMedication(t, db, "med1", "b1", "Ursodiol", []string{"08:00"}, "America/New_York", true)

	loc, _ := time.LoadLocation("America/New_York")
	scheduledUTC := time.Date(2026, 3, 18, 8, 0, 0, 0, loc).UTC()
	// Dose given 2 minutes after scheduled time
	seedMedLog(t, db, "med1", "b1", scheduledUTC, scheduledUTC.Add(2*time.Minute))

	// Check at +15 min
	now := scheduledUTC.Add(15 * time.Minute)
	mock := &MockPusher{}
	s := NewScheduler(db, mock)
	s.Tick(now)

	if len(mock.Sends) != 0 {
		t.Fatalf("expected 0 notifications at +15 (dose logged), got %d", len(mock.Sends))
	}

	// Check at +30 min
	mock2 := &MockPusher{}
	s2 := NewScheduler(db, mock2)
	s2.Tick(scheduledUTC.Add(30 * time.Minute))

	if len(mock2.Sends) != 0 {
		t.Fatalf("expected 0 notifications at +30 (dose logged), got %d", len(mock2.Sends))
	}
}

// TestScheduler_TimezoneConversion tests that timezone conversion is correct.
// A medication at 08:00 America/New_York should fire at 13:00 UTC (EST).
func TestScheduler_TimezoneConversion(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	seedUser(t, db, "u1", "g1", "u1@test.com")
	seedBaby(t, db, "b1", "u1")
	testutil.SeedPushSubscription(t, db, "u1", "https://push.example.com/u1")

	seedMedication(t, db, "med1", "b1", "Ursodiol", []string{"08:00"}, "America/New_York", true)

	// 13:00 UTC = 08:00 EST (during standard time, but March is EDT = UTC-4)
	// March 18, 2026 is during EDT, so 08:00 EDT = 12:00 UTC
	now := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)

	mock := &MockPusher{}
	s := NewScheduler(db, mock)
	s.Tick(now)

	if len(mock.Sends) != 1 {
		t.Fatalf("expected 1 notification at 12:00 UTC (08:00 EDT), got %d", len(mock.Sends))
	}

	// At 13:00 UTC (09:00 EDT) should NOT fire (not a scheduled time)
	mock2 := &MockPusher{}
	s2 := NewScheduler(db, mock2)
	s2.Tick(time.Date(2026, 3, 18, 13, 0, 0, 0, time.UTC))

	if len(mock2.Sends) != 0 {
		t.Fatalf("expected 0 notifications at 13:00 UTC (09:00 EDT), got %d", len(mock2.Sends))
	}
}

// TestScheduler_InactiveMedsSkipped tests that inactive medications are not scheduled.
func TestScheduler_InactiveMedsSkipped(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	seedUser(t, db, "u1", "g1", "u1@test.com")
	seedBaby(t, db, "b1", "u1")
	testutil.SeedPushSubscription(t, db, "u1", "https://push.example.com/u1")

	// Inactive medication
	seedMedication(t, db, "med1", "b1", "Ursodiol", []string{"08:00"}, "America/New_York", false)

	loc, _ := time.LoadLocation("America/New_York")
	now := time.Date(2026, 3, 18, 8, 0, 0, 0, loc).UTC()

	mock := &MockPusher{}
	s := NewScheduler(db, mock)
	s.Tick(now)

	if len(mock.Sends) != 0 {
		t.Fatalf("expected 0 notifications for inactive med, got %d", len(mock.Sends))
	}
}

// TestScheduler_PayloadIncludesRequiredFields verifies the notification payload
// contains scheduled_time (UTC), medication_id, medication name, and click URL.
func TestScheduler_PayloadIncludesRequiredFields(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	seedUser(t, db, "u1", "g1", "u1@test.com")
	seedBaby(t, db, "b1", "u1")
	testutil.SeedPushSubscription(t, db, "u1", "https://push.example.com/u1")

	seedMedication(t, db, "med1", "b1", "Ursodiol", []string{"08:00"}, "America/New_York", true)

	loc, _ := time.LoadLocation("America/New_York")
	now := time.Date(2026, 3, 18, 8, 0, 0, 0, loc).UTC()

	mock := &MockPusher{}
	s := NewScheduler(db, mock)
	s.Tick(now)

	if len(mock.Sends) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(mock.Sends))
	}

	p := mock.Sends[0].Payload
	// Title should include medication name
	if p.Title == "" {
		t.Error("expected non-empty title")
	}
	// Body should include medication name
	if p.Body == "" {
		t.Error("expected non-empty body")
	}
	// URL must be the click URL pattern
	if p.URL != "/log/med?medication_id=med1" {
		t.Errorf("expected URL=/log/med?medication_id=med1, got %q", p.URL)
	}

	// Verify structured Data fields
	expectedScheduledUTC := now.Format("2006-01-02T15:04:05Z")
	if p.Data["scheduled_time"] != expectedScheduledUTC {
		t.Errorf("expected Data[scheduled_time]=%s, got %q", expectedScheduledUTC, p.Data["scheduled_time"])
	}
	if p.Data["medication_id"] != "med1" {
		t.Errorf("expected Data[medication_id]=med1, got %q", p.Data["medication_id"])
	}
	// Body should contain dose and baby name
	if !strings.Contains(p.Body, "50mg") {
		t.Errorf("body should contain dose 50mg, got %q", p.Body)
	}
	if !strings.Contains(p.Body, "Baby") {
		t.Errorf("body should contain baby name Baby, got %q", p.Body)
	}
	// Data should include name
	if p.Data["name"] != "Ursodiol" {
		t.Errorf("expected Data[name]=Ursodiol, got %q", p.Data["name"])
	}
}

// TestScheduler_MultipleParentsReceiveNotification tests that all parents
// with push subscriptions for a baby receive notifications.
func TestScheduler_MultipleParentsReceiveNotification(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	seedUser(t, db, "u1", "g1", "u1@test.com")
	seedUser(t, db, "u2", "g2", "u2@test.com")
	seedBaby(t, db, "b1", "u1")
	// Link second parent
	_, err := db.Exec("INSERT INTO baby_parents (baby_id, user_id) VALUES ('b1', 'u2')")
	if err != nil {
		t.Fatalf("link parent2: %v", err)
	}
	testutil.SeedPushSubscription(t, db, "u1", "https://push.example.com/u1")
	testutil.SeedPushSubscription(t, db, "u2", "https://push.example.com/u2")

	seedMedication(t, db, "med1", "b1", "Ursodiol", []string{"08:00"}, "America/New_York", true)

	loc, _ := time.LoadLocation("America/New_York")
	now := time.Date(2026, 3, 18, 8, 0, 0, 0, loc).UTC()

	mock := &MockPusher{}
	s := NewScheduler(db, mock)
	s.Tick(now)

	if len(mock.Sends) != 2 {
		t.Fatalf("expected 2 notifications (one per parent), got %d", len(mock.Sends))
	}

	// Verify both endpoints received a notification
	endpoints := map[string]bool{}
	for _, send := range mock.Sends {
		endpoints[send.Sub.Endpoint] = true
	}
	if !endpoints["https://push.example.com/u1"] {
		t.Error("parent u1 did not receive notification")
	}
	if !endpoints["https://push.example.com/u2"] {
		t.Error("parent u2 did not receive notification")
	}
}

// TestScheduler_NoScheduleSkipped tests that medications without a schedule
// are silently skipped.
func TestScheduler_NoScheduleSkipped(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	seedUser(t, db, "u1", "g1", "u1@test.com")
	seedBaby(t, db, "b1", "u1")
	testutil.SeedPushSubscription(t, db, "u1", "https://push.example.com/u1")

	// Medication with no schedule times
	seedMedication(t, db, "med1", "b1", "Ursodiol", nil, "America/New_York", true)

	now := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)

	mock := &MockPusher{}
	s := NewScheduler(db, mock)
	s.Tick(now)

	if len(mock.Sends) != 0 {
		t.Fatalf("expected 0 notifications for med with no schedule, got %d", len(mock.Sends))
	}
}

// TestScheduler_MultipleScheduleTimes tests that a medication with multiple
// schedule times only fires for the one that matches.
func TestScheduler_MultipleScheduleTimes(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	seedUser(t, db, "u1", "g1", "u1@test.com")
	seedBaby(t, db, "b1", "u1")
	testutil.SeedPushSubscription(t, db, "u1", "https://push.example.com/u1")

	seedMedication(t, db, "med1", "b1", "Ursodiol", []string{"08:00", "20:00"}, "America/New_York", true)

	loc, _ := time.LoadLocation("America/New_York")
	// At 08:00 EDT
	now := time.Date(2026, 3, 18, 8, 0, 0, 0, loc).UTC()

	mock := &MockPusher{}
	s := NewScheduler(db, mock)
	s.Tick(now)

	if len(mock.Sends) != 1 {
		t.Fatalf("expected 1 notification for 08:00, got %d", len(mock.Sends))
	}
}

// TestScheduler_PushSendErrorDoesNotPanic tests that a push send error
// is handled gracefully without stopping other notifications.
func TestScheduler_PushSendErrorDoesNotPanic(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	seedUser(t, db, "u1", "g1", "u1@test.com")
	seedBaby(t, db, "b1", "u1")
	testutil.SeedPushSubscription(t, db, "u1", "https://push.example.com/u1")

	seedMedication(t, db, "med1", "b1", "Ursodiol", []string{"08:00"}, "America/New_York", true)

	loc, _ := time.LoadLocation("America/New_York")
	now := time.Date(2026, 3, 18, 8, 0, 0, 0, loc).UTC()

	mock := &MockPusher{Err: fmt.Errorf("network error")}
	s := NewScheduler(db, mock)
	// Should not panic
	s.Tick(now)
}

// TestScheduler_StartStop tests that the scheduler starts and can be stopped.
func TestScheduler_StartStop(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	mock := &MockPusher{}
	s := NewScheduler(db, mock)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.Start()
	}()

	// Give the goroutine a moment to start
	time.Sleep(10 * time.Millisecond)
	s.Stop()
	wg.Wait()
	// If we reach here without hanging, the test passes
}

// TestScheduler_ClosedDBHandlesError tests that Tick handles a closed DB gracefully.
func TestScheduler_ClosedDBHandlesError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	db.Close() // close immediately

	mock := &MockPusher{}
	s := NewScheduler(db, mock)
	// Should not panic, just log the error
	s.Tick(time.Now())

	if len(mock.Sends) != 0 {
		t.Errorf("expected 0 sends on DB error, got %d", len(mock.Sends))
	}
}

// TestScheduler_InvalidTimezoneSkipped tests that a medication with invalid timezone is skipped.
func TestScheduler_InvalidTimezoneSkipped(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	seedUser(t, db, "u1", "g1", "u1@test.com")
	seedBaby(t, db, "b1", "u1")
	testutil.SeedPushSubscription(t, db, "u1", "https://push.example.com/u1")

	// Insert medication with invalid timezone directly
	sched := `["08:00"]`
	_, err := db.Exec(
		`INSERT INTO medications (id, baby_id, logged_by, name, dose, frequency, schedule, timezone, active)
		 VALUES ('med1', 'b1', 'u1', 'Ursodiol', '50mg', 'twice_daily', ?, 'Invalid/Timezone', 1)`, sched)
	if err != nil {
		t.Fatalf("seed medication: %v", err)
	}

	now := time.Date(2026, 3, 18, 8, 0, 0, 0, time.UTC)
	mock := &MockPusher{}
	s := NewScheduler(db, mock)
	s.Tick(now)

	if len(mock.Sends) != 0 {
		t.Fatalf("expected 0 notifications for invalid timezone, got %d", len(mock.Sends))
	}
}

// TestScheduler_InvalidScheduleFormatSkipped tests that an invalid schedule JSON is skipped.
func TestScheduler_InvalidScheduleFormatSkipped(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	seedUser(t, db, "u1", "g1", "u1@test.com")
	seedBaby(t, db, "b1", "u1")
	testutil.SeedPushSubscription(t, db, "u1", "https://push.example.com/u1")

	// Insert medication with invalid schedule JSON
	_, err := db.Exec(
		`INSERT INTO medications (id, baby_id, logged_by, name, dose, frequency, schedule, timezone, active)
		 VALUES ('med1', 'b1', 'u1', 'Ursodiol', '50mg', 'twice_daily', 'not-json', 'America/New_York', 1)`)
	if err != nil {
		t.Fatalf("seed medication: %v", err)
	}

	now := time.Date(2026, 3, 18, 8, 0, 0, 0, time.UTC)
	mock := &MockPusher{}
	s := NewScheduler(db, mock)
	s.Tick(now)

	if len(mock.Sends) != 0 {
		t.Fatalf("expected 0 notifications for invalid schedule, got %d", len(mock.Sends))
	}
}

// TestScheduler_InvalidScheduleTimeFormatSkipped tests that an invalid time format
// in the schedule (e.g., "abc") is skipped.
func TestScheduler_InvalidScheduleTimeFormatSkipped(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	seedUser(t, db, "u1", "g1", "u1@test.com")
	seedBaby(t, db, "b1", "u1")
	testutil.SeedPushSubscription(t, db, "u1", "https://push.example.com/u1")

	// Schedule with bad time format: "abc" fails hour parse, "8:abc" fails minute parse, "8" has no colon
	sched := `["abc:00","08:abc","8"]`
	_, err := db.Exec(
		`INSERT INTO medications (id, baby_id, logged_by, name, dose, frequency, schedule, timezone, active)
		 VALUES ('med1', 'b1', 'u1', 'Ursodiol', '50mg', 'twice_daily', ?, 'America/New_York', 1)`, sched)
	if err != nil {
		t.Fatalf("seed medication: %v", err)
	}

	loc, _ := time.LoadLocation("America/New_York")
	now := time.Date(2026, 3, 18, 8, 0, 0, 0, loc).UTC()
	mock := &MockPusher{}
	s := NewScheduler(db, mock)
	s.Tick(now)

	if len(mock.Sends) != 0 {
		t.Fatalf("expected 0 notifications for invalid time format, got %d", len(mock.Sends))
	}
}

// TestScheduler_NoTimezoneUsesUTC tests that a medication with no timezone defaults to UTC.
func TestScheduler_NoTimezoneUsesUTC(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	seedUser(t, db, "u1", "g1", "u1@test.com")
	seedBaby(t, db, "b1", "u1")
	testutil.SeedPushSubscription(t, db, "u1", "https://push.example.com/u1")

	// Medication with no timezone
	sched := `["12:00"]`
	_, err := db.Exec(
		`INSERT INTO medications (id, baby_id, logged_by, name, dose, frequency, schedule, timezone, active)
		 VALUES ('med1', 'b1', 'u1', 'Ursodiol', '50mg', 'twice_daily', ?, NULL, 1)`, sched)
	if err != nil {
		t.Fatalf("seed medication: %v", err)
	}

	// 12:00 UTC should trigger
	now := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	mock := &MockPusher{}
	s := NewScheduler(db, mock)
	s.Tick(now)

	if len(mock.Sends) != 1 {
		t.Fatalf("expected 1 notification at 12:00 UTC, got %d", len(mock.Sends))
	}
}

// TestScheduler_ParentWithNoSubscriptions tests that a parent without push
// subscriptions does not cause errors.
func TestScheduler_ParentWithNoSubscriptions(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	seedUser(t, db, "u1", "g1", "u1@test.com")
	seedBaby(t, db, "b1", "u1")
	// No push subscription for u1

	seedMedication(t, db, "med1", "b1", "Ursodiol", []string{"08:00"}, "America/New_York", true)

	loc, _ := time.LoadLocation("America/New_York")
	now := time.Date(2026, 3, 18, 8, 0, 0, 0, loc).UTC()

	mock := &MockPusher{}
	s := NewScheduler(db, mock)
	s.Tick(now)

	// No crash, no sends
	if len(mock.Sends) != 0 {
		t.Fatalf("expected 0 notifications (no subscriptions), got %d", len(mock.Sends))
	}
}

// TestScheduler_MultipleDevicesPerParent tests that a parent with multiple
// push subscriptions receives notifications on all devices.
func TestScheduler_MultipleDevicesPerParent(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	seedUser(t, db, "u1", "g1", "u1@test.com")
	seedBaby(t, db, "b1", "u1")
	testutil.SeedPushSubscription(t, db, "u1", "https://push.example.com/device1")
	testutil.SeedPushSubscription(t, db, "u1", "https://push.example.com/device2")

	seedMedication(t, db, "med1", "b1", "Ursodiol", []string{"08:00"}, "America/New_York", true)

	loc, _ := time.LoadLocation("America/New_York")
	now := time.Date(2026, 3, 18, 8, 0, 0, 0, loc).UTC()

	mock := &MockPusher{}
	s := NewScheduler(db, mock)
	s.Tick(now)

	if len(mock.Sends) != 2 {
		t.Fatalf("expected 2 notifications (one per device), got %d", len(mock.Sends))
	}
}

// TestParseSchedule tests the parseSchedule function directly.
func TestParseSchedule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    *string
		expected int
	}{
		{"nil", nil, 0},
		{"empty string", strPtr(""), 0},
		{"valid single", strPtr(`["08:00"]`), 1},
		{"valid multiple", strPtr(`["08:00","20:00"]`), 2},
		{"invalid json", strPtr("not-json"), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSchedule(tt.input)
			if len(result) != tt.expected {
				t.Errorf("parseSchedule(%v) = %d items, want %d", tt.input, len(result), tt.expected)
			}
		})
	}
}

// TestBuildMedPayload tests the buildMedPayload function directly.
func TestBuildMedPayload(t *testing.T) {
	t.Parallel()

	med := activeMed{
		ID:       "med123",
		BabyID:   "b1",
		BabyName: "TestBaby",
		Name:     "Ursodiol",
		Dose:     "100mg",
	}
	scheduled := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)

	p := buildMedPayload(med, scheduled)

	expectedTitle := "\U0001F48A Ursodiol \u2014 Time for dose"
	if p.Title != expectedTitle {
		t.Errorf("wrong title: %q, expected %q", p.Title, expectedTitle)
	}
	if p.URL != "/log/med?medication_id=med123" {
		t.Errorf("wrong URL: %q", p.URL)
	}
	if !strings.Contains(p.Body, "TestBaby") {
		t.Errorf("body should contain baby name: %q", p.Body)
	}
	if !strings.Contains(p.Body, "100mg") {
		t.Errorf("body should contain dose: %q", p.Body)
	}
	if p.Data["name"] != "Ursodiol" {
		t.Errorf("expected Data[name]=Ursodiol, got %q", p.Data["name"])
	}
	if p.Data["scheduled_time"] != "2026-03-18T12:00:00Z" {
		t.Errorf("expected Data[scheduled_time]=2026-03-18T12:00:00Z, got %q", p.Data["scheduled_time"])
	}
	if p.Data["medication_id"] != "med123" {
		t.Errorf("expected Data[medication_id]=med123, got %q", p.Data["medication_id"])
	}
}

func strPtr(s string) *string {
	return &s
}

// TestScheduler_BabyWithNoParents tests that a baby with no parents
// linked does not cause errors.
func TestScheduler_BabyWithNoParents(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	seedUser(t, db, "u1", "g1", "u1@test.com")
	// Create baby without linking to any parent
	_, err := db.Exec("INSERT INTO babies (id, name, sex, date_of_birth) VALUES ('b1', 'Baby', 'female', '2025-01-01')")
	if err != nil {
		t.Fatalf("seed baby: %v", err)
	}

	sched := `["08:00"]`
	_, err = db.Exec(
		`INSERT INTO medications (id, baby_id, logged_by, name, dose, frequency, schedule, timezone, active)
		 VALUES ('med1', 'b1', 'u1', 'Ursodiol', '50mg', 'twice_daily', ?, 'America/New_York', 1)`, sched)
	if err != nil {
		t.Fatalf("seed medication: %v", err)
	}

	loc, _ := time.LoadLocation("America/New_York")
	now := time.Date(2026, 3, 18, 8, 0, 0, 0, loc).UTC()

	mock := &MockPusher{}
	s := NewScheduler(db, mock)
	s.Tick(now)

	if len(mock.Sends) != 0 {
		t.Fatalf("expected 0 notifications (no parents linked), got %d", len(mock.Sends))
	}
}

// TestScheduler_EmptyTimezoneDefaultsToUTC tests that an empty-string timezone
// defaults to UTC.
func TestScheduler_EmptyTimezoneDefaultsToUTC(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	seedUser(t, db, "u1", "g1", "u1@test.com")
	seedBaby(t, db, "b1", "u1")
	testutil.SeedPushSubscription(t, db, "u1", "https://push.example.com/u1")

	sched := `["12:00"]`
	_, err := db.Exec(
		`INSERT INTO medications (id, baby_id, logged_by, name, dose, frequency, schedule, timezone, active)
		 VALUES ('med1', 'b1', 'u1', 'Ursodiol', '50mg', 'twice_daily', ?, '', 1)`, sched)
	if err != nil {
		t.Fatalf("seed medication: %v", err)
	}

	now := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	mock := &MockPusher{}
	s := NewScheduler(db, mock)
	s.Tick(now)

	if len(mock.Sends) != 1 {
		t.Fatalf("expected 1 notification at 12:00 UTC with empty tz, got %d", len(mock.Sends))
	}
}

// TestScheduler_SkippedDoseSuppresses tests that a skipped med_log
// (skipped=true, given_at=NULL) suppresses the notification per spec §6.4.
func TestScheduler_SkippedDoseSuppresses(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	seedUser(t, db, "u1", "g1", "u1@test.com")
	seedBaby(t, db, "b1", "u1")
	testutil.SeedPushSubscription(t, db, "u1", "https://push.example.com/u1")

	seedMedication(t, db, "med1", "b1", "Ursodiol", []string{"08:00"}, "America/New_York", true)

	loc, _ := time.LoadLocation("America/New_York")
	scheduledUTC := time.Date(2026, 3, 18, 8, 0, 0, 0, loc).UTC()

	// Insert a skipped log (given_at is NULL) with created_at within the suppression window
	stStr := scheduledUTC.Format("2006-01-02T15:04:05Z")
	_, err := db.Exec(
		`INSERT INTO med_logs (id, medication_id, baby_id, logged_by, scheduled_time, given_at, skipped, skip_reason, created_at)
		 VALUES ('ml-skip', 'med1', 'b1', 'u1', ?, NULL, 1, 'Out of stock', ?)`, stStr, stStr)
	if err != nil {
		t.Fatalf("seed skipped med_log: %v", err)
	}

	now := scheduledUTC
	mock := &MockPusher{}
	s := NewScheduler(db, mock)
	s.Tick(now)

	// Skipped dose SHOULD suppress — spec §6.4: suppression uses created_at for skipped doses
	if len(mock.Sends) != 0 {
		t.Fatalf("expected 0 notifications (skipped dose should suppress), got %d", len(mock.Sends))
	}
}

// TestScheduler_DroppedPushSubsTableHandlesError tests that the scheduler
// handles missing push_subscriptions table gracefully.
func TestScheduler_DroppedPushSubsTableHandlesError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	seedUser(t, db, "u1", "g1", "u1@test.com")
	seedBaby(t, db, "b1", "u1")
	seedMedication(t, db, "med1", "b1", "Ursodiol", []string{"08:00"}, "America/New_York", true)

	// Drop push_subscriptions to trigger query error
	_, err := db.Exec("DROP TABLE push_subscriptions")
	if err != nil {
		t.Fatalf("drop table: %v", err)
	}

	loc, _ := time.LoadLocation("America/New_York")
	now := time.Date(2026, 3, 18, 8, 0, 0, 0, loc).UTC()

	mock := &MockPusher{}
	s := NewScheduler(db, mock)
	s.Tick(now) // Should not panic

	if len(mock.Sends) != 0 {
		t.Fatalf("expected 0 sends, got %d", len(mock.Sends))
	}
}

// TestScheduler_DroppedBabyParentsTableHandlesError tests that the scheduler
// handles missing baby_parents table gracefully.
func TestScheduler_DroppedBabyParentsTableHandlesError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	seedUser(t, db, "u1", "g1", "u1@test.com")
	seedBaby(t, db, "b1", "u1")
	testutil.SeedPushSubscription(t, db, "u1", "https://push.example.com/u1")
	seedMedication(t, db, "med1", "b1", "Ursodiol", []string{"08:00"}, "America/New_York", true)

	// Drop baby_parents to trigger query error in sendNotifications
	_, err := db.Exec("DROP TABLE baby_parents")
	if err != nil {
		t.Fatalf("drop table: %v", err)
	}

	loc, _ := time.LoadLocation("America/New_York")
	now := time.Date(2026, 3, 18, 8, 0, 0, 0, loc).UTC()

	mock := &MockPusher{}
	s := NewScheduler(db, mock)
	s.Tick(now) // Should not panic

	if len(mock.Sends) != 0 {
		t.Fatalf("expected 0 sends, got %d", len(mock.Sends))
	}
}

// TestScheduler_DroppedMedLogsTableHandlesError tests that the scheduler
// handles missing med_logs table gracefully (isDoseSuppressed error path).
func TestScheduler_DroppedMedLogsTableHandlesError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	seedUser(t, db, "u1", "g1", "u1@test.com")
	seedBaby(t, db, "b1", "u1")
	testutil.SeedPushSubscription(t, db, "u1", "https://push.example.com/u1")
	seedMedication(t, db, "med1", "b1", "Ursodiol", []string{"08:00"}, "America/New_York", true)

	// Drop med_logs to trigger query error in isDoseSuppressed
	_, err := db.Exec("DROP TABLE med_logs")
	if err != nil {
		t.Fatalf("drop table: %v", err)
	}

	loc, _ := time.LoadLocation("America/New_York")
	now := time.Date(2026, 3, 18, 8, 0, 0, 0, loc).UTC()

	mock := &MockPusher{}
	s := NewScheduler(db, mock)
	s.Tick(now) // Should not panic, suppression check fails -> sends notification

	if len(mock.Sends) != 1 {
		t.Fatalf("expected 1 send (suppression check failed, defaults to not suppressed), got %d", len(mock.Sends))
	}
}

// TestScheduler_NotDueTimeNoNotification tests that at a time that is NOT
// a scheduled time or follow-up, no notification is sent.
func TestScheduler_NotDueTimeNoNotification(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	seedUser(t, db, "u1", "g1", "u1@test.com")
	seedBaby(t, db, "b1", "u1")
	testutil.SeedPushSubscription(t, db, "u1", "https://push.example.com/u1")

	seedMedication(t, db, "med1", "b1", "Ursodiol", []string{"08:00"}, "America/New_York", true)

	loc, _ := time.LoadLocation("America/New_York")
	// 09:00 EDT is not 08:00, 08:15, or 08:30 - no notification
	now := time.Date(2026, 3, 18, 9, 0, 0, 0, loc).UTC()

	mock := &MockPusher{}
	s := NewScheduler(db, mock)
	s.Tick(now)

	if len(mock.Sends) != 0 {
		t.Fatalf("expected 0 notifications at non-due time, got %d", len(mock.Sends))
	}
}
