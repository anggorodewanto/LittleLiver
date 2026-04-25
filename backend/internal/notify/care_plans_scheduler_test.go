package notify

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

// seedCarePlanWithPhases creates a plan + phases, returning saved phases.
func seedCarePlanWithPhases(t *testing.T, db *sql.DB, parentID, babyID, planName, tz string, phases []model.CarePlanPhase) (*model.CarePlan, []model.CarePlanPhase) {
	t.Helper()
	plan, err := store.CreateCarePlan(db, babyID, parentID, planName, nil, tz)
	if err != nil {
		t.Fatalf("CreateCarePlan: %v", err)
	}
	saved, err := store.ReplaceCarePlanPhases(db, plan.ID, phases)
	if err != nil {
		t.Fatalf("ReplaceCarePlanPhases: %v", err)
	}
	return plan, saved
}

func TestScheduler_PhaseSwitchAt9amSendsPush(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	seedUser(t, db, "u1", "g1", "u1@test.com")
	seedBaby(t, db, "b1", "u1")
	testutil.SeedPushSubscription(t, db, "u1", "https://push.example.com/u1")

	loc, _ := time.LoadLocation("America/New_York")
	plan, saved := seedCarePlanWithPhases(t, db, "u1", "b1", "Antibiotic Rotation", "America/New_York",
		[]model.CarePlanPhase{
			{Seq: 1, Label: "Cefixime", StartDate: "2026-05-01"},
			{Seq: 2, Label: "Amoxicillin", StartDate: "2026-06-01"},
		})

	now := time.Date(2026, 6, 1, 9, 0, 0, 0, loc).UTC()

	mock := &MockPusher{}
	s := NewScheduler(db, mock)
	s.Tick(now)

	if len(mock.Sends) != 1 {
		t.Fatalf("expected 1 push, got %d", len(mock.Sends))
	}
	got := mock.Sends[0].Payload
	wantURL := "/care-plans/" + plan.ID
	if got.URL != wantURL {
		t.Errorf("URL = %q, want %q", got.URL, wantURL)
	}
	if !strings.Contains(got.Body, "Amoxicillin") {
		t.Errorf("body should mention Amoxicillin, got %q", got.Body)
	}
	if got.Data["phase_id"] != saved[1].ID {
		t.Errorf("data phase_id mismatch")
	}
	if got.Data["kind"] != "switch" {
		t.Errorf("data kind = %q, want switch", got.Data["kind"])
	}
}

func TestScheduler_PhaseHeadsUpTwoDaysPriorSendsPush(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	seedUser(t, db, "u1", "g1", "u1@test.com")
	seedBaby(t, db, "b1", "u1")
	testutil.SeedPushSubscription(t, db, "u1", "https://push.example.com/u1")

	loc, _ := time.LoadLocation("America/New_York")
	_, _ = seedCarePlanWithPhases(t, db, "u1", "b1", "Antibiotic Rotation", "America/New_York",
		[]model.CarePlanPhase{
			{Seq: 1, Label: "Cefixime", StartDate: "2026-05-01"},
			{Seq: 2, Label: "Amoxicillin", StartDate: "2026-06-01"},
		})

	// 2 days before phase 2 start, at 09:00 plan-tz.
	now := time.Date(2026, 5, 30, 9, 0, 0, 0, loc).UTC()

	mock := &MockPusher{}
	s := NewScheduler(db, mock)
	s.Tick(now)

	if len(mock.Sends) != 1 {
		t.Fatalf("expected 1 heads-up push, got %d", len(mock.Sends))
	}
	if mock.Sends[0].Payload.Data["kind"] != "heads_up" {
		t.Errorf("kind = %q, want heads_up", mock.Sends[0].Payload.Data["kind"])
	}
}

func TestScheduler_FirstPhaseDoesNotFire(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	seedUser(t, db, "u1", "g1", "u1@test.com")
	seedBaby(t, db, "b1", "u1")
	testutil.SeedPushSubscription(t, db, "u1", "https://push.example.com/u1")

	loc, _ := time.LoadLocation("America/New_York")
	_, _ = seedCarePlanWithPhases(t, db, "u1", "b1", "Antibiotic Rotation", "America/New_York",
		[]model.CarePlanPhase{
			{Seq: 1, Label: "Cefixime", StartDate: "2026-05-01"},
			{Seq: 2, Label: "Amoxicillin", StartDate: "2026-06-01"},
		})

	// 09:00 on the day phase 1 starts — should not push (creating the plan
	// implies the user already knows about phase 1).
	now := time.Date(2026, 5, 1, 9, 0, 0, 0, loc).UTC()

	mock := &MockPusher{}
	s := NewScheduler(db, mock)
	s.Tick(now)

	if len(mock.Sends) != 0 {
		t.Errorf("expected 0 pushes for first phase, got %d", len(mock.Sends))
	}
}

func TestScheduler_PhaseNotificationSuppressedSecondTick(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	seedUser(t, db, "u1", "g1", "u1@test.com")
	seedBaby(t, db, "b1", "u1")
	testutil.SeedPushSubscription(t, db, "u1", "https://push.example.com/u1")

	loc, _ := time.LoadLocation("America/New_York")
	_, _ = seedCarePlanWithPhases(t, db, "u1", "b1", "Antibiotic Rotation", "America/New_York",
		[]model.CarePlanPhase{
			{Seq: 1, Label: "Cefixime", StartDate: "2026-05-01"},
			{Seq: 2, Label: "Amoxicillin", StartDate: "2026-06-01"},
		})

	now := time.Date(2026, 6, 1, 9, 0, 0, 0, loc).UTC()

	mock := &MockPusher{}
	s := NewScheduler(db, mock)
	s.Tick(now)
	s.Tick(now) // immediate re-tick at the same instant

	if len(mock.Sends) != 1 {
		t.Errorf("expected 1 push across two ticks (idempotent), got %d", len(mock.Sends))
	}
}

func TestScheduler_OffPeakDoesNotFire(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	seedUser(t, db, "u1", "g1", "u1@test.com")
	seedBaby(t, db, "b1", "u1")
	testutil.SeedPushSubscription(t, db, "u1", "https://push.example.com/u1")

	loc, _ := time.LoadLocation("America/New_York")
	_, _ = seedCarePlanWithPhases(t, db, "u1", "b1", "Antibiotic Rotation", "America/New_York",
		[]model.CarePlanPhase{
			{Seq: 1, Label: "Cefixime", StartDate: "2026-05-01"},
			{Seq: 2, Label: "Amoxicillin", StartDate: "2026-06-01"},
		})

	// On the right day but at a non-trigger minute.
	now := time.Date(2026, 6, 1, 10, 0, 0, 0, loc).UTC()

	mock := &MockPusher{}
	s := NewScheduler(db, mock)
	s.Tick(now)

	if len(mock.Sends) != 0 {
		t.Errorf("expected 0 pushes off-peak, got %d", len(mock.Sends))
	}
}
