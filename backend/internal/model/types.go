package model

import "time"

// DefaultCalPerFeed is the default kcal estimate for breast-direct feeds without volume.
const DefaultCalPerFeed = 67.0

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
