package notify

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
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

	for _, st := range scheduleTimes {
		s.checkScheduleTime(med, st, now, loc)
	}
}

// checkScheduleTime checks if a single schedule time is due (at 0, +15, or
// +30 min offsets) and sends a notification if not suppressed.
func (s *Scheduler) checkScheduleTime(med activeMed, schedTime string, nowUTC time.Time, loc *time.Location) {
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

	// Compute the scheduled time in the medication's local timezone for today
	scheduledLocal := time.Date(
		nowLocal.Year(), nowLocal.Month(), nowLocal.Day(),
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
			// Check suppression: look for med_log within +/-30 min of scheduled time
			if s.isDoseSuppressed(med.ID, med.BabyID, scheduledUTC) {
				return
			}
			s.sendNotifications(med, scheduledUTC)
			return // Only send once per schedule time per tick
		}
	}
}

// isDoseSuppressed checks if a med_log exists for this medication within
// +/-30 minutes of the original scheduled time.
func (s *Scheduler) isDoseSuppressed(medID, babyID string, scheduledUTC time.Time) bool {
	windowStart := scheduledUTC.Add(-30 * time.Minute).Format(utcTimeFormat)
	windowEnd := scheduledUTC.Add(30 * time.Minute).Format(utcTimeFormat)

	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM med_logs
		 WHERE medication_id = ? AND baby_id = ?
		   AND (
		     (skipped = 0 AND given_at >= ? AND given_at <= ?)
		     OR
		     (skipped = 1 AND created_at >= ? AND created_at <= ?)
		   )`,
		medID, babyID, windowStart, windowEnd, windowStart, windowEnd,
	).Scan(&count)
	if err != nil {
		log.Printf("scheduler: check suppression for med %s: %v", medID, err)
		return false
	}
	return count > 0
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

	// Send to all push subscriptions for each parent
	for _, parentID := range parentIDs {
		subs, err := s.queryPushSubscriptions(parentID)
		if err != nil {
			log.Printf("scheduler: query subs for user %s: %v", parentID, err)
			continue
		}
		for _, sub := range subs {
			if _, err := s.pusher.Send(sub, payload); err != nil {
				log.Printf("scheduler: push send to %s: %v", sub.Endpoint, err)
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
