package companionhome

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/companion"
)

func TestDeriveSignals(t *testing.T) {
	assert.Equal(t, DaySignals{}, DeriveSignals(nil), "nothing logged → all false")

	cases := []struct {
		name string
		in   DayEntry
		want DaySignals
	}{
		{"pain location", DayEntry{Category: "pain", Param: "location", Item: "abdomen", Code: "mild"}, DaySignals{Pain: true}},
		{"energy low", DayEntry{Category: "appetite_energy", Param: "energy", Code: "low"}, DaySignals{Energy: true}},
		{"energy very_low", DayEntry{Category: "appetite_energy", Param: "energy", Code: "very_low"}, DaySignals{Energy: true}},
		{"energy medium", DayEntry{Category: "appetite_energy", Param: "energy", Code: "medium"}, DaySignals{}},
		{"sleep poor", DayEntry{Category: "sleep", Param: "quality", Code: "poor"}, DaySignals{Sleep: true}},
		{"sleep fair", DayEntry{Category: "sleep", Param: "quality", Code: "fair"}, DaySignals{}},
		{"short night", DayEntry{Category: "sleep", Param: "duration", Code: "3_6"}, DaySignals{Sleep: true}},
		{"long night", DayEntry{Category: "sleep", Param: "duration", Code: "6_9"}, DaySignals{}},
		{"pp hours short", DayEntry{Category: "sleep", Param: "hours", Num: "4.50"}, DaySignals{Sleep: true}},
		{"pp hours ok", DayEntry{Category: "sleep", Param: "hours", Num: "7.00"}, DaySignals{}},
		{"unrelated", DayEntry{Category: "note", Param: "text"}, DaySignals{}},
		// Mood is never shared (SPEC §1.7): even a low mood row derives nothing.
		{"mood never", DayEntry{Category: "mood", Param: "moods", Item: "sad", Code: "yes"}, DaySignals{}},
		{"meno low_mood never", DayEntry{Category: "symptoms", Param: "general", Item: "low_mood", Code: "severe"}, DaySignals{}},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, DeriveSignals([]DayEntry{tc.in}), tc.name)
	}

	all := DeriveSignals([]DayEntry{
		{Category: "pain", Param: "location", Item: "back", Code: "severe"},
		{Category: "appetite_energy", Param: "energy", Code: "low"},
		{Category: "sleep", Param: "quality", Code: "great"},
		{Category: "sleep", Param: "duration", Code: "0_3"},
	})
	assert.Equal(t, DaySignals{Pain: true, Energy: true, Sleep: true}, all)
}

func TestSignalsJSON_NullUnlessSharedAndShown(t *testing.T) {
	// share_mood on: still no mood key — mood never reaches a member (SPEC §1.7).
	scope := companion.SignalScope{companion.SignalPain: true, companion.SignalMood: true, companion.SignalSleep: true}
	day := DaySignals{Pain: true, Sleep: false}

	b, err := signalsJSON(scope, day, true).MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, `{"pain":true,"energy":null,"sleep":false}`, string(b), "card order, no mood key")

	b, err = signalsJSON(scope, day, false).MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, `{"pain":null,"energy":null,"sleep":null}`, string(b))
}

func TestSignalsShown(t *testing.T) {
	l := companion.Link{Status: companion.StatusActive, Grants: companion.Grants{companion.SectionCycle: companion.LevelView}}
	assert.True(t, signalsShown(l))
	l.Grants = companion.Grants{}
	assert.False(t, signalsShown(l), "cycle decided away (paused / out of mode / not granted) → no signals")
	l = companion.Link{Status: companion.StatusAwaitingOwner, Grants: companion.Grants{companion.SectionCycle: companion.LevelView}}
	assert.False(t, signalsShown(l))
}
