package engine

// The v6 enums (SPEC.md Part 2). String() is the spec spelling; the zero value is the "unknown"/"none" member
// where the spec has one, so a zero Status never claims a phase. Package-level name tables would be mutable
// globals (purity rule), so every String() picks from a literal list instead.

func pick(i int, names ...string) string {
	if i < 0 || i >= len(names) {
		return ""
	}
	return names[i]
}

// MainPhase is §2.13.
type MainPhase uint8

const (
	MainPhaseUnknown MainPhase = iota
	MainPhaseMenstrual
	MainPhaseFollicular
	MainPhaseFertile
	MainPhaseLuteal
	MainPhasePeriodExpected
)

func (v MainPhase) String() string {
	return pick(int(v), "unknown", "menstrual", "follicular", "fertile", "luteal", "period_expected")
}

// Subphase is §2.14.
type Subphase uint8

const (
	SubphaseUnknown Subphase = iota
	SubphaseMenstrual
	SubphaseMenstrualPossible
	SubphaseEarlyFollicular
	SubphaseMidFollicular
	SubphaseLateFollicularTransition
	SubphaseFertileRising
	SubphaseHighFertility
	SubphaseOvulationLikely
	SubphasePostOvulation
	SubphaseEarlyLuteal
	SubphaseMidLuteal
	SubphaseLateLuteal
	SubphasePMSPossible
	SubphasePeriodExpected
)

func (v Subphase) String() string {
	return pick(int(v), "unknown", "menstrual", "menstrual_possible", "early_follicular", "mid_follicular",
		"late_follicular_transition", "fertile_rising", "high_fertility", "ovulation_likely", "post_ovulation",
		"early_luteal", "mid_luteal", "late_luteal", "pms_possible", "period_expected")
}

// Main is the main phase a subphase belongs to (§2.15 table).
func (v Subphase) Main() MainPhase {
	switch {
	case v == SubphaseMenstrual || v == SubphaseMenstrualPossible:
		return MainPhaseMenstrual
	case v >= SubphaseEarlyFollicular && v <= SubphaseLateFollicularTransition:
		return MainPhaseFollicular
	case v >= SubphaseFertileRising && v <= SubphaseOvulationLikely:
		return MainPhaseFertile
	case v >= SubphasePostOvulation && v <= SubphasePMSPossible:
		return MainPhaseLuteal
	case v == SubphasePeriodExpected:
		return MainPhasePeriodExpected
	default:
		return MainPhaseUnknown
	}
}

// FertilityLevel is §2.26.
type FertilityLevel uint8

const (
	FertilityUnknown FertilityLevel = iota
	FertilityLow
	FertilityMedium
	FertilityHigh
	FertilityPeak
)

func (v FertilityLevel) String() string {
	return pick(int(v), "unknown", "low", "medium", "high", "peak")
}

// StartSource is current_period_start_source (§2.10).
type StartSource uint8

const (
	StartUnknown StartSource = iota
	StartUserLogged
	StartOnboardingDeclared
	StartPredictedReference
)

func (v StartSource) String() string {
	return pick(int(v), "unknown", "user_logged", "onboarding_declared", "predicted_reference")
}

// EndSource is current_period_end_source (§2.10).
type EndSource uint8

const (
	EndUnknown EndSource = iota
	EndUserLogged
	EndAssumedFromEffectiveDuration
)

func (v EndSource) String() string {
	return pick(int(v), "unknown", "user_logged", "assumed_from_effective_duration")
}

// OvulationSource is §2.10 / §2.17.
type OvulationSource uint8

const (
	OvulationUnknown OvulationSource = iota
	OvulationBBTConfirmed
	OvulationLHPositive
	OvulationCalendar
)

func (v OvulationSource) String() string {
	return pick(int(v), "unknown", "bbt_confirmed", "lh_positive", "calendar")
}

// CycleRegularity is §2.27.
type CycleRegularity uint8

const (
	RegularityInsufficientData CycleRegularity = iota
	RegularityRegular
	RegularityIrregular
)

func (v CycleRegularity) String() string {
	return pick(int(v), "insufficient_data", "regular", "irregular")
}

// DataQuality is §2.28.
type DataQuality uint8

const (
	DataQualityInsufficient DataQuality = iota
	DataQualityGood
	DataQualityPartial
	DataQualityPoor
)

func (v DataQuality) String() string {
	return pick(int(v), "insufficient", "good", "partial", "poor")
}

// Confidence is §2.29. The numeric order is the rank (unknown < low < medium < high), so min/max compare directly
// (E-03: min(today's confidence, medium)).
type Confidence uint8

const (
	ConfidenceUnknown Confidence = iota
	ConfidenceLow
	ConfidenceMedium
	ConfidenceHigh
)

func (v Confidence) String() string {
	return pick(int(v), "unknown", "low", "medium", "high")
}

// ResolutionSource is §2.30.
type ResolutionSource uint8

const (
	ResolutionUnknown ResolutionSource = iota
	ResolutionUserLogged
	ResolutionUserLoggedWithAssumedEnd
	ResolutionPrediction
	ResolutionOnboardingBased
	ResolutionDefaultBased
)

func (v ResolutionSource) String() string {
	return pick(int(v), "unknown", "user_logged", "user_logged_with_assumed_end", "prediction", "onboarding_based",
		"default_based")
}

// Frequency is clinical_flags.frequency (§2.27, E-06); absent values are Opt.None.
type Frequency uint8

const (
	FrequencyFrequent Frequency = iota
	FrequencyNormal
	FrequencyInfrequent
)

func (v Frequency) String() string { return pick(int(v), "frequent", "normal", "infrequent") }

// Duration is clinical_flags.duration (§2.27, E-07); absent values are Opt.None.
type Duration uint8

const (
	DurationNormal Duration = iota
	DurationProlonged
)

func (v Duration) String() string { return pick(int(v), "normal", "prolonged") }

// Mode is the user's life mode (§2.42, E-29). The zero value is cycle.
type Mode uint8

const (
	ModeCycle Mode = iota
	ModeTTC
	ModePregnancy
	ModePostpartum
	ModeMenopause
	ModeTeen
	modeCount
)

func (v Mode) String() string {
	return pick(int(v), "cycle", "ttc", "pregnancy", "postpartum", "menopause", "teen")
}

// ParseMode reads a mode spelling.
func ParseMode(s string) (Mode, bool) {
	for m := range modeCount {
		if m.String() == s {
			return m, true
		}
	}
	return ModeCycle, false
}

// SuppressionReason is limits.suppression_reason (E-24); SuppressionNone (String "") is JSON null.
type SuppressionReason uint8

const (
	SuppressionNone SuppressionReason = iota
	SuppressionHormonalContraception
	SuppressionPregnancy
	SuppressionPostpartumNotReturned
)

func (v SuppressionReason) String() string {
	return pick(int(v), "", "hormonal_contraception", "pregnancy", "postpartum_not_returned")
}

// PeriodSource is where a real period record comes from (both are actual data, §2.6; I-02 bridge).
type PeriodSource uint8

const (
	PeriodUserLogged PeriodSource = iota
	PeriodBleedingLog
)

func (v PeriodSource) String() string { return pick(int(v), "user_logged", "bleeding_log") }

// LengthSource is where an effective length comes from (§2.9, E-20).
type LengthSource uint8

const (
	LengthDefault LengthSource = iota
	LengthHistory
	LengthDeclared
)

func (v LengthSource) String() string { return pick(int(v), "default", "history", "declared") }

// RingAction is today_ui.ring_action (§2.43, E-17). The zero value is edit ("every other day").
type RingAction uint8

const (
	RingEdit RingAction = iota
	RingStart
	RingEnd
)

func (v RingAction) String() string { return pick(int(v), "edit", "start", "end") }

// Banner is today_ui.banner (§2.43, E-17), in the table's priority order; BannerNone (String "") is JSON null.
type Banner uint8

const (
	BannerNone Banner = iota
	BannerEndMissing
	BannerOverdue
	BannerAskLastPeriod
)

func (v Banner) String() string { return pick(int(v), "", "end_missing", "overdue", "ask_last_period") }
