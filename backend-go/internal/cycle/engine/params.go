package engine

import "github.com/ritme/backend-go/internal/cycle/engine/bbt"

// Params holds every number the engine uses. Product constants come from DefaultParams (code, X-11); the clinical
// values are filled by internal/cycle/engineparams.For from the clinical config valid on the target date, and their
// DefaultParams values equal the registry's version 1. Params is comparable (==).
type Params struct {
	// ── Product constants (code, X-11) ──

	FutureCycles int // future_cycles k = 1..FutureCycles (§2.16)
	PromptBefore int // ring «پریودم شروع شد» from predicted_next − PromptBefore … (§2.43)
	PromptAfter  int // … to predicted_next + PromptAfter
	AbsentDays   int // clinical_flags.absent after this many days without a period (§2.27, E-07)

	DefaultCycleLength  int // §2.9 fallback
	DefaultPeriodLength int
	HistoryWindow       int // medians and variability use the last ≤ HistoryWindow valid records (§2.9, §2.27)

	CycleMin  int // valid cycle length CycleMin..CycleMax; below = short outlier, above = long outlier (§2.7, §2.8)
	CycleMax  int
	PeriodMin int // valid period length PeriodMin..PeriodMax (§2.8)
	PeriodMax int

	OverdueDaysLate int // days_late ≥ this → predicted_period_overdue (§2.22 tier 8..14)

	PMSDays        int // days_until_period 1..PMSDays → pms_possible (§2.25)
	LateLutealDays int // days_until_period PMSDays+1..LateLutealDays → late_luteal

	TeenAge           int // age < TeenAge → teen flags (§2.42, E-08, E-27)
	PerimenopauseAge  int // age ≥ this → confidence ≤ medium + perimenopause_age_range
	PostpartumPeriods int // predictions return after this many real periods after delivery (§2.42, E-14)

	RegularityMonths       int // cycle_regularity window before the target (§2.27, E-09)
	RegularityMinCycles    int
	RegularityAdultFromAge int // FIGO bands: < AdultFrom → young, AdultFrom..LateFrom−1 → adult, ≥ LateFrom → late
	RegularityLateFromAge  int

	// ── Clinical config (engineparams.For) ──

	PeriodHardCap   int // engine.period_hard_cap.days (§2.12)
	WarningCapExtra int // engine.warning_cap.extra_days: warning_cap = max(effective_period + extra, min)
	WarningCapMin   int // engine.warning_cap.min_days
	MaxLateDays     int // engine.max_late_days.days: beyond → unknown (§2.22)
	FertileBefore   int // engine.fertile_window.before: window O−before..O (§2.18); `after` is ignored by v6

	Luteal            LutealParams
	BBT               BBTParams
	LHBBTConflictDays int // engine.lh_bbt_conflict.days: |O(lh) − O(bbt)| > this → BBT + lh_bbt_conflict (§2.17)

	PregnancyTestCycleDaysLate int // engine.pregnancy_test.cycle_days_late (§2.22)
	PregnancyTestTTCDaysLate   int // engine.pregnancy_test.ttc_days_late

	StaleAnchorExtraDays int // engine.stale_anchor.extra_days: stale when > effective_cycle + extra (§2.41)

	FIGO          FIGOParams
	TeenFrequency RangeParams // cycle.teen.frequency (ACOG CO 651, E-08)
}

// LutealParams is engine.luteal (§2.16, E-16).
type LutealParams struct {
	Default int
	Min     int
	Max     int
}

// BBTParams is engine.bbt (§2.17, E-16): Strict (the third high reading must be ≥ RiseCenti over the max of the
// previous 6), MinReadings (readings needed in the cycle), RiseCenti (hundredths of °C).
type BBTParams = bbt.Params

// RangeParams is a min/max pair of days.
type RangeParams struct {
	Min int
	Max int
}

// FIGOParams is cycle.figo.* (FIGO 2018).
type FIGOParams struct {
	Frequency       RangeParams // < Min frequent, > Max infrequent
	DurationMax     int         // period > this → prolonged
	RegularityYoung int         // max(cycle) − min(cycle) allowed per age band
	RegularityAdult int
	RegularityLate  int
}

// DefaultParams is the code default of every number (version 1 of every clinical key).
func DefaultParams() Params {
	return Params{
		FutureCycles: 6,
		PromptBefore: 3,
		PromptAfter:  14,
		AbsentDays:   90,

		DefaultCycleLength:  28,
		DefaultPeriodLength: 5,
		HistoryWindow:       3,

		CycleMin:  21,
		CycleMax:  45,
		PeriodMin: 2,
		PeriodMax: 10,

		OverdueDaysLate: 8,
		PMSDays:         3,
		LateLutealDays:  6,

		TeenAge:           18,
		PerimenopauseAge:  45,
		PostpartumPeriods: 2,

		RegularityMonths:       12,
		RegularityMinCycles:    3,
		RegularityAdultFromAge: 26,
		RegularityLateFromAge:  42,

		PeriodHardCap:   12,
		WarningCapExtra: 3,
		WarningCapMin:   8,
		MaxLateDays:     14,
		FertileBefore:   5,

		Luteal:            LutealParams{Default: 14, Min: 10, Max: 16},
		BBT:               BBTParams{Strict: true, MinReadings: 10, RiseCenti: 20},
		LHBBTConflictDays: 2,

		PregnancyTestCycleDaysLate: 8,
		PregnancyTestTTCDaysLate:   1,

		StaleAnchorExtraDays: 14,

		FIGO: FIGOParams{
			Frequency:       RangeParams{Min: 24, Max: 38},
			DurationMax:     8,
			RegularityYoung: 9,
			RegularityAdult: 7,
			RegularityLate:  9,
		},
		TeenFrequency: RangeParams{Min: 21, Max: 45},
	}
}
