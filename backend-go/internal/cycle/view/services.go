package view

import (
	"slices"

	"github.com/ritme/backend-go/internal/cycle/metrics"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// ── CyclePredictionService / CyclePrediction ──────────────────────────────────────────────

const (
	predictionLutealLength     = 14
	fertileDaysBeforeOvulation = 5
	fertileDaysAfterOvulation  = 1
)

// Prediction is CyclePrediction.php: the Start-anchored calendar prediction for the cycle
// containing a reference date (§2, §10, §13).
type Prediction struct {
	CurrentCycleStart      civildate.Date
	NextPeriodStart        civildate.Date
	NextPeriodEnd          civildate.Date
	EstimatedOvulationDate civildate.Date
	FertileWindowStart     civildate.Date
	FertileWindowEnd       civildate.Date
	Source                 enums.EffectiveSource
}

// Predict is CyclePredictionService::predict: the anchor is projected forward in whole cycles to
// the cycle containing reference; ovulation = next start − 14, fertile window O−5 .. O+1.
func Predict(anchorStart civildate.Date, cycleLength, periodDuration int, reference civildate.Date, source enums.EffectiveSource) Prediction {
	cycleLength = max(1, cycleLength)
	periodDuration = max(1, periodDuration)

	daysSinceAnchor := anchorStart.DiffDays(reference)
	completed := 0
	if daysSinceAnchor > 0 {
		completed = daysSinceAnchor / cycleLength
	}

	current := anchorStart.AddDays(completed * cycleLength)
	next := current.AddDays(cycleLength)
	ovulation := next.AddDays(-predictionLutealLength)

	return Prediction{
		CurrentCycleStart:      current,
		NextPeriodStart:        next,
		NextPeriodEnd:          next.AddDays(periodDuration - 1),
		EstimatedOvulationDate: ovulation,
		FertileWindowStart:     ovulation.AddDays(-fertileDaysBeforeOvulation),
		FertileWindowEnd:       ovulation.AddDays(fertileDaysAfterOvulation),
		Source:                 source,
	}
}

// ── OpenPeriodEvaluator / OpenPeriodState ─────────────────────────────────────────────────

const (
	openHardCapDays      = 12
	minMaxExpectedDays   = 8
	maxExpectedExtraDays = 3
)

// OpenPeriodState is OpenPeriodState.php: an ongoing (Start logged, no End) period relative to today (§4).
type OpenPeriodState struct {
	HasOpenPeriod bool
	// StartDate is the zero Date when there is no open period.
	StartDate civildate.Date
	// MenstrualDay is today − start + 1, nil when there is no open period.
	MenstrualDay             *int
	DisplayAsActiveMenstrual bool
	EndOverdue               bool
	PastHardCap              bool
	// AssumedEndDate is internal only (start + effective duration − 1); zero when none.
	AssumedEndDate   civildate.Date
	DataQualityFlags []string
}

// NoOpenPeriod is OpenPeriodState::none().
func NoOpenPeriod() OpenPeriodState { return OpenPeriodState{DataQualityFlags: []string{}} }

// HasFlag reports whether the read-time data-quality flag is set.
func (s OpenPeriodState) HasFlag(flag enums.DataQualityFlag) bool {
	return slices.Contains(s.DataQualityFlags, string(flag))
}

// EvaluateOpenPeriod is OpenPeriodEvaluator::evaluate. openStart zero = no open period;
// profilePeriodDuration is the raw profile column (nil = NULL; a stored 0 is used as 0, like PHP ??).
func EvaluateOpenPeriod(openStart civildate.Date, effectivePeriodDuration int, profilePeriodDuration *int, today civildate.Date) OpenPeriodState {
	if openStart.IsZero() || today.Before(openStart) {
		return NoOpenPeriod()
	}

	menstrualDay := openStart.DiffDays(today) + 1

	base := effectivePeriodDuration
	if profilePeriodDuration != nil {
		base = *profilePeriodDuration
	}
	maxExpected := max(base+maxExpectedExtraDays, minMaxExpectedDays)

	displayActive := menstrualDay <= min(maxExpected, openHardCapDays)
	flags := []string{}
	if !displayActive {
		flags = append(flags, string(enums.DataQualityFlagIncompleteEndMissing))
	}

	return OpenPeriodState{
		HasOpenPeriod:            true,
		StartDate:                openStart,
		MenstrualDay:             &menstrualDay,
		DisplayAsActiveMenstrual: displayActive,
		EndOverdue:               !displayActive,
		PastHardCap:              menstrualDay > openHardCapDays,
		AssumedEndDate:           openStart.AddDays(max(1, effectivePeriodDuration) - 1),
		DataQualityFlags:         flags,
	}
}

// ── CycleConfidenceCalculator / CycleConfidence ───────────────────────────────────────────

// Confidence-reason codes of CycleConfidenceCalculator.
const (
	ReasonThreeValidCycles = "based_on_three_valid_cycles"
	ReasonLowVariation     = "low_cycle_variation"
	ReasonCycleVariation   = "cycle_variation"
	ReasonNotEnoughCycles  = "not_enough_logged_cycles"
	ReasonProfileOnly      = "based_on_profile_data"
	ReasonEndMissingLong   = "incomplete_end_missing"
)

// Confidence is CycleConfidence.php.
type Confidence struct {
	Level   enums.ConfidenceLevel
	Reasons []string
}

// ConfidenceForPrediction is CycleConfidenceCalculator::forPrediction. endMissingLong (an open
// period past the hard cap) downgrades one level and annotates.
func ConfidenceForPrediction(m metrics.Metrics, endMissingLong bool) Confidence {
	var level enums.ConfidenceLevel
	var reasons []string
	switch {
	case m.ValidCyclesCount >= 3 && m.Variability == enums.CycleVariabilityRegular:
		level, reasons = enums.ConfidenceLevelHigh, []string{ReasonThreeValidCycles, ReasonLowVariation}
	case m.ValidCyclesCount >= 3:
		level, reasons = enums.ConfidenceLevelMedium, []string{ReasonThreeValidCycles, ReasonCycleVariation}
	case m.ValidCyclesCount >= 1:
		level, reasons = enums.ConfidenceLevelMedium, []string{ReasonNotEnoughCycles}
	default:
		level, reasons = enums.ConfidenceLevelLow, []string{ReasonProfileOnly, ReasonNotEnoughCycles}
	}

	if endMissingLong {
		if level == enums.ConfidenceLevelHigh {
			level = enums.ConfidenceLevelMedium
		} else {
			level = enums.ConfidenceLevelLow
		}
		reasons = append(reasons, ReasonEndMissingLong)
	}

	return Confidence{Level: level, Reasons: slices.Compact(reasons)}
}
