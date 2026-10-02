package analysis

// Pregnancy analysis thresholds (B-N3-12), all in one place. They are sent in the payload so the client
// never repeats a number; B-N9-11 moves them to the admin config table. Every value is descriptive
// («talk to your doctor»), never a diagnosis.

// IOM 2009 total gestational weight gain by pre-pregnancy BMI, singleton pregnancy (Institute of Medicine
// and National Research Council. «Weight Gain During Pregnancy: Reexamining the Guidelines.» Washington,
// DC: The National Academies Press, 2009, Table S-1). The first-trimester gain the report assumes is the
// same report's 0.5–2 kg.
type IOMBand struct {
	Category       string
	BMIMin, BMIMax float64 // BMIMin inclusive, BMIMax exclusive (0 = open)
	GainMin        float64 // total gain, kg
	GainMax        float64
}

// IOMBands are the four pre-pregnancy BMI categories in table order.
var IOMBands = []IOMBand{
	{Category: "underweight", BMIMin: 0, BMIMax: 18.5, GainMin: 12.5, GainMax: 18},
	{Category: "normal", BMIMin: 18.5, BMIMax: 25, GainMin: 11.5, GainMax: 16},
	{Category: "overweight", BMIMin: 25, BMIMax: 30, GainMin: 7, GainMax: 11.5},
	{Category: "obese", BMIMin: 30, BMIMax: 0, GainMin: 5, GainMax: 9},
}

const (
	// IOMFirstTrimesterGainMin / Max: the 0.5–2 kg the IOM assumes by the end of week 13.
	IOMFirstTrimesterGainMin = 0.5
	IOMFirstTrimesterGainMax = 2.0
	// IOMFirstTrimesterWeeks / IOMTermWeeks: the recommended band runs 0 → first-trimester gain over
	// weeks 0–13, then linearly to the total gain at week 40.
	IOMFirstTrimesterWeeks = 13
	IOMTermWeeks           = 40
	// IOMRecentDays: «۴ هفته اخیر» compares the latest weight with one logged 28 days earlier
	// (IOMRecentSlackDays more allowed when no weight was logged on that exact day).
	IOMRecentDays      = 28
	IOMRecentSlackDays = 14
	// BaselineLookbackDays: a weight logged in the 90 days before the pregnancy start is the
	// pre-pregnancy weight; failing that, the earliest first-trimester weight, then the profile weight.
	BaselineLookbackDays = 90
)

// Blood pressure in pregnancy (ACOG Practice Bulletin No. 222, «Gestational Hypertension and
// Preeclampsia», Obstet Gynecol 2020;135:e237–60): a systolic ≥ 140 or diastolic ≥ 90 mmHg is the
// hypertension threshold; ≥ 160 / ≥ 110 is the severe range.
const (
	BPHighSystolic    = 140
	BPHighDiastolic   = 90
	BPSevereSystolic  = 160
	BPSevereDiastolic = 110
	// BPChartReadings: the hub line shows the latest readings.
	BPChartReadings = 12
)

// Gestational diabetes glucose targets (American Diabetes Association, «Standards of Care in Diabetes —
// 2024», section 15 Management of Diabetes in Pregnancy, Diabetes Care 2024;47(Suppl 1):S282–S294):
// fasting < 95, 1 h after a meal < 140, 2 h after a meal < 120 mg/dL.
const (
	GDMFastingMax = 95
	GDMOneHourMax = 140
	GDMTwoHourMax = 120
)

// Glucose slots.
const (
	GlucoseFasting = "fasting"
	GlucoseOneHour = "one_hour"
	GlucoseTwoHour = "two_hour"
)

// GlucoseTargets are the slots in card order with their upper targets (mg/dL, exclusive).
var GlucoseTargets = []struct {
	Slot string
	Max  int
}{{GlucoseFasting, GDMFastingMax}, {GlucoseOneHour, GDMOneHourMax}, {GlucoseTwoHour, GDMTwoHourMax}}

// Fetal movement counting (ACOG, «Special Tests for Monitoring Fetal Well-Being», patient FAQ, and the
// «count to 10» method): 10 movements within 2 hours, usually from week 28.
const (
	KicksTarget        = 10
	KicksWindowMinutes = 120
	KicksFromWeek      = 28
	KicksChartDays     = 7
	// KicksSessionMaxMinutes: a first→last movement span longer than this is a whole-day tally (the v1
	// fetal-movement log), not a counting session — it shows the count but no «minutes to 10».
	KicksSessionMaxMinutes = 2 * KicksWindowMinutes
)

// Symptoms by trimester: the top items of each trimester (the card shows two, the payload sends three).
const PregnancyTopSymptoms = 3
