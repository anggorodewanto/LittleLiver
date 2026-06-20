package immunization

import "testing"

func TestSchedule_NotEmpty(t *testing.T) {
	if len(Schedule()) == 0 {
		t.Fatal("expected non-empty schedule")
	}
}

func TestSchedule_ReturnsCopy(t *testing.T) {
	s := Schedule()
	if len(s) == 0 {
		t.Fatal("expected non-empty schedule")
	}
	s[0].Name = "MUTATED"
	if Schedule()[0].Name == "MUTATED" {
		t.Error("Schedule() must return a copy, internal data was mutated")
	}
}

func TestSchedule_UniqueCodeDose(t *testing.T) {
	seen := map[string]bool{}
	for _, e := range Schedule() {
		key := e.Code + "/" + string(rune('0'+e.DoseNumber))
		if seen[key] {
			t.Errorf("duplicate (code, dose) slot: %s dose %d", e.Code, e.DoseNumber)
		}
		seen[key] = true
	}
}

func TestSchedule_FieldsValid(t *testing.T) {
	for _, e := range Schedule() {
		if e.Code == "" {
			t.Errorf("entry with empty code: %+v", e)
		}
		if e.Name == "" {
			t.Errorf("entry %s with empty name", e.Code)
		}
		if e.DoseNumber < 1 {
			t.Errorf("entry %s has invalid dose_number %d", e.Code, e.DoseNumber)
		}
		if e.AgeMonths < 0 {
			t.Errorf("entry %s has negative age_months %d", e.Code, e.AgeMonths)
		}
		if e.AgeLabel == "" {
			t.Errorf("entry %s/%d has empty age_label", e.Code, e.DoseNumber)
		}
	}
}

func TestSchedule_HasMandatoryAndOptional(t *testing.T) {
	var mandatory, optional bool
	for _, e := range Schedule() {
		if e.Mandatory {
			mandatory = true
		} else {
			optional = true
		}
	}
	if !mandatory {
		t.Error("expected at least one mandatory vaccine")
	}
	if !optional {
		t.Error("expected at least one optional vaccine")
	}
}
