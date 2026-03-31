package model_test

import (
	"testing"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

func TestValidSex(t *testing.T) {
	t.Parallel()
	tests := []struct {
		val  string
		want bool
	}{
		{"male", true},
		{"female", true},
		{"", false},
		{"other", false},
		{"Male", false},
	}
	for _, tt := range tests {
		if got := model.ValidSex(tt.val); got != tt.want {
			t.Errorf("ValidSex(%q) = %v, want %v", tt.val, got, tt.want)
		}
	}
}

func TestValidFeedType(t *testing.T) {
	t.Parallel()
	valid := []string{"breast_milk", "formula", "fortified_breast_milk", "solid", "other"}
	for _, v := range valid {
		if !model.ValidFeedType(v) {
			t.Errorf("ValidFeedType(%q) = false, want true", v)
		}
	}
	invalid := []string{"", "juice", "BreastMilk"}
	for _, v := range invalid {
		if model.ValidFeedType(v) {
			t.Errorf("ValidFeedType(%q) = true, want false", v)
		}
	}
}

func TestValidUrineColor(t *testing.T) {
	t.Parallel()
	valid := []string{"clear", "pale_yellow", "dark_yellow", "amber", "brown"}
	for _, v := range valid {
		if !model.ValidUrineColor(v) {
			t.Errorf("ValidUrineColor(%q) = false, want true", v)
		}
	}
	if model.ValidUrineColor("red") {
		t.Error("ValidUrineColor(\"red\") = true, want false")
	}
}

func TestValidStoolColorLabel(t *testing.T) {
	t.Parallel()
	valid := []string{"white", "clay", "pale_yellow", "yellow", "light_green", "green", "brown"}
	for _, v := range valid {
		if !model.ValidStoolColorLabel(v) {
			t.Errorf("ValidStoolColorLabel(%q) = false, want true", v)
		}
	}
	if model.ValidStoolColorLabel("red") {
		t.Error("ValidStoolColorLabel(\"red\") = true, want false")
	}
}

func TestValidStoolConsistency(t *testing.T) {
	t.Parallel()
	valid := []string{"watery", "loose", "soft", "formed", "hard"}
	for _, v := range valid {
		if !model.ValidStoolConsistency(v) {
			t.Errorf("ValidStoolConsistency(%q) = false, want true", v)
		}
	}
	if model.ValidStoolConsistency("runny") {
		t.Error("ValidStoolConsistency(\"runny\") = true, want false")
	}
}

func TestValidStoolVolume(t *testing.T) {
	t.Parallel()
	valid := []string{"small", "medium", "large"}
	for _, v := range valid {
		if !model.ValidStoolVolume(v) {
			t.Errorf("ValidStoolVolume(%q) = false, want true", v)
		}
	}
	if model.ValidStoolVolume("tiny") {
		t.Error("ValidStoolVolume(\"tiny\") = true, want false")
	}
}

func TestValidMeasurementSource(t *testing.T) {
	t.Parallel()
	valid := []string{"home_scale", "clinic"}
	for _, v := range valid {
		if !model.ValidMeasurementSource(v) {
			t.Errorf("ValidMeasurementSource(%q) = false, want true", v)
		}
	}
	if model.ValidMeasurementSource("pharmacy") {
		t.Error("ValidMeasurementSource(\"pharmacy\") = true, want false")
	}
}

func TestValidTemperatureMethod(t *testing.T) {
	t.Parallel()
	valid := []string{"rectal", "axillary", "ear", "forehead"}
	for _, v := range valid {
		if !model.ValidTemperatureMethod(v) {
			t.Errorf("ValidTemperatureMethod(%q) = false, want true", v)
		}
	}
	if model.ValidTemperatureMethod("oral") {
		t.Error("ValidTemperatureMethod(\"oral\") = true, want false")
	}
}

func TestValidFirmness(t *testing.T) {
	t.Parallel()
	valid := []string{"soft", "firm", "distended"}
	for _, v := range valid {
		if !model.ValidFirmness(v) {
			t.Errorf("ValidFirmness(%q) = false, want true", v)
		}
	}
	if model.ValidFirmness("hard") {
		t.Error("ValidFirmness(\"hard\") = true, want false")
	}
}

func TestValidJaundiceLevel(t *testing.T) {
	t.Parallel()
	valid := []string{"none", "mild_face", "moderate_trunk", "severe_limbs_and_trunk"}
	for _, v := range valid {
		if !model.ValidJaundiceLevel(v) {
			t.Errorf("ValidJaundiceLevel(%q) = false, want true", v)
		}
	}
	if model.ValidJaundiceLevel("moderate") {
		t.Error("ValidJaundiceLevel(\"moderate\") = true, want false")
	}
}

func TestValidBruisingSizeEstimate(t *testing.T) {
	t.Parallel()
	valid := []string{"small_<1cm", "medium_1-3cm", "large_>3cm"}
	for _, v := range valid {
		if !model.ValidBruisingSizeEstimate(v) {
			t.Errorf("ValidBruisingSizeEstimate(%q) = false, want true", v)
		}
	}
	if model.ValidBruisingSizeEstimate("tiny") {
		t.Error("ValidBruisingSizeEstimate(\"tiny\") = true, want false")
	}
}

func TestValidMedFrequency(t *testing.T) {
	t.Parallel()
	valid := []string{"once_daily", "twice_daily", "three_times_daily", "as_needed", "custom", "every_x_days"}
	for _, v := range valid {
		if !model.ValidMedFrequency(v) {
			t.Errorf("ValidMedFrequency(%q) = false, want true", v)
		}
	}
	if model.ValidMedFrequency("weekly") {
		t.Error("ValidMedFrequency(\"weekly\") = true, want false")
	}
}

func TestValidNoteCategory(t *testing.T) {
	t.Parallel()
	valid := []string{"behavior", "sleep", "vomiting", "irritability", "skin", "other"}
	for _, v := range valid {
		if !model.ValidNoteCategory(v) {
			t.Errorf("ValidNoteCategory(%q) = false, want true", v)
		}
	}
	if model.ValidNoteCategory("feeding") {
		t.Error("ValidNoteCategory(\"feeding\") = true, want false")
	}
}

func TestValidFluidDirection(t *testing.T) {
	t.Parallel()
	valid := []string{"intake", "output"}
	for _, v := range valid {
		if !model.ValidFluidDirection(v) {
			t.Errorf("ValidFluidDirection(%q) = false, want true", v)
		}
	}
	if model.ValidFluidDirection("both") {
		t.Error("ValidFluidDirection(\"both\") = true, want false")
	}
}

func TestValidFluidSourceType(t *testing.T) {
	t.Parallel()
	valid := []string{"feeding", "urine", "stool"}
	for _, v := range valid {
		if !model.ValidFluidSourceType(v) {
			t.Errorf("ValidFluidSourceType(%q) = false, want true", v)
		}
	}
	if model.ValidFluidSourceType("iv") {
		t.Error("ValidFluidSourceType(\"iv\") = true, want false")
	}
}
