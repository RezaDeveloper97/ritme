package cycletips

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/messages/facts"
	"github.com/ritme/backend-go/internal/messages/rules"
)

func TestDefs_ShapeAndCopy(t *testing.T) {
	assert.Len(t, Defs, 8+7+10)
	seen := map[string]bool{}
	for _, d := range Defs {
		assert.False(t, seen[d.Key], d.Key)
		seen[d.Key] = true
		assert.True(t, rules.LevelAllowed(Level, Kind), d.Key)
		assert.True(t, rules.KindAllowedIn(Slot, Kind), d.Key)
		lo, hi := rules.Band(Kind)
		assert.True(t, d.Prio >= lo && d.Prio <= hi, d.Key)
		for _, code := range Languages() {
			v, ok := d.variant(code)
			require.True(t, ok, "%s/%s", d.Key, code)
			assert.NotEmpty(t, v.Title, "%s/%s", d.Key, code)
			assert.NotEmpty(t, v.Body, "%s/%s", d.Key, code)
			assert.Empty(t, rules.Placeholders(v.Title, v.Body), "%s/%s: no variables in the legacy copy", d.Key, code)
		}
		// every fact is registered, and a fertility fact carries the fertile_window topic (M-06).
		for _, g := range d.Groups {
			for _, c := range g.Items {
				_, ok := facts.Lookup(c.Fact)
				assert.True(t, ok, "%s: fact %s", d.Key, c.Fact)
			}
		}
		var content map[string]any
		js, err := d.ContentJSON("fa")
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal([]byte(js), &content))
		assert.Contains(t, content, "translations", d.Key)
	}
	assert.Equal(t, []string{"en", "fa"}, Languages())
}

func match(d Def, f rules.Facts) bool {
	return rules.MatchRule(&rules.Rule{Groups: d.Groups}, f)
}

func byKey(t *testing.T, key string) Def {
	t.Helper()
	for _, d := range Defs {
		if d.Key == key {
			return d
		}
	}
	t.Fatalf("no def %s", key)
	return Def{}
}

// winner is the matching Def with the highest priority (the slot's single winner).
func winner(f rules.Facts) string {
	best := Def{}
	for _, d := range Defs {
		if match(d, f) && d.Prio > best.Prio {
			best = d
		}
	}
	return best.Key
}

func TestDefs_WhoWins(t *testing.T) {
	day := func(mode, phase string, symptoms ...string) rules.Facts {
		if symptoms == nil {
			symptoms = []string{}
		}
		return rules.Facts{"mode": mode, "main_phase": phase, facts.SymptomsToday: symptoms, "is_pms_window": false,
			"is_fertile_window": phase == "fertile", "is_premium": false}
	}
	// a quiet day: the phase message of her mode; the late luteal days read the luteal copy.
	assert.Equal(t, "tip.cycle.menstrual", winner(day("cycle", "menstrual")))
	assert.Equal(t, "tip.cycle.luteal", winner(day("cycle", "period_expected")))
	assert.Equal(t, "tip.cycle.default", winner(day("cycle", "unknown")))
	assert.Equal(t, "tip.ttc.follicular", winner(day("ttc", "follicular")))
	// one symptom beats the phase; the legacy order decides between two.
	assert.Equal(t, "tip.symptom.low_energy", winner(day("cycle", "luteal", "fatigue")))
	assert.Equal(t, "tip.symptom.heavy_flow", winner(day("cycle", "menstrual", "cramps", "heavy_flow")))
	// a combination beats a single symptom.
	assert.Equal(t, "tip.corr.sleep_mood", winner(day("cycle", "luteal", "poor_sleep", "mood_sad")))
	assert.Equal(t, "tip.corr.dehydration_indicator", winner(day("cycle", "follicular", "headache", "fatigue")))
	// TTC in the fertile window; premium-only insights need Plus.
	assert.Equal(t, "tip.corr.fertile_window_ttc", winner(day("ttc", "fertile")))
	assert.Equal(t, "tip.cycle.fertile", winner(day("cycle", "fertile")))
	free := day("cycle", "luteal", "low_energy")
	assert.Equal(t, "tip.symptom.low_energy", winner(free))
	free["is_premium"] = true
	assert.Equal(t, "tip.corr.luteal_energy_drop", winner(free))
	// other modes get none of these rules.
	assert.Empty(t, winner(day("pregnancy", "unknown", "fatigue")))
	assert.Equal(t, []string{rules.TopicFertileWindow}, byKey(t, "tip.corr.fertile_window_ttc").Topics)
	assert.Equal(t, []string{rules.TopicFertileWindow}, byKey(t, "tip.ttc.fertile").Topics)
}
