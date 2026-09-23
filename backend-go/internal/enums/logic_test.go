package enums

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Table tests for the hand-ported logic, one row per PHP branch (file:line cited per test).

// PHP: backend/app/Enums/CycleSubphase.php:78 (fertilityLevelV11), :93 (fertilityLevel), :115 (canonical).
func TestCycleSubphaseLogic(t *testing.T) {
	tests := []struct {
		in        CycleSubphase
		v11, lgcy FertilityLevel
		canonical CycleSubphase
	}{
		{CycleSubphaseMenstruation, FertilityLevelLow, FertilityLevelLow, CycleSubphaseMenstruation},
		{CycleSubphaseMenstrual, FertilityLevelLow, FertilityLevelLow, CycleSubphaseMenstruation},
		{CycleSubphaseMenstrualPossible, FertilityLevelLow, FertilityLevelLow, CycleSubphaseMenstruation},
		{CycleSubphaseEarlyFollicular, FertilityLevelLow, FertilityLevelLow, CycleSubphaseEarlyFollicular},
		{CycleSubphaseMidFollicular, FertilityLevelLow, FertilityLevelLow, CycleSubphaseMidFollicular},
		{CycleSubphaseLateFollicularTransition, FertilityLevelMedium, FertilityLevelLow, CycleSubphaseMidFollicular},
		{CycleSubphaseFertileRising, FertilityLevelMedium, FertilityLevelMedium, CycleSubphaseFertileRising},
		{CycleSubphaseHighFertility, FertilityLevelHigh, FertilityLevelHigh, CycleSubphaseHighFertility},
		{CycleSubphaseOvulationLikely, FertilityLevelPeak, FertilityLevelVeryHigh, CycleSubphaseOvulationLikely},
		{CycleSubphasePostOvulation, FertilityLevelLow, FertilityLevelMedium, CycleSubphasePostOvulation},
		{CycleSubphaseEarlyLuteal, FertilityLevelLow, FertilityLevelLow, CycleSubphaseEarlyLuteal},
		{CycleSubphaseMidLuteal, FertilityLevelLow, FertilityLevelLow, CycleSubphaseMidLuteal},
		{CycleSubphaseLateLuteal, FertilityLevelLow, FertilityLevelLow, CycleSubphaseLateLuteal},
		{CycleSubphasePmsPossible, FertilityLevelLow, FertilityLevelLow, CycleSubphasePmsPossible},
		{CycleSubphasePeriodExpected, FertilityLevelUnknown, FertilityLevelLow, CycleSubphasePeriodExpected},
		{CycleSubphaseUnknown, FertilityLevelUnknown, FertilityLevelLow, CycleSubphasePeriodExpected},
	}
	assert.Len(t, tests, len(CycleSubphaseCases()))
	for _, tt := range tests {
		assert.Equal(t, tt.v11, tt.in.FertilityLevelV11(), "v11 %s", tt.in)
		assert.Equal(t, tt.lgcy, tt.in.FertilityLevel(), "legacy %s", tt.in)
		assert.Equal(t, tt.canonical, tt.in.Canonical(), "canonical %s", tt.in)
	}
}

// PHP: backend/app/Enums/CycleSubphase.php:137 (options), :160 (contentBacked).
func TestCycleSubphaseContentBackedAndOptions(t *testing.T) {
	cb := CycleSubphaseContentBacked()
	assert.Len(t, cb, 12)
	assert.NotContains(t, cb, CycleSubphaseMenstrual)
	assert.NotContains(t, cb, CycleSubphaseUnknown)
	assert.Equal(t, CycleSubphaseMenstruation, cb[0])
	assert.Equal(t, CycleSubphasePeriodExpected, cb[len(cb)-1])

	opts := CycleSubphaseOptions("en")
	assert.Len(t, opts, 12)
	assert.Equal(t, Option{Value: "menstruation", Label: "Menstruation"}, opts[0])
}

// PHP: backend/app/Enums/CyclePhase.php:70 (subphases), :98 (allSubphases), :109 (subphaseValuesFor),
// :134 (labelFor).
func TestCyclePhaseSubphases(t *testing.T) {
	assert.Equal(t, []CycleSubphase{CycleSubphaseMenstruation}, CyclePhaseMenstruation.Subphases())
	assert.Equal(t, []CycleSubphase{CycleSubphaseOvulationLikely, CycleSubphasePostOvulation}, CyclePhaseOvulation.Subphases())
	assert.Len(t, CyclePhaseFollicular.Subphases(), 4)
	assert.Len(t, CyclePhaseLuteal.Subphases(), 4)
	assert.Len(t, CyclePhaseAllSubphases(), 11)

	all := valuesOf(CyclePhaseAllSubphases())
	assert.Equal(t, all, CyclePhaseSubphaseValuesFor(""), "null phase → every key")
	assert.Equal(t, all, CyclePhaseSubphaseValuesFor("bogus"), "unknown phase → every key")
	assert.Equal(t, []string{"ovulation_likely", "post_ovulation"}, CyclePhaseSubphaseValuesFor("ovulation"))

	l, ok := CyclePhaseLabelFor("luteal", "fa")
	assert.True(t, ok)
	assert.Equal(t, "لوتئال", l)
	_, ok = CyclePhaseLabelFor("", "fa")
	assert.False(t, ok)
	_, ok = CycleSubphaseLabelFor("bogus", "en")
	assert.False(t, ok)
}

// PHP: backend/app/Enums/MainPhase.php:47.
func TestMainPhaseLegacyPhase(t *testing.T) {
	tests := []struct {
		in   MainPhase
		want CyclePhase
		ok   bool
	}{
		{MainPhaseMenstrual, CyclePhaseMenstruation, true},
		{MainPhaseFollicular, CyclePhaseFollicular, true},
		{MainPhaseFertile, CyclePhaseOvulation, true},
		{MainPhaseLuteal, CyclePhaseLuteal, true},
		{MainPhasePeriodExpected, CyclePhaseLuteal, true},
		{MainPhaseUnknown, "", false},
	}
	for _, tt := range tests {
		got, ok := tt.in.LegacyPhase()
		assert.Equal(t, tt.want, got, "%s", tt.in)
		assert.Equal(t, tt.ok, ok, "%s", tt.in)
	}
}

// PHP: backend/app/Enums/CycleVariability.php:27 (uncertaintyRange), :36 (fromStdDev).
func TestCycleVariability(t *testing.T) {
	tests := []struct {
		sd   float64
		want CycleVariability
	}{
		{0, CycleVariabilityRegular},
		{4, CycleVariabilityRegular},
		{4.0001, CycleVariabilitySemiIrregular},
		{7, CycleVariabilitySemiIrregular},
		{7.01, CycleVariabilityIrregular},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, CycleVariabilityFromStdDev(tt.sd), "sd %v", tt.sd)
	}
	assert.Equal(t, 1, CycleVariabilityRegular.UncertaintyRange())
	assert.Equal(t, 2, CycleVariabilitySemiIrregular.UncertaintyRange())
	assert.Equal(t, 3, CycleVariabilityIrregular.UncertaintyRange())
}

// PHP: backend/app/Enums/RegularityStatus.php:44.
func TestRegularityStatusFromCycleLengths(t *testing.T) {
	tests := []struct {
		in   []int
		want RegularityStatus
	}{
		{nil, RegularityStatusNotEnoughData},
		{[]int{28, 30}, RegularityStatusNotEnoughData},
		{[]int{25, 32, 28}, RegularityStatusRelativelyRegular}, // spread 7
		{[]int{25, 33, 28}, RegularityStatusIrregularPossible}, // spread 8
		{[]int{30, 30, 30, 30}, RegularityStatusRelativelyRegular},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, RegularityStatusFromCycleLengths(tt.in), "%v", tt.in)
	}
}

// PHP: backend/app/Enums/CycleWarning.php:31.
func TestCycleWarningRequiresUserInput(t *testing.T) {
	want := map[CycleWarning]bool{
		CycleWarningPeriodEndMissingWarningCapExceeded: true,
		CycleWarningPeriodEndMissingHardCapExceeded:    true,
		CycleWarningCycleUnresolvedAfterExpectedPeriod: true,
		CycleWarningInsufficientAnchorData:             true,
	}
	for _, w := range CycleWarningCases() {
		assert.Equal(t, want[w], w.RequiresUserInput(), "%s", w)
	}
}

// PHP: backend/app/Enums/ResolutionSource.php:23, DataStatus.php:39, DataQualityFlag.php:46,
// DataSource.php:45, ClotsAmount.php:37, EnergyLevel.php:41.
func TestSmallPredicates(t *testing.T) {
	for _, r := range ResolutionSourceCases() {
		assert.Equal(t, r != ResolutionSourceUserLogged, r.IsPredicted(), "%s", r)
	}

	assert.Equal(t, []int{4, 3, 2, 1}, []int{
		DataStatusActual.Priority(), DataStatusIncomplete.Priority(),
		DataStatusNeedsConfirmation.Priority(), DataStatusPredicted.Priority(),
	})

	excl := map[DataQualityFlag]bool{
		DataQualityFlagOutlierLongCycle:     true,
		DataQualityFlagOutlierShortCycle:    true,
		DataQualityFlagIncompleteEndMissing: true,
	}
	for _, f := range DataQualityFlagCases() {
		assert.Equal(t, excl[f], f.ExcludesFromPrediction(), "%s", f)
	}
	assert.Equal(t, DataQualityFlag("long_period_flag"), DataQualityFlagLongPeriod, "case name ≠ value kept")

	assert.True(t, DataSourceUserLogged.IsActual())
	assert.True(t, DataSourceUserProfileConfirmed.IsActual())
	assert.False(t, DataSourceEnginePrediction.IsActual())

	assert.False(t, ClotsAmountNone.IsPresent())
	assert.True(t, ClotsAmountLow.IsPresent())

	assert.Equal(t, []int{15, 35, 60, 80, 100}, []int{
		EnergyLevelVeryLow.Score(), EnergyLevelLow.Score(), EnergyLevelMedium.Score(),
		EnergyLevelHigh.Score(), EnergyLevelVeryHigh.Score(),
	})
}

// PHP: backend/app/Enums/BmiCategory.php:23.
func TestBmiCategoryFromBmi(t *testing.T) {
	tests := []struct {
		bmi  float64
		want BmiCategory
	}{
		{18.49, BmiCategoryUnderweight},
		{18.5, BmiCategoryNormal},
		{24.99, BmiCategoryNormal},
		{25, BmiCategoryOverweight},
		{29.99, BmiCategoryOverweight},
		{30, BmiCategoryObese},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, BmiCategoryFromBmi(tt.bmi), "%v", tt.bmi)
	}
}

type fakeLog struct {
	headache, pelvic, stomach, sleep, bloating *string
	moods                                      []string
	fatigue                                    *bool
}

func (f fakeLog) HeadacheIntensity() *string    { return f.headache }
func (f fakeLog) PelvicPainIntensity() *string  { return f.pelvic }
func (f fakeLog) StomachAcheIntensity() *string { return f.stomach }
func (f fakeLog) SleepQuality() *string         { return f.sleep }
func (f fakeLog) Moods() []string               { return f.moods }
func (f fakeLog) BloatingIntensity() *string    { return f.bloating }
func (f fakeLog) Fatigue() *bool                { return f.fatigue }

// PHP: backend/app/Enums/RecommendationTrigger.php:46 (matches), :90 (activeFor).
func TestRecommendationTriggerMatches(t *testing.T) {
	s := func(v string) *string { return &v }
	b := func(v bool) *bool { return &v }
	tests := []struct {
		name string
		log  TriggerLog
		want []string
	}{
		{"no log", nil, []string{}},
		{"empty log", fakeLog{}, []string{}},
		{"headache", fakeLog{headache: s("low")}, []string{"headache"}},
		{"cramps via pelvic", fakeLog{pelvic: s("high")}, []string{"cramps"}},
		{"cramps via stomach", fakeLog{stomach: s("medium")}, []string{"cramps"}},
		{"poor sleep", fakeLog{sleep: s("bad")}, []string{"poor_sleep"}},
		{"medium sleep is fine", fakeLog{sleep: s("medium")}, []string{}},
		{"anxious", fakeLog{moods: []string{"happy", "anxious"}}, []string{"low_mood"}},
		{"sad", fakeLog{moods: []string{"sad"}}, []string{"low_mood"}},
		{"other moods", fakeLog{moods: []string{"calm"}}, []string{}},
		{"bloating", fakeLog{bloating: s("low")}, []string{"bloating"}},
		{"fatigue true", fakeLog{fatigue: b(true)}, []string{"fatigue"}},
		{"fatigue false", fakeLog{fatigue: b(false)}, []string{}},
		{"all, case order", fakeLog{
			headache: s("low"), stomach: s("low"), sleep: s("bad"), moods: []string{"sad"},
			bloating: s("low"), fatigue: b(true),
		}, []string{"headache", "cramps", "poor_sleep", "low_mood", "bloating", "fatigue"}},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, RecommendationTriggerActiveFor(tt.log), tt.name)
	}
}

// PHP: backend/app/Enums/RecommendationType.php:109 (labelFor), :115 (iconFor); ReminderType.php icon().
func TestRecommendationTypeFallbacksAndIcons(t *testing.T) {
	assert.Equal(t, "Tip", RecommendationTypeLabelFor("", "en"))
	assert.Equal(t, "توصیه", RecommendationTypeLabelFor("bogus", "fa"))
	assert.Equal(t, "Sleep", RecommendationTypeLabelFor("sleep", "de"), "non-fa locale → English")
	assert.Equal(t, "sparkle", RecommendationTypeIconFor("bogus"))
	assert.Equal(t, "moon", RecommendationTypeIconFor("sleep"))
	assert.Equal(t, "stethoscope", ReminderTypeDoctor.Icon())
	assert.Equal(t, "calendar-clock", ReminderTypeAppointment.Icon())
}

func TestLabelLocaleRules(t *testing.T) {
	assert.Equal(t, "شاد", MoodHappy.Label("en"), "Persian-only label ignores locale")
	assert.Equal(t, "High sexual desire", SexualActivityHighDesire.Label("fa"), "English-only label ignores locale")
	assert.Equal(t, "AB", BloodTypeAb.Label("fa"), "label() returns the value")
	assert.Equal(t, "Menstrual Cycle", MessageModeCycle.Label("de"))
	assert.Equal(t, "Light", ClotsAmountLow.Label("de"))
	_, ok := CycleWarningFrom("period_end_missing")
	assert.True(t, ok)
	_, ok = CycleWarningFrom("PERIOD_END_MISSING")
	assert.False(t, ok)
}
