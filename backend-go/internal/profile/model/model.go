// Package model holds the Eloquent-compatible JSON serialisation of the user-owned models
// (Model::toArray(): every column in table order, $hidden removed, $casts applied, loaded
// relations nested). It is the stable API other domains use to render a User or a
// UserProfile exactly as Laravel does.
//
//	model.UserJSON(u)                        // App\Models\User (auth.UserJSON)
//	model.UserWithProfileJSON(u, p)          // $user with the `profile` relation loaded (p may be nil)
//	model.ProfileJSON(p)                     // App\Models\UserProfile (nil → null)
//	model.ProfileWeight(p) / ProfileHeight(p) // the float / integer casts, for calculations
//
// Rows come from sqlc (`SELECT * FROM user_profiles …`). Every domain store generates its own
// but identical struct, so convert: model.ProfileJSON((*model.UserProfile)(&row)) or
// p := model.UserProfile(row).
//
// The generic serializer (Attributes / List with a Casts table) renders any other sqlc row
// struct from `SELECT *`; the cast tables of the models the data export needs
// (CycleHistoryCasts, DailyHealthLogCasts, Pregnancy*Casts, ReminderCasts) are here too.
package model

import (
	"strconv"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/profile/store"
)

// UserProfile is a user_profiles row (App\Models\UserProfile).
type UserProfile = store.UserProfile

// UserProfileCasts is UserProfile::casts().
var UserProfileCasts = Casts{
	"birthday":                 CastDateYMD,
	"weight":                   CastFloat,
	"height":                   CastInteger,
	"period_duration":          CastInteger,
	"cycle_duration":           CastInteger,
	"last_period_start":        CastDateYMD,
	"chronic_conditions":       CastArray,
	"calculation_started_at":   CastDateTime,
	"calculation_completed_at": CastDateTime,
	"calculation_version":      CastInteger,
}

// UserJSON is the User model's toArray() (password / remember_token hidden).
func UserJSON(u *auth.User) *jsonx.OrderedMap { return auth.UserJSON(u) }

// UserWithProfileJSON is $user->toArray() after $user->profile was read: the user's
// attributes followed by "profile" (null when the user has none).
func UserWithProfileJSON(u *auth.User, p *UserProfile) *jsonx.OrderedMap {
	return UserJSON(u).Set("profile", ProfileJSON(p))
}

// ProfileJSON is the UserProfile model's toArray(); nil encodes as null.
func ProfileJSON(p *UserProfile) any {
	if p == nil {
		return nil
	}
	return Attributes(p, UserProfileCasts)
}

// ProfileWeight is $profile->weight (float cast); ok=false when NULL.
func ProfileWeight(p *UserProfile) (float64, bool) {
	if p == nil || !p.Weight.Valid {
		return 0, false
	}
	w, err := strconv.ParseFloat(p.Weight.String, 64)
	return w, err == nil
}

// ProfileHeight is $profile->height (integer cast); ok=false when NULL.
func ProfileHeight(p *UserProfile) (int, bool) {
	if p == nil || !p.Height.Valid {
		return 0, false
	}
	return int(p.Height.Int16), true
}

// CycleHistoryCasts is CycleHistory::casts() (plain `date` casts: previous-day UTC instants).
var CycleHistoryCasts = Casts{
	"period_start_date":  CastDate,
	"period_end_date":    CastDate,
	"is_confirmed":       CastBoolean,
	"is_estimated":       CastBoolean,
	"data_quality_flags": CastArray,
}

// DailyHealthLogCasts is DailyHealthLog::casts().
var DailyHealthLogCasts = Casts{
	"log_date": CastDateYMD,

	"has_clots": CastBoolean, "spotting": CastBoolean, "diarrhea": CastBoolean,
	"constipation": CastBoolean, "food_craving": CastBoolean, "vaginal_dryness": CastBoolean,
	"vaginal_burning": CastBoolean, "vaginal_itching": CastBoolean, "vaginal_smell_change": CastBoolean,
	"acne": CastBoolean, "oily_skin": CastBoolean, "hair_loss": CastBoolean, "swelling": CastBoolean,
	"fatigue": CastBoolean, "dizziness": CastBoolean, "hot_flashes": CastBoolean, "chills": CastBoolean,
	"discharge_itching": CastBoolean, "discharge_burning": CastBoolean, "frequent_urination": CastBoolean,

	"weight":                 CastDecimal2,
	"basal_body_temperature": CastDecimal2,
	"blood_sugar":            CastDecimal1,

	"heart_rate":         CastInteger,
	"systolic_pressure":  CastInteger,
	"diastolic_pressure": CastInteger,
	"exercise_duration":  CastInteger,

	"moods":             CastArray,
	"exercise_type":     CastArray,
	"sexual_activities": CastArray,
	"medications":       CastArray,
}

// PregnancyProfileCasts is PregnancyProfile::casts().
var PregnancyProfileCasts = Casts{
	"pregnancy_mode": CastBoolean, "cycle_mode": CastBoolean, "is_locked": CastBoolean,
	"has_miscarriage_history": CastBoolean, "has_high_risk_history": CastBoolean,
	"rh_negative_care_flag": CastBoolean, "fetal_movement_felt": CastBoolean,
	"onboarding_completed": CastBoolean,

	"lmp_date": CastDate, "ultrasound_date": CastDate, "manual_entry_date": CastDate,
	"estimated_due_date": CastDate, "estimated_conception_date": CastDate,
	"first_fetal_movement_date": CastDate,
	"onboarding_completed_at":   CastDateTime,

	"pre_existing_conditions": CastArray,
}

// PregnancySymptomLogCasts is PregnancySymptomLog::casts().
var PregnancySymptomLogCasts = Casts{
	"log_date":   CastDateYMD,
	"has_nausea": CastBoolean, "has_vomiting": CastBoolean, "has_fatigue": CastBoolean,
	"has_headache": CastBoolean, "has_dizziness": CastBoolean, "has_breast_pain": CastBoolean,
	"has_lower_abdominal_pain": CastBoolean, "has_cramping": CastBoolean, "has_back_pain": CastBoolean,
	"has_pelvic_pressure": CastBoolean, "has_spotting": CastBoolean, "has_bleeding": CastBoolean,
	"has_fluid_leakage": CastBoolean, "has_severe_sudden_pain": CastBoolean,
}

// PregnancyWeeklyLogCasts is PregnancyWeeklyLog::casts().
var PregnancyWeeklyLogCasts = Casts{
	"log_date":     CastDateYMD,
	"has_swelling": CastBoolean, "has_shortness_of_breath": CastBoolean,
	"has_blood_pressure_device": CastBoolean, "has_anxiety": CastBoolean,
	"has_mood_swings": CastBoolean, "has_depression_feelings": CastBoolean,
	"weight": CastDecimal2, "fasting_blood_sugar": CastDecimal2, "post_meal_blood_sugar": CastDecimal2,
	"swelling_locations": CastArray,
}

// PregnancyFetalMovementCasts is PregnancyFetalMovement::casts().
var PregnancyFetalMovementCasts = Casts{"log_date": CastDateYMD}

// ReminderCasts is Reminder::casts() (recurrence_time deliberately uncast: "H:i:s").
var ReminderCasts = Casts{
	"scheduled_at": CastDateTime,
	"starts_on":    CastDate,
	"ends_on":      CastDate,
	"is_active":    CastBoolean,
	"meta":         CastArray,
}
