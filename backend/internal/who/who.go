// Package who provides WHO Child Growth Standards percentile calculations.
//
// It embeds WHO weight-for-age LMS tables (0-24 months, male + female)
// and provides functions to compute z-scores, percentiles, and
// percentile curves for charting.
package who

import (
	"embed"
	"encoding/csv"
	"errors"
	"fmt"
	"math"
	"strconv"
	"bytes"
)

//go:embed data/wfa_boys_lms.csv
var boysCSV embed.FS

//go:embed data/wfa_girls_lms.csv
var girlsCSV embed.FS

// lmsEntry holds the L, M, S parameters for a single age in days.
type lmsEntry struct {
	L float64
	M float64
	S float64
}

// CurvePoint represents a single point on a percentile curve.
type CurvePoint struct {
	AgeDays  int     `json:"age_days"`
	WeightKg float64 `json:"weight_kg"`
}

// PercentileCurve represents one percentile line (e.g., 50th percentile).
type PercentileCurve struct {
	Percentile float64      `json:"percentile"`
	Points     []CurvePoint `json:"points"`
}

// StandardPercentiles are the WHO percentile lines used for growth charts.
var StandardPercentiles = []float64{3, 15, 50, 85, 97}

// Package-level LMS tables, indexed by day (0-730).
var (
	maleLMS   []lmsEntry
	femaleLMS []lmsEntry
)

func init() {
	var err error
	maleLMS, err = loadLMS(boysCSV, "data/wfa_boys_lms.csv")
	if err != nil {
		panic(fmt.Sprintf("who: failed to load boys LMS data: %v", err))
	}
	femaleLMS, err = loadLMS(girlsCSV, "data/wfa_girls_lms.csv")
	if err != nil {
		panic(fmt.Sprintf("who: failed to load girls LMS data: %v", err))
	}
}

func loadLMS(fs embed.FS, path string) ([]lmsEntry, error) {
	data, err := fs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read embedded file %s: %w", path, err)
	}

	r := csv.NewReader(bytes.NewReader(data))
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse CSV %s: %w", path, err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV %s has no data rows", path)
	}

	// Skip header row
	entries := make([]lmsEntry, 0, len(records)-1)
	for i, rec := range records[1:] {
		if len(rec) < 4 {
			return nil, fmt.Errorf("CSV %s row %d: expected 4 columns, got %d", path, i+2, len(rec))
		}

		l, err := strconv.ParseFloat(rec[1], 64)
		if err != nil {
			return nil, fmt.Errorf("CSV %s row %d L value: %w", path, i+2, err)
		}
		m, err := strconv.ParseFloat(rec[2], 64)
		if err != nil {
			return nil, fmt.Errorf("CSV %s row %d M value: %w", path, i+2, err)
		}
		s, err := strconv.ParseFloat(rec[3], 64)
		if err != nil {
			return nil, fmt.Errorf("CSV %s row %d S value: %w", path, i+2, err)
		}

		entries = append(entries, lmsEntry{L: l, M: m, S: s})
	}

	return entries, nil
}

// getLMS returns the LMS table for the given sex.
func getLMS(sex string) ([]lmsEntry, error) {
	switch sex {
	case "male":
		return maleLMS, nil
	case "female":
		return femaleLMS, nil
	default:
		return nil, fmt.Errorf("invalid sex %q: must be \"male\" or \"female\"", sex)
	}
}

// ZScore computes the WHO z-score for a given sex, age in days, and weight in kg.
//
// The z-score is calculated using the LMS method:
//
//	z = ((y/M)^L - 1) / (L * S)  when L != 0
//	z = ln(y/M) / S              when L == 0
//
// where y is the measurement (weight), and L, M, S are the Box-Cox parameters.
func ZScore(sex string, ageDays int, weightKg float64) (float64, error) {
	if weightKg <= 0 {
		return 0, errors.New("weight must be positive")
	}
	if ageDays < 0 {
		return 0, errors.New("age in days must be non-negative")
	}

	table, err := getLMS(sex)
	if err != nil {
		return 0, err
	}

	if ageDays >= len(table) {
		return 0, fmt.Errorf("age %d days is out of range (max %d)", ageDays, len(table)-1)
	}

	entry := table[ageDays]
	return calcZScore(weightKg, entry.L, entry.M, entry.S), nil
}

// calcZScore computes the z-score from the LMS parameters.
func calcZScore(y, l, m, s float64) float64 {
	if math.Abs(l) < 1e-10 {
		return math.Log(y/m) / s
	}
	return (math.Pow(y/m, l) - 1) / (l * s)
}

// Percentile computes the WHO percentile for a given sex, age in days, and weight in kg.
// Returns a value between 0 and 100.
func Percentile(sex string, ageDays int, weightKg float64) (float64, error) {
	z, err := ZScore(sex, ageDays, weightKg)
	if err != nil {
		return 0, err
	}
	return zToPercentile(z), nil
}

// zToPercentile converts a z-score to a percentile using the standard normal CDF.
func zToPercentile(z float64) float64 {
	return 100.0 * normalCDF(z)
}

// normalCDF approximates the cumulative distribution function of the standard normal.
// Uses the Abramowitz and Stegun approximation (error < 1.5e-7).
func normalCDF(x float64) float64 {
	if x < -8 {
		return 0
	}
	if x > 8 {
		return 1
	}

	const (
		a1 = 0.254829592
		a2 = -0.284496736
		a3 = 1.421413741
		a4 = -1.453152027
		a5 = 1.061405429
		p  = 0.3275911
	)

	sign := 1.0
	if x < 0 {
		sign = -1.0
		x = -x
	}

	t := 1.0 / (1.0 + p*x)
	y := 1.0 - (((((a5*t+a4)*t)+a3)*t+a2)*t+a1)*t*math.Exp(-x*x/2)

	return 0.5 * (1.0 + sign*y)
}

// weightFromZScore computes the weight (kg) for a given z-score using the LMS method.
//
//	y = M * (1 + L*S*z)^(1/L)  when L != 0
//	y = M * exp(S*z)            when L == 0
func weightFromZScore(z float64, entry lmsEntry) float64 {
	if math.Abs(entry.L) < 1e-10 {
		return entry.M * math.Exp(entry.S*z)
	}
	return entry.M * math.Pow(1+entry.L*entry.S*z, 1/entry.L)
}

// percentileToZ converts a percentile (0-100) to a z-score using an approximation
// of the inverse standard normal (probit) function.
// Uses the rational approximation from Abramowitz and Stegun.
func percentileToZ(pct float64) float64 {
	p := pct / 100.0

	if p <= 0 {
		return -8
	}
	if p >= 1 {
		return 8
	}

	// Rational approximation for the inverse normal CDF
	// Peter Acklam's algorithm
	const (
		a1 = -3.969683028665376e+01
		a2 = 2.209460984245205e+02
		a3 = -2.759285104469687e+02
		a4 = 1.383577518672690e+02
		a5 = -3.066479806614716e+01
		a6 = 2.506628277459239e+00

		b1 = -5.447609879822406e+01
		b2 = 1.615858368580409e+02
		b3 = -1.556989798598866e+02
		b4 = 6.680131188771972e+01
		b5 = -1.328068155288572e+01

		c1 = -7.784894002430293e-03
		c2 = -3.223964580411365e-01
		c3 = -2.400758277161838e+00
		c4 = -2.549732539343734e+00
		c5 = 4.374664141464968e+00
		c6 = 2.938163982698783e+00

		d1 = 7.784695709041462e-03
		d2 = 3.224671290700398e-01
		d3 = 2.445134137142996e+00
		d4 = 3.754408661907416e+00

		pLow  = 0.02425
		pHigh = 1 - pLow
	)

	var q, r float64

	if p < pLow {
		q = math.Sqrt(-2 * math.Log(p))
		return (((((c1*q+c2)*q+c3)*q+c4)*q+c5)*q + c6) / ((((d1*q+d2)*q+d3)*q+d4)*q + 1)
	}

	if p <= pHigh {
		q = p - 0.5
		r = q * q
		return (((((a1*r+a2)*r+a3)*r+a4)*r+a5)*r + a6) * q / (((((b1*r+b2)*r+b3)*r+b4)*r+b5)*r + 1)
	}

	q = math.Sqrt(-2 * math.Log(1-p))
	return -(((((c1*q+c2)*q+c3)*q+c4)*q+c5)*q + c6) / ((((d1*q+d2)*q+d3)*q+d4)*q + 1)
}

// PercentileCurves generates the 3rd, 15th, 50th, 85th, and 97th weight-for-age
// percentile curves for a given sex and age range (inclusive).
func PercentileCurves(sex string, fromDays, toDays int) ([]PercentileCurve, error) {
	if fromDays < 0 {
		return nil, errors.New("from_days must be non-negative")
	}

	table, err := getLMS(sex)
	if err != nil {
		return nil, err
	}

	if toDays >= len(table) {
		return nil, fmt.Errorf("to_days %d is out of range (max %d)", toDays, len(table)-1)
	}

	if fromDays > toDays {
		return nil, fmt.Errorf("from_days %d must be <= to_days %d", fromDays, toDays)
	}

	percentiles := StandardPercentiles
	zScores := make([]float64, len(percentiles))
	for i, p := range percentiles {
		zScores[i] = percentileToZ(p)
	}

	numPoints := toDays - fromDays + 1
	curves := make([]PercentileCurve, len(percentiles))
	for i, p := range percentiles {
		points := make([]CurvePoint, numPoints)
		z := zScores[i]
		for day := fromDays; day <= toDays; day++ {
			w := weightFromZScore(z, table[day])
			points[day-fromDays] = CurvePoint{
				AgeDays:  day,
				WeightKg: math.Round(w*10000) / 10000, // 4 decimal places
			}
		}
		curves[i] = PercentileCurve{
			Percentile: p,
			Points:     points,
		}
	}

	return curves, nil
}
