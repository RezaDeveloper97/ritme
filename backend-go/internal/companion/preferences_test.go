package companion

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/companion/store"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/validation"
)

func TestNormalizeActions_DedupesKeepingOrder(t *testing.T) {
	got, err := NormalizeActions([]ActionPref{ActionCalm, ActionFood, ActionCalm, ActionHelp, ActionFood})
	require.NoError(t, err)
	assert.Equal(t, []ActionPref{ActionCalm, ActionFood, ActionHelp}, got)

	_, err = NormalizeActions([]ActionPref{ActionFood, "flowers"})
	require.ErrorIs(t, err, ErrInvalidActions)

	got, err = NormalizeActions(nil)
	require.NoError(t, err)
	assert.Equal(t, []ActionPref{}, got)
}

// Focus goals combine in the picked order; «info» and «unknown» only stand alone (unknown = the empty list).
func TestNormalizeGoals(t *testing.T) {
	for _, tc := range []struct{ in, want []Goal }{
		{nil, []Goal{}},
		{[]Goal{GoalUnknown}, []Goal{}},
		{[]Goal{GoalInfo}, []Goal{GoalInfo}},
		{[]Goal{GoalInfo, GoalUnknown}, []Goal{GoalInfo}},
		{[]Goal{GoalSupport, GoalCycle, GoalSupport}, []Goal{GoalSupport, GoalCycle}},
		{[]Goal{GoalInfo, GoalQualityTime, GoalUnknown, GoalLowEnergy}, []Goal{GoalQualityTime, GoalLowEnergy}},
		{Goals, []Goal{GoalCycle, GoalSupport, GoalLowEnergy, GoalQualityTime}},
	} {
		got, err := NormalizeGoals(tc.in)
		require.NoError(t, err)
		assert.Equal(t, tc.want, got, "%v", tc.in)
	}
	_, err := NormalizeGoals([]Goal{GoalCycle, "romance"})
	require.ErrorIs(t, err, ErrInvalidGoal)
}

func TestPreferencesValidate(t *testing.T) {
	p := Preferences{}
	require.NoError(t, p.Validate())
	assert.Equal(t, DefaultPreferences(), p, "empty → no goal (unknown), no signals, no actions")
	assert.Equal(t, GoalUnknown, p.Goal())

	p = Preferences{Goals: []Goal{GoalSupport, "romance"}}
	require.ErrorIs(t, p.Validate(), ErrInvalidGoal)

	p = Preferences{Signals: SignalScope{"weight": true}}
	require.ErrorIs(t, p.Validate(), ErrInvalidSignal)
}

func TestPreferencesApply_PartialPatch(t *testing.T) {
	base := Preferences{Goals: []Goal{GoalSupport}, Signals: SignalScope{SignalPain: true}, Actions: []ActionPref{ActionGift}}
	goals := []Goal{GoalQualityTime, GoalCycle}
	acts := []ActionPref{ActionTime, ActionTime, ActionCalm}

	got, err := base.Apply(PreferencesPatch{Signals: SignalScope{SignalMood: true}})
	require.NoError(t, err)
	assert.Equal(t, []Goal{GoalSupport}, got.Goals, "goals not sent → unchanged")
	assert.True(t, got.Signals.On(SignalPain), "unsent signal keeps its value")
	assert.True(t, got.Signals.On(SignalMood))
	assert.Equal(t, []ActionPref{ActionGift}, got.Actions)
	assert.False(t, base.Signals.On(SignalMood), "Apply never mutates the receiver")

	got, err = base.Apply(PreferencesPatch{Goals: &goals, Signals: SignalScope{SignalPain: false}, Actions: &acts})
	require.NoError(t, err)
	assert.Equal(t, goals, got.Goals)
	assert.Equal(t, GoalQualityTime, got.Goal(), "the single goal is the first one picked")
	assert.False(t, got.Signals.On(SignalPain))
	assert.Equal(t, []ActionPref{ActionTime, ActionCalm}, got.Actions)

	bad := []Goal{"x"}
	_, err = base.Apply(PreferencesPatch{Goals: &bad})
	require.ErrorIs(t, err, ErrInvalidGoal)

	assert.True(t, PreferencesPatch{}.Empty())
}

func TestPreferencesColumnsRoundTrip(t *testing.T) {
	assert.False(t, goalColumn(nil).Valid, "no goal (unknown) is stored as NULL")
	assert.Equal(t, "support,cycle", goalColumn([]Goal{GoalSupport, GoalCycle}).String)
	assert.Equal(t, sql.NullString{}, actionsColumn(nil))
	assert.Equal(t, "calm,food", actionsColumn([]ActionPref{ActionCalm, ActionFood}).String)

	c := store.Companion{Goal: sql.NullString{String: "low_energy", Valid: true}, ShareMood: true, ShareSleep: true,
		ActionPrefs: sql.NullString{String: "help,bogus,food,help", Valid: true}}
	p := preferencesOf(c)
	assert.Equal(t, []Goal{GoalLowEnergy}, p.Goals, "a row written before the multi-select reads as one goal")
	assert.Equal(t, SignalScope{SignalPain: false, SignalMood: true, SignalEnergy: false, SignalSleep: true}, p.Signals)
	assert.Equal(t, []ActionPref{ActionHelp, ActionFood}, p.Actions, "unknown codes dropped, order kept")

	assert.Empty(t, preferencesOf(store.Companion{Goal: sql.NullString{String: "gone", Valid: true}}).Goals)
	assert.Equal(t, []Goal{GoalQualityTime, GoalSupport},
		preferencesOf(store.Companion{Goal: sql.NullString{String: "quality_time,gone,support", Valid: true}}).Goals)
}

func TestParsePreferences(t *testing.T) {
	body := validation.DecodeBody([]byte(`{"goal":"support","signals":{"pain":true,"mood":"0","sleep":null},"action_prefs":["gift","calm","gift"]}`))
	p, err := parsePreferences("en", body)
	require.NoError(t, err)
	require.NotNil(t, p.Goals)
	assert.Equal(t, []Goal{GoalSupport}, *p.Goals, "the single goal of an older client")
	assert.Equal(t, SignalScope{SignalPain: true, SignalMood: false}, p.Signals, "null = not sent")
	require.NotNil(t, p.Actions)
	assert.Equal(t, []ActionPref{ActionGift, ActionCalm}, *p.Actions)

	p, err = parsePreferences("en", validation.DecodeBody([]byte(`{"goal":"info","goals":["low_energy","cycle","low_energy"]}`)))
	require.NoError(t, err)
	require.NotNil(t, p.Goals)
	assert.Equal(t, []Goal{GoalLowEnergy, GoalCycle}, *p.Goals, "goals wins over goal")

	p, err = parsePreferences("en", validation.DecodeBody([]byte(`{"goals":[]}`)))
	require.NoError(t, err)
	require.NotNil(t, p.Goals, "an empty list clears the goals")
	assert.Empty(t, *p.Goals)

	p, err = parsePreferences("en", validation.DecodeBody([]byte(`{}`)))
	require.NoError(t, err)
	assert.True(t, p.Empty())

	_, err = parsePreferences("en", validation.DecodeBody([]byte(`{"signals":{"weight":true}}`)))
	assert.Error(t, err)
	_, err = parsePreferences("en", validation.DecodeBody([]byte(`{"action_prefs":["flowers"]}`)))
	assert.Error(t, err)
}

// The Laravel-style rules refuse unknown goals, non-boolean signals, unknown or too many action kinds.
func TestPreferenceRules(t *testing.T) {
	for raw, field := range map[string]string{
		`{"goal":"romance"}`:                                           "goal",
		`{"goals":["cycle","romance"]}`:                                "goals.1",
		`{"goals":"cycle"}`:                                            "goals",
		`{"signals":{"pain":"maybe"}}`:                                 "signals.pain",
		`{"signals":"yes"}`:                                            "signals",
		`{"action_prefs":["flowers"]}`:                                 "action_prefs.0",
		`{"action_prefs":"food"}`:                                      "action_prefs",
		`{"action_prefs":["food","gift","time","calm","help","food"]}`: "action_prefs",
	} {
		v := validation.Make(lang.Default(), "en", validation.DecodeBody([]byte(raw)), preferenceRules())
		require.True(t, v.Fails(), raw)
		assert.Contains(t, v.Fields(), field, raw)
	}
	v := validation.Make(lang.Default(), "en", validation.DecodeBody([]byte(
		`{"goals":["quality_time","support"],"signals":{"pain":true,"energy":0},"action_prefs":["help"]}`)), preferenceRules())
	assert.False(t, v.Fails())
}

func TestPreferencesApply_PeriodDays(t *testing.T) {
	assert.False(t, DefaultPreferences().PeriodDays, "off by default")
	on := true
	got, err := DefaultPreferences().Apply(PreferencesPatch{PeriodDays: &on})
	require.NoError(t, err)
	assert.True(t, got.PeriodDays)
	assert.False(t, PreferencesPatch{PeriodDays: &on}.Empty())
	kept, err := got.Apply(PreferencesPatch{Signals: SignalScope{SignalPain: true}})
	require.NoError(t, err)
	assert.True(t, kept.PeriodDays, "an unrelated patch keeps the consent")
	assert.True(t, preferencesOf(store.Companion{SharePeriodDays: true}).PeriodDays)

	p, err := parsePreferences("en", validation.DecodeBody([]byte(`{"period_days":true}`)))
	require.NoError(t, err)
	require.NotNil(t, p.PeriodDays)
	assert.True(t, *p.PeriodDays)
}
