package who

import (
	"math"
	"testing"
)

// WHO reference values validated against WHO Child Growth Standards tables.
// Source: https://www.who.int/tools/child-growth-standards/standards/weight-for-age

func TestZScore_KnownValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sex       string
		ageDays   int
		weightKg  float64
		wantZ     float64
		tolerance float64
	}{
		{
			name:      "male birth median",
			sex:       "male",
			ageDays:   0,
			weightKg:  3.3464,
			wantZ:     0.0,
			tolerance: 0.05,
		},
		{
			name:      "female birth median",
			sex:       "female",
			ageDays:   0,
			weightKg:  3.2322,
			wantZ:     0.0,
			tolerance: 0.05,
		},
		{
			name:      "male 6 months (182 days) median",
			sex:       "male",
			ageDays:   182,
			weightKg:  7.926,
			wantZ:     0.0,
			tolerance: 0.05,
		},
		{
			name:      "female 12 months (365 days) median",
			sex:       "female",
			ageDays:   365,
			weightKg:  8.9462,
			wantZ:     0.0,
			tolerance: 0.05,
		},
		{
			name:      "male 24 months (730 days) median",
			sex:       "male",
			ageDays:   730,
			weightKg:  12.1482,
			wantZ:     0.0,
			tolerance: 0.05,
		},
		{
			name:      "male birth +1 SD",
			sex:       "male",
			ageDays:   0,
			weightKg:  3.8,
			wantZ:     1.0,
			tolerance: 0.15,
		},
		{
			name:      "male birth -2 SD",
			sex:       "male",
			ageDays:   0,
			weightKg:  2.5,
			wantZ:     -2.0,
			tolerance: 0.2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			z, err := ZScore(tt.sex, tt.ageDays, tt.weightKg)
			if err != nil {
				t.Fatalf("ZScore(%q, %d, %f) error: %v", tt.sex, tt.ageDays, tt.weightKg, err)
			}
			if math.Abs(z-tt.wantZ) > tt.tolerance {
				t.Errorf("ZScore(%q, %d, %f) = %f, want %f (±%f)",
					tt.sex, tt.ageDays, tt.weightKg, z, tt.wantZ, tt.tolerance)
			}
		})
	}
}

func TestPercentile_KnownValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		sex      string
		ageDays  int
		weightKg float64
		wantPct  float64
		tolPct   float64
	}{
		{
			name:     "male birth median is 50th percentile",
			sex:      "male",
			ageDays:  0,
			weightKg: 3.3464,
			wantPct:  50.0,
			tolPct:   2.0,
		},
		{
			name:     "female birth median is 50th percentile",
			sex:      "female",
			ageDays:  0,
			weightKg: 3.2322,
			wantPct:  50.0,
			tolPct:   2.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pct, err := Percentile(tt.sex, tt.ageDays, tt.weightKg)
			if err != nil {
				t.Fatalf("Percentile() error: %v", err)
			}
			if math.Abs(pct-tt.wantPct) > tt.tolPct {
				t.Errorf("Percentile() = %f, want %f (±%f)", pct, tt.wantPct, tt.tolPct)
			}
		})
	}
}

func TestZScore_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("day 0", func(t *testing.T) {
		t.Parallel()
		_, err := ZScore("male", 0, 3.3)
		if err != nil {
			t.Fatalf("ZScore at day 0 should not error: %v", err)
		}
	})

	t.Run("day 730", func(t *testing.T) {
		t.Parallel()
		_, err := ZScore("male", 730, 12.0)
		if err != nil {
			t.Fatalf("ZScore at day 730 should not error: %v", err)
		}
	})

	t.Run("invalid sex", func(t *testing.T) {
		t.Parallel()
		_, err := ZScore("other", 0, 3.3)
		if err == nil {
			t.Error("ZScore with invalid sex should return error")
		}
	})

	t.Run("negative age", func(t *testing.T) {
		t.Parallel()
		_, err := ZScore("male", -1, 3.3)
		if err == nil {
			t.Error("ZScore with negative age should return error")
		}
	})

	t.Run("age beyond 730", func(t *testing.T) {
		t.Parallel()
		_, err := ZScore("male", 731, 12.0)
		if err == nil {
			t.Error("ZScore with age > 730 should return error")
		}
	})

	t.Run("zero weight", func(t *testing.T) {
		t.Parallel()
		_, err := ZScore("male", 0, 0)
		if err == nil {
			t.Error("ZScore with zero weight should return error")
		}
	})

	t.Run("negative weight", func(t *testing.T) {
		t.Parallel()
		_, err := ZScore("male", 0, -1)
		if err == nil {
			t.Error("ZScore with negative weight should return error")
		}
	})
}

func TestPercentileCurves(t *testing.T) {
	t.Parallel()

	t.Run("male 0 to 365 days", func(t *testing.T) {
		t.Parallel()
		curves, err := PercentileCurves("male", 0, 365)
		if err != nil {
			t.Fatalf("PercentileCurves() error: %v", err)
		}
		if len(curves) != 5 {
			t.Fatalf("expected 5 curves, got %d", len(curves))
		}

		// Check curve labels
		expectedLabels := []float64{3, 15, 50, 85, 97}
		for i, curve := range curves {
			if curve.Percentile != expectedLabels[i] {
				t.Errorf("curve %d percentile = %f, want %f", i, curve.Percentile, expectedLabels[i])
			}
			// Each curve should have 366 points (0..365 inclusive)
			if len(curve.Points) != 366 {
				t.Errorf("curve %d has %d points, want 366", i, len(curve.Points))
			}
		}

		// 50th percentile at day 0 should be close to male birth median
		p50 := curves[2]
		if math.Abs(p50.Points[0].WeightKg-3.3464) > 0.01 {
			t.Errorf("50th percentile at day 0 = %f, want ~3.3464", p50.Points[0].WeightKg)
		}

		// Curves should be ordered: 3rd < 15th < 50th < 85th < 97th at any age
		for dayIdx := 0; dayIdx < len(curves[0].Points); dayIdx++ {
			for c := 1; c < len(curves); c++ {
				if curves[c].Points[dayIdx].WeightKg <= curves[c-1].Points[dayIdx].WeightKg {
					t.Errorf("at day %d, curve %d (%.0f%%) weight %.3f <= curve %d (%.0f%%) weight %.3f",
						dayIdx,
						c, curves[c].Percentile, curves[c].Points[dayIdx].WeightKg,
						c-1, curves[c-1].Percentile, curves[c-1].Points[dayIdx].WeightKg)
					break
				}
			}
		}
	})

	t.Run("female full range", func(t *testing.T) {
		t.Parallel()
		curves, err := PercentileCurves("female", 0, 730)
		if err != nil {
			t.Fatalf("PercentileCurves() error: %v", err)
		}
		if len(curves) != 5 {
			t.Fatalf("expected 5 curves, got %d", len(curves))
		}
		// 731 points (0..730)
		for i, curve := range curves {
			if len(curve.Points) != 731 {
				t.Errorf("curve %d has %d points, want 731", i, len(curve.Points))
			}
		}
	})

	t.Run("subset range", func(t *testing.T) {
		t.Parallel()
		curves, err := PercentileCurves("male", 100, 200)
		if err != nil {
			t.Fatalf("PercentileCurves() error: %v", err)
		}
		// 101 points (100..200)
		for i, curve := range curves {
			if len(curve.Points) != 101 {
				t.Errorf("curve %d has %d points, want 101", i, len(curve.Points))
			}
			if curve.Points[0].AgeDays != 100 {
				t.Errorf("first point age = %d, want 100", curve.Points[0].AgeDays)
			}
			if curve.Points[100].AgeDays != 200 {
				t.Errorf("last point age = %d, want 200", curve.Points[100].AgeDays)
			}
		}
	})

	t.Run("invalid sex", func(t *testing.T) {
		t.Parallel()
		_, err := PercentileCurves("other", 0, 365)
		if err == nil {
			t.Error("PercentileCurves with invalid sex should return error")
		}
	})

	t.Run("from > to", func(t *testing.T) {
		t.Parallel()
		_, err := PercentileCurves("male", 200, 100)
		if err == nil {
			t.Error("PercentileCurves with from > to should return error")
		}
	})

	t.Run("out of range", func(t *testing.T) {
		t.Parallel()
		_, err := PercentileCurves("male", 0, 731)
		if err == nil {
			t.Error("PercentileCurves with to > 730 should return error")
		}
	})
}

func TestCalcZScore_LZero(t *testing.T) {
	t.Parallel()
	// When L is 0, z = ln(y/M) / S
	z := calcZScore(10.0, 0, 10.0, 0.1)
	if math.Abs(z) > 0.001 {
		t.Errorf("calcZScore(10,0,10,0.1) = %f, want 0", z)
	}
	// Weight above median
	z = calcZScore(11.0, 0, 10.0, 0.1)
	if z <= 0 {
		t.Errorf("calcZScore(11,0,10,0.1) = %f, want > 0", z)
	}
}

func TestWeightFromZScore_LZero(t *testing.T) {
	t.Parallel()
	// When L is 0, y = M * exp(S*z)
	entry := lmsEntry{L: 0, M: 10.0, S: 0.1}
	w := weightFromZScore(0, entry)
	if math.Abs(w-10.0) > 0.001 {
		t.Errorf("weightFromZScore(0, L=0) = %f, want 10.0", w)
	}
	w = weightFromZScore(1, entry)
	if w <= 10.0 {
		t.Errorf("weightFromZScore(1, L=0) = %f, want > 10.0", w)
	}
}

func TestNormalCDF_ExtremeValues(t *testing.T) {
	t.Parallel()
	// Very negative z (< -8)
	if normalCDF(-10) != 0 {
		t.Errorf("normalCDF(-10) = %f, want 0", normalCDF(-10))
	}
	// Very positive z (> 8)
	if normalCDF(10) != 1 {
		t.Errorf("normalCDF(10) = %f, want 1", normalCDF(10))
	}
	// z=0 should be 0.5
	if math.Abs(normalCDF(0)-0.5) > 0.001 {
		t.Errorf("normalCDF(0) = %f, want 0.5", normalCDF(0))
	}
	// Positive z, not extreme
	cdf := normalCDF(1.0)
	if cdf < 0.84 || cdf > 0.88 {
		t.Errorf("normalCDF(1.0) = %f, want ~0.84-0.87", cdf)
	}
	// Negative z, not extreme
	cdf = normalCDF(-1.0)
	if cdf < 0.12 || cdf > 0.16 {
		t.Errorf("normalCDF(-1.0) = %f, want ~0.13-0.16", cdf)
	}
}


func TestPercentile_ErrorPassthrough(t *testing.T) {
	t.Parallel()
	// Verify error is passed through from ZScore
	_, err := Percentile("invalid", 0, 3.0)
	if err == nil {
		t.Error("Percentile with invalid sex should return error")
	}
}

func TestPercentileToZ_Extremes(t *testing.T) {
	t.Parallel()
	// 50th percentile -> z=0
	z := percentileToZ(50)
	if math.Abs(z) > 0.001 {
		t.Errorf("percentileToZ(50) = %f, want 0", z)
	}
	// 0th percentile
	z = percentileToZ(0)
	if z != -8 {
		t.Errorf("percentileToZ(0) = %f, want -8", z)
	}
	// 100th percentile
	z = percentileToZ(100)
	if z != 8 {
		t.Errorf("percentileToZ(100) = %f, want 8", z)
	}
	// Very low percentile (below pLow threshold)
	z = percentileToZ(1)
	if z >= -2 {
		t.Errorf("percentileToZ(1) = %f, want < -2", z)
	}
	// Very high percentile (above pHigh threshold)
	z = percentileToZ(99)
	if z <= 2 {
		t.Errorf("percentileToZ(99) = %f, want > 2", z)
	}
}

func TestPercentileCurves_NegativeFrom(t *testing.T) {
	t.Parallel()
	_, err := PercentileCurves("male", -1, 100)
	if err == nil {
		t.Error("PercentileCurves with negative from should return error")
	}
}

func TestDataLoaded(t *testing.T) {
	t.Parallel()
	// Verify that LMS data was loaded for both sexes
	if len(maleLMS) != 731 {
		t.Errorf("maleLMS has %d entries, want 731 (days 0-730)", len(maleLMS))
	}
	if len(femaleLMS) != 731 {
		t.Errorf("femaleLMS has %d entries, want 731 (days 0-730)", len(femaleLMS))
	}
}
