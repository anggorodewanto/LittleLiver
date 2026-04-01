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

func TestCategorizeLabTrends_KnownTests(t *testing.T) {
	t.Parallel()
	unit := "U/L"
	trends := map[string][]store.LabTrendEntry{
		"SGOT/AST":  {{TestName: "SGOT/AST", Value: "128", Unit: &unit}},
		"Hemoglobin": {{TestName: "Hemoglobin", Value: "9.6", Unit: &unit}},
		"Natrium":    {{TestName: "Natrium", Value: "131", Unit: &unit}},
	}

	result := categorizeLabTrends(trends)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if _, ok := result["Liver Function"]["SGOT/AST"]; !ok {
		t.Error("SGOT/AST should be in Liver Function")
	}
	if _, ok := result["Hematology"]["Hemoglobin"]; !ok {
		t.Error("Hemoglobin should be in Hematology")
	}
	if _, ok := result["Electrolytes"]["Natrium"]; !ok {
		t.Error("Natrium should be in Electrolytes")
	}
}

func TestCategorizeLabTrends_CaseInsensitive(t *testing.T) {
	t.Parallel()
	unit := "g/dL"
	trends := map[string][]store.LabTrendEntry{
		"albumin": {{TestName: "albumin", Value: "3.46", Unit: &unit}},
	}

	result := categorizeLabTrends(trends)
	if _, ok := result["Liver Function"]["albumin"]; !ok {
		t.Error("lowercase 'albumin' should match Liver Function category")
	}
}

func TestCategorizeLabTrends_UnknownTest(t *testing.T) {
	t.Parallel()
	trends := map[string][]store.LabTrendEntry{
		"BloodType": {{TestName: "BloodType", Value: "A+"}},
	}

	result := categorizeLabTrends(trends)
	if _, ok := result["Other"]["BloodType"]; !ok {
		t.Error("unknown test should be in Other category")
	}
}

func TestCategorizeLabTrends_Empty(t *testing.T) {
	t.Parallel()
	result := categorizeLabTrends(nil)
	if result != nil {
		t.Error("expected nil for empty input")
	}
}

func TestRenderLabTrendCharts_SeparateCategories(t *testing.T) {
	t.Parallel()
	unitUL := "U/L"
	unitGdL := "g/dL"
	trends := map[string][]store.LabTrendEntry{
		"SGOT/AST": {
			{Timestamp: "2025-08-01 08:00:00", TestName: "SGOT/AST", Value: "128", Unit: &unitUL},
			{Timestamp: "2025-08-15 08:00:00", TestName: "SGOT/AST", Value: "100", Unit: &unitUL},
		},
		"Hemoglobin": {
			{Timestamp: "2025-08-01 08:00:00", TestName: "Hemoglobin", Value: "9.6", Unit: &unitGdL},
			{Timestamp: "2025-08-15 08:00:00", TestName: "Hemoglobin", Value: "10.0", Unit: &unitGdL},
		},
	}

	charts, err := renderLabTrendCharts(trends)
	if err != nil {
		t.Fatalf("renderLabTrendCharts: %v", err)
	}
	if len(charts) != 2 {
		t.Fatalf("expected 2 category charts, got %d", len(charts))
	}
	if _, ok := charts["Liver Function"]; !ok {
		t.Error("expected Liver Function chart")
	}
	if _, ok := charts["Hematology"]; !ok {
		t.Error("expected Hematology chart")
	}
}

func TestRenderLabTrendCharts_Empty(t *testing.T) {
	t.Parallel()
	charts, err := renderLabTrendCharts(nil)
	if err != nil {
		t.Fatalf("renderLabTrendCharts with nil: %v", err)
	}
	if charts != nil {
		t.Fatal("expected nil for empty input")
	}
}

func TestRenderLabTrendCharts_NonNumericSkipped(t *testing.T) {
	t.Parallel()
	trends := map[string][]store.LabTrendEntry{
		"BloodType": {
			{Timestamp: "2025-08-01 08:00:00", TestName: "BloodType", Value: "A+"},
		},
	}

	charts, err := renderLabTrendCharts(trends)
	if err != nil {
		t.Fatalf("renderLabTrendCharts: %v", err)
	}
	if charts != nil {
		t.Fatal("expected nil when all values non-numeric")
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
