package enums

import "slices"

// Hand-ported behaviour of the cycle-engine enums. Every method cites the PHP it mirrors.

// FertilityLevelV11 is the v1.1 fertility level of the sub-phase (task.md §26): peaks at ovulation,
// medium only on the fertile ramp and the late-follicular transition, unknown when unresolved.
// PHP: backend/app/Enums/CycleSubphase.php:78.
func (e CycleSubphase) FertilityLevelV11() FertilityLevel {
	switch e {
	case CycleSubphaseOvulationLikely:
		return FertilityLevelPeak
	case CycleSubphaseHighFertility:
		return FertilityLevelHigh
	case CycleSubphaseFertileRising, CycleSubphaseLateFollicularTransition:
		return FertilityLevelMedium
	case CycleSubphasePeriodExpected, CycleSubphaseUnknown:
		return FertilityLevelUnknown
	default:
		return FertilityLevelLow
	}
}

// FertilityLevel is the legacy calendar-based fertility level (spec §17 mapping table).
// PHP: backend/app/Enums/CycleSubphase.php:93.
func (e CycleSubphase) FertilityLevel() FertilityLevel {
	switch e {
	case CycleSubphaseOvulationLikely:
		return FertilityLevelVeryHigh
	case CycleSubphaseHighFertility:
		return FertilityLevelHigh
	case CycleSubphaseFertileRising, CycleSubphasePostOvulation:
		return FertilityLevelMedium
	default:
		return FertilityLevelLow
	}
}

// Canonical is the content-equivalent sub-phase the v1.1 aliases collapse onto.
// PHP: backend/app/Enums/CycleSubphase.php:115.
func (e CycleSubphase) Canonical() CycleSubphase {
	switch e {
	case CycleSubphaseMenstrual, CycleSubphaseMenstrualPossible:
		return CycleSubphaseMenstruation
	case CycleSubphaseLateFollicularTransition:
		return CycleSubphaseMidFollicular
	case CycleSubphaseUnknown:
		return CycleSubphasePeriodExpected
	default:
		return e
	}
}

// CycleSubphaseContentBacked returns the sub-phases with their own seeded content, in case order
// (every case except the v1.1 aliases). PHP: backend/app/Enums/CycleSubphase.php:160.
func CycleSubphaseContentBacked() []CycleSubphase {
	aliases := []CycleSubphase{
		CycleSubphaseMenstrual,
		CycleSubphaseMenstrualPossible,
		CycleSubphaseLateFollicularTransition,
		CycleSubphaseUnknown,
	}
	out := make([]CycleSubphase, 0, len(cycleSubphaseCases))
	for _, c := range cycleSubphaseCases {
		if !slices.Contains(aliases, c) {
			out = append(out, c)
		}
	}
	return out
}

// CycleSubphaseOptions returns {value, label} pairs for the content-backed sub-phases only (the
// admin phase list). PHP: backend/app/Enums/CycleSubphase.php:137.
func CycleSubphaseOptions(locale string) []Option {
	return optionsOf(CycleSubphaseContentBacked(), locale)
}

// CycleSubphaseLabelFor labels a stored key; ok is false for an unknown or empty key (PHP null).
// PHP: backend/app/Enums/CycleSubphase.php:148.
func CycleSubphaseLabelFor(value, locale string) (string, bool) {
	return labelFor(CycleSubphaseFrom, value, locale)
}

// Subphases lists the sub-phases CyclePhaseMapper can emit inside this phase.
// PHP: backend/app/Enums/CyclePhase.php:70.
func (e CyclePhase) Subphases() []CycleSubphase {
	switch e {
	case CyclePhaseMenstruation:
		return []CycleSubphase{CycleSubphaseMenstruation}
	case CyclePhaseFollicular:
		return []CycleSubphase{
			CycleSubphaseEarlyFollicular,
			CycleSubphaseMidFollicular,
			CycleSubphaseFertileRising,
			CycleSubphaseHighFertility,
		}
	case CyclePhaseOvulation:
		return []CycleSubphase{CycleSubphaseOvulationLikely, CycleSubphasePostOvulation}
	case CyclePhaseLuteal:
		return []CycleSubphase{
			CycleSubphaseEarlyLuteal,
			CycleSubphaseMidLuteal,
			CycleSubphaseLateLuteal,
			CycleSubphasePmsPossible,
		}
	}
	return nil
}

// CyclePhaseAllSubphases returns every emittable sub-phase, phase by phase.
// PHP: backend/app/Enums/CyclePhase.php:98.
func CyclePhaseAllSubphases() []CycleSubphase {
	var out []CycleSubphase
	for _, p := range cyclePhaseCases {
		out = append(out, p.Subphases()...)
	}
	return out
}

// CyclePhaseSubphaseValuesFor returns the sub-phase keys valid for phase, or every emittable key when
// phase is empty (PHP null) or unknown. PHP: backend/app/Enums/CyclePhase.php:109.
func CyclePhaseSubphaseValuesFor(phase string) []string {
	subs := CyclePhaseAllSubphases()
	if p, ok := CyclePhaseFrom(phase); ok {
		subs = p.Subphases()
	}
	return valuesOf(subs)
}

// CyclePhaseLabelFor labels a stored phase key; ok is false for an unknown or empty key (PHP null).
// PHP: backend/app/Enums/CyclePhase.php:134.
func CyclePhaseLabelFor(value, locale string) (string, bool) {
	return labelFor(CyclePhaseFrom, value, locale)
}

// LegacyPhase is the pre-v1.1 four-phase value; ok is false for unknown (PHP null).
// PHP: backend/app/Enums/MainPhase.php:47.
func (e MainPhase) LegacyPhase() (CyclePhase, bool) {
	switch e {
	case MainPhaseMenstrual:
		return CyclePhaseMenstruation, true
	case MainPhaseFollicular:
		return CyclePhaseFollicular, true
	case MainPhaseFertile:
		return CyclePhaseOvulation, true
	case MainPhaseLuteal, MainPhasePeriodExpected:
		return CyclePhaseLuteal, true
	}
	return "", false
}

// UncertaintyRange is the ± day window for this variability. PHP: backend/app/Enums/CycleVariability.php:27.
func (e CycleVariability) UncertaintyRange() int {
	switch e {
	case CycleVariabilityRegular:
		return 1
	case CycleVariabilitySemiIrregular:
		return 2
	case CycleVariabilityIrregular:
		return 3
	}
	return 0
}

// CycleVariabilityFromStdDev classifies a cycle-length standard deviation: ≤4 regular, ≤7 semi-irregular.
// PHP: backend/app/Enums/CycleVariability.php:36.
func CycleVariabilityFromStdDev(stdDev float64) CycleVariability {
	switch {
	case stdDev <= 4:
		return CycleVariabilityRegular
	case stdDev <= 7:
		return CycleVariabilitySemiIrregular
	}
	return CycleVariabilityIrregular
}

// RegularityStatusFromCycleLengths: fewer than 3 valid cycles is not_enough_data; otherwise a spread
// (max-min) of ≤7 days reads as relatively regular. PHP: backend/app/Enums/RegularityStatus.php:44.
func RegularityStatusFromCycleLengths(lengths []int) RegularityStatus {
	const minCycles, regularRangeDays = 3, 7
	if len(lengths) < minCycles {
		return RegularityStatusNotEnoughData
	}
	if slices.Max(lengths)-slices.Min(lengths) <= regularRangeDays {
		return RegularityStatusRelativelyRegular
	}
	return RegularityStatusIrregularPossible
}

// RequiresUserInput reports warnings that only clear once the user logs or confirms data (§33).
// PHP: backend/app/Enums/CycleWarning.php:31.
func (e CycleWarning) RequiresUserInput() bool {
	switch e {
	case CycleWarningPeriodEndMissingWarningCapExceeded,
		CycleWarningPeriodEndMissingHardCapExceeded,
		CycleWarningCycleUnresolvedAfterExpectedPeriod,
		CycleWarningInsufficientAnchorData:
		return true
	}
	return false
}

// IsPredicted is the backward-compat is_predicted flag: everything but a fully user-logged resolution.
// PHP: backend/app/Enums/ResolutionSource.php:23.
func (e ResolutionSource) IsPredicted() bool {
	return e != ResolutionSourceUserLogged
}

// Priority is the display priority when several statuses describe one day (highest wins).
// PHP: backend/app/Enums/DataStatus.php:39.
func (e DataStatus) Priority() int {
	switch e {
	case DataStatusActual:
		return 4
	case DataStatusIncomplete:
		return 3
	case DataStatusNeedsConfirmation:
		return 2
	case DataStatusPredicted:
		return 1
	}
	return 0
}

// ExcludesFromPrediction reports flags whose records are held out of the prediction medians (spec §8).
// PHP: backend/app/Enums/DataQualityFlag.php:46.
func (e DataQualityFlag) ExcludesFromPrediction() bool {
	switch e {
	case DataQualityFlagOutlierLongCycle, DataQualityFlagOutlierShortCycle, DataQualityFlagIncompleteEndMissing:
		return true
	}
	return false
}

// IsActual reports a real, user-owned period fact (not an estimate or prediction).
// PHP: backend/app/Enums/DataSource.php:45.
func (e DataSource) IsActual() bool {
	return e == DataSourceUserLogged || e == DataSourceUserProfileConfirmed
}

// labelFor is the shared body of the PHP labelFor(?string, locale) helpers:
// tryFrom($value)?->label($locale), with null (here: ok=false) for an unknown key.
func labelFor[E labeled](from func(string) (E, bool), value, locale string) (string, bool) {
	e, ok := from(value)
	if !ok {
		return "", false
	}
	return e.Label(locale), true
}
