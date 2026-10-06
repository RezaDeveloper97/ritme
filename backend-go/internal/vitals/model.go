// Package vitals is «علائم حیاتی» (bloom B-N6-01, D-63): timed blood-pressure, glucose and heart-rate readings with
// ACC/AHA 2017 / ADA classification (thresholds.go), glucose unit conversion, 7/14/30/90-day reports, a weekly
// measurement plan with reminders gated by notifications.Decide, and the urgent safety message (BP > 180/120,
// glucose < 54 mg/dL) whose copy is admin-editable (message_contents vitals_alert).
//
// Existing day-level vitals are not migrated: the log sheet's taxonomy v2 measurements.* slots (and their legacy
// daily_health_logs projection) stay where analysis, the pregnancy analysis and the log sheet read them, and are read
// merged into the vitals views as `source: "log"` readings without a time (merge.go). See D-63.
package vitals

import (
	"database/sql"
	"strconv"
	"time"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/phpround"
	"github.com/ritme/backend-go/internal/vitals/store"
)

// Types.
const (
	TypeBP      = "bp"
	TypeGlucose = "glucose"
	TypeHR      = "hr"
)

// Types is every type, in hub order (BP · heart rate · glucose).
var Types = []string{TypeBP, TypeHR, TypeGlucose}

// BP measuring conditions.
var (
	Arms      = []string{"left", "right"}
	Positions = []string{"sitting", "standing", "lying"}
)

// Glucose contexts, units and methods.
const (
	ContextFasting    = "fasting"
	ContextBeforeMeal = "before_meal"
	ContextAfterMeal  = "after_meal"
	ContextBedtime    = "bedtime"
	ContextRandom     = "random"

	UnitMgDl  = "mg_dl"
	UnitMmolL = "mmol_l"
)

var (
	GlucoseContexts = []string{ContextFasting, ContextBeforeMeal, ContextAfterMeal, ContextBedtime, ContextRandom}
	GlucoseUnits    = []string{UnitMgDl, UnitMmolL}
	GlucoseMethods  = []string{"glucometer", "lab", "sensor"}
)

// Heart-rate contexts.
const (
	HRResting       = "resting"
	HRAfterExercise = "after_exercise"
	HRAfterWaking   = "after_waking"
	HRStress        = "stress"
)

var HRContexts = []string{HRResting, HRAfterExercise, HRAfterWaking, HRStress}

// Input ranges (validation).
const (
	MinSystolic  = 50
	MaxSystolic  = 300
	MinDiastolic = 30
	MaxDiastolic = 200
	MinPulse     = 30
	MaxPulse     = 250
	MinMgDl      = 20
	MaxMgDl      = 600
	MinMmolL     = 1.1
	MaxMmolL     = 33.3
	MaxNoteLen   = 500
	// MaxBackfillDays: a reading may be entered up to two years back.
	MaxBackfillDays = 730
)

// Sources of a reading.
const (
	SourceVitals = "vitals" // a vital_readings row
	SourceLog    = "log"    // a day value of the log sheet (taxonomy measurements.*), no time
)

// Reading is one reading of any source.
type Reading struct {
	ID         uint64 // 0 for log readings
	Source     string
	Type       string
	Date       civildate.Date
	MeasuredAt time.Time // zero for log readings
	Systolic   float64
	Diastolic  float64
	Pulse      float64 // BP pulse or heart rate; 0 = none
	Arm        string
	Position   string
	MgDl       float64 // glucose, canonical
	Unit       string  // glucose unit as typed
	Context    string
	Method     string
	Note       string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Timed reports whether the reading has a time of day (log readings have none).
func (r Reading) Timed() bool { return !r.MeasuredAt.IsZero() }

// Class is the reading's class ("" when unclassified).
func (r Reading) Class() string {
	switch r.Type {
	case TypeBP:
		return ClassifyBP(r.Systolic, r.Diastolic)
	case TypeGlucose:
		return ClassifyGlucose(r.MgDl, r.Context)
	case TypeHR:
		return ClassifyHR(r.Pulse, r.Context)
	}
	return ""
}

// Urgent reports whether the reading crosses a safety threshold.
func (r Reading) Urgent() bool {
	switch r.Type {
	case TypeBP:
		return UrgentBP(r.Systolic, r.Diastolic)
	case TypeGlucose:
		return UrgentGlucose(r.MgDl)
	}
	return false
}

// MgDlFrom converts a typed value to canonical mg/dL (1 decimal).
func MgDlFrom(value float64, unit string) float64 {
	if unit == UnitMmolL {
		return phpround.Round(value*MmolFactor, 1)
	}
	return phpround.Round(value, 1)
}

// MmolL is a mg/dL value in mmol/L (1 decimal).
func MmolL(mgdl float64) float64 { return phpround.Round(mgdl/MmolFactor, 1) }

// MgDlRounded is a mg/dL value as shown (whole number).
func MgDlRounded(mgdl float64) float64 { return phpround.Round(mgdl, 0) }

// InUnit is a mg/dL value in unit, rounded as shown (mg/dL whole, mmol/L 1 decimal).
func InUnit(mgdl float64, unit string) float64 {
	if unit == UnitMmolL {
		return MmolL(mgdl)
	}
	return MgDlRounded(mgdl)
}

// Slot of the day of a timed reading: morning 04:00–11:59, afternoon 12:00–17:59, night 18:00–03:59.
const (
	PeriodMorning   = "morning"
	PeriodAfternoon = "afternoon"
	PeriodNight     = "night"
)

// Period is the part of the day of a timed reading ("" for log readings).
func (r Reading) Period() string {
	if !r.Timed() {
		return ""
	}
	h := r.MeasuredAt.In(civildate.Tehran).Hour()
	switch {
	case h >= 4 && h < 12:
		return PeriodMorning
	case h >= 12 && h < 18:
		return PeriodAfternoon
	}
	return PeriodNight
}

func i16(n sql.NullInt16) float64 {
	if !n.Valid {
		return 0
	}
	return float64(n.Int16)
}

func strOf(s sql.NullString) string {
	if !s.Valid {
		return ""
	}
	return s.String
}

func timeOf(t sql.NullTime) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}

// FromRow is a vital_readings row as a Reading.
func FromRow(r store.VitalReading) Reading {
	var mgdl float64
	if r.GlucoseMgDl.Valid {
		mgdl, _ = strconv.ParseFloat(r.GlucoseMgDl.String, 64)
	}
	unit := strOf(r.GlucoseUnit)
	if r.Type == TypeGlucose && unit == "" {
		unit = UnitMgDl
	}
	at := r.MeasuredAt.In(civildate.Tehran)
	return Reading{
		ID: r.ID, Source: SourceVitals, Type: r.Type, Date: civildate.FromTime(at), MeasuredAt: at,
		Systolic: i16(r.Systolic), Diastolic: i16(r.Diastolic), Pulse: i16(r.Pulse),
		Arm: strOf(r.Arm), Position: strOf(r.Position), MgDl: mgdl, Unit: unit,
		Context: strOf(r.Context), Method: strOf(r.Method), Note: strOf(r.Note),
		CreatedAt: timeOf(r.CreatedAt), UpdatedAt: timeOf(r.UpdatedAt),
	}
}
