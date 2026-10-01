// Package model holds the inputs of the cycle engine (T-M2-13/14), decoupled from sqlc so the
// engine packages (metrics, resolver, view, legacy) stay pure: no DB, no clock, no HTTP.
//
// The fields mirror what the PHP engine reads off the Eloquent models
// (backend/app/Models/CycleHistory.php, UserProfile.php) after their casts.
package model

import "github.com/ritme/backend-go/internal/platform/civildate"

// History is one cycle_histories row (a logged, estimated or auto-detected period).
type History struct {
	ID int64
	// PeriodStart is period_start_date (`date` cast, never null).
	PeriodStart civildate.Date
	// PeriodEnd is period_end_date; the zero Date is NULL (an open period).
	PeriodEnd civildate.Date
	// IsConfirmed marks a user-confirmed start; only these feed the medians and anchors.
	IsConfirmed bool
	// IsEstimated marks the onboarding seed row (source=onboarding_estimate).
	IsEstimated bool
	// Source is the `source` column (DataSource value, e.g. user_logged, onboarding_estimate).
	Source string
	// BleedingLength is bleeding_length (nil = NULL).
	BleedingLength *int
	// CycleLength is the stored cycle_length (nil = NULL). The v1.1 engine ignores it.
	CycleLength *int
	// DataQualityFlags is the data_quality_flags JSON array (nil = NULL).
	DataQualityFlags []string
}

// HasEnd reports whether the period has a logged end (period_end_date IS NOT NULL).
func (h History) HasEnd() bool { return !h.PeriodEnd.IsZero() }

// Profile is the subset of user_profiles the cycle engine reads. A nil *Profile is a user
// without a profile row (PHP `$user->profile === null`).
type Profile struct {
	// LastPeriodStart is last_period_start; the zero Date is NULL.
	LastPeriodStart civildate.Date
	// CycleDuration is cycle_duration (integer cast; nil = NULL). 0 counts as missing where
	// PHP tests truthiness.
	CycleDuration *int
	// PeriodDuration is period_duration (integer cast; nil = NULL).
	PeriodDuration *int
	// Birthday is birthday; the zero Date is NULL.
	Birthday civildate.Date
	// Goal is the `goal` column (UserGoal value, "" = NULL).
	Goal string
	// LengthsManual is «خودکار از داده‌ها» switched off (B-N1-09, cycle_preferences.lengths_auto = 0): the
	// profile's cycle / period length win over the medians of the history.
	LengthsManual bool
	// NoFertilityCopy is a stored life-stage mode that never gets fertility content (teen, menopause —
	// enums.LifeMode.AllowsFertilityContent, B-N2-11b): the daily card's copy stays off the fertile window.
	NoFertilityCopy bool
}

// Int returns a pointer to v (for literals in tables and tests).
func Int(v int) *int { return &v }
