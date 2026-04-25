package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

const carePlanColumns = `id, baby_id, logged_by, updated_by, name, notes, timezone, active, created_at, updated_at`
const carePlanPhaseColumns = `id, care_plan_id, seq, label, start_date, ends_on, notes, created_at, updated_at`

// scanCarePlan scans a single care_plans row from the given scanner.
func scanCarePlan(s scanner) (*model.CarePlan, error) {
	var p model.CarePlan
	var updatedBy, notes sql.NullString
	var createdStr, updatedStr string
	var active bool

	err := s.Scan(
		&p.ID, &p.BabyID, &p.LoggedBy, &updatedBy,
		&p.Name, &notes, &p.Timezone, &active, &createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}

	p.Active = active
	p.UpdatedBy = nullStr(updatedBy)
	p.Notes = nullStr(notes)

	p.CreatedAt, err = ParseTime(createdStr)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	p.UpdatedAt, err = ParseTime(updatedStr)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at: %w", err)
	}

	return &p, nil
}

// scanCarePlanPhase scans a single care_plan_phases row.
func scanCarePlanPhase(s scanner) (*model.CarePlanPhase, error) {
	var p model.CarePlanPhase
	var endsOn, notes sql.NullString
	var createdStr, updatedStr string

	err := s.Scan(
		&p.ID, &p.CarePlanID, &p.Seq, &p.Label, &p.StartDate,
		&endsOn, &notes, &createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}

	p.EndsOn = nullStr(endsOn)
	p.Notes = nullStr(notes)
	p.CreatedAt, err = ParseTime(createdStr)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	p.UpdatedAt, err = ParseTime(updatedStr)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at: %w", err)
	}

	return &p, nil
}

// CreateCarePlan inserts a new care plan and returns it.
func CreateCarePlan(db *sql.DB, babyID, loggedBy, name string, notes *string, timezone string) (*model.CarePlan, error) {
	id := model.NewULID()

	_, err := db.Exec(
		`INSERT INTO care_plans (id, baby_id, logged_by, name, notes, timezone)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id, babyID, loggedBy, name, notes, timezone,
	)
	if err != nil {
		return nil, fmt.Errorf("create care plan: %w", err)
	}

	return GetCarePlanByID(db, babyID, id)
}

// GetCarePlanByID retrieves a care plan by its ID, scoped to the given baby.
func GetCarePlanByID(db *sql.DB, babyID, planID string) (*model.CarePlan, error) {
	row := db.QueryRow(
		`SELECT `+carePlanColumns+` FROM care_plans WHERE id = ? AND baby_id = ?`,
		planID, babyID,
	)
	return scanCarePlan(row)
}

// ListCarePlans returns all plans (active + inactive) for a baby, newest first.
func ListCarePlans(db *sql.DB, babyID string) ([]model.CarePlan, error) {
	rows, err := db.Query(
		`SELECT `+carePlanColumns+` FROM care_plans WHERE baby_id = ? ORDER BY created_at DESC, id DESC`,
		babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("list care plans: %w", err)
	}
	defer rows.Close()

	plans := make([]model.CarePlan, 0)
	for rows.Next() {
		p, err := scanCarePlan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan care plan: %w", err)
		}
		plans = append(plans, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return plans, nil
}

// UpdateCarePlan mutates a plan. Pass active=nil to leave unchanged; pass
// timezone="" to preserve the existing tz (mirrors the Medication pattern).
func UpdateCarePlan(db *sql.DB, babyID, planID, updatedBy, name string, notes *string, timezone string, active *bool) (*model.CarePlan, error) {
	existing, err := GetCarePlanByID(db, babyID, planID)
	if err != nil {
		return nil, err
	}

	tz := existing.Timezone
	if timezone != "" {
		tz = timezone
	}
	activeVal := existing.Active
	if active != nil {
		activeVal = *active
	}

	res, err := db.Exec(
		`UPDATE care_plans SET
			updated_by = ?, name = ?, notes = ?, timezone = ?, active = ?,
			updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND baby_id = ?`,
		updatedBy, name, notes, tz, activeVal, planID, babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("update care plan: %w", err)
	}
	if err := checkRowsAffected(res, "update care plan"); err != nil {
		return nil, err
	}

	return GetCarePlanByID(db, babyID, planID)
}

// DeleteCarePlan hard-deletes a plan; FK CASCADE drops phases + audit rows.
func DeleteCarePlan(db *sql.DB, babyID, planID string) error {
	res, err := db.Exec(
		`DELETE FROM care_plans WHERE id = ? AND baby_id = ?`,
		planID, babyID,
	)
	if err != nil {
		return fmt.Errorf("delete care plan: %w", err)
	}
	return checkRowsAffected(res, "delete care plan")
}

// ReplaceCarePlanPhases atomically replaces the full phase list for a plan.
// Validates contiguous seq 1..N and strictly increasing start_date in YYYY-MM-DD.
func ReplaceCarePlanPhases(db *sql.DB, planID string, phases []model.CarePlanPhase) ([]model.CarePlanPhase, error) {
	if len(phases) == 0 {
		return nil, errors.New("replace care plan phases: at least one phase required")
	}

	var prevDate time.Time
	for i, ph := range phases {
		if ph.Seq != i+1 {
			return nil, fmt.Errorf("replace care plan phases: phase %d has seq=%d, expected %d", i, ph.Seq, i+1)
		}
		if ph.Label == "" {
			return nil, fmt.Errorf("replace care plan phases: phase %d has empty label", i+1)
		}
		d, err := time.Parse(model.DateFormat, ph.StartDate)
		if err != nil {
			return nil, fmt.Errorf("replace care plan phases: phase %d bad start_date %q: %w", i+1, ph.StartDate, err)
		}
		if i > 0 && !d.After(prevDate) {
			return nil, fmt.Errorf("replace care plan phases: phase %d start_date %q not after phase %d", i+1, ph.StartDate, i)
		}
		if ph.EndsOn != nil {
			if _, err := time.Parse(model.DateFormat, *ph.EndsOn); err != nil {
				return nil, fmt.Errorf("replace care plan phases: phase %d bad ends_on %q: %w", i+1, *ph.EndsOn, err)
			}
		}
		prevDate = d
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM care_plan_phases WHERE care_plan_id = ?`, planID); err != nil {
		return nil, fmt.Errorf("clear phases: %w", err)
	}

	for _, ph := range phases {
		id := model.NewULID()
		if _, err := tx.Exec(
			`INSERT INTO care_plan_phases (id, care_plan_id, seq, label, start_date, ends_on, notes)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			id, planID, ph.Seq, ph.Label, ph.StartDate, ph.EndsOn, ph.Notes,
		); err != nil {
			return nil, fmt.Errorf("insert phase %d: %w", ph.Seq, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit phase replace: %w", err)
	}

	return ListCarePlanPhases(db, planID)
}

// GetCurrentCarePlanPhase returns the phase active at asOf, evaluated in loc
// (the plan's timezone). Returns (nil, nil) when asOf is before the first
// phase or after a closed final phase.
func GetCurrentCarePlanPhase(db *sql.DB, planID string, asOf time.Time, loc *time.Location) (*model.CarePlanPhase, error) {
	today := asOf.In(loc).Format(model.DateFormat)

	row := db.QueryRow(
		`SELECT `+carePlanPhaseColumns+`
		 FROM care_plan_phases
		 WHERE care_plan_id = ? AND start_date <= ?
		 ORDER BY start_date DESC, seq DESC
		 LIMIT 1`,
		planID, today,
	)
	phase, err := scanCarePlanPhase(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get current phase: %w", err)
	}

	if phase.EndsOn != nil && *phase.EndsOn < today {
		return nil, nil
	}
	return phase, nil
}

// RecordPhaseNotification inserts an audit row for (phase_id, kind). Returns
// true when this call was the one to insert; false when a row already existed.
// The scheduler uses this to gate exactly-once Web Push delivery.
func RecordPhaseNotification(db *sql.DB, phaseID, kind string) (bool, error) {
	res, err := db.Exec(
		`INSERT OR IGNORE INTO care_plan_phase_notifications (phase_id, kind) VALUES (?, ?)`,
		phaseID, kind,
	)
	if err != nil {
		return false, fmt.Errorf("record phase notification: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}
	return affected == 1, nil
}

// CurrentCarePlanPhase is the dashboard projection of a plan's currently
// active phase, with optional countdown to the next switch.
type CurrentCarePlanPhase struct {
	PlanID        string  `json:"plan_id"`
	PlanName      string  `json:"plan_name"`
	PhaseID       string  `json:"phase_id"`
	Label         string  `json:"label"`
	EndsOn        *string `json:"ends_on,omitempty"`
	DaysRemaining *int    `json:"days_remaining,omitempty"`
}

// GetCurrentCarePlanPhasesForBaby resolves the active phase per active plan
// for a baby. Each plan is evaluated in its own timezone — dashboard tz is
// not used here so that a parent in one tz sees the same phase boundaries
// as the plan was authored in.
func GetCurrentCarePlanPhasesForBaby(db *sql.DB, babyID string, asOf time.Time) ([]CurrentCarePlanPhase, error) {
	plans, err := ListCarePlans(db, babyID)
	if err != nil {
		return nil, err
	}

	out := make([]CurrentCarePlanPhase, 0)
	for _, plan := range plans {
		if !plan.Active {
			continue
		}
		loc, err := time.LoadLocation(plan.Timezone)
		if err != nil {
			loc = time.UTC
		}
		current, err := GetCurrentCarePlanPhase(db, plan.ID, asOf, loc)
		if err != nil {
			return nil, fmt.Errorf("current phase for plan %s: %w", plan.ID, err)
		}
		if current == nil {
			continue
		}

		endsOn := current.EndsOn
		if endsOn == nil {
			phases, err := ListCarePlanPhases(db, plan.ID)
			if err != nil {
				return nil, fmt.Errorf("phases for plan %s: %w", plan.ID, err)
			}
			for _, p := range phases {
				if p.Seq == current.Seq+1 {
					next := p.StartDate
					endsOn = &next
					break
				}
			}
		}

		item := CurrentCarePlanPhase{
			PlanID:   plan.ID,
			PlanName: plan.Name,
			PhaseID:  current.ID,
			Label:    current.Label,
			EndsOn:   endsOn,
		}
		if endsOn != nil {
			today := asOf.In(loc).Format(model.DateFormat)
			todayT, errToday := time.Parse(model.DateFormat, today)
			endT, errEnd := time.Parse(model.DateFormat, *endsOn)
			if errToday == nil && errEnd == nil {
				days := int(endT.Sub(todayT).Hours() / 24)
				if days < 0 {
					days = 0
				}
				item.DaysRemaining = &days
			}
		}

		out = append(out, item)
	}
	return out, nil
}

// ListCarePlanPhases returns phases for a plan ordered by seq.
func ListCarePlanPhases(db *sql.DB, planID string) ([]model.CarePlanPhase, error) {
	rows, err := db.Query(
		`SELECT `+carePlanPhaseColumns+` FROM care_plan_phases WHERE care_plan_id = ? ORDER BY seq ASC`,
		planID,
	)
	if err != nil {
		return nil, fmt.Errorf("list care plan phases: %w", err)
	}
	defer rows.Close()

	phases := make([]model.CarePlanPhase, 0)
	for rows.Next() {
		ph, err := scanCarePlanPhase(rows)
		if err != nil {
			return nil, fmt.Errorf("scan phase: %w", err)
		}
		phases = append(phases, *ph)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return phases, nil
}
