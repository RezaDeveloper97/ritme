package vitals

// Classification thresholds of every vital (B-N6-01). One table, with its sources, echoed by GET /vitals/thresholds
// so the clients draw the same bands the server classifies with. Moved to versioned admin config in B-N9-11; until
// then a change here re-classifies history (classes are computed on read, never stored). [needs clinical review]
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

// ClassifyBP is the ACC/AHA 2017 category; the higher of the two numbers decides.
func ClassifyBP(systolic, diastolic float64) string {
	switch {
	case systolic > BPCrisisSystolic || diastolic > BPCrisisDiastolic:
		return ClassCrisis
	case systolic >= BPStage2Systolic || diastolic >= BPStage2Diastolic:
		return ClassStage2
	case systolic >= BPStage1Systolic || diastolic >= BPStage1Diastolic:
		return ClassStage1
	case systolic >= BPElevatedSystolic:
		return ClassElevated
	}
	return ClassNormal
}

// ClassifyGlucose is the ADA class of a mg/dL value in a context (unknown contexts use the random band).
func ClassifyGlucose(mgdl float64, context string) string {
	b, ok := GlucoseBands[context]
	if !ok {
		b = GlucoseBands[ContextRandom]
	}
	switch {
	case mgdl < GlucoseUrgentBelow:
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
func ClassifyHR(bpm float64, context string) string {
	if !HRClassified(context) {
		return ""
	}
	switch {
	case bpm < HRRestingMin:
		return ClassLow
	case bpm > HRRestingMax:
		return ClassHigh
	}
	return ClassNormal
}

// UrgentBP: systolic > 180 and/or diastolic > 120.
func UrgentBP(systolic, diastolic float64) bool {
	return systolic > BPCrisisSystolic || diastolic > BPCrisisDiastolic
}

// UrgentGlucose: below 54 mg/dL.
func UrgentGlucose(mgdl float64) bool { return mgdl < GlucoseUrgentBelow }
