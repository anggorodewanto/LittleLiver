package notify

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

// utcTimeFormat is the standard UTC time format used for scheduling timestamps.
const utcTimeFormat = "2006-01-02T15:04:05Z"

// Scheduler checks active medications every minute and sends push
// notifications when a dose is due. It is stateless — each tick
// re-derives what notifications should be sent.
type Scheduler struct {
	db     *sql.DB
	pusher Pusher
	stop   chan struct{}
}

// NewScheduler creates a new medication reminder scheduler.
func NewScheduler(db *sql.DB, pusher Pusher) *Scheduler {
	return &Scheduler{
		db:     db,
		pusher: pusher,
		stop:   make(chan struct{}),
	}
}

// Start runs the scheduler loop, ticking every minute. It blocks until Stop
// is called.
func (s *Scheduler) Start() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case t := <-ticker.C:
			s.Tick(t)
		case <-s.stop:
			return
		}
	}
}

// Stop signals the scheduler to stop.
func (s *Scheduler) Stop() {
	close(s.stop)
}

// activeMed represents an active medication with schedule information.
type activeMed struct {
	ID       string
	BabyID   string
	BabyName string
	Name     string
	Dose     string
	Schedule *string
	Timezone *string
}

// carePlanNotifyHour is the local-time hour at which care plan boundary
// notifications fire. Tied to plan timezone, not user timezone.
const carePlanNotifyHour = 9

// Tick performs a single scheduling check at the given time.
// It queries all active medications with schedules, checks which ones
// are due (at 0, +15, or +30 minutes), suppresses those with recent
// med_logs, and sends push notifications.
func (s *Scheduler) Tick(now time.Time) {
	meds, err := s.queryActiveMeds()
	if err != nil {
		log.Printf("scheduler: query active meds: %v", err)
		return
	}

	for _, med := range meds {
		s.processMedication(med, now)
	}

	s.processCarePlans(now)
}

// processCarePlans walks every active care plan and fires phase-boundary
// notifications when nowLocal lands on the trigger hour for a phase's
// switch-day or two-days-prior heads-up.
func (s *Scheduler) processCarePlans(now time.Time) {
	plans, err := s.queryActiveCarePlans()
	if err != nil {
		log.Printf("scheduler: query care plans: %v", err)
		return
	}

	for _, plan := range plans {
		loc, err := time.LoadLocation(plan.Timezone)
		if err != nil {
			log.Printf("scheduler: invalid timezone %q for care plan %s: %v", plan.Timezone, plan.ID, err)
			continue
		}
		nowLocal := now.In(loc)
		if nowLocal.Hour() != carePlanNotifyHour || nowLocal.Minute() != 0 {
			continue
		}

		phases, err := store.ListCarePlanPhases(s.db, plan.ID)
		if err != nil {
			log.Printf("scheduler: list phases for plan %s: %v", plan.ID, err)
			continue
		}

		for _, phase := range phases {
			if phase.Seq == 1 {
				// First phase fires no notification — creating the plan
				// implies the user already knows about it.
				continue
			}
			startDate, err := time.ParseInLocation("2006-01-02", phase.StartDate, loc)
			if err != nil {
				log.Printf("scheduler: bad phase start_date %q: %v", phase.StartDate, err)
				continue
			}
			today := time.Date(nowLocal.Year(), nowLocal.Month(), nowLocal.Day(), 0, 0, 0, 0, loc)

			switch {
			case today.Equal(startDate):
				s.firePhaseNotification(plan, phase, "switch")
			case today.Equal(startDate.AddDate(0, 0, -2)):
				s.firePhaseNotification(plan, phase, "heads_up")
			}
		}
	}
}

// firePhaseNotification gates on the audit ledger so each (phase, kind) is
// pushed exactly once, then sends to all parents of the baby.
func (s *Scheduler) firePhaseNotification(plan carePlanRow, phase model.CarePlanPhase, kind string) {
	sent, err := store.RecordPhaseNotification(s.db, phase.ID, kind)
	if err != nil {
		log.Printf("scheduler: record phase notification: %v", err)
		return
	}
	if !sent {
		return
	}

	parentIDs, err := s.queryBabyParents(plan.BabyID)
	if err != nil {
		log.Printf("scheduler: query parents for baby %s: %v", plan.BabyID, err)
		return
	}
	if len(parentIDs) == 0 {
		return
	}

	payload := buildCarePlanPayload(plan, phase, kind)
	for _, parentID := range parentIDs {
		subs, err := s.queryPushSubscriptions(parentID)
		if err != nil {
			log.Printf("scheduler: query subs for user %s: %v", parentID, err)
			continue
		}
		for _, sub := range subs {
			resp, err := s.pusher.Send(sub, payload)
			if err != nil {
				log.Printf("scheduler: push send to %s: %v", sub.Endpoint, err)
				continue
			}
			if resp != nil && (resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound) {
				if delErr := store.DeletePushSubscriptionByEndpoint(s.db, sub.Endpoint); delErr != nil {
					log.Printf("scheduler: failed to delete stale subscription: %v", delErr)
				}
			}
		}
	}
}

type carePlanRow struct {
	ID       string
	BabyID   string
	BabyName string
	Name     string
	Timezone string
}

func (s *Scheduler) queryActiveCarePlans() ([]carePlanRow, error) {
	rows, err := s.db.Query(
		`SELECT p.id, p.baby_id, b.name, p.name, p.timezone
		 FROM care_plans p
		 JOIN babies b ON p.baby_id = b.id
		 WHERE p.active = 1`,
	)
	if err != nil {
		return nil, fmt.Errorf("query care plans: %w", err)
	}
	defer rows.Close()

	var plans []carePlanRow
	for rows.Next() {
		var p carePlanRow
		if err := rows.Scan(&p.ID, &p.BabyID, &p.BabyName, &p.Name, &p.Timezone); err != nil {
			return nil, fmt.Errorf("scan care plan: %w", err)
		}
		plans = append(plans, p)
	}
	return plans, rows.Err()
}

func (s *Scheduler) queryBabyParents(babyID string) ([]string, error) {
	rows, err := s.db.Query("SELECT user_id FROM baby_parents WHERE baby_id = ?", babyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		ids = append(ids, uid)
	}
	return ids, rows.Err()
}

// buildCarePlanPayload constructs the Web Push payload for a phase boundary.
func buildCarePlanPayload(plan carePlanRow, phase model.CarePlanPhase, kind string) Payload {
	title := fmt.Sprintf("Care plan: %s", plan.Name)
	body := fmt.Sprintf("'%s' starts %s", phase.Label, phase.StartDate)
	if kind == "heads_up" {
		body = fmt.Sprintf("Heads up: '%s' starts %s", phase.Label, phase.StartDate)
	}
	return Payload{
		Title: title,
		Body:  body,
		URL:   fmt.Sprintf("/care-plans/%s", plan.ID),
		Data: map[string]string{
			"care_plan_id": plan.ID,
			"phase_id":     phase.ID,
			"kind":         kind,
			"baby_name":    plan.BabyName,
		},
	}
}

// queryActiveMeds returns all active medications that have a schedule.
func (s *Scheduler) queryActiveMeds() ([]activeMed, error) {
	rows, err := s.db.Query(
		`SELECT m.id, m.baby_id, b.name, m.name, m.dose, m.schedule, m.timezone
		 FROM medications m
		 JOIN babies b ON m.baby_id = b.id
		 WHERE m.active = 1 AND m.schedule IS NOT NULL AND m.schedule != ''`,
	)
	if err != nil {
		return nil, fmt.Errorf("query medications: %w", err)
	}
	defer rows.Close()

	var meds []activeMed
	for rows.Next() {
		var m activeMed
		if err := rows.Scan(&m.ID, &m.BabyID, &m.BabyName, &m.Name, &m.Dose, &m.Schedule, &m.Timezone); err != nil {
			return nil, fmt.Errorf("scan medication: %w", err)
		}
		meds = append(meds, m)
	}
	return meds, rows.Err()
}

// processMedication checks if a medication is due at the given time and
// sends notifications if needed.
func (s *Scheduler) processMedication(med activeMed, now time.Time) {
	scheduleTimes := parseSchedule(med.Schedule)
	if len(scheduleTimes) == 0 {
		return
	}

	loc := time.UTC
	if med.Timezone != nil && *med.Timezone != "" {
		parsed, err := time.LoadLocation(*med.Timezone)
		if err != nil {
			log.Printf("scheduler: invalid timezone %q for med %s: %v", *med.Timezone, med.ID, err)
			return
		}
		loc = parsed
	}

	sort.Strings(scheduleTimes)
	for i, st := range scheduleTimes {
		s.checkScheduleTime(med, scheduleTimes, i, st, now, loc)
	}
}

// checkScheduleTime checks if a single schedule time is due (at 0, +15, or
// +30 min offsets) and sends a notification if not suppressed.
// It checks both today and yesterday in the medication's local timezone to
// handle cross-midnight follow-ups (e.g., dose at 23:45, tick at 00:15).
func (s *Scheduler) checkScheduleTime(med activeMed, sortedTimes []string, slotIdx int, schedTime string, nowUTC time.Time, loc *time.Location) {
	nowLocal := nowUTC.In(loc)
	parts := strings.SplitN(schedTime, ":", 2)
	if len(parts) != 2 {
		return
	}
	var hour, minute int
	if _, err := fmt.Sscanf(parts[0], "%d", &hour); err != nil {
		return
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &minute); err != nil {
		return
	}

	// Check both yesterday and today to handle cross-midnight follow-ups
	days := []time.Time{
		nowLocal.AddDate(0, 0, -1),
		nowLocal,
	}

	for _, day := range days {
		scheduledLocal := time.Date(
			day.Year(), day.Month(), day.Day(),
			hour, minute, 0, 0, loc,
		)
		scheduledUTC := scheduledLocal.UTC()

		// Check offsets: 0, +15, +30 minutes
		offsets := []time.Duration{0, 15 * time.Minute, 30 * time.Minute}
		for _, offset := range offsets {
			triggerTime := scheduledUTC.Add(offset)
			// Check if "now" matches this trigger time (within the same minute)
			if nowUTC.Hour() == triggerTime.Hour() && nowUTC.Minute() == triggerTime.Minute() &&
				nowUTC.Year() == triggerTime.Year() && nowUTC.Month() == triggerTime.Month() &&
				nowUTC.Day() == triggerTime.Day() {
				// Check suppression using shared IsDoseCovered with midpoint coverage window
				dayStr := day.Format("2006-01-02")
				coverStart := store.SlotCoverageStart(sortedTimes, slotIdx, dayStr, loc).UTC()
				covered, err := store.IsDoseCovered(s.db, med.ID, scheduledUTC, coverStart)
				if err != nil {
					log.Printf("scheduler: check suppression for med %s: %v", med.ID, err)
					covered = true
				}
				if covered {
					return
				}
				s.sendNotifications(med, scheduledUTC)
				return // Only send once per schedule time per tick
			}
		}
	}
}

// sendNotifications sends push notifications to all subscribed parents of the baby.
func (s *Scheduler) sendNotifications(med activeMed, scheduledUTC time.Time) {
	// Get all parent user IDs for this baby
	rows, err := s.db.Query(
		"SELECT user_id FROM baby_parents WHERE baby_id = ?",
		med.BabyID,
	)
	if err != nil {
		log.Printf("scheduler: query parents for baby %s: %v", med.BabyID, err)
		return
	}
	defer rows.Close()

	var parentIDs []string
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			log.Printf("scheduler: scan parent: %v", err)
			continue
		}
		parentIDs = append(parentIDs, uid)
	}

	if len(parentIDs) == 0 {
		return
	}

	payload := buildMedPayload(med, scheduledUTC)

	// Send to all push subscriptions for each parent.
	// Clean up stale subscriptions (410 Gone / 404 Not Found from push service).
	for _, parentID := range parentIDs {
		subs, err := s.queryPushSubscriptions(parentID)
		if err != nil {
			log.Printf("scheduler: query subs for user %s: %v", parentID, err)
			continue
		}
		for _, sub := range subs {
			resp, err := s.pusher.Send(sub, payload)
			if err != nil {
				log.Printf("scheduler: push send to %s: %v", sub.Endpoint, err)
				continue
			}
			if resp != nil && (resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound) {
				log.Printf("scheduler: removing stale subscription %s (HTTP %d)", sub.Endpoint, resp.StatusCode)
				if delErr := store.DeletePushSubscriptionByEndpoint(s.db, sub.Endpoint); delErr != nil {
					log.Printf("scheduler: failed to delete stale subscription: %v", delErr)
				}
			}
		}
	}
}

// queryPushSubscriptions returns all push subscriptions for a user.
func (s *Scheduler) queryPushSubscriptions(userID string) ([]Subscription, error) {
	rows, err := s.db.Query(
		"SELECT endpoint, p256dh, auth FROM push_subscriptions WHERE user_id = ?",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []Subscription
	for rows.Next() {
		var sub Subscription
		if err := rows.Scan(&sub.Endpoint, &sub.P256dh, &sub.Auth); err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}

// buildMedPayload constructs the notification payload for a medication reminder.
// The body includes scheduled_time (UTC), medication_id, and medication name.
func buildMedPayload(med activeMed, scheduledUTC time.Time) Payload {
	scheduledStr := scheduledUTC.Format(utcTimeFormat)
	return Payload{
		Title: fmt.Sprintf("\U0001F48A %s \u2014 Time for dose", med.Name),
		Body:  fmt.Sprintf("%s for %s. Tap to log.", med.Dose, med.BabyName),
		URL:   fmt.Sprintf("/log/med?medication_id=%s", med.ID),
		Data: map[string]string{
			"scheduled_time": scheduledStr,
			"medication_id":  med.ID,
			"name":           med.Name,
		},
	}
}

// parseSchedule extracts schedule time strings from the JSON array stored
// in the medication's schedule column.
func parseSchedule(schedule *string) []string {
	if schedule == nil || *schedule == "" {
		return nil
	}
	var times []string
	if err := json.Unmarshal([]byte(*schedule), &times); err != nil {
		log.Printf("scheduler: parse schedule: %v", err)
		return nil
	}
	return times
}
