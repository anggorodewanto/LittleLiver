package model

import (
	"testing"
)

func TestCalculateCalories_FormulaWithCalDensity(t *testing.T) {
	t.Parallel()

	vol := 120.0
	calDen := 0.8 // kcal/mL
	feedType := "formula"

	result, err := CalculateCalories(feedType, &vol, &calDen, 67.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := 120.0 * 0.8 // 96.0
	if result.Calories == nil {
		t.Fatal("expected non-nil calories")
	}
	if diff := *result.Calories - expected; diff > 0.01 || diff < -0.01 {
		t.Errorf("expected calories ~%.2f, got %.2f", expected, *result.Calories)
	}
	if result.UsedDefaultCal {
		t.Error("expected used_default_cal=false for formula with cal_density")
	}
	if result.CalDensity == nil || *result.CalDensity != 0.8 {
		t.Errorf("expected cal_density=0.8, got %v", result.CalDensity)
	}
}

func TestCalculateCalories_BreastMilkDefaultsToStandardDensity(t *testing.T) {
	t.Parallel()

	vol := 100.0
	feedType := "breast_milk"

	result, err := CalculateCalories(feedType, &vol, nil, 67.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := 100.0 * DefaultCalDensity // ~67.6
	if result.Calories == nil {
		t.Fatal("expected non-nil calories")
	}
	if diff := *result.Calories - expected; diff > 0.01 || diff < -0.01 {
		t.Errorf("expected calories ~%.2f, got %.2f", expected, *result.Calories)
	}
	if result.UsedDefaultCal {
		t.Error("expected used_default_cal=false for breast_milk with volume (type-based default)")
	}
	if result.CalDensity == nil || *result.CalDensity != DefaultCalDensity {
		t.Errorf("expected cal_density=DefaultCalDensity (auto-applied), got %v", result.CalDensity)
	}
}

func TestCalculateCalories_FormulaDefaultsToStandardDensity(t *testing.T) {
	t.Parallel()

	vol := 60.0
	feedType := "formula"

	result, err := CalculateCalories(feedType, &vol, nil, 67.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := 60.0 * DefaultCalDensity
	if result.Calories == nil {
		t.Fatal("expected non-nil calories")
	}
	if diff := *result.Calories - expected; diff > 0.01 || diff < -0.01 {
		t.Errorf("expected calories ~%.2f, got %.2f", expected, *result.Calories)
	}
	if result.UsedDefaultCal {
		t.Error("expected used_default_cal=false")
	}
}

func TestCalculateCalories_BreastDirectUsesDefaultCalPerFeed(t *testing.T) {
	t.Parallel()

	feedType := "breast_milk"
	defaultCal := 67.0

	result, err := CalculateCalories(feedType, nil, nil, defaultCal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Calories == nil {
		t.Fatal("expected non-nil calories")
	}
	if *result.Calories != 67.0 {
		t.Errorf("expected calories=67.0, got %.2f", *result.Calories)
	}
	if !result.UsedDefaultCal {
		t.Error("expected used_default_cal=true for breast-direct")
	}
}

func TestCalculateCalories_UsedDefaultCalFlagSetCorrectly(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		feedType       string
		volumeMl       *float64
		calDensity     *float64
		defaultCal     float64
		wantUsedDefault bool
	}{
		{
			name:           "formula with cal_density",
			feedType:       "formula",
			volumeMl:       ptrFloat(120),
			calDensity:     ptrFloat(0.8),
			defaultCal:     67.0,
			wantUsedDefault: false,
		},
		{
			name:           "breast_milk with volume, no cal_density (type-default)",
			feedType:       "breast_milk",
			volumeMl:       ptrFloat(100),
			calDensity:     nil,
			defaultCal:     67.0,
			wantUsedDefault: false,
		},
		{
			name:           "breast-direct (no volume)",
			feedType:       "breast_milk",
			volumeMl:       nil,
			calDensity:     nil,
			defaultCal:     67.0,
			wantUsedDefault: true,
		},
		{
			name:           "formula with volume, no cal_density",
			feedType:       "formula",
			volumeMl:       ptrFloat(60),
			calDensity:     nil,
			defaultCal:     67.0,
			wantUsedDefault: false,
		},
		{
			name:           "solid with volume and cal_density",
			feedType:       "solid",
			volumeMl:       ptrFloat(50),
			calDensity:     ptrFloat(1.0),
			defaultCal:     67.0,
			wantUsedDefault: false,
		},
		{
			name:           "solid without volume or cal_density",
			feedType:       "solid",
			volumeMl:       nil,
			calDensity:     nil,
			defaultCal:     67.0,
			wantUsedDefault: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := CalculateCalories(tc.feedType, tc.volumeMl, tc.calDensity, tc.defaultCal)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.UsedDefaultCal != tc.wantUsedDefault {
				t.Errorf("expected used_default_cal=%v, got %v", tc.wantUsedDefault, result.UsedDefaultCal)
			}
		})
	}
}

func TestCalculateCalories_BreastDirectWithCalDensity_Returns400(t *testing.T) {
	t.Parallel()

	feedType := "breast_milk"
	calDen := 0.8

	_, err := CalculateCalories(feedType, nil, &calDen, 67.0)
	if err == nil {
		t.Fatal("expected error for breast-direct with cal_density, got nil")
	}
}

func TestCalculateCalories_OtherTypeNoVolumeNoCal(t *testing.T) {
	t.Parallel()

	result, err := CalculateCalories("other", nil, nil, 67.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Calories != nil {
		t.Errorf("expected nil calories for other type without volume, got %v", *result.Calories)
	}
	if result.UsedDefaultCal {
		t.Error("expected used_default_cal=false")
	}
}

func TestCalculateCalories_SolidWithBothFields(t *testing.T) {
	t.Parallel()

	vol := 50.0
	calDen := 1.0 // kcal/mL

	result, err := CalculateCalories("solid", &vol, &calDen, 67.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := 50.0 * 1.0 // 50.0
	if result.Calories == nil {
		t.Fatal("expected non-nil calories")
	}
	if diff := *result.Calories - expected; diff > 0.01 || diff < -0.01 {
		t.Errorf("expected calories ~%.2f, got %.2f", expected, *result.Calories)
	}
}

func TestCalculateCalories_NonBreastMilkNoVolumeWithCalDensity(t *testing.T) {
	t.Parallel()

	// cal_density provided but no volume for non-breast_milk type
	// For non breast_milk types without volume, cal_density with no volume
	// can't compute calories — return error (400)
	calDen := 0.8
	_, err := CalculateCalories("formula", nil, &calDen, 67.0)
	if err == nil {
		t.Fatal("expected error for no volume with cal_density, got nil")
	}
}

func ptrFloat(f float64) *float64 {
	return &f
}
