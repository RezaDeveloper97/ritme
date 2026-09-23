// Package resolver is the port of the v1.1 cycle engine core
// (backend/app/Services/HealthEngine/CycleStatusResolver.php, CycleStatus.php, CyclePhaseMapper.php).
//
// Resolve priority (§20): actual/assumed menstrual → period_expected → fertile display zone →
// luteal → follicular → unknown, with the §12 soft/warning/hard caps for a missing period end, the
// §22 overdue model, then the onboarding/default resolution override, warning dedup (first
// occurrence kept), data_quality (§28) and confidence (§29) + reasons. Future targets roll the
// anchor forward by the effective cycle length (predicted_reference); targets ≤ today do not.
//
// # Test mapping
//
// backend/tests/Unit/CycleStatusResolverTest.php → TestCycleStatusResolver (same-named subtests):
//
//	test_21_day_cycle_walk_matches_the_spec_example
//	test_menstrual_day_with_logged_end_is_user_logged_and_high_confidence
//	test_luteal_day_resolves_via_prediction_even_with_full_logs
//	test_open_period_within_soft_cap_is_menstrual_with_assumed_end
//	test_open_period_within_warning_cap_is_menstrual_possible
//	test_open_period_past_warning_cap_degrades_quality_and_asks_for_input
//	test_open_period_past_hard_cap_resolves_the_cycle_onward
//	test_period_expected_at_seven_days_late_is_calm
//	test_period_expected_at_eight_days_late_warns_and_drops_confidence
//	test_period_expected_past_fourteen_days_late_goes_unknown_but_keeps_anchors
//	test_a_newly_logged_period_cancels_period_expected
//	test_long_bleed_empties_the_display_fertile_window
//	test_28_day_cycle_follicular_split_and_fertile_tiers
//	test_onboarding_only_resolve_is_onboarding_based_and_low_confidence
//	test_defaults_only_resolve_is_default_based
//	test_no_anchor_at_all_is_insufficient_and_unknown
//	test_future_dates_roll_the_anchor_forward_as_predicted_reference
//	test_cycle_day_is_one_based
//
// backend/tests/Unit/CyclePhaseMapperTest.php → TestCyclePhaseMapper (same-named subtests):
//
//	test_phase_boundaries_follow_the_ovulation_day
//	test_subphase_walks_the_whole_cycle
//	test_every_day_maps_to_some_subphase_without_gaps
//	test_subphase_maps_to_the_spec_fertility_level
package resolver

import (
	"math"
	"slices"

	"github.com/ritme/backend-go/internal/cycle/metrics"
	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

const (
	lutealLength               = 14
	fertileDaysBeforeOvulation = 5
	periodHardCap              = 12
	warningCapExtraDays        = 3
	warningCapMinDays          = 8
	maxLateDays                = 14
)

// Confidence-reason codes (§29.6), CycleStatusResolver::REASON_*.
const (
	ReasonLowHistory              = "low_history"
	ReasonHighCycleVariability    = "high_cycle_variability"
	ReasonMissingPeriodEnd        = "missing_period_end"
	ReasonMissingEndBeyondWarnCap = "missing_period_end_beyond_warning_cap"
	ReasonMissingEndBeyondHardCap = "missing_period_end_beyond_hard_cap"
	ReasonOnboardingDataUsed      = "onboarding_data_used"
	ReasonDefaultValuesUsed       = "default_values_used"
	ReasonPredictionBasedOutput   = "prediction_based_output"
	ReasonPeriodOverdue           = "period_overdue"
	ReasonUnresolvedCycle         = "unresolved_cycle"
	ReasonPoorDataQuality         = "poor_data_quality"
	ReasonInsufficientAnchorData  = "insufficient_anchor_data"
)

func ptr(v int) *int { return &v }

// between is Carbon betweenIncluded.
func between(d, from, to civildate.Date) bool { return !d.Before(from) && !d.After(to) }

// daysBetween is the signed whole-day difference to − from: (int) round(diffInDays).
func daysBetween(from, to civildate.Date) int { return from.DiffDays(to) }

// Resolve is CycleStatusResolver::resolve. histories in any order; profile nil = no profile row.
// today separates the prediction regimes (future dates ride predicted cycles).
func Resolve(histories []model.History, profile *model.Profile, target, today civildate.Date, m metrics.Metrics) Status {
	start, startSource, loggedEnd, ok := anchorFor(histories, profile, target)
	if !ok {
		return unresolved(target, m)
	}

	ecl := max(1, m.EffectiveCycleLength)
	epl := max(1, m.EffectivePeriodDuration)

	if target.After(today) {
		for !start.AddDays(ecl).After(target) {
			start = start.AddDays(ecl)
			startSource = enums.PeriodStartSourcePredictedReference
			loggedEnd = civildate.Date{}
		}
	}

	endConfirmed := !loggedEnd.IsZero() && !loggedEnd.Before(start)
	end := start.AddDays(epl - 1)
	endSource := enums.PeriodEndSourceAssumedFromEffectiveDuration
	if endConfirmed {
		end = loggedEnd
		endSource = enums.PeriodEndSourceUserLogged
	}

	predictedNext := start.AddDays(ecl)
	ovulation := predictedNext.AddDays(-lutealLength)
	biologicalFertileStart := ovulation.AddDays(-fertileDaysBeforeOvulation)
	dayAfterPeriodEnd := end.AddDays(1)
	displayFertileStart := dayAfterPeriodEnd
	if biologicalFertileStart.After(dayAfterPeriodEnd) {
		displayFertileStart = biologicalFertileStart
	}
	displayFertileEnd := ovulation
	fertileWindowEmpty := displayFertileEnd.Before(displayFertileStart)

	cycleDay := daysBetween(start, target) + 1
	rawDaysToPeriod := daysBetween(target, predictedNext)
	daysToPeriod := max(0, rawDaysToPeriod)
	daysLate := max(0, -rawDaysToPeriod)
	daysToOvulation := daysBetween(target, ovulation)

	softCap := epl
	warningCap := max(epl+warningCapExtraDays, warningCapMinDays)
	isRealOpenPeriod := startSource == enums.PeriodStartSourceUserLogged && loggedEnd.IsZero()
	pastWarningCapOpen := isRealOpenPeriod && cycleDay > warningCap

	var warnings []enums.CycleWarning
	requiresInput := false
	forcedPoor := false
	forcedLowConfidence := false
	anchorContradiction := false

	var main enums.MainPhase
	var sub enums.CycleSubphase
	var resolution enums.ResolutionSource // "" = PHP null

	// 1. actual / assumed-active menstrual (§20, §21)
	if endConfirmed && between(target, start, end) {
		main, sub, resolution = enums.MainPhaseMenstrual, enums.CycleSubphaseMenstrual, enums.ResolutionSourceUserLogged
	} else if !endConfirmed && !target.Before(start) {
		switch {
		case isRealOpenPeriod:
			switch {
			case cycleDay <= softCap:
				main, sub, resolution = enums.MainPhaseMenstrual, enums.CycleSubphaseMenstrual, enums.ResolutionSourceUserLoggedWithAssumedEnd
			case cycleDay <= warningCap:
				main, sub, resolution = enums.MainPhaseMenstrual, enums.CycleSubphaseMenstrualPossible, enums.ResolutionSourceUserLoggedWithAssumedEnd
				warnings = append(warnings, enums.CycleWarningPeriodEndMissing)
			case cycleDay <= periodHardCap:
				main, sub, resolution = enums.MainPhaseMenstrual, enums.CycleSubphaseMenstrualPossible, enums.ResolutionSourceUserLoggedWithAssumedEnd
				warnings = append(warnings, enums.CycleWarningPeriodEndMissing, enums.CycleWarningPeriodEndMissingWarningCapExceeded)
				forcedPoor, forcedLowConfidence, requiresInput = true, true, true
			default:
				warnings = append(warnings, enums.CycleWarningPeriodEndMissingHardCapExceeded)
				forcedPoor, requiresInput = true, true
			}
		case cycleDay <= softCap:
			main, sub = enums.MainPhaseMenstrual, enums.CycleSubphaseMenstrual
			if startSource == enums.PeriodStartSourceUserLogged {
				resolution = enums.ResolutionSourceUserLoggedWithAssumedEnd
			}
		}
	}

	// 2. period_expected (§22)
	if main == "" && !target.Before(predictedNext) {
		switch {
		case daysLate <= 7:
			main, sub, resolution = enums.MainPhasePeriodExpected, enums.CycleSubphasePeriodExpected, enums.ResolutionSourcePrediction
		case daysLate <= maxLateDays:
			main, sub, resolution = enums.MainPhasePeriodExpected, enums.CycleSubphasePeriodExpected, enums.ResolutionSourcePrediction
			warnings = append(warnings, enums.CycleWarningPredictedPeriodOverdue)
			forcedLowConfidence = true
		default:
			main, sub, resolution = enums.MainPhaseUnknown, enums.CycleSubphaseUnknown, enums.ResolutionSourcePrediction
			warnings = append(warnings, enums.CycleWarningPredictedPeriodOverdue, enums.CycleWarningCycleUnresolvedAfterExpectedPeriod)
			forcedLowConfidence, forcedPoor, requiresInput = true, true, true
		}
	}

	// 3. fertile display zone (§19, §24)
	if main == "" && !fertileWindowEmpty && between(target, displayFertileStart, displayFertileEnd) {
		main = enums.MainPhaseFertile
		switch {
		case daysToOvulation >= 3:
			sub = enums.CycleSubphaseFertileRising
		case daysToOvulation >= 1:
			sub = enums.CycleSubphaseHighFertility
		default:
			sub = enums.CycleSubphaseOvulationLikely
		}
		resolution = enums.ResolutionSourcePrediction
	}

	// 4. luteal (§25)
	if main == "" && target.After(ovulation) && target.Before(predictedNext) {
		daysSinceOvulation := daysBetween(ovulation, target)
		daysUntilPeriod := daysBetween(target, predictedNext)
		main = enums.MainPhaseLuteal
		switch {
		case daysSinceOvulation == 1:
			sub = enums.CycleSubphasePostOvulation
		case daysUntilPeriod >= 1 && daysUntilPeriod <= 3:
			sub = enums.CycleSubphasePmsPossible
		case daysUntilPeriod >= 4 && daysUntilPeriod <= 6:
			sub = enums.CycleSubphaseLateLuteal
		default:
			sub = earlyOrMidLuteal(target, ovulation, predictedNext)
		}
		resolution = enums.ResolutionSourcePrediction
	}

	// 5. follicular (§23)
	if main == "" {
		follicularStart := end.AddDays(1)
		follicularEnd := displayFertileStart.AddDays(-1)
		if !follicularEnd.Before(follicularStart) && between(target, follicularStart, follicularEnd) {
			main = enums.MainPhaseFollicular
			sub = follicularSubphase(target, follicularStart, follicularEnd)
			resolution = enums.ResolutionSourcePrediction
		}
	}

	// 6. unknown fallback (§20)
	if main == "" {
		main, sub, resolution = enums.MainPhaseUnknown, enums.CycleSubphaseUnknown, enums.ResolutionSourceUnknown
		warnings = append(warnings, enums.CycleWarningInsufficientAnchorData)
		requiresInput = true
		anchorContradiction = true
	}

	// §30 onboarding/default override.
	if startSource == enums.PeriodStartSourceOnboardingDeclared &&
		resolution != enums.ResolutionSourceUnknown &&
		main != enums.MainPhasePeriodExpected &&
		(main != enums.MainPhaseUnknown || daysLate <= maxLateDays) {
		resolution = onboardingResolution(m)
	} else if resolution == "" {
		resolution = enums.ResolutionSourcePrediction
	}

	// §32 data-signal warnings.
	rng := m.CycleVariabilityRange
	highVariability := rng != nil && *rng > 5
	if m.ValidCyclesCount < 2 {
		warnings = append(warnings, enums.CycleWarningLowHistory)
	}
	if highVariability {
		warnings = append(warnings, enums.CycleWarningHighCycleVariability)
	}
	if m.HasShortCycleOutlier {
		warnings = append(warnings, enums.CycleWarningShortCycleOutlierDetected)
	}
	if m.HasLongCycleOutlier {
		warnings = append(warnings, enums.CycleWarningLongCycleOutlierDetected)
	}
	onboardingUsed := startSource == enums.PeriodStartSourceOnboardingDeclared ||
		m.CycleLengthSource == enums.EffectiveSourceProfile ||
		m.PeriodDurationSource == enums.EffectiveSourceProfile
	defaultsUsed := m.CycleLengthSource == enums.EffectiveSourceDefault ||
		m.PeriodDurationSource == enums.EffectiveSourceDefault
	if onboardingUsed {
		warnings = append(warnings, enums.CycleWarningOnboardingDataUsed)
	}
	if defaultsUsed {
		warnings = append(warnings, enums.CycleWarningDefaultValuesUsed)
	}
	if resolution == enums.ResolutionSourcePrediction || startSource == enums.PeriodStartSourcePredictedReference {
		warnings = append(warnings, enums.CycleWarningPredictionBasedOutput)
	}

	warningValues := make([]string, 0, len(warnings))
	for _, w := range warnings {
		if !slices.Contains(warningValues, string(w)) {
			warningValues = append(warningValues, string(w))
		}
		if w.RequiresUserInput() {
			requiresInput = true
		}
	}

	// data_quality (§28): poor → good → partial.
	endInvolved := main == enums.MainPhaseMenstrual ||
		main == enums.MainPhaseFollicular ||
		(main == enums.MainPhaseFertile && displayFertileStart == end.AddDays(1))
	var dataQuality enums.DataQualityLevel
	switch {
	case forcedPoor || pastWarningCapOpen || daysLate > maxLateDays || anchorContradiction || m.OnlyOutlierHistory:
		dataQuality = enums.DataQualityLevelPoor
	case startSource == enums.PeriodStartSourceUserLogged &&
		m.ValidCyclesCount >= 2 &&
		main != enums.MainPhaseUnknown &&
		daysLate <= 7 &&
		m.CycleLengthSource == enums.EffectiveSourceRecentValidCycles &&
		m.PeriodDurationSource == enums.EffectiveSourceRecentValidCycles &&
		(!endInvolved || endConfirmed):
		dataQuality = enums.DataQualityLevelGood
	default:
		dataQuality = enums.DataQualityLevelPartial
	}

	// confidence (§29): unknown → low → high → medium.
	var confidence enums.ConfidenceLevel
	switch {
	case resolution == enums.ResolutionSourceUnknown:
		confidence = enums.ConfidenceLevelUnknown
	case forcedLowConfidence ||
		main == enums.MainPhaseUnknown ||
		daysLate > 7 ||
		dataQuality == enums.DataQualityLevelPoor ||
		pastWarningCapOpen ||
		m.ValidCyclesCount < 2 ||
		highVariability ||
		resolution == enums.ResolutionSourceDefaultBased || resolution == enums.ResolutionSourceOnboardingBased:
		confidence = enums.ConfidenceLevelLow
	case dataQuality == enums.DataQualityLevelGood &&
		m.ValidCyclesCount >= 2 &&
		m.ValidPeriodDurationsCount >= 2 &&
		startSource == enums.PeriodStartSourceUserLogged &&
		(!endInvolved || endConfirmed) &&
		rng != nil && *rng <= 5 &&
		main != enums.MainPhasePeriodExpected && main != enums.MainPhaseUnknown:
		confidence = enums.ConfidenceLevelHigh
	default:
		confidence = enums.ConfidenceLevelMedium
	}

	reasons := confidenceReasons(m, highVariability, isRealOpenPeriod, cycleDay, warningCap, onboardingUsed,
		defaultsUsed, resolution, startSource, daysLate, main, dataQuality, anchorContradiction)

	fertility := sub.FertilityLevelV11()
	if main == enums.MainPhaseUnknown {
		fertility = enums.FertilityLevelUnknown
	}
	var dto *int
	if daysLate <= maxLateDays {
		dto = ptr(daysToOvulation)
	}

	return Status{
		Date:                        target,
		CycleDay:                    ptr(cycleDay),
		MainPhase:                   main,
		Subphase:                    sub,
		FertilityLevel:              fertility,
		DaysToOvulation:             dto,
		DaysToPeriod:                ptr(daysToPeriod),
		DaysLate:                    ptr(daysLate),
		CurrentPeriodStart:          start,
		CurrentPeriodStartSource:    startSource,
		CurrentPeriodEnd:            end,
		CurrentPeriodEndSource:      endSource,
		CurrentPeriodEndIsConfirmed: endConfirmed,
		PredictedNextPeriodStart:    predictedNext,
		EstimatedOvulationDate:      ovulation,
		EffectiveCycleLength:        ecl,
		EffectivePeriodLength:       epl,
		CycleVariability:            rng,
		Confidence:                  confidence,
		ConfidenceReasons:           reasons,
		DataQuality:                 dataQuality,
		ResolutionSource:            resolution,
		RequiresUserInput:           requiresInput,
		Warnings:                    warningValues,
	}
}

// anchorFor is the current-cycle start anchor (§10.1): the latest confirmed start ≤ target (with
// its logged end), else the latest is_estimated start ≤ target, else the profile LMP ≤ target.
// Ties on the start keep the first row in input order (PHP 8 stable sortByDesc + first()).
func anchorFor(histories []model.History, profile *model.Profile, target civildate.Date) (civildate.Date, enums.PeriodStartSource, civildate.Date, bool) {
	if h, ok := latestStart(histories, target, func(h model.History) bool { return h.IsConfirmed }); ok {
		return h.PeriodStart, enums.PeriodStartSourceUserLogged, h.PeriodEnd, true
	}
	if h, ok := latestStart(histories, target, func(h model.History) bool { return h.IsEstimated }); ok {
		return h.PeriodStart, enums.PeriodStartSourceOnboardingDeclared, civildate.Date{}, true
	}
	if profile != nil && !profile.LastPeriodStart.IsZero() && !profile.LastPeriodStart.After(target) {
		return profile.LastPeriodStart, enums.PeriodStartSourceOnboardingDeclared, civildate.Date{}, true
	}
	return civildate.Date{}, "", civildate.Date{}, false
}

func latestStart(histories []model.History, onOrBefore civildate.Date, keep func(model.History) bool) (model.History, bool) {
	var best model.History
	found := false
	for _, h := range histories {
		if !keep(h) || h.PeriodStart.After(onOrBefore) {
			continue
		}
		if !found || h.PeriodStart.After(best.PeriodStart) {
			best, found = h, true
		}
	}
	return best, found
}

// unresolved: no anchor at all — the output is still well-formed (§28.2 / §29.2 / §30).
func unresolved(target civildate.Date, m metrics.Metrics) Status {
	return Status{
		Date:                     target,
		MainPhase:                enums.MainPhaseUnknown,
		Subphase:                 enums.CycleSubphaseUnknown,
		FertilityLevel:           enums.FertilityLevelUnknown,
		CurrentPeriodStartSource: enums.PeriodStartSourceUnknown,
		CurrentPeriodEndSource:   enums.PeriodEndSourceUnknown,
		EffectiveCycleLength:     max(1, m.EffectiveCycleLength),
		EffectivePeriodLength:    max(1, m.EffectivePeriodDuration),
		CycleVariability:         m.CycleVariabilityRange,
		Confidence:               enums.ConfidenceLevelUnknown,
		ConfidenceReasons:        []string{ReasonInsufficientAnchorData},
		DataQuality:              enums.DataQualityLevelInsufficient,
		ResolutionSource:         enums.ResolutionSourceUnknown,
		RequiresUserInput:        true,
		Warnings:                 []string{string(enums.CycleWarningInsufficientAnchorData)},
	}
}

// follicularSubphase splits the follicular range 40/40/20 with the 1- and 2-day special cases (§23).
func follicularSubphase(target, rangeStart, rangeEnd civildate.Date) enums.CycleSubphase {
	length := daysBetween(rangeStart, rangeEnd) + 1
	position := daysBetween(rangeStart, target) + 1

	if length == 1 {
		return enums.CycleSubphaseMidFollicular
	}
	if length == 2 {
		if position == 1 {
			return enums.CycleSubphaseEarlyFollicular
		}
		return enums.CycleSubphaseLateFollicularTransition
	}

	early := max(1, int(math.Floor(float64(length)*0.4)))
	mid := max(1, int(math.Floor(float64(length)*0.4)))
	switch {
	case position <= early:
		return enums.CycleSubphaseEarlyFollicular
	case position <= early+mid:
		return enums.CycleSubphaseMidFollicular
	default:
		return enums.CycleSubphaseLateFollicularTransition
	}
}

// earlyOrMidLuteal splits O+2 .. P−7 into early / mid halves; an odd extra day goes early (§25.4).
func earlyOrMidLuteal(target, ovulation, predictedNext civildate.Date) enums.CycleSubphase {
	assignableStart := ovulation.AddDays(2)
	assignableEnd := predictedNext.AddDays(-7)

	if assignableEnd.Before(assignableStart) || !between(target, assignableStart, assignableEnd) {
		return enums.CycleSubphaseMidLuteal
	}

	length := daysBetween(assignableStart, assignableEnd) + 1
	earlyLength := int(math.Ceil(float64(length) / 2))
	if daysBetween(assignableStart, target) < earlyLength {
		return enums.CycleSubphaseEarlyLuteal
	}
	return enums.CycleSubphaseMidLuteal
}

// onboardingResolution: an onboarding anchor without declared lengths degrades to default_based (§30.1).
func onboardingResolution(m metrics.Metrics) enums.ResolutionSource {
	if m.CycleLengthSource == enums.EffectiveSourceDefault && m.PeriodDurationSource == enums.EffectiveSourceDefault {
		return enums.ResolutionSourceDefaultBased
	}
	return enums.ResolutionSourceOnboardingBased
}

// confidenceReasons lists every active cause behind the confidence grade (§29.6), deduplicated.
func confidenceReasons(
	m metrics.Metrics,
	highVariability, isRealOpenPeriod bool,
	cycleDay, warningCap int,
	onboardingUsed, defaultsUsed bool,
	resolution enums.ResolutionSource,
	startSource enums.PeriodStartSource,
	daysLate int,
	main enums.MainPhase,
	dataQuality enums.DataQualityLevel,
	anchorContradiction bool,
) []string {
	var reasons []string
	add := func(r string) {
		if !slices.Contains(reasons, r) {
			reasons = append(reasons, r)
		}
	}

	if m.ValidCyclesCount < 2 {
		add(ReasonLowHistory)
	}
	if highVariability {
		add(ReasonHighCycleVariability)
	}
	if isRealOpenPeriod {
		add(ReasonMissingPeriodEnd)
		if cycleDay > warningCap && cycleDay <= periodHardCap {
			add(ReasonMissingEndBeyondWarnCap)
		} else if cycleDay > periodHardCap {
			add(ReasonMissingEndBeyondHardCap)
		}
	}
	if onboardingUsed {
		add(ReasonOnboardingDataUsed)
	}
	if defaultsUsed {
		add(ReasonDefaultValuesUsed)
	}
	if resolution == enums.ResolutionSourcePrediction || startSource == enums.PeriodStartSourcePredictedReference {
		add(ReasonPredictionBasedOutput)
	}
	if daysLate > 0 {
		add(ReasonPeriodOverdue)
	}
	if main == enums.MainPhaseUnknown && daysLate > maxLateDays {
		add(ReasonUnresolvedCycle)
	}
	if dataQuality == enums.DataQualityLevelPoor {
		add(ReasonPoorDataQuality)
	}
	if anchorContradiction {
		add(ReasonInsufficientAnchorData)
	}
	if reasons == nil {
		return []string{}
	}
	return reasons
}
