package alerts

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// T-M9-28 (I-02): a `flow` day-log value in pregnancy raises the v1 bleeding alert — heavy flow (or any flow in a
// high-risk pregnancy) the emergency, the rest the warning; no flow, nothing.
func TestForLoggedFlow(t *testing.T) {
	low := Context{Locale: "fa", Week: 10, Profile: lowRisk}
	cases := []struct {
		ctx       Context
		intensity string
		level     enums.AlertLevel
		rank      int
	}{
		{low, "low", enums.AlertLevelWarning, 1},
		{low, "medium", enums.AlertLevelWarning, 1},
		{low, "high", enums.AlertLevelEmergency, 2},
		{low, "very_high", enums.AlertLevelEmergency, 2},
		{Context{Locale: "en", Week: 20, Profile: highRisk}, "low", enums.AlertLevelEmergency, 1},
	}
	for _, c := range cases {
		d, ok := ForLoggedFlow(c.ctx, c.intensity)
		if !assert.True(t, ok, c.intensity) {
			continue
		}
		assert.Equal(t, c.level, d.Level, c.intensity)
		assert.Equal(t, "symptom_based", d.Type)
		assert.Equal(t, c.ctx.Week, d.Week)
		assert.NotEmpty(t, d.Title)
		assert.NotEmpty(t, d.Actions)
		b, err := jsonx.Marshal(d.Trigger, 0)
		assert.NoError(t, err)
		assert.Contains(t, string(b), `"flow":"`+c.intensity+`"`)
		assert.Equal(t, c.rank, FlowRank(c.intensity))
	}
	en, _ := ForLoggedFlow(Context{Locale: "en", Profile: lowRisk}, "medium")
	assert.Equal(t, "Warning: Bleeding", en.Title, "the existing bleeding copy")

	_, ok := ForLoggedFlow(low, "")
	assert.False(t, ok)
	_, ok = ForLoggedFlow(low, "unknown")
	assert.False(t, ok)
	assert.Zero(t, FlowRank(""))
}
