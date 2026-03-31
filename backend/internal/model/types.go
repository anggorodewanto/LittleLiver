package model

import "time"

// DefaultCalPerFeed is the default kcal estimate for breast-direct feeds without volume.
const DefaultCalPerFeed = 67.0

const (
	DateFormat          = "2006-01-02"
	DateTimeFormat      = "2006-01-02T15:04:05Z"
	DeletedUserSentinel = "deleted_user"
)

// User represents a parent user authenticated via Google OAuth.
type User struct {
	ID        string    `json:"id"`
	GoogleID  string    `json:"google_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Timezone  *string   `json:"timezone,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Baby represents a baby profile being tracked.
type Baby struct {
	ID                string     `json:"id"`
	Name              string     `json:"name"`
	Sex               string     `json:"sex"`
	DateOfBirth       time.Time  `json:"date_of_birth"`
	DiagnosisDate     *time.Time `json:"diagnosis_date,omitempty"`
	KasaiDate         *time.Time `json:"kasai_date,omitempty"`
	DefaultCalPerFeed float64    `json:"default_cal_per_feed"`
	Notes             *string    `json:"notes,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

// BabyParent represents the link between a baby and a parent user.
type BabyParent struct {
	BabyID   string    `json:"baby_id"`
	UserID   string    `json:"user_id"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

// Session represents a server-side session for an authenticated user.
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// Feeding represents a single feeding entry for a baby.
type Feeding struct {
	ID             string    `json:"id"`
	BabyID         string    `json:"baby_id"`
	LoggedBy       string    `json:"logged_by"`
	UpdatedBy      *string   `json:"updated_by,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
	FeedType       string    `json:"feed_type"`
	VolumeMl       *float64  `json:"volume_ml,omitempty"`
	CalDensity     *float64  `json:"cal_density,omitempty"`
	Calories       *float64  `json:"calories,omitempty"`
	UsedDefaultCal bool      `json:"used_default_cal"`
	DurationMin    *int      `json:"duration_min,omitempty"`
	Notes          *string   `json:"notes,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// MetricPage represents a paginated list of metric entries.
type MetricPage[T any] struct {
	Data       []T     `json:"data"`
	NextCursor *string `json:"next_cursor"`
}

// Stool represents a single stool entry for a baby.
type Stool struct {
	ID             string    `json:"id"`
	BabyID         string    `json:"baby_id"`
	LoggedBy       string    `json:"logged_by"`
	UpdatedBy      *string   `json:"updated_by,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
	ColorRating    int       `json:"color_rating"`
	ColorLabel     *string   `json:"color_label,omitempty"`
	Consistency    *string   `json:"consistency,omitempty"`
	VolumeEstimate *string   `json:"volume_estimate,omitempty"`
	VolumeMl       *float64  `json:"volume_ml,omitempty"`
	PhotoKeys      *string   `json:"photo_keys,omitempty"`
	Notes          *string   `json:"notes,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Urine represents a single urine entry for a baby.
type Urine struct {
	ID        string    `json:"id"`
	BabyID    string    `json:"baby_id"`
	LoggedBy  string    `json:"logged_by"`
	UpdatedBy *string   `json:"updated_by,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Color     *string   `json:"color,omitempty"`
	VolumeMl  *float64  `json:"volume_ml,omitempty"`
	Notes     *string   `json:"notes,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FluidLog represents a single fluid intake or output entry for fluid balance tracking.
type FluidLog struct {
	ID         string    `json:"id"`
	BabyID     string    `json:"baby_id"`
	LoggedBy   string    `json:"logged_by"`
	UpdatedBy  *string   `json:"updated_by,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	Direction  string    `json:"direction"`
	Method     string    `json:"method"`
	VolumeMl   *float64  `json:"volume_ml,omitempty"`
	SourceType *string   `json:"source_type,omitempty"`
	SourceID   *string   `json:"source_id,omitempty"`
	Notes      *string   `json:"notes,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Weight represents a single weight measurement for a baby.
type Weight struct {
	ID                string    `json:"id"`
	BabyID            string    `json:"baby_id"`
	LoggedBy          string    `json:"logged_by"`
	UpdatedBy         *string   `json:"updated_by,omitempty"`
	Timestamp         time.Time `json:"timestamp"`
	WeightKg          float64   `json:"weight_kg"`
	MeasurementSource *string   `json:"measurement_source,omitempty"`
	Notes             *string   `json:"notes,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// Temperature represents a single temperature reading for a baby.
type Temperature struct {
	ID        string    `json:"id"`
	BabyID    string    `json:"baby_id"`
	LoggedBy  string    `json:"logged_by"`
	UpdatedBy *string   `json:"updated_by,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Method    string    `json:"method"`
	Notes     *string   `json:"notes,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AbdomenObservation represents a single abdomen observation for a baby.
type AbdomenObservation struct {
	ID         string    `json:"id"`
	BabyID     string    `json:"baby_id"`
	LoggedBy   string    `json:"logged_by"`
	UpdatedBy  *string   `json:"updated_by,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	Firmness   string    `json:"firmness"`
	Tenderness bool      `json:"tenderness"`
	GirthCm    *float64  `json:"girth_cm,omitempty"`
	PhotoKeys  *string   `json:"photo_keys,omitempty"`
	Notes      *string   `json:"notes,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// SkinObservation represents a single skin observation for a baby.
type SkinObservation struct {
	ID             string    `json:"id"`
	BabyID         string    `json:"baby_id"`
	LoggedBy       string    `json:"logged_by"`
	UpdatedBy      *string   `json:"updated_by,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
	JaundiceLevel  *string   `json:"jaundice_level,omitempty"`
	ScleralIcterus bool      `json:"scleral_icterus"`
	Rashes         *string   `json:"rashes,omitempty"`
	Bruising       *string   `json:"bruising,omitempty"`
	PhotoKeys      *string   `json:"photo_keys,omitempty"`
	Notes          *string   `json:"notes,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// BruisingObservation represents a single bruising observation for a baby.
type BruisingObservation struct {
	ID           string    `json:"id"`
	BabyID       string    `json:"baby_id"`
	LoggedBy     string    `json:"logged_by"`
	UpdatedBy    *string   `json:"updated_by,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
	Location     string    `json:"location"`
	SizeEstimate string    `json:"size_estimate"`
	SizeCm       *float64  `json:"size_cm,omitempty"`
	Color        *string   `json:"color,omitempty"`
	PhotoKeys    *string   `json:"photo_keys,omitempty"`
	Notes        *string   `json:"notes,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// LabResult represents a single lab result entry for a baby (EAV-style).
type LabResult struct {
	ID          string    `json:"id"`
	BabyID      string    `json:"baby_id"`
	LoggedBy    string    `json:"logged_by"`
	UpdatedBy   *string   `json:"updated_by,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	TestName    string    `json:"test_name"`
	Value       string    `json:"value"`
	Unit        *string   `json:"unit,omitempty"`
	NormalRange *string   `json:"normal_range,omitempty"`
	Notes       *string   `json:"notes,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// GeneralNote represents a general note entry for a baby.
type GeneralNote struct {
	ID        string    `json:"id"`
	BabyID    string    `json:"baby_id"`
	LoggedBy  string    `json:"logged_by"`
	UpdatedBy *string   `json:"updated_by,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Content   string    `json:"content"`
	PhotoKeys *string   `json:"photo_keys,omitempty"`
	Category  *string   `json:"category,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Invite represents an invite code for linking a parent to a baby.
type Invite struct {
	Code      string     `json:"code"`
	BabyID    string     `json:"baby_id"`
	CreatedBy string     `json:"created_by"`
	UsedBy    *string    `json:"used_by,omitempty"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	ExpiresAt time.Time  `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// Medication represents a medication definition/schedule for a baby.
type Medication struct {
	ID        string    `json:"id"`
	BabyID    string    `json:"baby_id"`
	LoggedBy  string    `json:"logged_by"`
	UpdatedBy *string   `json:"updated_by,omitempty"`
	Name      string    `json:"name"`
	Dose      string    `json:"dose"`
	Frequency    string    `json:"frequency"`
	Schedule     *string   `json:"schedule,omitempty"`
	Timezone     *string   `json:"timezone,omitempty"`
	IntervalDays *int      `json:"interval_days,omitempty"`
	StartsFrom   *string   `json:"starts_from,omitempty"`
	Active       bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MedLog represents a single medication administration log entry.
type MedLog struct {
	ID            string     `json:"id"`
	MedicationID  string     `json:"medication_id"`
	BabyID        string     `json:"baby_id"`
	LoggedBy      string     `json:"logged_by"`
	UpdatedBy     *string    `json:"updated_by,omitempty"`
	ScheduledTime *time.Time `json:"scheduled_time,omitempty"`
	GivenAt       *time.Time `json:"given_at,omitempty"`
	Skipped       bool       `json:"skipped"`
	SkipReason    *string    `json:"skip_reason,omitempty"`
	Notes         *string    `json:"notes,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// PhotoUpload represents a photo upload staging row.
type PhotoUpload struct {
	ID           string     `json:"id"`
	BabyID       *string    `json:"baby_id,omitempty"`
	R2Key        string     `json:"r2_key"`
	ThumbnailKey *string    `json:"thumbnail_key,omitempty"`
	UploadedAt   time.Time  `json:"uploaded_at"`
	LinkedAt     *time.Time `json:"linked_at,omitempty"`
}

// PushSubscription represents a Web Push subscription for a user's device.
type PushSubscription struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Endpoint  string    `json:"endpoint"`
	P256dh    string    `json:"p256dh"`
	Auth      string    `json:"auth"`
	CreatedAt time.Time `json:"created_at"`
}
