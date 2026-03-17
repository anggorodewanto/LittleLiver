package model

import "fmt"

const (
	// MlPerOz is the conversion factor from oz to mL.
	MlPerOz = 29.5735
	// DefaultCalDensity is the default caloric density (kcal/oz) for breast_milk and formula.
	DefaultCalDensity = 20.0
)

// CalorieResult holds the computed calorie values for a feeding entry.
type CalorieResult struct {
	Calories       *float64
	CalDensity     *float64
	UsedDefaultCal bool
}

// CalculateCalories computes calories for a feeding entry based on feed type,
// volume, caloric density, and the baby's default_cal_per_feed.
// Returns an error if cal_density is provided with no volume (invalid combination).
func CalculateCalories(feedType string, volumeMl, calDensity *float64, defaultCalPerFeed float64) (*CalorieResult, error) {
	// Validation: cal_density with no volume is invalid
	if calDensity != nil && volumeMl == nil {
		return nil, fmt.Errorf("cal_density cannot be provided without volume_ml")
	}

	result := &CalorieResult{
		CalDensity: calDensity,
	}

	// Breast-direct: breast_milk with no volume
	if feedType == "breast_milk" && volumeMl == nil {
		cal := defaultCalPerFeed
		result.Calories = &cal
		result.UsedDefaultCal = true
		return result, nil
	}

	// Has volume — compute calories
	if volumeMl != nil {
		density := calDensity
		// Auto-apply default 20 kcal/oz for breast_milk and formula when cal_density not provided
		if density == nil && (feedType == "breast_milk" || feedType == "formula") {
			d := DefaultCalDensity
			density = &d
			result.CalDensity = &d
		}
		if density != nil {
			cal := *volumeMl * (*density / MlPerOz)
			result.Calories = &cal
		}
		return result, nil
	}

	// No volume, no breast-direct — calories left nil
	return result, nil
}
