package model

const (
	// FeedTypeBreastMilk is the feed type for breast milk.
	FeedTypeBreastMilk = "breast_milk"
	// FeedTypeFormula is the feed type for formula.
	FeedTypeFormula = "formula"
)

// validSet is a helper that builds a set lookup for enum validation.
func validSet(values ...string) map[string]bool {
	m := make(map[string]bool, len(values))
	for _, v := range values {
		m[v] = true
	}
	return m
}

var (
	sexValues              = validSet("male", "female")
	feedTypeValues         = validSet("breast_milk", "formula", "fortified_breast_milk", "solid", "other")
	urineColorValues       = validSet("clear", "pale_yellow", "dark_yellow", "amber", "brown")
	stoolColorLabelValues  = validSet("white", "clay", "pale_yellow", "yellow", "light_green", "green", "brown")
	stoolConsistencyValues = validSet("watery", "loose", "soft", "formed", "hard")
	stoolVolumeValues      = validSet("small", "medium", "large")
	measurementSrcValues   = validSet("home_scale", "clinic")
	tempMethodValues       = validSet("rectal", "axillary", "ear", "forehead")
	firmnessValues         = validSet("soft", "firm", "distended")
	jaundiceLevelValues    = validSet("none", "mild_face", "moderate_trunk", "severe_limbs_and_trunk")
	bruisingSizeValues     = validSet("small_<1cm", "medium_1-3cm", "large_>3cm")
	medFrequencyValues     = validSet("once_daily", "twice_daily", "three_times_daily", "as_needed", "custom")
	noteCategoryValues     = validSet("behavior", "sleep", "vomiting", "irritability", "skin", "other")
	fluidDirectionValues   = validSet("intake", "output")
	fluidSourceTypeValues  = validSet("feeding", "urine", "stool")
)

// ValidSex reports whether v is a valid sex value.
func ValidSex(v string) bool { return sexValues[v] }

// ValidFeedType reports whether v is a valid feed_type value.
func ValidFeedType(v string) bool { return feedTypeValues[v] }

// ValidUrineColor reports whether v is a valid urine color value.
func ValidUrineColor(v string) bool { return urineColorValues[v] }

// ValidStoolColorLabel reports whether v is a valid stool color label.
func ValidStoolColorLabel(v string) bool { return stoolColorLabelValues[v] }

// ValidStoolConsistency reports whether v is a valid stool consistency value.
func ValidStoolConsistency(v string) bool { return stoolConsistencyValues[v] }

// ValidStoolVolume reports whether v is a valid stool volume estimate.
func ValidStoolVolume(v string) bool { return stoolVolumeValues[v] }

// ValidMeasurementSource reports whether v is a valid weight measurement source.
func ValidMeasurementSource(v string) bool { return measurementSrcValues[v] }

// ValidTemperatureMethod reports whether v is a valid temperature measurement method.
func ValidTemperatureMethod(v string) bool { return tempMethodValues[v] }

// ValidFirmness reports whether v is a valid abdomen firmness value.
func ValidFirmness(v string) bool { return firmnessValues[v] }

// ValidJaundiceLevel reports whether v is a valid jaundice level value.
func ValidJaundiceLevel(v string) bool { return jaundiceLevelValues[v] }

// ValidBruisingSizeEstimate reports whether v is a valid bruising size estimate.
func ValidBruisingSizeEstimate(v string) bool { return bruisingSizeValues[v] }

// ValidMedFrequency reports whether v is a valid medication frequency.
func ValidMedFrequency(v string) bool { return medFrequencyValues[v] }

// ValidNoteCategory reports whether v is a valid general note category.
func ValidNoteCategory(v string) bool { return noteCategoryValues[v] }

// ValidFluidDirection reports whether v is a valid fluid log direction.
func ValidFluidDirection(v string) bool { return fluidDirectionValues[v] }

// ValidFluidSourceType reports whether v is a valid fluid log source type.
func ValidFluidSourceType(v string) bool { return fluidSourceTypeValues[v] }
