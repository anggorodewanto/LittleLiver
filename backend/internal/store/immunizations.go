package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/immunization"
	"github.com/ablankz/LittleLiver/backend/internal/model"
)

// Immunization status values for a computed schedule slot.
const (
	ImmunizationStatusDone     = "done"
	ImmunizationStatusDue      = "due"
	ImmunizationStatusUpcoming = "upcoming"
)

const immunizationColumns = `id, baby_id, logged_by, updated_by, vaccine_code,
	vaccine_name, dose_number, administered_date, provider, lot_number, notes,
	created_at, updated_at`

// ImmunizationSlot is one computed schedule slot for a baby: a reference dose
// (or an off-schedule administered record) plus its done/due/upcoming status.
type ImmunizationSlot struct {
	Code             string  `json:"code"`
	Name             string  `json:"name"`
	DoseNumber       int     `json:"dose_number"`
	DoseLabel        string  `json:"dose_label"`
	AgeMonths        int     `json:"age_months"`
	AgeLabel         string  `json:"age_label"`
	Mandatory        bool    `json:"mandatory"`
	Status           string  `json:"status"`
	DueDate          string  `json:"due_date,omitempty"`
	AdministeredDate *string `json:"administered_date,omitempty"`
	RecordID         *string `json:"record_id,omitempty"`
	OffSchedule      bool    `json:"off_schedule"`
}

func scanImmunization(s scanner) (*model.Immunization, error) {
	var m model.Immunization
	var updatedBy, provider, lotNumber, notes sql.NullString
	var doseNumber sql.NullInt64
	var createdStr, updatedStr string

	err := s.Scan(
		&m.ID, &m.BabyID, &m.LoggedBy, &updatedBy, &m.VaccineCode,
		&m.VaccineName, &doseNumber, &m.AdministeredDate, &provider, &lotNumber, &notes,
		&createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}

	m.UpdatedBy = nullStr(updatedBy)
	m.DoseNumber = nullInt(doseNumber)
	m.Provider = nullStr(provider)
	m.LotNumber = nullStr(lotNumber)
	m.Notes = nullStr(notes)

	m.CreatedAt, err = ParseTime(createdStr)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	m.UpdatedAt, err = ParseTime(updatedStr)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at: %w", err)
	}

	return &m, nil
}

// CreateImmunization inserts a new immunization record and returns it.
func CreateImmunization(db *sql.DB, babyID, loggedBy, vaccineCode, vaccineName string, doseNumber *int, administeredDate string, provider, lotNumber, notes *string) (*model.Immunization, error) {
	id := model.NewULID()

	_, err := db.Exec(
		`INSERT INTO immunizations (id, baby_id, logged_by, vaccine_code, vaccine_name, dose_number, administered_date, provider, lot_number, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, babyID, loggedBy, vaccineCode, vaccineName, doseNumber, administeredDate, provider, lotNumber, notes,
	)
	if err != nil {
		return nil, fmt.Errorf("create immunization: %w", err)
	}

	return GetImmunizationByID(db, babyID, id)
}

// GetImmunizationByID retrieves an immunization by its ID, scoped to the baby.
func GetImmunizationByID(db *sql.DB, babyID, id string) (*model.Immunization, error) {
	row := db.QueryRow(
		`SELECT `+immunizationColumns+` FROM immunizations WHERE id = ? AND baby_id = ?`,
		id, babyID,
	)
	return scanImmunization(row)
}

// ListImmunizations returns all immunization records for a baby, newest first.
func ListImmunizations(db *sql.DB, babyID string) ([]model.Immunization, error) {
	rows, err := db.Query(
		`SELECT `+immunizationColumns+` FROM immunizations WHERE baby_id = ? ORDER BY administered_date DESC, id DESC`,
		babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("list immunizations: %w", err)
	}
	defer rows.Close()

	records := make([]model.Immunization, 0)
	for rows.Next() {
		rec, err := scanImmunization(rows)
		if err != nil {
			return nil, fmt.Errorf("scan immunization: %w", err)
		}
		records = append(records, *rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return records, nil
}

// UpdateImmunization updates an immunization record.
func UpdateImmunization(db *sql.DB, babyID, id, updatedBy, vaccineCode, vaccineName string, doseNumber *int, administeredDate string, provider, lotNumber, notes *string) (*model.Immunization, error) {
	res, err := db.Exec(
		`UPDATE immunizations SET
			updated_by = ?, vaccine_code = ?, vaccine_name = ?, dose_number = ?,
			administered_date = ?, provider = ?, lot_number = ?, notes = ?,
			updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND baby_id = ?`,
		updatedBy, vaccineCode, vaccineName, doseNumber,
		administeredDate, provider, lotNumber, notes,
		id, babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("update immunization: %w", err)
	}
	if err := checkRowsAffected(res, "update immunization"); err != nil {
		return nil, err
	}

	return GetImmunizationByID(db, babyID, id)
}

// DeleteImmunization hard-deletes an immunization record.
func DeleteImmunization(db *sql.DB, babyID, id string) error {
	return deleteByID(db, "immunizations", babyID, id)
}

// immKey identifies a dose slot by vaccine code + dose number.
type immKey struct {
	code string
	dose int
}

func recordDose(r *model.Immunization) int {
	if r.DoseNumber != nil {
		return *r.DoseNumber
	}
	return 0
}

// BuildImmunizationSchedule overlays administered records onto the reference
// schedule to produce per-dose status. It is pure (no DB) for easy testing.
// "today" is evaluated in loc; a dose is done if a record matches its
// (code, dose_number), due if its due date is on/before today, otherwise
// upcoming. Administered records not matching any reference slot are appended
// as off-schedule done entries so the completed list stays comprehensive.
func BuildImmunizationSchedule(entries []immunization.ScheduleEntry, records []model.Immunization, dob, asOf time.Time, loc *time.Location) []ImmunizationSlot {
	if loc == nil {
		loc = time.UTC
	}
	today := asOf.In(loc).Format(model.DateFormat)

	recByKey := make(map[immKey]*model.Immunization, len(records))
	for i := range records {
		k := immKey{records[i].VaccineCode, recordDose(&records[i])}
		if _, exists := recByKey[k]; !exists {
			recByKey[k] = &records[i]
		}
	}

	used := make(map[immKey]bool, len(records))
	slots := make([]ImmunizationSlot, 0, len(entries)+len(records))

	for _, e := range entries {
		slot := ImmunizationSlot{
			Code:       e.Code,
			Name:       e.Name,
			DoseNumber: e.DoseNumber,
			DoseLabel:  e.DoseLabel,
			AgeMonths:  e.AgeMonths,
			AgeLabel:   e.AgeLabel,
			Mandatory:  e.Mandatory,
		}
		if !dob.IsZero() {
			slot.DueDate = dob.AddDate(0, e.AgeMonths, 0).Format(model.DateFormat)
		}

		k := immKey{e.Code, e.DoseNumber}
		if rec, ok := recByKey[k]; ok {
			used[k] = true
			ad := rec.AdministeredDate
			id := rec.ID
			slot.Status = ImmunizationStatusDone
			slot.AdministeredDate = &ad
			slot.RecordID = &id
		} else if slot.DueDate != "" && slot.DueDate <= today {
			slot.Status = ImmunizationStatusDue
		} else {
			slot.Status = ImmunizationStatusUpcoming
		}

		slots = append(slots, slot)
	}

	for i := range records {
		k := immKey{records[i].VaccineCode, recordDose(&records[i])}
		if used[k] {
			continue
		}
		ad := records[i].AdministeredDate
		id := records[i].ID
		slots = append(slots, ImmunizationSlot{
			Code:             records[i].VaccineCode,
			Name:             records[i].VaccineName,
			DoseNumber:       recordDose(&records[i]),
			Status:           ImmunizationStatusDone,
			AdministeredDate: &ad,
			RecordID:         &id,
			OffSchedule:      true,
		})
	}

	return slots
}

// GetImmunizationSchedule loads a baby's records and computes the schedule
// against the baby's date of birth and the IDAI reference schedule.
func GetImmunizationSchedule(db *sql.DB, baby *model.Baby, asOf time.Time, loc *time.Location) ([]ImmunizationSlot, error) {
	records, err := ListImmunizations(db, baby.ID)
	if err != nil {
		return nil, err
	}
	return BuildImmunizationSchedule(immunization.Schedule(), records, baby.DateOfBirth, asOf, loc), nil
}
