package engine

import "github.com/ritme/backend-go/internal/cycle/engine/bbt"

// Period is one real period record (§2.2: actual data only — never a prediction, never the onboarding date).
type Period struct {
	Start  Day
	End    Day // NoDay = open (no period_end logged yet)
	Source PeriodSource
}

// Open reports whether no end is logged.
func (p Period) Open() bool { return p.End == NoDay }

// Declared is the onboarding/profile data (§2.6 "declared — not an actual log").
type Declared struct {
	LastPeriodStart Opt[Day] // last_period_start_from_onboarding; never a real period (I-01)
	CycleLength     Opt[int] // declared_cycle_length
	PeriodLength    Opt[int] // declared_period_length
	// Manual is the user's "use my lengths" switch (E-20): the declared lengths override history medians.
	Manual bool
}

// LHTest is one LH test (E-16: only positive counts; faint is stored as not positive).
type LHTest struct {
	Date     Day
	Positive bool
}

// BBTReading is one basal body temperature {Date, Centi} in hundredths of °C (E-16).
type BBTReading = bbt.Reading[Day]

// PregnancyTest is one home pregnancy test (E-26).
type PregnancyTest struct {
	Date     Day
	Positive bool
}

// Input is everything the engine knows about one user. Slices may be unsorted; Prepare copies what it keeps. The
// zero Input means "no data": every optional date is an Opt[Day] (Day 0 is the real date 1970-01-01), and Some(NoDay)
// is treated as absent too.
type Input struct {
	Periods  []Period
	Skips    []Day // the predicted starts the user marked «پریود نداشتم» (E-15)
	Declared Declared

	BirthDate    Opt[Day] // none = unknown (no age rules, E-27)
	TeenBandOnly bool     // only a teen age band is known: treated as < 18 (E-27)

	Mode          Mode
	Hormonal      bool     // active hormonal contraception (I-03)
	DeliveryDate  Opt[Day] // none = no delivery (E-14)
	Breastfeeding bool

	LH             []LHTest
	BBT            []BBTReading
	PregnancyTests []PregnancyTest
}

// Anchors is the §2.35 anchors block.
type Anchors struct {
	CurrentPeriodStart          Day
	CurrentPeriodStartSource    StartSource
	CurrentPeriodEnd            Day
	CurrentPeriodEndSource      EndSource
	CurrentPeriodEndIsConfirmed bool
	PredictedNextPeriodStart    Day
	EstimatedOvulationDate      Day
	OvulationSource             OvulationSource
}

// Metrics is the §2.35 metrics block.
type Metrics struct {
	EffectiveCycleLength  int
	EffectivePeriodLength int
	CycleVariability      Opt[int]
	CycleRegularity       CycleRegularity
}

// ClinicalFlags is the §2.27 clinical_flags block (E-06, E-07; null when there is no basis).
type ClinicalFlags struct {
	Frequency Opt[Frequency]
	Duration  Opt[Duration]
	Absent    Opt[bool]
}

// FutureCycle is one future_cycles item (E-05): period k and the ovulation that precedes it. Display only, never
// stored (§2.40). Under fertility suppression the fertile/ovulation days are NoDay (E-13).
type FutureCycle struct {
	K               int
	PeriodStart     Day
	PeriodEnd       Day
	FertileStart    Day
	FertileEnd      Day
	Ovulation       Day
	OvulationSource OvulationSource
	Confidence      Confidence
}

// Limits is the E-24 limits block.
type Limits struct {
	FertilitySuppressed bool
	SuppressionReason   SuppressionReason
	EngineSuspended     bool
	PromptPregnancyMode bool
}

// Status is the engine output for one target date: every §2.35 field plus Limits (E-24) and FutureK.
type Status struct {
	Date            Day
	CycleDay        Opt[int]
	MainPhase       MainPhase
	Subphase        Subphase
	FertilityLevel  FertilityLevel
	DaysToOvulation Opt[int]
	DaysToPeriod    Opt[int]
	DaysLate        Opt[int]

	Anchors       Anchors
	Metrics       Metrics
	ClinicalFlags ClinicalFlags
	// FutureCycles is today's future_cycles list (State.FutureCycles). Resolve leaves it nil so a single-day resolve
	// does not allocate; the today payload fills it (T-M9-20/21).
	FutureCycles []FutureCycle

	SuggestPregnancyTest bool
	Confidence           Confidence
	ConfidenceReasons    ReasonSet
	DataQuality          DataQuality
	ResolutionSource     ResolutionSource
	RequiresUserInput    bool
	Warnings             WarningSet

	Limits Limits
	// FutureK is the future cycle the target was resolved from (E-03): 0 = the current cycle, 1..6 = future_cycles[k].
	FutureK int
}

// IsPredicted is the legacy is_predicted flag (§2.31): false only for user_logged.
func (s *Status) IsPredicted() bool { return s.ResolutionSource != ResolutionUserLogged }

// DayRange is an inclusive range of days.
type DayRange struct {
	Start Day
	End   Day
}

// OvulationMark is an ovulation date with its source.
type OvulationMark struct {
	Date   Day
	Source OvulationSource
}

// PredictionCard is today_ui.prediction_card (§2.43, E-17).
type PredictionCard struct {
	Period          DayRange
	PMS             DayRange
	Fertile         Opt[DayRange]
	Ovulation       Opt[OvulationMark]
	CycleLength     int
	CycleRegularity CycleRegularity
}

// TodayUI is the E-17 today_ui block (the §2.43 mapping owned by the engine).
type TodayUI struct {
	RingAction              RingAction
	Banner                  Banner
	ShowPregnancyTestButton bool
	PredictionCard          PredictionCard
}
