package legacy

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/enums"
)

func TestNeutralPhase(t *testing.T) {
	assert.Equal(t, enums.CyclePhaseFollicular, NeutralPhase(enums.CyclePhaseOvulation, 13, 14))
	assert.Equal(t, enums.CyclePhaseLuteal, NeutralPhase(enums.CyclePhaseOvulation, 14, 14))
	assert.Equal(t, enums.CyclePhaseLuteal, NeutralPhase(enums.CyclePhaseOvulation, 15, 14))
	for _, p := range []enums.CyclePhase{enums.CyclePhaseMenstruation, enums.CyclePhaseFollicular, enums.CyclePhaseLuteal} {
		assert.Equal(t, p, NeutralPhase(p, 14, 14))
	}
}

// CB-TEEN-04b: an ovulation-day calculation loses every fertility / pregnancy-chance string; the
// numbers and the other flags and tips stay.
func TestWithoutFertilityCopy(t *testing.T) {
	c := Calculation{
		Complete: true, Day: 15, Phase: enums.CyclePhaseOvulation, CurrentSubphase: enums.CycleSubphaseOvulationLikely,
		OvulationDay: 15, CycleLength: 28, IsFertileWindow: true, IsPmsWindow: false, FinalProbability: 23.1,
		TextFlags: TextFlags(enums.CyclePhaseOvulation, enums.CycleSubphaseOvulationLikely, true, false, true, 0.231, enums.CycleVariabilityRegular),
		DailyTips: phaseTips(enums.CyclePhaseOvulation, enums.CycleSubphaseOvulationLikely),
	}
	for _, locale := range []string{"fa", "en"} {
		full, err := json.Marshal(c.Localize(locale))
		require.NoError(t, err)
		assert.Contains(t, strings.ToLower(string(full)), map[string]string{"fa": "باروری", "en": "fertile"}[locale])

		out, err := json.Marshal(c.WithoutFertilityCopy().Localize(locale))
		require.NoError(t, err)
		var body struct {
			Flags map[string]string `json:"text_flags"`
			Tips  []map[string]any  `json:"daily_tips"`
			Prob  float64           `json:"final_probability"`
		}
		require.NoError(t, json.Unmarshal(out, &body), string(out))
		assert.NotContains(t, body.Flags, "fertility_status")
		assert.NotContains(t, body.Flags, "probability_message")
		assert.Contains(t, body.Flags, "period_prediction", "other flags stay")
		assert.Equal(t, map[string]string{"fa": "فاز فعلی: " + enums.CyclePhaseLuteal.Label("fa"), "en": "Current phase: " + enums.CyclePhaseLuteal.Label("en")}[locale], body.Flags["phase_info"])
		assert.Len(t, body.Tips, 3, "only the fertility tip goes")
		for _, tip := range body.Tips {
			assert.NotEqual(t, "fertility", tip["type"])
		}
		assert.InDelta(t, 23.1, body.Prob, 0.001, "numbers are data")
		copyJSON, err := json.Marshal([]any{body.Flags, body.Tips}) // the copy only: keys like estimated_ovulation_day are data
		require.NoError(t, err)
		text := strings.ToLower(string(copyJSON))
		for _, w := range []string{"باروری", "تخمک", "بارداری", "fertil", "ovulat", "pregnan", "conceive"} {
			assert.NotContains(t, text, w, locale)
		}
	}
	// The receiver is untouched.
	assert.Len(t, c.TextFlags, 4)
	assert.Len(t, c.DailyTips, 4)
}
