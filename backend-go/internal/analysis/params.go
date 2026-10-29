package analysis

// Params are the SPEC v6 §4.1 analysis minimums and methods in one place (A-12, D-175, T-M10-26): product constants,
// not clinical config — documented here, pinned by params_test.go. Below a minimum a report says "not enough data"
// (ready false + days_needed) instead of a weak finding.
//
//	| analysis               | method                                                    | minimum                              |
//	|------------------------|-----------------------------------------------------------|--------------------------------------|
//	| length, flags          | from the engine (§2.9)                                    | 2 cycles                             |
//	| regularity             | the engine's cycle_regularity (§2.27, A-01)               | 3 cycles within 12 months            |
//	| period intensity       | mean flow per period day                                  | 2 periods                            |
//	| symptom pattern        | relative frequency per cycle day                          | symptom in ≥ 3 cycles                |
//	| mood by phase          | % of days with each mood in each phase                    | ≥ 10 logged days in a phase          |
//	| correlations           | rate ratio of two day groups (A-02); φ kept as `value`    | 30 paired days; strong = 30 d & ≥ 2× |
//	| weight                 | 7-day moving average, Δ7 and Δ30                          | 3 logs in the last 14 days           |
//	| blood pressure, sugar  | mean, min–max, class distribution (vitals classifier)     | 2 logs on 2 different days           |
//	| lab trend              | the same marker across consecutive confirmed reports      | 2 reports                            |
//	| BBT                    | 3-over-6 (engine/bbt); the result feeds the engine        | ≥ 10 temperatures in the cycle       |
//	| pregnancy weight       | against the IOM band                                      | pre-pregnancy BMI + 2 logs           |
//	| EPDS                   | score and trend                                           | 1 score; a trend with 2              |
//	| monthly report         | the month's figures + change vs the previous month        | 10 logged days                       |
//
// Top finding (§4.1): score = strength × recency, recency = 0.5^(age / TopFindingHalfLifeDays) where age is the days
// from the finding's latest evidence day to today; ties go to the fixed order of findingOrder (finding.go). The
// strength of each finding kind is the table findingStrength (finding.go).
type Params struct {
	CycleLengthMinCycles int // length / FIGO flags
	RegularityMinCycles  int // regularity verdict (the engine's own rule under v6)
	RegularityMonths     int // … counted within this many months before the report day

	PeriodIntensityMinPeriods int
	SymptomPatternMinCycles   int
	MoodPhaseMinDays          int // a phase's mood shares need this many mood-logged days
	MoodPhaseMinPhases        int // the mood-by-phase card compares at least this many qualifying phases

	CorrMinPairedDays int     // a correlation needs this many paired days (every logged day with both variables)
	CorrStrongDays    int     // strong = at least this many paired days …
	CorrStrongRatio   float64 // … and a group-rate ratio of at least this (either direction)
	CorrMediumRatio   float64 // medium: ratio ≥ this (and below strong)
	CorrWeakRatio     float64 // weak: ratio ≥ this; below = no association
	CorrMinGroupDays  int     // every compared group at least this many days (a 29-vs-1 split is no comparison)
	CorrMinOutcome    int     // the outcome logged at least this many times

	WeightMinLogs    int // weight logs …
	WeightWindowDays int // … within the last this many days

	VitalsMinLogs int // BP / glucose readings …
	VitalsMinDays int // … on at least this many different days

	LabTrendMinReports int
	BBTMinTemps        int // temperatures in the cycle (engine/bbt MinReadings under v6)
	PregWeightMinLogs  int // with a pre-pregnancy BMI
	EPDSMinScores      int
	EPDSTrendMinScores int
	MonthlyMinDays     int

	TopFindingHalfLifeDays float64
}

// P is the one parameter set (§4.1).
var P = Params{
	CycleLengthMinCycles: 2,
	RegularityMinCycles:  3,
	RegularityMonths:     12,

	PeriodIntensityMinPeriods: 2,
	SymptomPatternMinCycles:   3,
	MoodPhaseMinDays:          10,
	MoodPhaseMinPhases:        2,

	CorrMinPairedDays: 30,
	CorrStrongDays:    30,
	CorrStrongRatio:   2,
	CorrMediumRatio:   1.5,
	CorrWeakRatio:     1.2,
	CorrMinGroupDays:  5,
	CorrMinOutcome:    3,

	WeightMinLogs:    3,
	WeightWindowDays: 14,

	VitalsMinLogs: 2,
	VitalsMinDays: 2,

	LabTrendMinReports: 2,
	BBTMinTemps:        10,
	PregWeightMinLogs:  2,
	EPDSMinScores:      1,
	EPDSTrendMinScores: 2,
	MonthlyMinDays:     10,

	TopFindingHalfLifeDays: 30,
}

// daysForCycles converts a cycle minimum into days with the effective cycle length (rounded up, at least 1 day).
func daysForCycles(cycles, cycleLength int) int {
	if cycles <= 0 {
		return 0
	}
	return max(1, cycles*max(1, cycleLength))
}
