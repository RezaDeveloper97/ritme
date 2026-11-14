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

func TestPreferencesValidate(t *testing.T) {
	p := Preferences{}
	require.NoError(t, p.Validate())
	assert.Equal(t, DefaultPreferences(), p, "empty → goal unknown, no signals, no actions")

	p = Preferences{Goal: "romance"}
	require.ErrorIs(t, p.Validate(), ErrInvalidGoal)

	p = Preferences{Signals: SignalScope{"weight": true}}
	require.ErrorIs(t, p.Validate(), ErrInvalidSignal)
}

func TestPreferencesApply_PartialPatch(t *testing.T) {
	base := Preferences{Goal: GoalSupport, Signals: SignalScope{SignalPain: true}, Actions: []ActionPref{ActionGift}}
	g := GoalInfo
	acts := []ActionPref{ActionTime, ActionTime, ActionCalm}

	got, err := base.Apply(PreferencesPatch{Signals: SignalScope{SignalMood: true}})
	require.NoError(t, err)
	assert.Equal(t, GoalSupport, got.Goal, "goal not sent → unchanged")
	assert.True(t, got.Signals.On(SignalPain), "unsent signal keeps its value")
	assert.True(t, got.Signals.On(SignalMood))
	assert.Equal(t, []ActionPref{ActionGift}, got.Actions)
	assert.False(t, base.Signals.On(SignalMood), "Apply never mutates the receiver")

	got, err = base.Apply(PreferencesPatch{Goal: &g, Signals: SignalScope{SignalPain: false}, Actions: &acts})
	require.NoError(t, err)
	assert.Equal(t, GoalInfo, got.Goal)
	assert.False(t, got.Signals.On(SignalPain))
	assert.Equal(t, []ActionPref{ActionTime, ActionCalm}, got.Actions)

	bad := Goal("x")
	_, err = base.Apply(PreferencesPatch{Goal: &bad})
	require.ErrorIs(t, err, ErrInvalidGoal)

	assert.True(t, PreferencesPatch{}.Empty())
}

func TestPreferencesColumnsRoundTrip(t *testing.T) {
	assert.False(t, goalColumn(GoalUnknown).Valid, "unknown is stored as NULL")
	assert.Equal(t, sql.NullString{}, actionsColumn(nil))
	assert.Equal(t, "calm,food", actionsColumn([]ActionPref{ActionCalm, ActionFood}).String)

	c := store.Companion{Goal: goalColumn(GoalLowEnergy), ShareMood: true, ShareSleep: true,
		ActionPrefs: sql.NullString{String: "help,bogus,food,help", Valid: true}}
	p := preferencesOf(c)
	assert.Equal(t, GoalLowEnergy, p.Goal)
	assert.Equal(t, SignalScope{SignalPain: false, SignalMood: true, SignalEnergy: false, SignalSleep: true}, p.Signals)
	assert.Equal(t, []ActionPref{ActionHelp, ActionFood}, p.Actions, "unknown codes dropped, order kept")

	assert.Equal(t, DefaultPreferences().Goal, preferencesOf(store.Companion{Goal: sql.NullString{String: "gone", Valid: true}}).Goal)
}

func TestParsePreferences(t *testing.T) {
	body := validation.DecodeBody([]byte(`{"goal":"support","signals":{"pain":true,"mood":"0","sleep":null},"action_prefs":["gift","calm","gift"]}`))
	p, err := parsePreferences("en", body)
	require.NoError(t, err)
	require.NotNil(t, p.Goal)
	assert.Equal(t, GoalSupport, *p.Goal)
	assert.Equal(t, SignalScope{SignalPain: true, SignalMood: false}, p.Signals, "null = not sent")
	require.NotNil(t, p.Actions)
	assert.Equal(t, []ActionPref{ActionGift, ActionCalm}, *p.Actions)

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
		`{"goal":"romance"}`:           "goal",
		`{"signals":{"pain":"maybe"}}`: "signals.pain",
		`{"signals":"yes"}`:            "signals",
		`{"action_prefs":["flowers"]}`: "action_prefs.0",
		`{"action_prefs":"food"}`:      "action_prefs",
		`{"action_prefs":["food","gift","time","calm","help","food"]}`: "action_prefs",
	} {
		v := validation.Make(lang.Default(), "en", validation.DecodeBody([]byte(raw)), preferenceRules())
		require.True(t, v.Fails(), raw)
		assert.Contains(t, v.Fields(), field, raw)
	}
	v := validation.Make(lang.Default(), "en", validation.DecodeBody([]byte(
		`{"goal":"quality_time","signals":{"pain":true,"energy":0},"action_prefs":["help"]}`)), preferenceRules())
	assert.False(t, v.Fails())
}
