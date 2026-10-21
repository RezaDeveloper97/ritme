package engine

import "iter"

// Warning is one value of §2.32 plus contradictory_anchor_data (E-02), in spec order.
type Warning uint8

const (
	WarnPeriodEndMissing Warning = iota
	WarnPeriodEndMissingWarningCapExceeded
	WarnPeriodEndMissingHardCapExceeded
	WarnLowHistory
	WarnHighCycleVariability
	WarnShortCycleOutlierDetected
	WarnLongCycleOutlierDetected
	WarnOnboardingDataUsed
	WarnDefaultValuesUsed
	WarnPredictionBasedOutput
	WarnPredictedPeriodOverdue
	WarnCycleUnresolvedAfterExpectedPeriod
	WarnInsufficientAnchorData
	WarnStaleOnboardingAnchor
	WarnHormonalContraceptionActive
	WarnPostpartumCyclesNotReturned
	WarnPerimenopauseAgeRange
	WarnLongAbsenceNoPeriod
	WarnSkippedCycleRecorded
	WarnLHBBTConflict
	WarnContradictoryAnchorData
	warningCount
)

func (v Warning) String() string {
	return pick(int(v), "period_end_missing", "period_end_missing_warning_cap_exceeded",
		"period_end_missing_hard_cap_exceeded", "low_history", "high_cycle_variability",
		"short_cycle_outlier_detected", "long_cycle_outlier_detected", "onboarding_data_used", "default_values_used",
		"prediction_based_output", "predicted_period_overdue", "cycle_unresolved_after_expected_period",
		"insufficient_anchor_data", "stale_onboarding_anchor", "hormonal_contraception_active",
		"postpartum_cycles_not_returned", "perimenopause_age_range", "long_absence_no_period",
		"skipped_cycle_recorded", "lh_bbt_conflict", "contradictory_anchor_data")
}

// RequiresUserInput reports whether the warning can only be cleared by the user (§2.33).
func (v Warning) RequiresUserInput() bool {
	switch v {
	case WarnPeriodEndMissingWarningCapExceeded, WarnPeriodEndMissingHardCapExceeded,
		WarnCycleUnresolvedAfterExpectedPeriod, WarnInsufficientAnchorData, WarnContradictoryAnchorData,
		WarnStaleOnboardingAnchor:
		return true
	default:
		return false
	}
}

// WarningSet is a set of warnings; it iterates in enum order (deterministic output, no allocations).
type WarningSet uint32

// Add puts w into the set.
func (s *WarningSet) Add(w Warning) { *s |= 1 << w }

// Remove takes w out of the set.
func (s *WarningSet) Remove(w Warning) { *s &^= 1 << w }

// Has reports whether w is in the set.
func (s WarningSet) Has(w Warning) bool { return s&(1<<w) != 0 }

// Len is the number of warnings in the set.
func (s WarningSet) Len() int {
	n := 0
	for x := s; x != 0; x &= x - 1 {
		n++
	}
	return n
}

// All yields the warnings in enum order.
func (s WarningSet) All() iter.Seq[Warning] {
	return func(yield func(Warning) bool) {
		for w := range warningCount {
			if s.Has(w) && !yield(w) {
				return
			}
		}
	}
}

// RequiresUserInput reports whether any warning of the set requires user input (§2.33).
func (s WarningSet) RequiresUserInput() bool {
	for w := range s.All() {
		if w.RequiresUserInput() {
			return true
		}
	}
	return false
}

// Reason is one confidence_reasons value of §2.29, in spec order.
type Reason uint8

const (
	ReasonLowHistory Reason = iota
	ReasonHighCycleVariability
	ReasonMissingPeriodEnd
	ReasonMissingPeriodEndBeyondWarningCap
	ReasonMissingPeriodEndBeyondHardCap
	ReasonOnboardingDataUsed
	ReasonDefaultValuesUsed
	ReasonPredictionBasedOutput
	ReasonPeriodOverdue
	ReasonUnresolvedCycle
	ReasonPoorDataQuality
	ReasonInsufficientAnchorData
	ReasonStaleOnboardingAnchor
	ReasonHormonalContraceptionActive
	reasonCount
)

func (v Reason) String() string {
	return pick(int(v), "low_history", "high_cycle_variability", "missing_period_end",
		"missing_period_end_beyond_warning_cap", "missing_period_end_beyond_hard_cap", "onboarding_data_used",
		"default_values_used", "prediction_based_output", "period_overdue", "unresolved_cycle", "poor_data_quality",
		"insufficient_anchor_data", "stale_onboarding_anchor", "hormonal_contraception_active")
}

// ReasonSet is a set of confidence reasons; it iterates in enum order (no allocations).
type ReasonSet uint16

// Add puts r into the set.
func (s *ReasonSet) Add(r Reason) { *s |= 1 << r }

// Remove takes r out of the set.
func (s *ReasonSet) Remove(r Reason) { *s &^= 1 << r }

// Has reports whether r is in the set.
func (s ReasonSet) Has(r Reason) bool { return s&(1<<r) != 0 }

// Len is the number of reasons in the set.
func (s ReasonSet) Len() int {
	n := 0
	for x := s; x != 0; x &= x - 1 {
		n++
	}
	return n
}

// All yields the reasons in enum order.
func (s ReasonSet) All() iter.Seq[Reason] {
	return func(yield func(Reason) bool) {
		for r := range reasonCount {
			if s.Has(r) && !yield(r) {
				return
			}
		}
	}
}
