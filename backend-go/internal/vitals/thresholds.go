package vitals

import (
	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Classification thresholds of every vital (B-N6-01). One table, with its sources, echoed by GET /vitals/thresholds
// so the clients draw the same bands the server classifies with. Since B-N9-11 the numbers are versioned clinical
// config (internal/clinicalconfig, D-88): the constants below are version 1 (the code default), and a reading is
// classified with the version valid on its own day, so history keeps its classes when a threshold changes.
//
// Intervals are half-open: a band's Max is exclusive («۷۰–۱۰۰» = 70 ≤ v < 100).

// ThresholdsVersion identifies this table in payloads (clients and the B-N9-11 config compare it).
const ThresholdsVersion = "2026-10-acc-aha-2017.ada-2025.v1"

// Sources of the bands.
const (
	SourceBP      = "Whelton PK et al. 2017 ACC/AHA Guideline for the Prevention, Detection, Evaluation, and Management of High Blood Pressure in Adults. Hypertension 2018;71:e13–e115 (adults; AAP 2017 uses the same cut-offs from age 13)"
	SourceGlucose = "American Diabetes Association. Standards of Care in Diabetes — 2025: §2 Diagnosis and Classification (fasting 100 / 126, 2-h 140 / 200 mg/dL) and §6 Glycemic Goals and Hypoglycemia (level 1 < 70, level 2 < 54 mg/dL). Diabetes Care 2025;48(Suppl 1)"
	SourceHR      = "American Heart Association — All About Heart Rate (Pulse): normal resting heart rate for adults 60–100 bpm"
)

// Blood pressure (mmHg), ACC/AHA 2017.
const (
	BPElevatedSystolic = 120 // 120–129 and diastolic < 80
	BPStage1Systolic   = 130 // 130–139 or diastolic 80–89
	BPStage1Diastolic  = 80
	BPStage2Systolic   = 140 // ≥ 140 or diastolic ≥ 90
	BPStage2Diastolic  = 90
	// Hypertensive crisis: systolic > 180 and/or diastolic > 120 → the urgent message.
	BPCrisisSystolic  = 180
	BPCrisisDiastolic = 120
)

// Glucose (mg/dL), ADA 2025.
const (
	// GlucoseUrgentBelow is level 2 hypoglycaemia (< 54) → the urgent message.
	GlucoseUrgentBelow = 54
	// GlucoseLowBelow is level 1 hypoglycaemia (< 70).
	GlucoseLowBelow = 70
	// MmolFactor converts mmol/L to mg/dL (glucose 180.16 g/mol ÷ 10, rounded as in the ADA tables).
	MmolFactor = 18.0
)

// GlucoseBand is a context's target band and its upper classes (mg/dL; Max / HighFrom / VeryHighFrom inclusive lower
// bounds of the next class).
type GlucoseBand struct {
	TargetMin, TargetMax float64 // in range: TargetMin ≤ v < TargetMax
	VeryHighFrom         float64 // high: TargetMax ≤ v < VeryHighFrom; very high: ≥ VeryHighFrom
}

// GlucoseBands per context: fasting / before a meal use the fasting cut-offs (normal < 100, diabetes range ≥ 126);
// after a meal (2 h), bedtime and random use the 2-hour cut-offs (normal < 140, ≥ 200).
var GlucoseBands = map[string]GlucoseBand{
	ContextFasting:    {TargetMin: GlucoseLowBelow, TargetMax: 100, VeryHighFrom: 126},
	ContextBeforeMeal: {TargetMin: GlucoseLowBelow, TargetMax: 100, VeryHighFrom: 126},
	ContextAfterMeal:  {TargetMin: GlucoseLowBelow, TargetMax: 140, VeryHighFrom: 200},
	ContextBedtime:    {TargetMin: GlucoseLowBelow, TargetMax: 140, VeryHighFrom: 200},
	ContextRandom:     {TargetMin: GlucoseLowBelow, TargetMax: 140, VeryHighFrom: 200},
}

// Heart rate (bpm), AHA: resting 60–100; only resting / after-waking readings are classified.
const (
	HRRestingMin = 60
	HRRestingMax = 100 // normal: 60 ≤ v ≤ 100 (the AHA range is inclusive)
)

// Class codes.
const (
	ClassNormal   = "normal"
	ClassElevated = "elevated"
	ClassStage1   = "stage1"
	ClassStage2   = "stage2"
	ClassCrisis   = "crisis"

	ClassUrgentLow = "urgent_low"
	ClassLow       = "low"
	ClassInRange   = "in_range"
	ClassHigh      = "high"
	ClassVeryHigh  = "very_high"
)

// Tones colour a class on the clients: ok · watch · high · urgent.
const (
	ToneOK     = "ok"
	ToneWatch  = "watch"
	ToneHigh   = "high"
	ToneUrgent = "urgent"
)

// BPClasses / GlucoseClasses / HRClasses are the distribution orders.
var (
	BPClasses      = []string{ClassNormal, ClassElevated, ClassStage1, ClassStage2, ClassCrisis}
	GlucoseClasses = []string{ClassUrgentLow, ClassLow, ClassInRange, ClassHigh, ClassVeryHigh}
	HRClasses      = []string{ClassLow, ClassNormal, ClassHigh}
)

var tones = map[string]map[string]string{
	TypeBP: {ClassNormal: ToneOK, ClassElevated: ToneWatch, ClassStage1: ToneHigh, ClassStage2: ToneHigh, ClassCrisis: ToneUrgent},
	TypeGlucose: {ClassUrgentLow: ToneUrgent, ClassLow: ToneHigh, ClassInRange: ToneOK, ClassHigh: ToneWatch,
		ClassVeryHigh: ToneHigh},
	TypeHR: {ClassLow: ToneWatch, ClassNormal: ToneOK, ClassHigh: ToneWatch},
}

// Tone is the colour of a class of a type ("" when unclassified).
func Tone(typ, class string) string { return tones[typ][class] }

// Thresholds are the bands valid on one day.
type Thresholds struct {
	On                                  civildate.Date
	BPElevatedSystolic                  float64
	BPStage1Systolic, BPStage1Diastolic float64
	BPStage2Systolic, BPStage2Diastolic float64
	BPCrisisSystolic, BPCrisisDiastolic float64
	GlucoseUrgentBelow, GlucoseLowBelow float64
	GlucoseBands                        map[string]GlucoseBand
	HRRestingMin, HRRestingMax          float64
}

// thresholdKeys are the clinical config keys of the vitals table.
var thresholdKeys = []string{
	clinicalconfig.KeyBPElevated, clinicalconfig.KeyBPStage1, clinicalconfig.KeyBPStage2, clinicalconfig.KeyBPCrisis,
	clinicalconfig.KeyGlucoseUrgentLow, clinicalconfig.KeyGlucoseLow, clinicalconfig.KeyGlucoseFasting,
	clinicalconfig.KeyGlucosePostMeal, clinicalconfig.KeyHRResting,
}

// ThresholdsOn reads the bands valid on d from the clinical config (a zero d = version 1, the constants above).
func ThresholdsOn(d civildate.Date) Thresholds {
	n := func(key, field string) float64 { return clinicalconfig.Num(key, field, d) }
	low := n(clinicalconfig.KeyGlucoseLow, "mgdl")
	fasting := GlucoseBand{TargetMin: low, TargetMax: n(clinicalconfig.KeyGlucoseFasting, "target_max"),
		VeryHighFrom: n(clinicalconfig.KeyGlucoseFasting, "very_high_from")}
	post := GlucoseBand{TargetMin: low, TargetMax: n(clinicalconfig.KeyGlucosePostMeal, "target_max"),
		VeryHighFrom: n(clinicalconfig.KeyGlucosePostMeal, "very_high_from")}
	return Thresholds{
		On:                 d,
		BPElevatedSystolic: n(clinicalconfig.KeyBPElevated, "systolic"),
		BPStage1Systolic:   n(clinicalconfig.KeyBPStage1, "systolic"),
		BPStage1Diastolic:  n(clinicalconfig.KeyBPStage1, "diastolic"),
		BPStage2Systolic:   n(clinicalconfig.KeyBPStage2, "systolic"),
		BPStage2Diastolic:  n(clinicalconfig.KeyBPStage2, "diastolic"),
		BPCrisisSystolic:   n(clinicalconfig.KeyBPCrisis, "systolic"),
		BPCrisisDiastolic:  n(clinicalconfig.KeyBPCrisis, "diastolic"),
		GlucoseUrgentBelow: n(clinicalconfig.KeyGlucoseUrgentLow, "mgdl"),
		GlucoseLowBelow:    low,
		GlucoseBands: map[string]GlucoseBand{
			ContextFasting: fasting, ContextBeforeMeal: fasting,
			ContextAfterMeal: post, ContextBedtime: post, ContextRandom: post,
		},
		HRRestingMin: n(clinicalconfig.KeyHRResting, "min"),
		HRRestingMax: n(clinicalconfig.KeyHRResting, "max"),
	}
}

// Version is ThresholdsVersion, suffixed with the config versions in force when any key is past version 1.
func (t Thresholds) Version() string {
	if fp := clinicalconfig.Fingerprint(t.On, thresholdKeys...); fp != "" {
		return ThresholdsVersion + "+" + fp
	}
	return ThresholdsVersion
}

// ClassifyBP is the ACC/AHA 2017 category; the higher of the two numbers decides.
func (t Thresholds) ClassifyBP(systolic, diastolic float64) string {
	switch {
	case systolic > t.BPCrisisSystolic || diastolic > t.BPCrisisDiastolic:
		return ClassCrisis
	case systolic >= t.BPStage2Systolic || diastolic >= t.BPStage2Diastolic:
		return ClassStage2
	case systolic >= t.BPStage1Systolic || diastolic >= t.BPStage1Diastolic:
		return ClassStage1
	case systolic >= t.BPElevatedSystolic:
		return ClassElevated
	}
	return ClassNormal
}

// ClassifyGlucose is the ADA class of a mg/dL value in a context (unknown contexts use the random band).
func (t Thresholds) ClassifyGlucose(mgdl float64, context string) string {
	b, ok := t.GlucoseBands[context]
	if !ok {
		b = t.GlucoseBands[ContextRandom]
	}
	switch {
	case mgdl < t.GlucoseUrgentBelow:
		return ClassUrgentLow
	case mgdl < b.TargetMin:
		return ClassLow
	case mgdl < b.TargetMax:
		return ClassInRange
	case mgdl < b.VeryHighFrom:
		return ClassHigh
	}
	return ClassVeryHigh
}

// HRClassified reports whether a heart-rate context is compared with the resting range.
func HRClassified(context string) bool {
	return context == HRResting || context == HRAfterWaking || context == ""
}

// ClassifyHR is the resting-range class, "" for contexts the range does not apply to (after exercise, stress).
func (t Thresholds) ClassifyHR(bpm float64, context string) string {
	if !HRClassified(context) {
		return ""
	}
	switch {
	case bpm < t.HRRestingMin:
		return ClassLow
	case bpm > t.HRRestingMax:
		return ClassHigh
	}
	return ClassNormal
}

// UrgentBP: systolic > 180 and/or diastolic > 120.
func (t Thresholds) UrgentBP(systolic, diastolic float64) bool {
	return systolic > t.BPCrisisSystolic || diastolic > t.BPCrisisDiastolic
}

// UrgentGlucose: below 54 mg/dL.
func (t Thresholds) UrgentGlucose(mgdl float64) bool { return mgdl < t.GlucoseUrgentBelow }

// v1 is the code default table (version 1).
func v1() Thresholds { return ThresholdsOn(civildate.Date{}) }

// ClassifyBP / ClassifyGlucose / ClassifyHR / UrgentBP / UrgentGlucose use version 1; readings use their day's
// version (Reading.Class / Reading.Urgent).
func ClassifyBP(systolic, diastolic float64) string { return v1().ClassifyBP(systolic, diastolic) }

// ClassifyGlucose is Thresholds.ClassifyGlucose of version 1.
func ClassifyGlucose(mgdl float64, context string) string { return v1().ClassifyGlucose(mgdl, context) }

// ClassifyHR is Thresholds.ClassifyHR of version 1.
func ClassifyHR(bpm float64, context string) string { return v1().ClassifyHR(bpm, context) }

// UrgentBP is Thresholds.UrgentBP of version 1.
func UrgentBP(systolic, diastolic float64) bool { return v1().UrgentBP(systolic, diastolic) }

// UrgentGlucose is Thresholds.UrgentGlucose of version 1.
func UrgentGlucose(mgdl float64) bool { return v1().UrgentGlucose(mgdl) }
