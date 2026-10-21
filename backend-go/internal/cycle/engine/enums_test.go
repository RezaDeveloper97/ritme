package engine

import (
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func names[E ~uint8](n int) []string {
	out := make([]string, 0, n+1)
	for i := range n + 1 { // one past the end must answer ""
		out = append(out, any(E(i)).(fmt.Stringer).String())
	}
	return out
}

func TestEnums_SpecSpelling(t *testing.T) {
	assert.Equal(t, []string{"unknown", "menstrual", "follicular", "fertile", "luteal", "period_expected", ""},
		names[MainPhase](6), "§2.13")
	assert.Equal(t, []string{"unknown", "menstrual", "menstrual_possible", "early_follicular", "mid_follicular",
		"late_follicular_transition", "fertile_rising", "high_fertility", "ovulation_likely", "post_ovulation",
		"early_luteal", "mid_luteal", "late_luteal", "pms_possible", "period_expected", ""}, names[Subphase](15), "§2.14")
	assert.Equal(t, []string{"unknown", "low", "medium", "high", "peak", ""}, names[FertilityLevel](5), "§2.26")
	assert.Equal(t, []string{"unknown", "user_logged", "onboarding_declared", "predicted_reference", ""},
		names[StartSource](4), "§2.10")
	assert.Equal(t, []string{"unknown", "user_logged", "assumed_from_effective_duration", ""}, names[EndSource](3))
	assert.Equal(t, []string{"unknown", "bbt_confirmed", "lh_positive", "calendar", ""}, names[OvulationSource](4))
	assert.Equal(t, []string{"insufficient_data", "regular", "irregular", ""}, names[CycleRegularity](3), "§2.27")
	assert.Equal(t, []string{"insufficient", "good", "partial", "poor", ""}, names[DataQuality](4), "§2.28")
	assert.Equal(t, []string{"unknown", "low", "medium", "high", ""}, names[Confidence](4), "§2.29")
	assert.Equal(t, []string{"unknown", "user_logged", "user_logged_with_assumed_end", "prediction", "onboarding_based",
		"default_based", ""}, names[ResolutionSource](6), "§2.30")
	assert.Equal(t, []string{"frequent", "normal", "infrequent", ""}, names[Frequency](3))
	assert.Equal(t, []string{"normal", "prolonged", ""}, names[Duration](2))
	assert.Equal(t, []string{"cycle", "ttc", "pregnancy", "postpartum", "menopause", "teen", ""}, names[Mode](6))
	assert.Equal(t, []string{"", "hormonal_contraception", "pregnancy", "postpartum_not_returned", ""},
		names[SuppressionReason](4), "E-24")
	assert.Equal(t, []string{"user_logged", "bleeding_log", ""}, names[PeriodSource](2))
	assert.Equal(t, []string{"default", "history", "declared", ""}, names[LengthSource](3))
	assert.Equal(t, []string{"edit", "start", "end", ""}, names[RingAction](3), "E-17")
	assert.Equal(t, []string{"", "end_missing", "overdue", "ask_last_period", ""}, names[Banner](4), "E-17")
}

func TestEnums_ConfidenceRank(t *testing.T) {
	assert.Equal(t, ConfidenceMedium, min(ConfidenceHigh, ConfidenceMedium), "E-03 min(today, medium)")
	assert.Equal(t, ConfidenceLow, min(ConfidenceLow, ConfidenceMedium))
	assert.Equal(t, ConfidenceUnknown, min(ConfidenceUnknown, ConfidenceHigh))
}

func TestSubphase_Main(t *testing.T) {
	want := map[MainPhase][]Subphase{ // §2.15 table
		MainPhaseMenstrual:      {SubphaseMenstrual, SubphaseMenstrualPossible},
		MainPhaseFollicular:     {SubphaseEarlyFollicular, SubphaseMidFollicular, SubphaseLateFollicularTransition},
		MainPhaseFertile:        {SubphaseFertileRising, SubphaseHighFertility, SubphaseOvulationLikely},
		MainPhaseLuteal:         {SubphasePostOvulation, SubphaseEarlyLuteal, SubphaseMidLuteal, SubphaseLateLuteal, SubphasePMSPossible},
		MainPhasePeriodExpected: {SubphasePeriodExpected},
		MainPhaseUnknown:        {SubphaseUnknown},
	}
	n := 0
	for main, subs := range want {
		for _, s := range subs {
			assert.Equal(t, main, s.Main(), s.String())
			n++
		}
	}
	assert.Equal(t, 15, n)
}

func TestParseMode(t *testing.T) {
	for m := range modeCount {
		got, ok := ParseMode(m.String())
		assert.True(t, ok)
		assert.Equal(t, m, got)
	}
	_, ok := ParseMode("nope")
	assert.False(t, ok)
}

func TestWarningSet(t *testing.T) {
	assert.Equal(t, 21, int(warningCount), "§2.32 (20) + contradictory_anchor_data (E-02)")
	assert.Equal(t, "lh_bbt_conflict", WarnLHBBTConflict.String())
	assert.Equal(t, "contradictory_anchor_data", WarnContradictoryAnchorData.String())

	var s WarningSet
	s.Add(WarnContradictoryAnchorData)
	s.Add(WarnPeriodEndMissing)
	s.Add(WarnLowHistory)
	s.Add(WarnLowHistory)
	assert.Equal(t, 3, s.Len())
	assert.True(t, s.Has(WarnLowHistory))
	assert.False(t, s.Has(WarnLHBBTConflict))
	assert.Equal(t, []Warning{WarnPeriodEndMissing, WarnLowHistory, WarnContradictoryAnchorData}, slices.Collect(s.All()),
		"enum order, not insertion order")
	assert.True(t, s.RequiresUserInput(), "contradictory_anchor_data (§2.33)")
	s.Remove(WarnContradictoryAnchorData)
	assert.False(t, s.RequiresUserInput())

	all := WarningSet(1<<warningCount - 1)
	var ui []string
	for w := range all.All() {
		if w.RequiresUserInput() {
			ui = append(ui, w.String())
		}
	}
	assert.ElementsMatch(t, []string{"period_end_missing_warning_cap_exceeded", "period_end_missing_hard_cap_exceeded",
		"cycle_unresolved_after_expected_period", "insufficient_anchor_data", "contradictory_anchor_data",
		"stale_onboarding_anchor"}, ui, "§2.33")

	allocs := testing.AllocsPerRun(100, func() {
		n := 0
		for w := range all.All() {
			n += len(w.String())
		}
		_ = n
	})
	assert.Zero(t, allocs)
}

func TestReasonSet(t *testing.T) {
	assert.Equal(t, 14, int(reasonCount), "§2.29")
	want := []string{"low_history", "high_cycle_variability", "missing_period_end",
		"missing_period_end_beyond_warning_cap", "missing_period_end_beyond_hard_cap", "onboarding_data_used",
		"default_values_used", "prediction_based_output", "period_overdue", "unresolved_cycle", "poor_data_quality",
		"insufficient_anchor_data", "stale_onboarding_anchor", "hormonal_contraception_active"}
	all := ReasonSet(1<<reasonCount - 1)
	var got []string
	for r := range all.All() {
		got = append(got, r.String())
	}
	assert.Equal(t, want, got)
	assert.Equal(t, 14, all.Len())

	var s ReasonSet
	s.Add(ReasonHormonalContraceptionActive)
	s.Add(ReasonLowHistory)
	assert.Equal(t, []Reason{ReasonLowHistory, ReasonHormonalContraceptionActive}, slices.Collect(s.All()))
	s.Remove(ReasonLowHistory)
	assert.False(t, s.Has(ReasonLowHistory))
	assert.Zero(t, testing.AllocsPerRun(100, func() {
		for r := range all.All() {
			_ = r
		}
	}))
}

func TestStatus_IsPredicted(t *testing.T) {
	s := Status{ResolutionSource: ResolutionUserLogged}
	assert.False(t, s.IsPredicted(), "§2.31")
	for _, r := range []ResolutionSource{ResolutionUserLoggedWithAssumedEnd, ResolutionPrediction,
		ResolutionOnboardingBased, ResolutionDefaultBased, ResolutionUnknown} {
		s.ResolutionSource = r
		assert.True(t, s.IsPredicted(), r.String())
	}
}
