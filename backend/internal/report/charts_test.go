package report

import (
	"bytes"
	"image/png"
	"testing"

	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/who"
)

func TestRenderStoolChart_ValidPNG(t *testing.T) {
	t.Parallel()
	stools := []store.StoolColorSeriesEntry{
		{Timestamp: "2025-08-01 08:00:00", ColorScore: 3},
		{Timestamp: "2025-08-01 12:00:00", ColorScore: 4},
		{Timestamp: "2025-08-02 09:00:00", ColorScore: 2},
		{Timestamp: "2025-08-02 14:00:00", ColorScore: 5},
		{Timestamp: "2025-08-03 10:00:00", ColorScore: 3},
	}

	data, err := renderStoolChart(stools)
	if err != nil {
		t.Fatalf("renderStoolChart: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty PNG data")
	}

	// Verify it's a valid PNG
	_, err = png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("output is not valid PNG: %v", err)
	}
}

func TestRenderStoolChart_Empty(t *testing.T) {
	t.Parallel()
	data, err := renderStoolChart(nil)
	if err != nil {
		t.Fatalf("renderStoolChart with nil: %v", err)
	}
	if data != nil {
		t.Fatal("expected nil data for empty input")
	}
}

func TestRenderWeightChart_ValidPNG(t *testing.T) {
	t.Parallel()
	weights := []store.WeightSeriesEntry{
		{Timestamp: "2025-08-01 08:00:00", WeightKg: 4.5},
		{Timestamp: "2025-08-15 08:00:00", WeightKg: 4.8},
		{Timestamp: "2025-09-01 08:00:00", WeightKg: 5.2},
	}

	curves, err := who.PercentileCurves("female", 0, 90)
	if err != nil {
		t.Fatalf("PercentileCurves: %v", err)
	}

	data, err := renderWeightChart(weights, curves, "2025-06-15")
	if err != nil {
		t.Fatalf("renderWeightChart: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty PNG data")
	}

	_, err = png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("output is not valid PNG: %v", err)
	}
}

func TestRenderWeightChart_IncludesWHOPercentiles(t *testing.T) {
	t.Parallel()
	// Even with no weight data points, the WHO curves should still render
	curves, err := who.PercentileCurves("male", 0, 60)
	if err != nil {
		t.Fatalf("PercentileCurves: %v", err)
	}

	data, err := renderWeightChart(nil, curves, "2025-06-15")
	if err != nil {
		t.Fatalf("renderWeightChart: %v", err)
	}
	// With no weight data but WHO curves, we still get a chart
	if len(data) == 0 {
		t.Fatal("expected non-empty PNG for WHO-only chart")
	}
}

func TestRenderWeightChart_Empty(t *testing.T) {
	t.Parallel()
	// No weights AND no WHO curves -> nil
	data, err := renderWeightChart(nil, nil, "2025-06-15")
	if err != nil {
		t.Fatalf("renderWeightChart: %v", err)
	}
	if data != nil {
		t.Fatal("expected nil data for empty input")
	}
}

func TestRenderLabTrendsChart_ValidPNG(t *testing.T) {
	t.Parallel()
	unit := "mg/dL"
	trends := map[string][]store.LabTrendEntry{
		"Bilirubin": {
			{Timestamp: "2025-08-01 08:00:00", TestName: "Bilirubin", Value: "2.5", Unit: &unit},
			{Timestamp: "2025-08-15 08:00:00", TestName: "Bilirubin", Value: "2.1", Unit: &unit},
		},
		"ALT": {
			{Timestamp: "2025-08-01 08:00:00", TestName: "ALT", Value: "45", Unit: &unit},
			{Timestamp: "2025-08-15 08:00:00", TestName: "ALT", Value: "38", Unit: &unit},
		},
	}

	data, err := renderLabTrendsChart(trends)
	if err != nil {
		t.Fatalf("renderLabTrendsChart: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty PNG data")
	}

	_, err = png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("output is not valid PNG: %v", err)
	}
}

func TestRenderLabTrendsChart_GroupsByTestName(t *testing.T) {
	t.Parallel()
	unit := "U/L"
	trends := map[string][]store.LabTrendEntry{
		"ALT": {
			{Timestamp: "2025-08-01 08:00:00", TestName: "ALT", Value: "45", Unit: &unit},
		},
		"AST": {
			{Timestamp: "2025-08-01 08:00:00", TestName: "AST", Value: "30", Unit: &unit},
		},
		"GGT": {
			{Timestamp: "2025-08-01 08:00:00", TestName: "GGT", Value: "120", Unit: &unit},
		},
	}

	data, err := renderLabTrendsChart(trends)
	if err != nil {
		t.Fatalf("renderLabTrendsChart: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty PNG data for multiple test names")
	}
}

func TestRenderLabTrendsChart_Empty(t *testing.T) {
	t.Parallel()
	data, err := renderLabTrendsChart(nil)
	if err != nil {
		t.Fatalf("renderLabTrendsChart with nil: %v", err)
	}
	if data != nil {
		t.Fatal("expected nil data for empty input")
	}
}

func TestRenderLabTrendsChart_NonNumericValues(t *testing.T) {
	t.Parallel()
	// Lab values that aren't parseable as float should be skipped
	trends := map[string][]store.LabTrendEntry{
		"BloodType": {
			{Timestamp: "2025-08-01 08:00:00", TestName: "BloodType", Value: "A+"},
		},
	}

	data, err := renderLabTrendsChart(trends)
	if err != nil {
		t.Fatalf("renderLabTrendsChart with non-numeric: %v", err)
	}
	// Non-numeric values result in no plottable data
	if data != nil {
		t.Fatal("expected nil data for non-numeric values only")
	}
}
